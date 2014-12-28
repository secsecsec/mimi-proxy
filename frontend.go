package main

import (
	"crypto/tls"
	b64 "encoding/base64"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

func NewFrontend(id string) *Frontend {
	fr := &Frontend{
		Id:   id,
		ch:   make(chan bool),
		wait: &sync.WaitGroup{},
	}

	return fr
}

type Frontend struct {
	Id     string   `json:"id"`
	Hosts  []string `json:"hosts"`
	TLSCrt string   `json:"tls_crt"`
	TLSKey string   `json:"tls_key"`

	strategy  BackendStrategy
	tlsConfig *tls.Config
	backends  []Backend
	server    *Server
	running   bool

	hostListeners []net.Listener
	ch            chan bool
	wait          *sync.WaitGroup
}

func (s *Frontend) SetTLS(TLSCrt, TLSKey string) (err error) {
	var cert, key []byte

	cert, err = b64.StdEncoding.DecodeString(TLSCrt)
	if err != nil {
		return err
	}

	key, err = b64.StdEncoding.DecodeString(TLSKey)
	if err != nil {
		return err
	}

	cfg, errLoad := loadTLSConfig(cert, key)
	if errLoad != nil {
		return errLoad
	}
	s.tlsConfig = cfg

	return nil
}

func (f *Frontend) SetBackends(backends []Backend) {
	// always round-robin strategy for now
	f.strategy = &RoundRobinStrategy{
		backends: backends,
	}
	f.backends = backends
}

func (f *Frontend) isSecure() bool {
	return f.tlsConfig != nil
}

func (s *Frontend) Start() {
	s.running = true
	s.ch = make(chan bool)
}

func (s *Frontend) Create(start bool) error {
	if start {
		s.Start()
	}
	s.wait.Add(len(s.Hosts))

	for _, host := range s.Hosts {
		fl, err := s.prepareHost(host)
		if err != nil {
			s.server.Printf("Failed to mux listen: %s", err)
			return err
		}

		// proxy the connection to an backend
		go s.RunHost(host, fl)
	}

	s.wait.Wait()

	return nil
}

func (s *Frontend) prepareHost(host string) (l net.Listener, err error) {
	if s.server.Secure && s.isSecure() {
		l, err = s.server.muxTLS.Listen(host)
	} else {
		l, err = s.server.muxHTTP.Listen(host)
	}

	s.hostListeners = append(s.hostListeners, l)

	return l, err
}

func (s *Frontend) RunHost(host string, l net.Listener) {
	// mark finished when done so Run() can return
	defer s.server.wait.Done()

	s.server.Printf("Handling connections to %v", host)
	for {
		var err error
		var conn net.Conn

		s.server.Printf("Request on frontend %s. Is running %v", s.Id, s.running)
		// accept next connection to this frontend
		conn, err = l.Accept()

		if s.running == false {
			conn.Close()
			continue
		}

		select {
		case <-s.ch:
			s.server.Printf("Stopping frontend %s: %s", s.Id, s.Hosts)
			conn.Close()
			for i := 0; i < len(s.Hosts); i++ {
				s.wait.Done()
			}
			return
		default:
		}

		if err != nil {
			s.server.Printf("Failed to accept new connection for '%v': %v", conn.RemoteAddr())
			if e, ok := err.(net.Error); ok {
				if e.Temporary() {
					continue
				}
			}
			continue
		}
		s.server.Printf("Accepted new connection for %v from %v", host, conn.RemoteAddr())

		s.wait.Add(1)
		// proxy the connection to an backend
		go s.proxyConnection(host, conn)
	}
}

func (s *Frontend) proxyConnection(host string, c net.Conn) (err error) {
	defer s.wait.Done()

	// unwrap if tls cert/key was specified
	if s.isSecure() { //
		if s.server.Secure {
			c = tls.Server(c, s.tlsConfig)
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
	backend := s.strategy.NextBackend()

	// dial the backend
	upConn, err := net.DialTimeout("tcp", backend.Url, time.Duration(backend.ConnectTimeout)*time.Millisecond)
	if err != nil {
		s.server.Printf("Failed to dial backend connection %v: %v", backend.Url, err)
		if s.server.ErrorPage != "" {
			fmt.Fprintf(c, `HTTP/1.0 200
Content-Length: %d
Content-Type: text/html; charset=utf-8

%s
`, len(s.server.ErrorPage), s.server.ErrorPage)
		}
		c.Close()
		return err
	}
	s.server.Printf("Initiated new connection to backend: %v %v", upConn.LocalAddr(), upConn.RemoteAddr())

	// join the connections
	s.joinConnections(c, upConn)

	return nil
}

func (s *Frontend) joinConnections(c1 net.Conn, c2 net.Conn) {
	var wg sync.WaitGroup
	halfJoin := func(dst net.Conn, src net.Conn) {
		defer wg.Done()
		defer dst.Close()
		defer src.Close()
		n, err := io.Copy(dst, src)
		s.server.Printf("Copy from %v to %v failed after %d bytes with error %v", src.RemoteAddr(), dst.RemoteAddr(), n, err)
	}

	s.server.Printf("Joining connections: %v %v", c1.RemoteAddr(), c2.RemoteAddr())
	wg.Add(2)
	go halfJoin(c1, c2)
	go halfJoin(c2, c1)
	wg.Wait()
}

func (s *Frontend) Stop() {
	s.running = false

	s.server.Printf("Stop frontend %s", s.Id)

	close(s.ch)
	s.wait.Wait()

	for _, l := range s.hostListeners {
		l.Close()
	}
}
