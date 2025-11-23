package core

import (
	"io"
	"net"
)

// RelayConn implements full-duplex (bidirectional) byte stream forwarding between two TCP connections A and B, until either side closes.
// This function blocks after being called until either connection is closed. Suitable for all full passthrough services like ssh/http.
func RelayConn(a, b net.Conn) {
	// a->b
	go func() {
		_, _ = io.Copy(b, a)
		_ = b.Close()
	}()
	// b->a
	_, _ = io.Copy(a, b)
	_ = a.Close()
}
