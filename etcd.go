package main

import (
	b64 "encoding/base64"
	"encoding/json"
	"github.com/coreos/go-etcd/etcd"
	"log"
)

const (
	defaultConnectTimeout = 10000 // milliseconds
)

func ResolveApps(client *etcd.Client, etcdKey string) map[string][]*Frontend {
	var backends = make(map[string][]Backend)
	var frontends = make(map[string][]*Frontend)

	r, err := client.Get("/"+etcdKey, false, false)
	if err != nil {
		return nil
	}

	for _, n := range r.Node.Nodes {
		appId := n.Key[len(etcdKey)+2:]
		log.Printf("%s %s %s", n.Key, n.Value, appId)

		backendsEtcd, err := client.Get("/"+etcdKey+"/"+appId+"/backends", true, false)
		if err != nil {
			continue
		}

		for _, t := range backendsEtcd.Node.Nodes {
			log.Printf("%s %s", t.Key, t.Value)
			if _, ok := backends[appId]; !ok {
				backends[appId] = []Backend{}
			}

			var backend Backend
			if err := json.Unmarshal([]byte(t.Value), &backend); err != nil {
				log.Printf("Skip backend %s", err)
				continue
			}
			log.Printf("%s", backend.Url)
			if backend.Url == "" {
				log.Printf("Skip backend with incorrect url %s", backend)
				continue
			}
			if backend.ConnectTimeout == 0 {
				backend.ConnectTimeout = 100
			}
			backends[appId] = append(backends[appId], backend)
		}

		frontendsEtcd, err := client.Get("/"+etcdKey+"/"+appId+"/frontends", true, false)
		if err != nil {
			continue
		}

		for _, t := range frontendsEtcd.Node.Nodes {
			log.Printf("%s %s", t.Key, t.Value)
			if _, ok := frontends[appId]; !ok {
				frontends[appId] = []*Frontend{}
			}

			frontend := new(Frontend)
			if err := json.Unmarshal([]byte(t.Value), &frontend); err != nil {
				log.Printf("Skip frontend %s", err)
				continue
			}

			// always round-robin strategy for now
			frontend.strategy = &RoundRobinStrategy{
				backends: backends[appId],
			}
			frontend.backends = backends[appId]
			if frontend.TLSCrt != "" || frontend.TLSKey != "" {
				cert, err := b64.StdEncoding.DecodeString(frontend.TLSCrt)
				if err != nil {
					log.Printf("Failed to decode certificate '%v': %v", frontend.Hosts, err)
					continue
				}
				key, err := b64.StdEncoding.DecodeString(frontend.TLSKey)
				if err != nil {
					log.Printf("Failed to decode key '%v': %v", frontend.Hosts, err)
					continue
				}
				log.Printf("%s %s", cert, key)
				cfg, err := loadTLSConfig(cert, key)
				if err != nil {
					log.Printf("Failed to load TLS configuration for frontend '%v': %v", frontend.Hosts, err)
					continue
				}
				frontend.tlsConfig = cfg
			}
			frontends[appId] = append(frontends[appId], frontend)
		}
	}

	return frontends
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

func watchApps(client *etcd.Client, etcdKey string) {
	for {
		resp, err := client.Watch("/"+etcdKey, 0, true, nil, nil)
		if err != nil {
			panic(err)
		}
		log.Printf("%s %s", resp.Node.Key, resp.Node.Value)
	}
}
