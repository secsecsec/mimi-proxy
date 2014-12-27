package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	vhost "github.com/inconshreveable/go-vhost"
	"github.com/mimicloud/easyconfig"
	"log"
	"os"
	"time"
)

var config = struct {
	SecureBindAddr   string               `json:"secure_bind_addr"`
	InsecureBindAddr string               `json:"insecure_bind_addr"`
	Frontends        map[string]*Frontend `json:"frontends"`
	defaultFrontend  *Frontend
}{}

func init() {
	easyconfig.Parse("./example.json", &config)
}

const (
	muxTimeout            = 10 * time.Second
	defaultConnectTimeout = 10000 // milliseconds
)

var (
	secureFrontends   map[string]*Frontend
	insecureFrontends map[string]*Frontend
)

type loadTLSConfigFn func(crtPath, keyPath string) (*tls.Config, error)

type Options struct {
	configPath string
}

type Backend struct {
	Addr           string `"json:addr"`
	ConnectTimeout int    `json:connect_timeout"`
}

type Frontend struct {
	name     string
	Backends []Backend `json:"backends"`
	Strategy string    `json:"strategy"`
	TLSCrt   string    `json:"tls_crt"`
	TLSKey   string    `json:"tls_key"`
	muxTLS   *vhost.TLSMuxer
	muxHTTP  *vhost.HTTPMuxer
	Default  bool `json:"default"`

	strategy  BackendStrategy `json:"-"`
	tlsConfig *tls.Config     `json:"-"`
}

func (f *Frontend) isSecure() bool {
	return f.tlsConfig != nil
}

type BackendStrategy interface {
	NextBackend() Backend
}

func parseArgs() (*Options, error) {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <config file>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "%s is a simple TLS reverse proxy that can multiplex TLS connections\n"+
			"by inspecting the SNI extension on each incoming connection. This\n"+
			"allows you to accept connections to many different backend TLS\n"+
			"applications on a single port.\n\n"+
			"%s takes a single argument: the path to a JSON configuration file.\n\n", os.Args[0], os.Args[0])
	}
	flag.Parse()

	if len(flag.Args()) != 1 {
		return nil, fmt.Errorf("You must specify a single argument, the path to the configuration file.")
	}

	return &Options{
		configPath: flag.Arg(0),
	}, nil

}

func parseConfig(loadTLS loadTLSConfigFn) error {
	secureFrontends = make(map[string]*Frontend)
	insecureFrontends = make(map[string]*Frontend)

	for name, front := range config.Frontends {
		front.name = name
		if len(front.Backends) == 0 {
			return fmt.Errorf("You must specify at least one backend for frontend '%v'", name)
		}

		if front.Default {
			if config.defaultFrontend != nil {
				err = fmt.Errorf("Only one frontend may be the default")
				return
			}
			config.defaultFrontend = front
		}

		for _, back := range front.Backends {
			if back.ConnectTimeout == 0 {
				back.ConnectTimeout = defaultConnectTimeout
			}

			if back.Addr == "" {
				return fmt.Errorf("You must specify an addr for each backend on frontend '%v'", name)
			}
		}

		if front.TLSCrt != "" || front.TLSKey != "" {
			cfg, err := loadTLS(front.TLSCrt, front.TLSKey)
			if err != nil {
				return fmt.Errorf("Failed to load TLS configuration for frontend '%v': %v", name, err)
			}
			front.tlsConfig = cfg
			secureFrontends[name] = front
			insecureFrontends[name] = front
		} else {
			insecureFrontends[name] = front
		}
	}

	return nil
}

func loadTLSConfig(crtPath, keyPath string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(crtPath, keyPath)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}, nil
}

func main() {
	// parse configuration file
	err := parseConfig(loadTLSConfig)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	go func() {
		// run server
		secureServer := &Server{
			Listen: config.SecureBindAddr,
			Secure: true,
			Logger: log.New(os.Stdout, "slt ", log.LstdFlags|log.Lshortfile),
		}

		// this blocks unless there's a startup error
		err = secureServer.Run(secureFrontends)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start slt: %v\n", err)
			os.Exit(1)
		}
	}()

	go func() {
		// run server
		insecureServer := &Server{
			Listen: config.InsecureBindAddr,
			Secure: false,
			Logger: log.New(os.Stdout, "slt ", log.LstdFlags|log.Lshortfile),
		}

		// this blocks unless there's a startup error
		err = insecureServer.Run(insecureFrontends)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start slt: %v\n", err)
			os.Exit(1)
		}
	}()

	checkAlive()
}

func checkAlive() {
	ticker := time.NewTicker(1 * time.Minute)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			log.Printf("Run periodic check alive")
		case <-quit:
			ticker.Stop()
			return
		}
	}
}
