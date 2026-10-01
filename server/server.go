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
	if isCLIClient(r.Header.Get("User-Agent")) {
		// Command-line client: serve the file directly, no UI
		s.handleSendFile(w, r)
		return
	}
	// Anything else (browsers, link previewers, QR apps) gets the send UI,
	// which does not count as a transfer
	serveTemplate("send", web.Send, w, struct {
		DownloadURL string
		Filename    string
	}{
		DownloadURL: s.SendFileURL,
		Filename:    s.body.Filename,
	})
}

func (s *Server) handleSendFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Disposition", "attachment; filename=\""+
		s.body.Filename+
		"\"; filename*=UTF-8''"+
		url.QueryEscape(s.body.Filename))
	cw := &countingResponseWriter{ResponseWriter: w}
	http.ServeFile(cw, r, s.body.Path)
	// HEAD probes, partial ranges and aborted downloads (e.g. a mobile browser
	// handing the download over to its download manager) must not stop the
	// server: only a response that delivered the end of the file does
	if s.deliveredLastByte(r, cw) {
		s.triggerShutdownOnce()
	}
}

// deliveredLastByte reports whether the response fully delivered a body
// ending with the last byte of the file being sent
func (s *Server) deliveredLastByte(r *http.Request, cw *countingResponseWriter) bool {
	if r.Method == http.MethodHead {
		return false
	}
	info, err := os.Stat(s.body.Path)
	if err != nil {
		return false
	}
	size := info.Size()
	switch cw.status {
	case http.StatusOK:
		return cw.written == size
	case http.StatusPartialContent:
		var start, end, total int64
		if _, err := fmt.Sscanf(cw.Header().Get("Content-Range"),
			"bytes %d-%d/%d", &start, &end, &total); err != nil {
			return false
		}
		return end == size-1 && cw.written == end-start+1
	}
	return false
}

func (s *Server) triggerShutdownOnce() {
	s.fileServeOnce.Do(func() {
		s.waitgroup.Done()
	})
}

// countingResponseWriter records the status code and the number of body
// bytes written to the client
type countingResponseWriter struct {
	http.ResponseWriter
	status  int
	written int64
}

func (cw *countingResponseWriter) WriteHeader(status int) {
	if cw.status == 0 {
		cw.status = status
	}
	cw.ResponseWriter.WriteHeader(status)
}

func (cw *countingResponseWriter) Write(b []byte) (int, error) {
	if cw.status == 0 {
		cw.status = http.StatusOK
	}
	n, err := cw.ResponseWriter.Write(b)
	cw.written += int64(n)
	return n, err
}

// cliUserAgents are lowercase User-Agent prefixes of command-line clients,
// which receive the file directly instead of the send UI
var cliUserAgents = []string{
	"curl/", "wget/", "httpie/", "aria2/", "lynx/", "w3m/", "links",
	"elinks/", "go-http-client/", "python-requests/", "python-urllib/",
}

// isCLIClient reports whether the User-Agent belongs to a command-line client
func isCLIClient(userAgent string) bool {
	ua := strings.ToLower(strings.TrimSpace(userAgent))
	if ua == "" || strings.Contains(ua, "powershell/") {
		return true
	}
	for _, prefix := range cliUserAgents {
		if strings.HasPrefix(ua, prefix) {
			return true
		}
	}
	return false
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
			http.Error(w, fmt.Sprintf("Upload error: %v", err), http.StatusBadRequest)
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
			if err != nil {
				http.Error(w, fmt.Sprintf("Upload error: %v", err), http.StatusBadRequest)
				log.Printf("Upload error: %v\n", err)
				s.stopChannel <- true
				return
			}
			if part.FileName() == "" {
				continue
			}
			fileName := getFileName(filepath.Base(part.FileName()), filenames)
			out, err := os.Create(filepath.Join(s.outputDir, fileName))
			if err != nil {
				http.Error(w, fmt.Sprintf("Unable to create the file for writing: %s", err), http.StatusInternalServerError)
				log.Printf("Unable to create the file for writing: %s\n", err)
				s.stopChannel <- true
				return
			}
			defer func() { _ = out.Close() }()
			filenames = append(filenames, fileName)
			fmt.Println("Transferring file: ", out.Name())
			progressBar.Prefix(out.Name())
			progressBar.Start()
			buf := make([]byte, 1024)
			for {
				n, err := part.Read(buf)
				if err != nil && err != io.EOF {
					http.Error(w, fmt.Sprintf("Upload interrupted: %v", err), http.StatusBadRequest)
					fmt.Printf("Unable to write file to disk: %v", err)
					s.stopChannel <- true
					return
				}
				if n == 0 {
					break
				}
				if _, err := out.Write(buf[:n]); err != nil {
					http.Error(w, fmt.Sprintf("Unable to write file to disk: %v", err), http.StatusInternalServerError)
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
