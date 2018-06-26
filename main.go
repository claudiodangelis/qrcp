package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/mdp/qrterminal"
)

const (
	// TODO: windows may require a different path
	// https cert path
	certPath = "/var/tmp/qr-filetransfer.cert"
	// https key path
	keyPath = "/var/tmp/qr-filetransfer.key"
)

var zipFlag = flag.Bool("zip", false, "zip the contents to be transfered")
var forceFlag = flag.Bool("force", false, "ignore saved configuration")
var debugFlag = flag.Bool("debug", false, "increase verbosity")
var portFlag = flag.Int("port", 9527, "specify port, default is a 9527")
var remoteFlag = flag.Bool("remote", false, "if set true, will use public ip address, default is false")

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
	var address string
	var err error
	if *remoteFlag {
		if address, err = GetPublicIP(); err != nil {
			log.Fatalln(err)
		}
	} else {
		if address, err = getAddress(&config); err != nil {
			log.Fatalln(err)
		}
	}

	// generate cert file
	if err := Generate(certPath, keyPath); err != nil {
		log.Fatalln(err)
	}

	content, err := getContent(flag.Args())
	if err != nil {
		log.Fatalln(err)
	}

	// Generate the QR code
	fmt.Println("Scan the following QR to start the download.")
	protocol := "http"
	if *remoteFlag {
		protocol += "s"
	}
	qrterminal.GenerateHalfBlock(fmt.Sprintf("%s://%s:%d", protocol, address, *portFlag),
		qrterminal.L, os.Stdout)

	// Define a default handler for the requests
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition",
			"attachment; filename="+content.Name())

		w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
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
	// Start a new server bound to the chosen address on a 9527 or specified port
	log.Fatalln(
		http.ListenAndServeTLS(
			fmt.Sprintf(":%d", *portFlag),
			certPath,
			keyPath,
			nil,
		),
	)
}
