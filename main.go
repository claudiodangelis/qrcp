package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/mdp/qrterminal"
)

var zipFlag = flag.Bool("zip", false, "zip the contents to be transfered")
var debugFlag = flag.Bool("debug", false, "increase verbosity")

func main() {
	flag.Parse()

	// Check how many arguments are passed
	if len(flag.Args()) == 0 {
		log.Fatalln("At least one argument is required")
	}

	// Get addresses
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatalln(err)
	}
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	conn.Close()
	address := strings.Split(localAddr.String(), ":")[0]

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
	qrterminal.GenerateHalfBlock(fmt.Sprintf("http://%s", listener.Addr().String()),
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
		os.Exit(0)
	})
	// Start a new server using the listener bound to the choosen address on a random port
	log.Fatalln(http.Serve(listener, nil))

}
