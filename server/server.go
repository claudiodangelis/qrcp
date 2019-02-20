package server

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"

	"github.com/claudiodangelis/qr-filetransfer/config"
	"github.com/claudiodangelis/qr-filetransfer/content"
	l "github.com/claudiodangelis/qr-filetransfer/log"
	"github.com/claudiodangelis/qr-filetransfer/util"
)

// New returns http server, tcp listner, address of server, route, and channel used for graceful shutdown
func New(cfg config.Config) (srv *http.Server, listener net.Listener, generatedAddress, route string, stop chan bool, wg *sync.WaitGroup) {
	// Get address
	address, err := util.GetAddress(&cfg)
	if err != nil {
		log.Fatalln(err)
	}
	listener, err = net.Listen("tcp", fmt.Sprintf("%s:%d", address, cfg.Port))
	if err != nil {
		log.Fatalln(err)
	}
	address = fmt.Sprintf("%s:%d", address, listener.Addr().(*net.TCPAddr).Port)

	randomPath := util.GetRandomURLPath()

	generatedAddress = fmt.Sprintf("http://%s/%s", listener.Addr().String(), randomPath)

	// Create a server
	srv = &http.Server{Addr: address}

	// Define a default handler for the requests
	route = fmt.Sprintf("/%s", randomPath)
	// Create channel to send message to stop server
	stop = make(chan bool)

	// Wait for stop and then shutdown the server,
	go func() {
		<-stop
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Println(err)
		}
	}()

	// Gracefully shutdown when an OS signal is received
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig
		stop <- true
	}()

	// The handler adds and removes from the sync.WaitGroup
	// When the group is zero all requests are completed
	// and the server is shutdown
	var waitgroup sync.WaitGroup
	wg = &waitgroup // little hack to return wg as pointer
	(*wg).Add(1)
	go func() {
		(*wg).Wait()
		if flag.Lookup("keep-alive").Value.(flag.Getter).Get().(bool) == false {
			stop <- true
		}
	}()
	return
}

// Serve serves files
func Serve(generatedAddress, route string, content content.Content, wg *sync.WaitGroup, stop chan bool) {
	logger := l.New()
	logger.Info("Scan the following QR to start the download.")
	logger.Info("Make sure that your smartphone is connected to the same WiFi network as this computer.")
	logger.Info("Size of transfer:", util.HumanReadableSizeOf(content.Path))
	logger.Info("Your generated address is", generatedAddress)

	// Create cookie used to verify request is coming from first client to connect
	cookie := http.Cookie{Name: "qr-filetransfer", Value: ""}

	var initCookie sync.Once

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
				value, err := util.GetSessionID()
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
}

//getFileName generates a file name based on the existing files in the directory
// if name isn't taken leave it unchanged
// else change name to format "name(number).ext"
func getFileName(newFilename string, fileNamesInTargetDir []string) string {
	fileExt := filepath.Ext(newFilename)
	fileName := strings.TrimSuffix(newFilename, fileExt)
	number := 1
	i := 0
	for i < len(fileNamesInTargetDir) {
		if newFilename == fileNamesInTargetDir[i] {
			newFilename = fmt.Sprintf("%s(%v)%s", fileName, number, fileExt)
			number++
			i = 0 // start search again
		}
		i++
	}
	return newFilename
}
