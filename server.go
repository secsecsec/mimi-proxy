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

	Listen    string
	Secure    bool
	ErrorPage string
	Frontends []*Frontend

	muxTLS  *vhost.TLSMuxer
	muxHTTP *vhost.HTTPMuxer
	wait    sync.WaitGroup

	// these are for easier testing
	ready chan int
}

func (s *Server) ListenAndServe() error {
	// bind a port to handle HTTP / TLS connections
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
		go frontend.Start()
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
	s.Printf("Start error handler")
	for {
		var err error
		var conn net.Conn

		if s.Secure {
			conn, err = s.muxTLS.NextError()
		} else {
			conn, err = s.muxHTTP.NextError()
		}

		switch err.(type) {
		case vhost.BadRequest:
			s.Printf("got a bad request!")
		case vhost.NotFound:
			s.Printf("got a connection for an unknown vhost")
		case vhost.Closed:
			s.Printf("Closed conn: %s", err)
		default:
			if conn != nil {
				s.Printf("Unknown server error")
			}
		}

		if conn != nil {
			s.Printf("Failed to mux connection from %v, error: %v", conn.RemoteAddr(), err)
			// XXX: respond with valid TLS close messages
			conn.Close()
		}
	}
}
