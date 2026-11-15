package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"image"
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
	"gopkg.in/cheggaaa/pb.v1"
)

// Server is the server
type Server struct {
	BaseURL string
	// SendURL is the URL used to send the file
	SendURL string
	// ReceiveURL is the URL used to Receive the file
	ReceiveURL  string
	instance    *http.Server
	body        body.Body
	outputDir   string
	stopChannel chan bool
	// mux routes the requests of this server. Only the route of the current
	// mode is registered, by Send or ReceiveTo.
	mux            *http.ServeMux
	path           string
	sendHandler    http.HandlerFunc
	receiveHandler http.HandlerFunc
	// sendRoute and receiveRoute register each route once, even if Send or
	// ReceiveTo is called again
	sendRoute    sync.Once
	receiveRoute sync.Once
	// mu protects sessionID and activeRequests
	mu sync.Mutex
	// sessionID identifies the browser the file is sent to
	sessionID string
	// activeRequests counts the requests of that browser in progress. The
	// server stops when it drops to zero.
	activeRequests int
	sessionStop    sync.Once
}

// ReceiveTo sets the output directory
func (s *Server) ReceiveTo(dir string) error {
	output, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	// Check if the output dir exists
	fileinfo, err := os.Stat(output)
	if err != nil {
		return err
	}
	if !fileinfo.IsDir() {
		return fmt.Errorf("%s is not a valid directory", output)
	}
	// Received files are named after the existing ones, so the directory
	// must be readable
	if err := checkReadable(output); err != nil {
		return fmt.Errorf("unable to read %s: %w", output, err)
	}
	s.outputDir = output
	s.receiveRoute.Do(func() {
		s.mux.HandleFunc("/receive/"+s.path, s.receiveHandler)
	})
	return nil
}

// checkReadable reports whether the entries of dir can be listed, reading at
// most one of them
func checkReadable(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	if _, err := f.ReadDir(1); err != nil && err != io.EOF {
		return err
	}
	return nil
}

// Send adds a handler for sending the file
func (s *Server) Send(p body.Body) {
	s.body = p
	s.sendRoute.Do(func() {
		s.mux.HandleFunc("/send/"+s.path, s.sendHandler)
	})
}

// startSessionRequest admits a request of the browser the file is sent to.
// The first request starts the session and gets its new ID, to be set as a
// cookie; the following ones must present it and get an empty newID.
// Exactly one request gets a new ID, even when several arrive at once.
func (s *Server) startSessionRequest(cookieValue string) (newID string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sessionID == "" {
		id, err := util.GetSessionID()
		if err != nil {
			return "", err
		}
		s.sessionID = id
		s.activeRequests++
		return id, nil
	}
	if cookieValue != s.sessionID {
		return "", errMismatchingSession
	}
	s.activeRequests++
	return "", nil
}

// endSessionRequest stops the server when the last request of the session
// is completed
func (s *Server) endSessionRequest() {
	s.mu.Lock()
	s.activeRequests--
	done := s.activeRequests == 0
	s.mu.Unlock()
	if done {
		// Like before, the end of the session stops the server once
		s.sessionStop.Do(func() {
			go func() { s.stopChannel <- true }()
		})
	}
}

var errMismatchingSession = errors.New("mismatching cookie")

// cookieName is the name of the cookie that identifies the browser the file
// is sent to
const cookieName = "qrcp"

// DisplayQR creates a handler for serving the QR code in the browser
func (s *Server) DisplayQR(url string) {
	const PATH = "/qr"
	s.mux.HandleFunc(PATH, qrHandler(qr.RenderImage(url)))
	openBrowser(s.BaseURL + PATH)
}

// qrHandler serves the QR code as an image
func qrHandler(img image.Image) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		if err := jpeg.Encode(w, img, nil); err != nil {
			// The browser went away
			log.Printf("Unable to send the QR code: %v", err)
		}
	}
}

// Wait for transfer to be completed, it waits forever if kept awlive
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

