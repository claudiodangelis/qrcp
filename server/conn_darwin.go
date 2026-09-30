//go:build darwin

package server

import "net"

// wrapConn hides ReadFrom from net/http so that file responses are written
// with a plain Write loop instead of sendfile(2).
//
// On some macOS setups, responses served over non-loopback interfaces arrive
// reordered as [file[512:], headers, file[:512]] when sendfile is used
// (see #382). The exact cause is not confirmed; it may be a network extension
// or content filter interfering with sendfile rather than a Go bug. Skipping
// sendfile avoids it at a negligible cost for LAN transfers.
func wrapConn(tc *net.TCPConn) net.Conn {
	return noReadFromConn{Conn: tc, tc: tc}
}

type noReadFromConn struct {
	net.Conn
	tc *net.TCPConn
}

// CloseWrite is forwarded so net/http can still half-close the connection
// before closing it, avoiding a TCP reset while the client is still sending.
func (c noReadFromConn) CloseWrite() error {
	return c.tc.CloseWrite()
}
