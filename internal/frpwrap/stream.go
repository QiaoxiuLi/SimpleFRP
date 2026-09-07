package frpwrap

import (
	"bufio"
	"io"
	"net"
	"sync"
)

// Preserve application bytes read alongside the newline-delimited handshake.
type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *bufferedConn) Read(p []byte) (int, error) { return c.reader.Read(p) }
func (c *bufferedConn) CloseWrite() error {
	if c, ok := c.Conn.(interface{ CloseWrite() error }); ok {
		return c.CloseWrite()
	}
	return nil
}

// A TCP FIN closes one direction only; the peer may still be sending its response.
func proxy(a, b net.Conn) (int64, int64) {
	defer a.Close()
	defer b.Close()
	var wg sync.WaitGroup
	var uploaded, downloaded int64
	copyOne := func(dst, src net.Conn, n *int64) {
		defer wg.Done()
		var err error
		*n, err = io.Copy(dst, src)
		if err != nil {
			_ = a.Close()
			_ = b.Close()
			return
		}
		if c, ok := dst.(interface{ CloseWrite() error }); ok {
			_ = c.CloseWrite()
		} else {
			_ = dst.Close()
		}
	}
	wg.Add(2)
	go copyOne(b, a, &uploaded)
	go copyOne(a, b, &downloaded)
	wg.Wait()
	return uploaded, downloaded
}
