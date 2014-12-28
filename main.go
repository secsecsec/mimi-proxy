package main

import (
	"fmt"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mimicloud/easyconfig"
	"io/ioutil"
	"log"
	"os"
)

const (
	defaultConnectTimeout = 10000 // milliseconds
)

func init() {
	easyconfig.Parse("./example.json", &config)
}

func initializeApplications(frontendsRaw map[string][]*Frontend) (secureFrontends []*Frontend, insecureFrontends []*Frontend) {
	for _, frontends := range frontendsRaw {
		for _, front := range frontends {

			log.Printf("%v", front.backends == nil)
			for _, back := range front.backends {
				if back.ConnectTimeout == 0 {
					back.ConnectTimeout = defaultConnectTimeout
				}

				if back.Url == "" {
					log.Printf("You must specify an addr for each backend on frontend '%v'", front)
					continue
				}
			}

			insecureFrontends = append(insecureFrontends, front)
			if front.isSecure() {
				secureFrontends = append(secureFrontends, front)
			}
		}
	}

	return secureFrontends, insecureFrontends
}

var etcdClient *etcd.Client

func main() {
	var err error
	var errorPage []byte
	if config.ErrorPage != "" {
		errorPage, err = ioutil.ReadFile(config.ErrorPage)
		if err != nil {
			panic(err)
		}
	}

	etcdClient = etcd.NewClient(config.EtcdServers)
	frontends := ResolveApps(etcdClient, config.EtcdKey)
	secureFrontends, insecureFrontends := initializeApplications(frontends)

	go func() {
		// run server
		secureServer := &Server{
			Listen:    config.SecureBindAddr,
			Secure:    true,
			ErrorPage: string(errorPage),
			Logger:    log.New(os.Stdout, "slt ", log.LstdFlags|log.Lshortfile),
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
			Listen:    config.InsecureBindAddr,
			Secure:    false,
			ErrorPage: string(errorPage),
			Logger:    log.New(os.Stdout, "slt ", log.LstdFlags|log.Lshortfile),
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
