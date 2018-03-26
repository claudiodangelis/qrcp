package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"path/filepath"

	"github.com/mdp/qrterminal"
)

var zipFlag = flag.Bool("zip", false, "zip the contents to be transfered")
var forceFlag = flag.Bool("force", false, "ignore saved configuration")
var debugFlag = flag.Bool("debug", false, "increase verbosity")
var portFlag = flag.Int("port", 9527, "specify port, default is a 9527")
var remoteFlag = flag.Bool("remote", false, "if set true, will use public ip address, default is false")

// TODO this feature is not done
var sshPortFlag = flag.Int("ssh", 22, "specify ssh port, default is 22, this is for generate scp command")

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

	content, err := getContent(flag.Args())
	if err != nil {
		log.Fatalln(err)
	}

	// Get absolute file path for generating scp command
	dir, err := filepath.Abs(content.Name())
	if err != nil {
		fmt.Println(err)
		return
	}

	// Generate the QR code
	fmt.Println("Scan the following QR to start the download.")
	fmt.Printf("scp %s:%s ./\n", address, dir)
	qrterminal.GenerateHalfBlock(fmt.Sprintf("http://%s:%d", address, *portFlag),
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
	log.Fatalln(http.ListenAndServe(fmt.Sprintf(":%d", *portFlag), nil))
}
