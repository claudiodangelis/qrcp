//go:build !darwin

package server

import "net"

// wrapConn returns the connection as is; see conn_darwin.go.
func wrapConn(tc *net.TCPConn) net.Conn {
	return tc
}
