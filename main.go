package main

import (
	"fmt"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mimicloud/easyconfig"
	"io/ioutil"
	"log"
	"os"
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

func init() {
	easyconfig.Parse("./example.json", &config)
	etcdClient = etcd.NewClient(config.EtcdServers)
	collection = &Collection{
		Applications: make(map[string]*Application),
		Backends:     make(map[string]Backend),
		Frontends:    make(map[string]*Frontend),
	}
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

	// watchApps(etcdClient, config.EtcdKey)
	// os.Exit(0)

	frontends := ResolveApps(etcdClient, config.EtcdKey)
	secureFrontends, insecureFrontends := initializeApplications(frontends)

	secureServer := &Server{
		Listen:    config.SecureBindAddr,
		Secure:    true,
		ErrorPage: string(errorPage),
		Frontends: secureFrontends,
		Logger:    log.New(os.Stdout, "secure ", log.LstdFlags|log.Lshortfile),
	}

	// Start secure (:443 port) server
	go func() {
		err = secureServer.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start slt: %v\n", err)
			os.Exit(1)
		}
	}()

	insecureServer := &Server{
		Listen:    config.InsecureBindAddr,
		Secure:    false,
		ErrorPage: string(errorPage),
		Frontends: insecureFrontends,
		Logger:    log.New(os.Stdout, "insecure ", log.LstdFlags|log.Lshortfile),
	}

	// Start insecure (:80 port) server
	go func() {
		err = insecureServer.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start slt: %v\n", err)
			os.Exit(1)
		}
	}()

	// Watch new applications / frontends / backends in etcd server
	go watchApps(etcdClient, config.EtcdKey)

	apiServer := &ApiServer{
		EnableCheckAlive: true,
	}
	apiServer.Run(config.ApiServerAddr)
}
