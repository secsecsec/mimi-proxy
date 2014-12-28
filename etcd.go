package main

import (
	"encoding/json"
	"github.com/coreos/go-etcd/etcd"
	"log"
	"strings"
)

const (
	defaultConnectTimeout = 10000 // milliseconds
)

type FrontendTmp struct {
	Hosts  []string `json:"hosts"`
	TLSCrt string   `json:"tls_crt"`
	TLSKey string   `json:"tls_key"`
}

func ResolveApps(client *etcd.Client, etcdKey string) map[string][]*Frontend {
	var backends = make(map[string][]Backend)
	var frontends = make(map[string][]*Frontend)

	r, err := client.Get("/"+etcdKey, false, false)
	if err != nil {
		return nil
	}

	for _, n := range r.Node.Nodes {
		appId := n.Key[strings.LastIndex(n.Key, "/")+1:]

		backendsEtcd, err := client.Get("/"+etcdKey+"/"+appId+"/backends", true, false)
		if err != nil {
			continue
		}

		for _, t := range backendsEtcd.Node.Nodes {
			backendId := t.Key[strings.LastIndex(t.Key, "/")+1:]

			if _, ok := backends[appId]; !ok {
				backends[appId] = []Backend{}
			}

			var backend Backend
			if err := json.Unmarshal([]byte(t.Value), &backend); err != nil {
				log.Printf("Skip backend %s", err)
				continue
			}
			if backend.Url == "" {
				log.Printf("Skip backend with incorrect url %s", backend)
				continue
			}
			if backend.ConnectTimeout == 0 {
				backend.ConnectTimeout = 100
			}
			backends[appId] = append(backends[appId], backend)
			collection.Backends[backendId] = backend
		}

		frontendsEtcd, err := client.Get("/"+etcdKey+"/"+appId+"/frontends", true, false)
		if err != nil {
			continue
		}

		for _, t := range frontendsEtcd.Node.Nodes {
			frontendId := t.Key[strings.LastIndex(t.Key, "/")+1:]

			if _, ok := frontends[appId]; !ok {
				frontends[appId] = []*Frontend{}
			}

			var tmp FrontendTmp
			frontend := NewFrontend(frontendId)
			if err := json.Unmarshal([]byte(t.Value), &tmp); err != nil {
				log.Printf("Skip frontend %s", err)
				continue
			}
			frontend.Hosts = tmp.Hosts
			frontend.SetBackends(backends[appId])

			if tmp.TLSCrt != "" || tmp.TLSKey != "" {
				err = frontend.SetTLS(tmp.TLSCrt, tmp.TLSCrt)
				if err != nil {
					log.Printf("Failed to decode certificate / key: %v", err)
					continue
				}
			}

			frontends[appId] = append(frontends[appId], frontend)
			collection.Frontends[frontendId] = frontend
		}
	}

	return frontends
}

func initializeApplications(frontendsRaw map[string][]*Frontend) (secureFrontends []*Frontend, insecureFrontends []*Frontend) {
	for _, frontends := range frontendsRaw {
		for _, front := range frontends {

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
