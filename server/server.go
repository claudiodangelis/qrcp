package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"image/jpeg"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/claudiodangelis/qrcp/qr"

	"github.com/claudiodangelis/qrcp/body"
	"github.com/claudiodangelis/qrcp/config"
	"github.com/claudiodangelis/qrcp/util"
	"github.com/claudiodangelis/qrcp/web"
	"gopkg.in/cheggaaa/pb.v1"
)

// Server is the server
type Server struct {
	BaseURL string
	// SendURL is the URL used to send the file
	SendURL string
	// SendFileURL is the direct file download URL (used by the send UI)
	SendFileURL string
	// ReceiveURL is the URL used to receive the file
	ReceiveURL             string
	instance               *http.Server
	body                   body.Body
	outputDir              string
	stopChannel            chan bool
	expectParallelRequests bool
	cfg                    *config.Config
	waitgroup              sync.WaitGroup
	fileServeOnce          sync.Once
	receiveRoute           string
}

// ReceiveTo sets the output directory
func (s *Server) ReceiveTo(dir string) error {
	output, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	fileinfo, err := os.Stat(output)
	if err != nil {
		return err
	}
	if !fileinfo.IsDir() {
		return fmt.Errorf("%s is not a valid directory", output)
	}
	s.outputDir = output
	return nil
}

// Send configures the server to serve a file for download
func (s *Server) Send(p body.Body) {
	s.body = p
	s.expectParallelRequests = true
}

// DisplayQR creates a handler for serving the QR code in the browser
func (s *Server) DisplayQR(url string) {
	const PATH = "/qr"
	qrImg := qr.RenderImage(url)
	http.HandleFunc(PATH, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		if err := jpeg.Encode(w, qrImg, nil); err != nil {
			panic(err)
		}
	})
	openBrowser(s.BaseURL + PATH)
}

// Wait blocks until the transfer completes, then shuts down the server
func (s *Server) Wait() error {
	<-s.stopChannel
	if err := s.instance.Shutdown(context.Background()); err != nil {
		log.Println(err)
	}
	if s.body.DeleteAfterTransfer {
		if err := s.body.Delete(); err != nil {
			panic(err)
		}
	}
	return nil
}

// Shutdown the server
func (s *Server) Shutdown() {
	s.stopChannel <- true
}

func (s *Server) handleSend(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.Header.Get("User-Agent"), "Mozilla") {
		// Terminal browser: serve file directly and signal shutdown
		defer s.triggerShutdownOnce()
		w.Header().Set("Content-Disposition", "attachment; filename=\""+
			s.body.Filename+
			"\"; filename*=UTF-8''"+
			url.QueryEscape(s.body.Filename))
		http.ServeFile(w, r, s.body.Path)
		return
	}
	// Web browser: serve the send UI
	serveTemplate("send", web.Send, w, struct {
		DownloadURL string
		Filename    string
	}{
		DownloadURL: s.SendFileURL,
		Filename:    s.body.Filename,
	})
}

func (s *Server) handleSendFile(w http.ResponseWriter, r *http.Request) {
	defer s.triggerShutdownOnce()
	w.Header().Set("Content-Disposition", "attachment; filename=\""+
		s.body.Filename+
		"\"; filename*=UTF-8''"+
		url.QueryEscape(s.body.Filename))
	http.ServeFile(w, r, s.body.Path)
}

func (s *Server) triggerShutdownOnce() {
	s.fileServeOnce.Do(func() {
		s.waitgroup.Done()
	})
}

