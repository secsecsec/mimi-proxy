package main

import (
	vhost "github.com/inconshreveable/go-vhost"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

const (
	muxTimeout = 10 * time.Second
)

func NewServer(listen string, secure bool, errorPage string) *Server {
	return &Server{
		Listen:    listen,
		Secure:    secure,
		ErrorPage: errorPage,
		Frontends: make(map[string]*Frontend),
		Logger:    log.New(os.Stdout, config.SecureBindAddr+" ", log.LstdFlags|log.Lshortfile),
	}
}

type Server struct {
	*log.Logger

	Listen    string
	Secure    bool
	ErrorPage string
	Frontends map[string]*Frontend

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
		go s.RunFrontend(frontend)
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

func (s *Server) AddFrontend(frontend *Frontend) {
	if f, ok := s.Frontends[frontend.Id]; ok {
		f.Stop()
	}

	s.Frontends[frontend.Id] = frontend
}

func (s *Server) RunFrontend(frontend *Frontend) {
	s.wait.Add(1)
	frontend.server = s
	frontend.Start()
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
