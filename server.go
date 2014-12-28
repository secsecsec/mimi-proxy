package main

import (
	"crypto/tls"
	"fmt"
	vhost "github.com/inconshreveable/go-vhost"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	muxTimeout = 10 * time.Second
)

type Server struct {
	*log.Logger
	wait      sync.WaitGroup
	Listen    string
	Secure    bool
	ErrorPage string

	// these are for easier testing
	muxTLS  *vhost.TLSMuxer
	muxHTTP *vhost.HTTPMuxer
	ready   chan int
}

func (s *Server) Match(host, route string) bool {
	s.Printf("Match %s %s", host, route)
	return true
}

func (s *Server) Run(frontends []*Frontend) error {
	// bind a port to handle TLS connections
	l, err := net.Listen("tcp", s.Listen)
	if err != nil {
		return err
	}

	s.Printf("Serving connections on %v, frontends: %d", l.Addr(), len(frontends))

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
			continue
		}

		if len(frontends) == 0 {
			conn.Close()
			continue
		}

		var host string
		if s.Secure {
			vhostConn, err := vhost.TLS(conn)
			if err != nil {
				log.Printf("Not a valid http connection")
				conn.Close()
			}
			host = strings.ToLower(vhostConn.Host())
			vhostConn.Free()
		} else {
			vhostConn, err := vhost.HTTP(conn)
			if err != nil {
				log.Printf("Not a valid http connection")
				conn.Close()
			}
			host = strings.ToLower(vhostConn.Host())
			vhostConn.Free()
		}

		// setup muxing for each frontend
		for _, frontend := range frontends {
			var fl net.Listener
			var err error

			if s.Secure && frontend.isSecure() {
				fl, err = s.muxTLS.Listen(host)
			} else {
				fl, err = s.muxHTTP.Listen(host)
			}

			if err != nil {
				s.Printf("Failed to mux listen: %s", err)
				return err
			}

			if s.Match(host, frontend.Route) {
				// wait for all frontends to finish
				s.wait.Add(1)
				// proxy the connection to an backend
				go s.runFrontend(host, frontend, fl)
			}
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
					s.Printf("Failed to mux connection from %v, error: %v", conn.RemoteAddr(), err)
					// XXX: respond with valid TLS close messages
					conn.Close()
				}
			}
		}()

		// we're ready, signal it for testing
		if s.ready != nil {
			close(s.ready)
		}

		s.wait.Wait()
	}

	return nil
}

func (s *Server) runFrontend(host string, frontend *Frontend, fl net.Listener) {
	// mark finished when done so Run() can return
	defer s.wait.Done()

	wait := sync.WaitGroup{}

	s.Printf("Handling connections to %v", host)
	for {
		log.Printf("%d", 1)
		// accept next connection to this frontend
		conn, err := fl.Accept()

		wait.Wait()
		log.Printf("%d", 2)
		if err != nil {
			log.Printf("%d", 3)
			s.Printf("Failed to accept new connection for '%v': %v", conn.RemoteAddr())
			log.Printf("%d", 4)
			if e, ok := err.(net.Error); ok {
				log.Printf("%d", 5)
				if e.Temporary() {
					log.Printf("%d", 6)
					continue
				}
				log.Printf("%d", 7)
			}
			log.Printf("%d", 8)
			continue
		}
		log.Printf("%d", 9)
		s.Printf("Accepted new connection for %v from %v", host, conn.RemoteAddr())

		// proxy the connection to an backend
		go s.proxyConnection(host, conn, frontend)
	}
}

func (s *Server) proxyConnection(host string, c net.Conn, frontend *Frontend) (err error) {
	// unwrap if tls cert/key was specified
	if frontend.isSecure() { //
		if s.Secure {
			c = tls.Server(c, frontend.tlsConfig)
		} else {
			// Redirect to secure host
			fmt.Fprintf(c, `HTTP/1.0 301 Moved Permanently
Location: https://%s
`, host)
			c.Close()
			return nil
		}
	}

	// pick the backend
	backend := frontend.strategy.NextBackend()

	// dial the backend
	upConn, err := net.DialTimeout("tcp", backend.Url, time.Duration(backend.ConnectTimeout)*time.Millisecond)
	if err != nil {
		s.Printf("Failed to dial backend connection %v: %v", backend.Url, err)
		if s.ErrorPage != "" {
			fmt.Fprintf(c, `HTTP/1.0 200
Content-Length: %d
Content-Type: text/html; charset=utf-8

%s
`, len(s.ErrorPage), s.ErrorPage)
		}
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

func loadTLSConfig(cert, key []byte) (*tls.Config, error) {
	certificate, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{certificate},
	}, nil
}
