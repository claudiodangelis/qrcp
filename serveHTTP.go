package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
)

// returns http server, tcp listner, address of server, route, and channel used for gracefull shutdown
func setupHTTPServer(config Config) (srv *http.Server, listener net.Listener, generatedAddress, route string, stop chan bool, wg *sync.WaitGroup) {
	// Get address
	address, err := getAddress(&config)
	if err != nil {
		log.Fatalln(err)
	}
	listener, err = net.Listen("tcp", fmt.Sprintf("%s:%d", address, config.Port))
	if err != nil {
		log.Fatalln(err)
	}
	address = fmt.Sprintf("%s:%d", address, listener.Addr().(*net.TCPAddr).Port)

	randomPath := getRandomURLPath()

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
	signal.Notify(sig)
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
		stop <- true
	}()
	return
}

func serveFilesHTTP(generatedAddress, route string, content Content, wg *sync.WaitGroup, stop chan bool) {
	info("Scan the following QR to start the download.")
	info("Make sure that your smartphone is connected to the same WiFi network as this computer.")
	info("Size of transfer:", humanReadableSizeOf(content.Path))
	info("Your generated address is", generatedAddress)

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
}

func receiveFilesHTTP(generatedAddress, route, dirToStore string, wg *sync.WaitGroup, stop chan bool) {
	info("Scan the following QR to start the upload.")
	info("Make sure that your smartphone is connected to the same WiFi network as this computer.")
	info("Your generated address is", generatedAddress)

	http.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			fmt.Fprintf(w, `<html>
<head>
  <title>qr-filetransfer</title>
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body>
<h2>Upload a file</h2>
<form action="`+route+`" method="post" enctype="multipart/form-data">
  <label for="file">Filename:</label>
  <input type="file" name="files" id="files" multiple>
  <br>
  <input type="submit" name="submit" value="Submit">
</form>
</body>
</html>`)
		}
		if r.Method == "POST" {
			defer wg.Done()

			// make sure dirToStore is exist
			filesInfo, err := ioutil.ReadDir(dirToStore)
			if err != nil && os.IsNotExist(err) {
				// if not exist try to create directories in path to dirToStore
				if err := os.MkdirAll(dirToStore, os.ModePerm); err != nil {
					fmt.Fprintf(w, "Unable to create specifyed dir: %s\n", err) //output to server
					log.Printf("Unable to create specifyed dir: %v\n", err)     //output to console
					stop <- true                                                // send signal to server to shutdown
					return
				}
			}

			// create array of names of files which are stored in dirToStore
			// used later to set valid name for received files
			fileNamesInTargetDir := make([]string, len(filesInfo))
			for _, fi := range filesInfo {
				fileNamesInTargetDir = append(fileNamesInTargetDir, fi.Name())
			}

			r.ParseMultipartForm(32 << 20) // 32MB is the default used by http.Request.FormFile()
			fileHeaders := r.MultipartForm.File["files"]
			for _, fileHeader := range fileHeaders {
				// open provided file
				file, err := fileHeader.Open()
				defer file.Close()
				if err != nil {
					fmt.Fprintf(w, "Unable to read provided file: %v\n", err) //output to server
					log.Printf("Unable to read provided file: %v\n", err)     //output to console
					stop <- true                                              // send signal to server to shutdown
					return
				}

				// seting name of new file
				// if name isn't taken leave it unchanged
				// else change name to format "name(number).ext"
				newFilename := fileHeader.Filename
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

				// try to create output file
				out, err := os.Create(filepath.Join(dirToStore, newFilename))
				if err != nil {
					fmt.Fprintf(w, "Unable to create the file for writing: %s\n", err) //output to server
					log.Printf("Unable to create the file for writing: %s\n", err)     //output to console
					stop <- true                                                       // send signal to server to shutdown
					return
				}
				defer out.Close()

				// add name of new file
				fileNamesInTargetDir = append(fileNamesInTargetDir, newFilename)

				// write the content from POSTed file to the out
				if _, err = io.Copy(out, file); err != nil {
					fmt.Fprintf(w, "Unable to write file to disk: %v", err) //output to server
					log.Printf("Unable to write file to disk: %v", err)     //output to console
					stop <- true                                            // send signal to server to shutdown
					return
				}

				fmt.Fprintf(w, "File uploaded successfully: %s\n", out.Name()) //ouput to server
				fmt.Printf("File uploaded successfully: %s\n", out.Name())     //output to console
			}
		}
	})
}
