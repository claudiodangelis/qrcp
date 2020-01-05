package server

import (
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/claudiodangelis/qrcp/config"
	"github.com/claudiodangelis/qrcp/payload"
	"github.com/claudiodangelis/qrcp/util"
)

// Server is the server
type Server struct {
	wg sync.WaitGroup
	// SendURL is the URL used to send the file
	SendURL string
	// ReceiveURL is the URL used to receive the file(s)
	ReceiveURL string
}

// Wait for the transfer to be completed
func (s *Server) Wait() {
	s.wg.Wait()
}

// Start a new instance of the server in background
func Start(cfg *config.Config, payload *payload.Payload) (*Server, error) {
	s := &Server{}
	// Create listener
	cfg.Port = 8080
	address, err := util.AddressByInterfaceName(cfg.Interface)
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", address, cfg.Port))
	if err != nil {
		return nil, err
	}
	s.wg = sync.WaitGroup{}
	s.wg.Add(1)
	host := listener.Addr().String()
	// TODO: Not entirely sure I want the send/receive prefixes
	sendpath := "/send" + util.GetRandomURLPath()
	receivepath := "/receive" + util.GetRandomURLPath()
	// Handlers
	// TODO: Handle concurrency
	http.HandleFunc(sendpath, func(w http.ResponseWriter, r *http.Request) {
		// defer s.wg.Done()
		w.Header().Set("Content-Disposition",
			fmt.Sprintf("attachment; filename=%s", payload.Filename))
		http.ServeFile(w, r, payload.Path)
	})
	http.HandleFunc(receivepath, func(w http.ResponseWriter, r *http.Request) {
		defer s.wg.Done()
		w.WriteHeader(500)
	})
	fmt.Println(host)
	srv := &http.Server{Addr: address}
	// TODO: Improve this
	s.SendURL = fmt.Sprintf("http://%s%s", host, sendpath)
	go srv.Serve(tcpKeepAliveListener{listener.(*net.TCPListener)})
	return s, nil
}
