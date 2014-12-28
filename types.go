package main

import (
	"crypto/tls"
	vhost "github.com/inconshreveable/go-vhost"
)

var config = struct {
	SecureBindAddr   string               `json:"secure_bind_addr"`
	InsecureBindAddr string               `json:"insecure_bind_addr"`
	Frontends        map[string]*Frontend `json:"frontends"`
	EtcdKey          string               `json:"etcd_key"`
	EtcdServers      []string             `json:"etcd_servers"`
	ErrorPage        string               `json:"error_page"`
}{}

type Backend struct {
	Url            string `"json:url"`
	ConnectTimeout int    `json:connect_timeout"`
}

type Frontend struct {
	Route    string `json:"route"`
	Strategy string `json:"strategy"`
	TLSCrt   string `json:"tls_crt"`
	TLSKey   string `json:"tls_key"`

	muxTLS    *vhost.TLSMuxer
	muxHTTP   *vhost.HTTPMuxer
	strategy  BackendStrategy
	tlsConfig *tls.Config
	backends  []Backend
}

func (f *Frontend) isSecure() bool {
	return f.tlsConfig != nil
}
