package main

import (
	"flag"
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
var quietFlag = flag.Bool("quiet", false, "ignores non critical output")
var portFlag = flag.Int("port", 0, "port to bind the server to")

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

	if *portFlag > 0 {
		config.Port = *portFlag
	}

	srv, listener, generatedAddress := ServeFileAsHTTPServer(address, config.Port, content)

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
