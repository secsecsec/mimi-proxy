package main

import (
	"flag"
	"fmt"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mimicloud/easyconfig"
	"io/ioutil"
	"os"
)

const (
	usage = `mimiproxy [--path] file

`
)

var config = struct {
	ApiServerAddr    string               `json:"api_server_addr"`
	SecureBindAddr   string               `json:"secure_bind_addr"`
	InsecureBindAddr string               `json:"insecure_bind_addr"`
	Frontends        map[string]*Frontend `json:"frontends"`
	EtcdKey          string               `json:"etcd_key"`
	EtcdServers      []string             `json:"etcd_servers"`
	ErrorPage        string               `json:"error_page"`
}{}

var etcdClient *etcd.Client
var collection *Collection
var configPath string

func init() {
	flag.StringVar(&configPath, "path", "", "Path to config file")

	flag.Parse()
	if flag.NFlag() == 0 {
		os.Stderr.WriteString(usage)
		flag.PrintDefaults()
		os.Exit(0)
	}

	easyconfig.Parse(configPath, &config)

	etcdClient = etcd.NewClient(config.EtcdServers)
	collection = NewCollection()
}

func main() {
	var err error
	var errorPage []byte
	if config.ErrorPage != "" {
		errorPage, err = ioutil.ReadFile(config.ErrorPage)
		if err != nil {
			panic(err)
		}
	}

	secureFrontends, insecureFrontends := ResolveApps(etcdClient, config.EtcdKey)

	secureServer := NewServer(config.SecureBindAddr, true, string(errorPage))
	secureServer.Frontends = secureFrontends

	// Start secure (:443 port) server
	go func() {
		err = secureServer.ListenAndServe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start slt: %v\n", err)
			os.Exit(1)
		}
	}()

	insecureServer := NewServer(config.InsecureBindAddr, false, string(errorPage))
	insecureServer.Frontends = insecureFrontends

	// Start insecure (:80 port) server
	go func() {
		err = insecureServer.ListenAndServe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start slt: %v\n", err)
			os.Exit(1)
		}
	}()

	// Watch new applications / frontends / backends in etcd server
	go watchApps(etcdClient, config.EtcdKey, secureServer, insecureServer)

	apiServer := &ApiServer{
		EnableCheckAlive: true,
	}
	apiServer.ListenAndServe(config.ApiServerAddr)
}
