package main

import (
	vhost "github.com/inconshreveable/go-vhost"
	"log"
	"net"
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
	Frontends []*Frontend
	muxTLS    *vhost.TLSMuxer
	muxHTTP   *vhost.HTTPMuxer
	ready     chan int
}

func (s *Server) Run() error {
	// bind a port to handle TLS connections
	l, err := net.Listen("tcp", s.Listen)
	if err != nil {
		return err
	}

	s.Printf("Serving connections on %v, frontends: %d", l.Addr(), len(s.Frontends))

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

	// wait for all s.Frontends to finish
	s.wait.Add(len(s.Frontends))

	// setup muxing for each frontend
	for _, frontend := range s.Frontends {
		frontend.server = s
		frontend.Start()
	}

	// custom error handler so we can log errors
	go s.ErrorHandler()

	// we're ready, signal it for testing
	if s.ready != nil {
		close(s.ready)
	}

	s.wait.Wait()

	return nil
}

func (s *Server) ErrorHandler() {
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
}
