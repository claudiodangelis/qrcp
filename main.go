package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/mattn/go-colorable"
	"github.com/mdp/qrterminal"
)

var zipFlag = flag.Bool("zip", false, "zip the contents to be transfered")
var forceFlag = flag.Bool("force", false, "ignore saved configuration")
var debugFlag = flag.Bool("debug", false, "increase verbosity")
var quietFlag = flag.Bool("quiet", false, "ignores non critical output")

func main() {
	flag.Parse()
	config := LoadConfig()
	if *forceFlag == true {
		config.Delete()
		config = LoadConfig()
	}

	// Check how many arguments are passed
	if len(flag.Args()) == 0 {
		log.Fatalln("At least one argument is required")
	}

	// Get Content
	content, err := getContent(flag.Args())
	if err != nil {
		log.Fatalln(err)
	}

	// Get address
	address, err := getAddress(&config)
	if err != nil {
		log.Fatalln(err)
	}

	port := ":0"
	if config.Port > 0 {
		port = ":" + strconv.FormatInt(int64(config.Port), 10)
	}

	// Get a TCP Listener bound to a random port, or the user specificed port
	listener, err := net.Listen("tcp", address+port)
	if err != nil {
		log.Fatalln(err)
	}
	address = fmt.Sprintf("%s:%d", address, listener.Addr().(*net.TCPAddr).Port)

	randomPath := getRandomURLPath()

	generatedAddress := fmt.Sprintf("http://%s/%s", listener.Addr().String(), randomPath)

	// Generate the QR code
	info("Scan the following QR to start the download.")
	info("Make sure that your smartphone is connected to the same WiFi network as this computer.")
	info("Your generated address is", generatedAddress)

	qrConfig := qrterminal.Config{
		HalfBlocks:     true,
		Level:          qrterminal.L,
		Writer:         os.Stdout,
		BlackWhiteChar: "\u001b[37m\u001b[40m\u2584\u001b[0m",
		BlackChar:      "\u001b[30m\u001b[40m\u2588\u001b[0m",
		WhiteBlackChar: "\u001b[30m\u001b[47m\u2585\u001b[0m",
		WhiteChar:      "\u001b[37m\u001b[47m\u2588\u001b[0m",
	}
	if runtime.GOOS == "windows" {
		qrConfig.HalfBlocks = false
		qrConfig.Writer = colorable.NewColorableStdout()
		qrConfig.BlackChar = qrterminal.BLACK
		qrConfig.WhiteChar = qrterminal.WHITE
	}

	qrterminal.GenerateWithConfig(generatedAddress, qrConfig)

	// Create a server
	srv := &http.Server{Addr: address}

	// Create channel to send message to stop server
	stop := make(chan bool)

	// Wait for stop and then shutdown the server,
	go func() {
		<-stop
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Println(err)
		}
	}()

	// Gracefully shutdown when an OS signal is received
	sig := make(chan os.Signal, 1)
	signal.Notify(sig)
	go func() {
		<-sig
		stop <- true
	}()

	// The handler adds and removes from the sync.WaitGroup
	// When the group is zero all requests are completed
	// and the server is shutdown
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		wg.Wait()
		stop <- true
	}()

	// Create cookie used to verify request is coming from first client to connect
	cookie := http.Cookie{Name: "qr-filetransfer", Value: ""}

	var initCookie sync.Once

	// Define a default handler for the requests
	route := fmt.Sprintf("/%s", randomPath)
	http.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		// If the cookie's value is empty this is the first connection
		// and the initialize the cookie.
		// Wrapped in a sync.Once to avoid potential race conditions
		if cookie.Value == "" {
			if !strings.HasPrefix(r.Header.Get("User-Agent"), "Mozilla") {
				http.Error(w, "", http.StatusOK)
				return
			}
			initCookie.Do(func() {
				value, err := getSessionID()
				if err != nil {
					log.Println("Unable to generate session ID", err)
					stop <- true
				}
				cookie.Value = value
				http.SetCookie(w, &cookie)
			})
		} else {
			// Check for the expected cookie and value
			// If it is missing or doesn't match
			// return a 404 status
			rcookie, err := r.Cookie(cookie.Name)
			if err != nil || rcookie.Value != cookie.Value {
				http.Error(w, "", http.StatusNotFound)
				return
			}
			// If the cookie exits and matches
			// this is an aadditional request.
			// Increment the waitgroup
			wg.Add(1)
		}

		defer wg.Done()
		w.Header().Set("Content-Disposition",
			"attachment; filename="+content.Name())
		http.ServeFile(w, r, content.Path)
	})

	// Enable TCP keepalives on the listener and start serving requests
	if err := (srv.Serve(tcpKeepAliveListener{listener.(*net.TCPListener)})); err != http.ErrServerClosed {
		log.Fatalln(err)
	}

	if content.ShouldBeDeleted {
		if err := content.Delete(); err != nil {
			log.Println("Unable to delete the content from disk", err)
		}
	}
	if err := config.Update(); err != nil {
		log.Println("Unable to update configuration", err)
	}

}
