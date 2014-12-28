package main

import (
	"crypto/tls"
)

func loadTLSConfig(cert, key []byte) (*tls.Config, error) {
	certificate, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{certificate},
	}, nil
}
