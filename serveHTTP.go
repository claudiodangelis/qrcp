package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
)

func ServeFileAsHTTPServer(address string, port int, content Content) (srv *http.Server, listener net.Listener, generatedAddress string) {
	// Get a TCP Listener bound to a random port, or the user specificed port
	// listener, err := net.Listen("tcp", address+port)
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", address, port))
	if err != nil {
		log.Fatalln(err)
	}
	address = fmt.Sprintf("%s:%d", address, listener.Addr().(*net.TCPAddr).Port)

	randomPath := getRandomURLPath()

	generatedAddress = fmt.Sprintf("http://%s/%s", listener.Addr().String(), randomPath)

	// Create a server
	srv = &http.Server{Addr: address}

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

	return
}