// New instance of the server
func New(cfg *config.Config) (*Server, error) {

	app := &Server{mux: http.NewServeMux()}
	// Get the address of the configured interface to bind the server to.
	// If `bind` configuration parameter has been configured, it takes precedence
	bind, err := util.GetInterfaceAddress(cfg.Interface)
	if err != nil {
		return &Server{}, err
	}
	if cfg.Bind != "" {
		bind = cfg.Bind
	}
	// Create a listener. If `port: 0`, a random one is chosen
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", bind, cfg.Port))
	if err != nil {
		return nil, err
	}
	// Set the value of computed port
	port := listener.Addr().(*net.TCPAddr).Port
	// Set the host
	host := fmt.Sprintf("%s:%d", bind, port)
	// Get a random path to use
	path := cfg.Path
	if path == "" {
		path = util.GetRandomURLPath()
	}
	app.path = path
	// Set the hostname
	hostname := fmt.Sprintf("%s:%d", bind, port)
	// Use external IP when using `interface: any`, unless a FQDN is set
	if bind == "0.0.0.0" && cfg.FQDN == "" {
		fmt.Println("Retrieving the external IP...")
		extIP, err := util.GetExternalIP()
		if err != nil {
			panic(err)
		}
		extIPString := extIP.String()
		fmtstring := "%s:%d"
		if strings.Count(extIPString, ":") >= 2 {
			// IPv6 address, wrap it in [] to add a port
			fmtstring = "[%s]:%d"
		}
		hostname = fmt.Sprintf(fmtstring, extIPString, port)
	}
	// Use a fully-qualified domain name if set
	if cfg.FQDN != "" {
		hostname = fmt.Sprintf("%s:%d", cfg.FQDN, port)
	}
	// Set URLs
	protocol := "http"
	if cfg.Secure {
		protocol = "https"
	}
	app.BaseURL = fmt.Sprintf("%s://%s", protocol, hostname)
	app.SendURL = fmt.Sprintf("%s/send/%s",
		app.BaseURL, path)
	app.ReceiveURL = fmt.Sprintf("%s/receive/%s",
		app.BaseURL, path)
	// Create a server
	httpserver := &http.Server{
		Addr:    host,
		Handler: app.mux,
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
	// Create channel to send message to stop server
	app.stopChannel = make(chan bool)
	// Gracefully shutdown when an OS signal is received or when "q" is pressed
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig
		app.stopChannel <- true
	}()
	// Create handlers
	// Send handler (sends file to caller)
	app.sendHandler = func(w http.ResponseWriter, r *http.Request) {
		// Without keep-alive, the file is sent to the first browser that
		// connects, which gets a cookie, and the server stops when that
		// browser's requests (possibly several, for parallel chunks) are done
		if !cfg.KeepAlive && strings.HasPrefix(r.Header.Get("User-Agent"), "Mozilla") {
			var cookieValue string
			if rcookie, err := r.Cookie(cookieName); err == nil {
				cookieValue = rcookie.Value
			}
			newID, err := app.startSessionRequest(cookieValue)
			switch {
			case errors.Is(err, errMismatchingSession):
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			case err != nil:
				// As before, stop the server but still serve this request
				log.Println("Unable to generate session ID", err)
				app.sessionStop.Do(func() {
					go func() { app.stopChannel <- true }()
				})
			default:
				if newID != "" {
					http.SetCookie(w, &http.Cookie{Name: cookieName, Value: newID})
				}
				defer app.endSessionRequest()
			}
		}
		w.Header().Set("Content-Disposition", "attachment; filename=\""+
			app.body.Filename+
			"\"; filename*=UTF-8''"+
			url.QueryEscape(app.body.Filename))
		http.ServeFile(w, r, app.body.Path)
	}
	// Upload handler (serves the upload page)
	app.receiveHandler = func(w http.ResponseWriter, r *http.Request) {
		htmlVariables := struct {
			Route string
			File  string
		}{}
		htmlVariables.Route = "/receive/" + path
		switch r.Method {
		case "POST":
			filenames, err := util.ReadFilenames(app.outputDir)
			if err != nil {
				// E.g. the output directory was removed: answer, and keep
				// serving in case it comes back
				log.Printf("Unable to read the output directory: %v", err)
				http.Error(w, "Unable to read the output directory", http.StatusInternalServerError)
				return
			}
			reader, err := r.MultipartReader()
			if err != nil {
				_, _ = fmt.Fprintf(w, "Upload error: %v\n", err)
				log.Printf("Upload error: %v\n", err)
				app.stopChannel <- true
				return
			}
			transferredFiles := []string{}
			progressBar := pb.New64(r.ContentLength)
			progressBar.ShowCounters = false
			barStarted := false
			for {
				part, err := reader.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					// A malformed upload: answer, and keep serving so that
					// the user can try again
					if barStarted {
						progressBar.Finish()
					}
					log.Printf("Upload error: %v\n", err)
					http.Error(w, fmt.Sprintf("Upload error: %v", err), http.StatusBadRequest)
					return
				}
				// iIf part.FileName() is empty, skip this iteration.
				if part.FileName() == "" {
					continue
				}
				// Prepare the destination
				fileName := getFileName(filepath.Base(part.FileName()), filenames)
				out, err := os.Create(filepath.Join(app.outputDir, fileName))
				if err != nil {
					// Output to server
					_, _ = fmt.Fprintf(w, "Unable to create the file for writing: %s\n", err)
					// Output to console
					log.Printf("Unable to create the file for writing: %s\n", err)
					// Send signal to server to shutdown
					app.stopChannel <- true
					return
				}
				defer func() { _ = out.Close() }()
				// Add name of new file
				filenames = append(filenames, fileName)
				// Write the content from POSTed file to the out
				fmt.Println("Transferring file: ", out.Name())
				progressBar.Prefix(out.Name())
				progressBar.Start()
				barStarted = true
				buf := make([]byte, 1024)
				for {
					// Read a chunk
					n, err := part.Read(buf)
					if err != nil && err != io.EOF {
						// Output to server
						_, _ = fmt.Fprintf(w, "Unable to write file to disk: %v", err)
						// Output to console
						fmt.Printf("Unable to write file to disk: %v", err)
						// Send signal to server to shutdown
						app.stopChannel <- true
						return
					}
					if n == 0 {
						break
					}
					// Write a chunk
					if _, err := out.Write(buf[:n]); err != nil {
						// Output to server
						_, _ = fmt.Fprintf(w, "Unable to write file to disk: %v", err)
						// Output to console
						log.Printf("Unable to write file to disk: %v", err)
						// Send signal to server to shutdown
						app.stopChannel <- true
						return
					}
					progressBar.Add(n)
				}
				transferredFiles = append(transferredFiles, out.Name())
			}
			if barStarted {
				progressBar.FinishPrint("File transfer completed")
			}
			// Set the value of the variable to the actually transferred files
			htmlVariables.File = strings.Join(transferredFiles, ", ")
			serveTemplate(donePage, w, htmlVariables)
			if !cfg.KeepAlive {
				app.stopChannel <- true
			}
		case "GET":
			serveTemplate(uploadPage, w, htmlVariables)
		}
	}
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
