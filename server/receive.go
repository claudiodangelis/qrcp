package server

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	l "github.com/claudiodangelis/qr-filetransfer/log"
	"github.com/claudiodangelis/qr-filetransfer/page"
	"github.com/claudiodangelis/qr-filetransfer/util"
	pb "gopkg.in/cheggaaa/pb.v1"
)

// Receive receives files
func Receive(generatedAddress, route, dirToStore string, wg *sync.WaitGroup, stop chan bool) {
	logger := l.New()
	logger.Info("Scan the following QR to start the upload.")
	logger.Info("Make sure that your smartphone is connected to the same WiFi network as this computer.")
	logger.Info("Your generated address is", generatedAddress)

	http.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Route string
			File  string
		}{}
		data.Route = route
		if r.Method == "GET" {
			serveTemplate("upload", page.Upload, w, data)
		}
		if r.Method == "POST" {
			defer wg.Done()

			// make sure dirToStore is exist
			if ok, err := util.EnsureDirExists(dirToStore); !ok {
				fmt.Fprintf(w, "Unable to create specified dir: %s\n", err) //output to server
				log.Printf("Unable to create specified dir: %v\n", err)     //output to console
				stop <- true                                                // send signal to server to shutdown
				return
			}
			filenames := util.ReadFilenames(dirToStore)

			reader, err := r.MultipartReader()
			if err != nil {
				fmt.Fprintf(w, "Upload error: %v\n", err)
				log.Printf("Upload error: %v\n", err)
				stop <- true
				return
			}

			transferedFiles := []string{}
			logger.Info("Transferring files...")
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
					stop <- true                                                       // send signal to server to shutdown
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
						stop <- true                                            // send signal to server to shutdown
						return
					}
					if n == 0 {
						break
					}
					// write a chunk
					if _, err := out.Write(buf[:n]); err != nil {
						fmt.Fprintf(w, "Unable to write file to disk: %v", err) //output to server
						log.Printf("Unable to write file to disk: %v", err)     //output to console
						stop <- true                                            // send signal to server to shutdown
						return
					}
					progressBar.Add(n)
				}

				transferedFiles = append(transferedFiles, out.Name())
			}

			progressBar.FinishPrint("File transfer completed")

			data.File = strings.Join(transferedFiles, ", ")
			serveTemplate("done", page.Done, w, data)
		}
	})
}
