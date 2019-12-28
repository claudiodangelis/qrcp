package server

import "fmt"

// Server is the server
type Server struct {
	filename  string
	directory string
	// SendURL is the URL used to send the file
	SendURL string
	// ReceiveURL is the URL used to receive the file(s)
	ReceiveURL string
}

// SetFilename to send
func (s Server) SetFilename(filename string) error {
	s.filename = filename
	return nil
}

// SetOutput where to receive the files to
func (s Server) SetOutput(directory string) error {
	s.directory = directory
	return nil
}

// Start the server
func Start(address string, port int) (Server, error) {
	return Server{
		SendURL:    fmt.Sprintf("http://%s:%d", address, port),
		ReceiveURL: fmt.Sprintf("http://%s:%d/upload", address, port),
	}, nil
}
