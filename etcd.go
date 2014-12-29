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

func ResolveApps(client *etcd.Client, etcdKey string) (map[string]*Frontend, map[string]*Frontend) {
	var backends = make(map[string][]Backend)
	var frontendsApp = make(map[string]map[string]*Frontend)
	var frontends = make(map[string]*Frontend)

	r, err := client.Get("/"+etcdKey, false, false)
	if err != nil {
		panic(err)
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
				backend.ConnectTimeout = defaultConnectTimeout
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

			if _, ok := frontendsApp[appId]; !ok {
				frontendsApp[appId] = make(map[string]*Frontend)
			}

			frontend, err := newFrontendFromJson(frontendId, t.Value)
			if err != nil {
				log.Printf("Skip frontend due error: %s", err)
				continue
			}
			frontend.SetBackends(backends[appId])
			frontendsApp[appId][frontend.Id] = frontend
			frontends[frontend.Id] = frontend
			collection.Frontends[frontendId] = frontend
		}

		app := NewApplication(appId)
		app.Frontends = collection.Frontends
		app.Backends = collection.Backends
		collection.Applications[appId] = app
	}

	return InitApplications(frontends)
}

func newFrontendFromJson(id, data string) (*Frontend, error) {
	var tmp FrontendTmp

	frontend := NewFrontend(id)
	if err := json.Unmarshal([]byte(data), &tmp); err != nil {
		return nil, err
	}

	frontend.Hosts = tmp.Hosts

	if tmp.TLSCrt != "" || tmp.TLSKey != "" {
		err := frontend.SetTLS(tmp.TLSCrt, tmp.TLSCrt)
		if err != nil {
			return nil, err
		}
	}

	return frontend, nil
}

func InitApplications(frontends map[string]*Frontend) (map[string]*Frontend, map[string]*Frontend) {
	secureFrontends := make(map[string]*Frontend)
	insecureFrontends := make(map[string]*Frontend)

	for id, f := range frontends {
		insecureFrontends[id] = f
		if f.isSecure() {
			secureFrontends[id] = f
		}
	}

	return secureFrontends, insecureFrontends
}

func watchApps(client *etcd.Client, etcdKey string, secureServer, insecureServer *Server) {
	for {
		r, err := client.Watch("/"+etcdKey, 0, true, nil, nil)
		if err != nil {
			log.Printf("Incorrect json: %s", err)
			continue
		}

		parts := strings.Split(r.Node.Key, "/")
		appId := parts[2]
		tmpId := r.Node.Key[strings.LastIndex(r.Node.Key, "/")+1:]
		if strings.Contains(r.Node.Key, "backends") {
			// Create / Update / Delete backend
		} else if strings.Contains(r.Node.Key, "frontends") {
			// Create / Update / Delete frontend
			frontend, err := newFrontendFromJson(tmpId, r.Node.Value)
			if err != nil {
				log.Printf("Skip frontend %s:%s", appId, tmpId)
				continue
			}

			var backendList []Backend
			for _, back := range collection.Applications[appId].Backends {
				backendList = append(backendList, back)
			}
			frontend.SetBackends(backendList)

			collection.Frontends[tmpId] = frontend

			if frontend.isSecure() {
				secureServer.AddFrontend(frontend)
				go secureServer.RunFrontend(frontend)
			} else {
				insecureServer.AddFrontend(frontend)
				go insecureServer.RunFrontend(frontend)
			}
		} else {
			// Create / Update / Delete application
		}

		log.Printf("%s %s %s", tmpId, r.Node.Key, r.Node.Value)
	}
}
