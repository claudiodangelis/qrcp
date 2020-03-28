package server

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"

	"github.com/claudiodangelis/qr-filetransfer/page"
	"github.com/claudiodangelis/qrcp/logger"
	"github.com/claudiodangelis/qrcp/pages"
	"github.com/claudiodangelis/qrcp/payload"
	"github.com/claudiodangelis/qrcp/util"
	"gopkg.in/cheggaaa/pb.v1"
)

// Server is the server
type Server struct {
	// SendURL is the URL used to send the file
	SendURL string
	// ReceiveURL is the URL used to Receive the file
	ReceiveURL  string
	instance    *http.Server
	p           payload.Payload
	stopchannel chan bool
}

// SetForSend adds a handler for sending the file
func (s *Server) SetForSend(p payload.Payload) error {
	// Add handler
	s.p = p
	return nil
}

// Wait for transfer to be completed
func (s Server) Wait() error {
	<-s.stopchannel
	if err := s.instance.Shutdown(context.Background()); err != nil {
		log.Println(err)
	}
	return nil
}

// New instance of the server
func New(iface string, port int) (*Server, error) {
	logger := logger.New()

	theserver := &Server{}
	// Create the server
	address, err := util.GetInterfaceAddress(iface)
	if err != nil {
		return &Server{}, err
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", address, port))
	if err != nil {
		log.Fatalln(err)
	}
	// TODO: Refactor this
	address = fmt.Sprintf("%s:%d", address, listener.Addr().(*net.TCPAddr).Port)

	randomPath := util.GetRandomURLPath()
	// TODO: Refactor this
	theserver.SendURL = fmt.Sprintf("http://%s/send/%s",
		listener.Addr().String(), randomPath)
	theserver.ReceiveURL = fmt.Sprintf("http://%s/receive/%s",
		listener.Addr().String(), randomPath)

	// Create a server
	s := &http.Server{Addr: address}
	// Create channel to send message to stop server
	theserver.stopchannel = make(chan bool)
	// Create handlers
	// Send handler (sends file to caller)
	http.HandleFunc("/send/"+randomPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", "attachment; filename="+
			theserver.p.Filename)
		http.ServeFile(w, r, theserver.p.Path)
		theserver.stopchannel <- true
	})
	// Upload handler (serves the upload page)
	http.HandleFunc("/receive/"+randomPath, func(w http.ResponseWriter, r *http.Request) {
		// TODO: This can be refactored
		data := struct {
			Route string
			File  string
		}{}
		data.Route = "/receive/" + randomPath
		switch r.Method {
		case "POST":
			// TODO: DO this
			dirToStore := "/tmp"
			filenames := util.ReadFilenames(dirToStore)
			reader, err := r.MultipartReader()
			if err != nil {
				fmt.Fprintf(w, "Upload error: %v\n", err)
				log.Printf("Upload error: %v\n", err)
				theserver.stopchannel <- true
				return
			}

			transferedFiles := []string{}
			progressBar := pb.New64(r.ContentLength)
			progressBar.ShowCounters = false
			if flag.Lookup("quiet").Value.(flag.Getter).Get().(bool) == true {
				progressBar.NotPrint = true
			}

			for {
				part, err := reader.NextPart()

				if err == io.EOF {
					break
				}
				// if part.FileName() is empty, skip this iteration.
				if part.FileName() == "" {
					continue
				}

				// prepare the dst
				fileName := getFileName(part.FileName(), filenames)
				out, err := os.Create(filepath.Join(dirToStore, fileName))
				if err != nil {
					fmt.Fprintf(w, "Unable to create the file for writing: %s\n", err) //output to server
					log.Printf("Unable to create the file for writing: %s\n", err)     //output to console
					theserver.stopchannel <- true                                      // send signal to server to shutdown
					return
				}
				defer out.Close()

				// add name of new file
				filenames = append(filenames, fileName)

				// write the content from POSTed file to the out
				logger.Info("Transferring file: ", out.Name())
				progressBar.Prefix(out.Name())
				progressBar.Start()
				buf := make([]byte, 1024)
				for {
					// read a chunk
					n, err := part.Read(buf)
					if err != nil && err != io.EOF {
						fmt.Fprintf(w, "Unable to write file to disk: %v", err) //output to server
						log.Printf("Unable to write file to disk: %v", err)     //output to console
						theserver.stopchannel <- true                           // send signal to server to shutdown
						return
					}
					if n == 0 {
						break
					}
					// write a chunk
					if _, err := out.Write(buf[:n]); err != nil {
						fmt.Fprintf(w, "Unable to write file to disk: %v", err) //output to server
						log.Printf("Unable to write file to disk: %v", err)     //output to console
						theserver.stopchannel <- true                           // send signal to server to shutdown
						return
					}
					progressBar.Add(n)
				}

				transferedFiles = append(transferedFiles, out.Name())
			}

			progressBar.FinishPrint("File transfer completed")

			data.File = strings.Join(transferedFiles, ", ")
			serveTemplate("done", page.Done, w, data)
		case "GET":
			serveTemplate("upload", pages.Upload, w, data)
		}
		theserver.stopchannel <- true
	})
	// Receive handler (receives file from caller)

	// Gracefully shutdown when an OS signal is received
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig
		theserver.stopchannel <- true
	}()

	// The handler adds and removes from the sync.WaitGroup
	// When the group is zero all requests are completed
	// and the server is shutdown
	// TODO: Refactor this
	var waitgroup sync.WaitGroup
	wg := &waitgroup // little hack to return wg as pointer
	(*wg).Add(1)
	go func() {
		(*wg).Wait()
		// TODO: what is this
		if flag.Lookup("keep-alive").Value.(flag.Getter).Get().(bool) == false {
			theserver.stopchannel <- true
		}
	}()
	fmt.Println("about to serve", theserver.SendURL)
	go func() {
		if err := (s.Serve(tcpKeepAliveListener{listener.(*net.TCPListener)})); err != http.ErrServerClosed {
			log.Fatalln(err)
		}
	}()
	theserver.instance = s
	return theserver, nil
}
