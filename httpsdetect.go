package main

import (
	"fmt"
	"io"
	"net"
)

// A listener that detects the incoming data is TLS encrypted or
// plaintext and emits a HTTP 301 redirect if appropriate.
type HijackHTTPListener struct {
	net.Listener
}

type Conn struct {
	net.Conn
	b byte
	e error
	f bool
}

func (c *Conn) Read(b []byte) (int, error) {
	if c.f {
		c.f = false
		b[0] = c.b
		if len(b) > 1 && c.e == nil {
			n, e := c.Conn.Read(b[1:])
			if e != nil {
				c.Conn.Close()
			}
			return n + 1, e
		} else {
			return 1, c.e
		}
	}
	return c.Conn.Read(b)
}

func (l *HijackHTTPListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	b := make([]byte, 1)
	_, err = c.Read(b)
	if err != nil {
		c.Close()
		if err != io.EOF {
			return nil, err
		}
	}

	con := &Conn{
		Conn: c,
		b:    b[0],
		e:    err,
		f:    true,
	}

	// First byte == 22 means it's HTTPS
	if b[0] == 22 {
		return con, nil
	}

	// Otherwise it’s HTTP
	con.Write([]byte(fmt.Sprintf("HTTP/1.1 301 Moved Permanently\nLocation: https://%s/\n", *listen)))
	con.Close()
	return con, nil
}