func (s *Server) handleReceive(w http.ResponseWriter, r *http.Request) {
	htmlVariables := struct {
		Route string
		File  string
	}{Route: s.receiveRoute}

	switch r.Method {
	case "POST":
		filenames := util.ReadFilenames(s.outputDir)
		reader, err := r.MultipartReader()
		if err != nil {
			fmt.Fprintf(w, "Upload error: %v\n", err)
			log.Printf("Upload error: %v\n", err)
			s.stopChannel <- true
			return
		}
		transferredFiles := []string{}
		progressBar := pb.New64(r.ContentLength)
		progressBar.ShowCounters = false
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if part.FileName() == "" {
				continue
			}
			fileName := getFileName(filepath.Base(part.FileName()), filenames)
			out, err := os.Create(filepath.Join(s.outputDir, fileName))
			if err != nil {
				fmt.Fprintf(w, "Unable to create the file for writing: %s\n", err)
				log.Printf("Unable to create the file for writing: %s\n", err)
				s.stopChannel <- true
				return
			}
			defer out.Close()
			filenames = append(filenames, fileName)
			fmt.Println("Transferring file: ", out.Name())
			progressBar.Prefix(out.Name())
			progressBar.Start()
			buf := make([]byte, 1024)
			for {
				n, err := part.Read(buf)
				if err != nil && err != io.EOF {
					fmt.Fprintf(w, "Unable to write file to disk: %v", err)
					fmt.Printf("Unable to write file to disk: %v", err)
					s.stopChannel <- true
					return
				}
				if n == 0 {
					break
				}
				if _, err := out.Write(buf[:n]); err != nil {
					fmt.Fprintf(w, "Unable to write file to disk: %v", err)
					log.Printf("Unable to write file to disk: %v", err)
					s.stopChannel <- true
					return
				}
				progressBar.Add(n)
			}
			transferredFiles = append(transferredFiles, out.Name())
		}
		progressBar.FinishPrint("File transfer completed")
		htmlVariables.File = strings.Join(transferredFiles, ", ")
		serveTemplate("done", web.Done, w, htmlVariables)
		if !s.cfg.KeepAlive {
			s.stopChannel <- true
		}
	case "GET":
		serveTemplate("upload", web.Upload, w, htmlVariables)
	}
}

// New instance of the server
func New(cfg *config.Config) (*Server, error) {
	app := &Server{
		cfg: cfg,
	}

	bind, err := util.GetInterfaceAddress(cfg.Interface)
	if err != nil {
		return &Server{}, err
	}
	if cfg.Bind != "" {
		bind = cfg.Bind
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", bind, cfg.Port))
	if err != nil {
		return nil, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	host := fmt.Sprintf("%s:%d", bind, port)

	path := cfg.Path
	if path == "" {
		path = util.GetRandomURLPath()
	}

	hostname := fmt.Sprintf("%s:%d", bind, port)
	if bind == "0.0.0.0" && cfg.FQDN == "" {
		fmt.Println("Retrieving the external IP...")
		extIP, err := util.GetExternalIP()
		if err != nil {
			panic(err)
		}
		extIPString := extIP.String()
		fmtstring := "%s:%d"
		if strings.Count(extIPString, ":") >= 2 {
			fmtstring = "[%s]:%d"
		}
		hostname = fmt.Sprintf(fmtstring, extIPString, port)
	}
	if cfg.FQDN != "" {
		hostname = fmt.Sprintf("%s:%d", cfg.FQDN, port)
	}

	protocol := "http"
	if cfg.Secure {
		protocol = "https"
	}
	app.BaseURL = fmt.Sprintf("%s://%s", protocol, hostname)
	app.SendURL = fmt.Sprintf("%s/send/%s", app.BaseURL, path)
	app.SendFileURL = fmt.Sprintf("%s/send/%s/file", app.BaseURL, path)
	app.ReceiveURL = fmt.Sprintf("%s/receive/%s", app.BaseURL, path)
	app.receiveRoute = "/receive/" + path

	httpserver := &http.Server{
		Addr: host,
		TLSConfig: &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
		},
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	app.stopChannel = make(chan bool)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig
		app.stopChannel <- true
	}()

	app.waitgroup.Add(1)
	http.HandleFunc("/send/"+path, app.handleSend)
	http.HandleFunc("/send/"+path+"/file", app.handleSendFile)
	http.HandleFunc("/receive/"+path, app.handleReceive)

	go func() {
		app.waitgroup.Wait()
		if cfg.KeepAlive || !app.expectParallelRequests {
			return
		}
		app.stopChannel <- true
	}()

	go func() {
		netListener := tcpKeepAliveListener{listener.(*net.TCPListener)}
		if cfg.Secure {
			if err := httpserver.ServeTLS(netListener, cfg.TlsCert, cfg.TlsKey); err != http.ErrServerClosed {
				log.Fatalln("error starting the server:", err)
			}
		} else {
			if err := httpserver.Serve(netListener); err != http.ErrServerClosed {
				log.Fatalln("error starting the server", err)
			}
		}
	}()

	app.instance = httpserver
	return app, nil
}

// openBrowser navigates to a url using the default system browser
func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("failed to open browser on platform: %s", runtime.GOOS)
	}
	if err != nil {
		log.Fatal(err)
	}
}
