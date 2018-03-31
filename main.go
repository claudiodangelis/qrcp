package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"

	"github.com/mattn/go-colorable"
	"github.com/mdp/qrterminal"
)

var zipFlag = flag.Bool("zip", false, "zip the contents to be transfered")
var forceFlag = flag.Bool("force", false, "ignore saved configuration")
var debugFlag = flag.Bool("debug", false, "increase verbosity")

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

	// Get addresses
	address, err := getAddress(&config)
	if err != nil {
		log.Fatalln(err)
	}

	// Create a net.Listener bound to the choosen address on a random port
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:0", address))
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	content, err := getContent(flag.Args())
	if err != nil {
		log.Fatalln(err)
	}

	// Generate the QR code
	fmt.Println("Scan the following QR to start the download.")
	fmt.Println("Make sure that your smartphone is connected to the same WiFi network as this computer.")
	qrcfg := qrterminal.Config{
		Level:          qrterminal.L,
		HalfBlocks:     true,
		Writer:         os.Stdout,
		BlackChar:      qrterminal.BLACK_BLACK,
		WhiteBlackChar: qrterminal.WHITE_BLACK,
		WhiteChar:      qrterminal.WHITE_WHITE,
		BlackWhiteChar: qrterminal.BLACK_WHITE,
	}
	if runtime.GOOS == "windows" {
		qrcfg.HalfBlocks = false
		qrcfg.Writer = colorable.NewColorableStdout()
		qrcfg.BlackChar = qrterminal.BLACK
		qrcfg.WhiteChar = qrterminal.WHITE
	}

	qrterminal.GenerateWithConfig(fmt.Sprintf("http://%s", listener.Addr().String()), qrcfg)

	// Define a default handler for the requests
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition",
			"attachment; filename="+content.Name())

		http.ServeFile(w, r, content.Path)
		if content.ShouldBeDeleted {
			if err := content.Delete(); err != nil {
				log.Println("Unable to delete the content from disk", err)
			}
		}
		if err := config.Update(); err != nil {
			log.Println("Unable to update configuration", err)
		}
		os.Exit(0)
	})
	// Start a new server using the listener bound to the choosen address on a random port
	log.Fatalln(http.Serve(listener, nil))

}
