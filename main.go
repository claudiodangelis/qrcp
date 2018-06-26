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
var externalFlag = flag.Bool("external", false, "if set true, will use public ip address, default is false")
var certFlag = flag.Bool("cert", false, "if set true, will start a qrcode for downloading the certification file for https, default is false")
var address string
var config Config
var content Content

func main() {
	flag.Parse()
	config = LoadConfig()
	if *forceFlag == true {
		config.Delete()
		config = LoadConfig()
	}

	// Check how many arguments are passed
	if len(flag.Args()) == 0 && !*certFlag {
		log.Fatalln("At least one argument is required")
	}

	// Get addresses
	var err error
	if *externalFlag || *certFlag {
		if address, err = GetPublicIP(); err != nil {
			log.Fatalln(err)
		}
		if *certFlag {
			// generate cert file
			_, certStatsErr := os.Stat(certPath)
			_, keyStatsErr := os.Stat(certPath)
			if certStatsErr != nil || keyStatsErr != nil {
				if err := Generate(certPath, keyPath); err != nil {
					log.Fatalln(err)
				}
			}
			serveCert()
			return
		}
	} else {
		if address, err = getAddress(&config); err != nil {
			log.Fatalln(err)
		}
	}

	content, err = getContent(flag.Args())
	if err != nil {
		log.Fatalln(err)
	}

	if *externalFlag {
		serveHTTPS()
	} else {
		serveHTTP()
	}
}

func serveHTTP() {
	// Generate the QR code
	fmt.Println("Scan the following QR to start the download.")
	qrterminal.GenerateHalfBlock(
		fmt.Sprintf("http://%s:%d/", address, *portFlag),
		qrterminal.L,
		os.Stdout,
	)

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
	log.Fatalln(
		http.ListenAndServe(
			fmt.Sprintf(":%d", *portFlag),
			nil,
		),
	)
}

func serveCert() {
	fmt.Println("Scan the following QR to download the certification file")
	qrterminal.GenerateHalfBlock(
		fmt.Sprintf("http://%s:%d/cert", address, *portFlag),
		qrterminal.L,
		os.Stdout,
	)

	// Define a cert handler for cert download
	http.HandleFunc("/cert", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition",
			"attachment; filename="+content.Name())

		w.Header().Set("Content-Type", "application/x-x509-ca-cert")
		http.ServeFile(w, r, certPath)
		if err := config.Update(); err != nil {
			log.Println("Unable to update configuration", err)
		}
	})

	// Start a new server bound to the chosen address on a 9527 or specified port
	log.Fatalln(
		http.ListenAndServe(
			fmt.Sprintf(":%d", *portFlag),
			nil,
		),
	)
}

func serveHTTPS() {
	fmt.Println("Scan the following QR to start the download.")
	qrterminal.GenerateHalfBlock(
		fmt.Sprintf("https://%s:%d/", address, *portFlag),
		qrterminal.L,
		os.Stdout,
	)

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
