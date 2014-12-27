package main

import (
	"crypto/tls"
	"fmt"
	vhost "github.com/inconshreveable/go-vhost"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type Server struct {
	*log.Logger
	wait   sync.WaitGroup
	Listen string
	Secure bool

	// these are for easier testing
	muxTLS  *vhost.TLSMuxer
	muxHTTP *vhost.HTTPMuxer
	ready   chan int
}

func (s *Server) Run(frontends map[string]*Frontend) error {
	// bind a port to handle TLS connections
	l, err := net.Listen("tcp", s.Listen)
	if err != nil {
		return err
	}
	s.Printf("Serving connections on %v", l.Addr())

	if s.Secure {
		// start muxing on it
		s.muxTLS, err = vhost.NewTLSMuxer(l, muxTimeout)
		if err != nil {
			return err
		}
	} else {
		// start muxing on it
		s.muxHTTP, err = vhost.NewHTTPMuxer(l, muxTimeout)
		if err != nil {
			return err
		}
	}

	// wait for all frontends to finish
	s.wait.Add(len(frontends))

	// setup muxing for each frontend
	for name, front := range frontends {
		var fl net.Listener
		var err error
		if s.Secure && front.isSecure() {
			fl, err = s.muxTLS.Listen(name)
		} else {
			fl, err = s.muxHTTP.Listen(name)
		}

		if err != nil {
			return err
		}
		go s.runFrontend(name, front, fl)
	}

	// custom error handler so we can log errors
	go func() {
		var err error
		var conn net.Conn

		for {
			if s.Secure {
				conn, err = s.muxTLS.NextError()
			} else {
				conn, err = s.muxHTTP.NextError()
			}

			if conn == nil {
				s.Printf("Failed to mux next connection, error: %v", err)
				if _, ok := err.(vhost.Closed); ok {
					return
				} else {
					continue
				}
			} else {
				if _, ok := err.(vhost.NotFound); ok && config.defaultFrontend != nil {
					go s.proxyConnection(conn, config.defaultFrontend)
				} else {
					s.Printf("Failed to mux connection from %v, error: %v", conn.RemoteAddr(), err)
					// XXX: respond with valid TLS close messages
					conn.Close()
				}
			}
		}
	}()

	// we're ready, signal it for testing
	if s.ready != nil {
		close(s.ready)
	}

	s.wait.Wait()

	return nil
}

func (s *Server) runFrontend(name string, front *Frontend, l net.Listener) {
	// mark finished when done so Run() can return
	defer s.wait.Done()

	// always round-robin strategy for now
	front.strategy = &RoundRobinStrategy{backends: front.Backends}

	s.Printf("Handling connections to %v", name)
	for {
		// accept next connection to this frontend
		conn, err := l.Accept()
		if err != nil {
			s.Printf("Failed to accept new connection for '%v': %v", conn.RemoteAddr())
			if e, ok := err.(net.Error); ok {
				if e.Temporary() {
					continue
				}
			}
			return
		}
		s.Printf("Accepted new connection for %v from %v", name, conn.RemoteAddr())

		// proxy the connection to an backend
		go s.proxyConnection(conn, front)
	}
}

func (s *Server) proxyConnection(c net.Conn, front *Frontend) (err error) {
	log.Printf("%v", c.RemoteAddr())

	// unwrap if tls cert/key was specified
	if front.isSecure() { //
		if s.Secure {
			c = tls.Server(c, front.tlsConfig)
		} else {
			// Redirect to secure host
			fmt.Fprintf(c, `HTTP/1.0 301 Moved Permanently
Location: https://%s
`, front.name)
			c.Close()
			return nil
		}
	}

	// pick the backend
	backend := front.strategy.NextBackend()

	// dial the backend
	upConn, err := net.DialTimeout("tcp", backend.Addr, time.Duration(backend.ConnectTimeout)*time.Millisecond)
	if err != nil {
		s.Printf("Failed to dial backend connection %v: %v", backend.Addr, err)
		c.Close()
		return
	}
	s.Printf("Initiated new connection to backend: %v %v", upConn.LocalAddr(), upConn.RemoteAddr())

	// join the connections
	s.joinConnections(c, upConn)
	return
}

func (s *Server) joinConnections(c1 net.Conn, c2 net.Conn) {
	var wg sync.WaitGroup
	halfJoin := func(dst net.Conn, src net.Conn) {
		defer wg.Done()
		defer dst.Close()
		defer src.Close()
		n, err := io.Copy(dst, src)
		s.Printf("Copy from %v to %v failed after %d bytes with error %v", src.RemoteAddr(), dst.RemoteAddr(), n, err)
	}

	s.Printf("Joining connections: %v %v", c1.RemoteAddr(), c2.RemoteAddr())
	wg.Add(2)
	go halfJoin(c1, c2)
	go halfJoin(c2, c1)
	wg.Wait()
}
