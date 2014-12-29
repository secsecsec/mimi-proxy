package main

import (
	"encoding/json"
	"errors"
	"fmt"
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

type BackendTmp struct {
	Url            string `"json:url"`
	ConnectTimeout int    `json:connect_timeout"`
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

			backend, err := NewBackendFromJson(backendId, t.Value)
			if err != nil {
				log.Printf("Skip backend due error: %s", err)
				continue
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

func NewBackendFromJson(id, data string) (Backend, error) {
	var tmp BackendTmp

	backend := NewBackend(id)
	if err := json.Unmarshal([]byte(data), &tmp); err != nil {
		return backend, err
	}

	if tmp.Url == "" {
		return backend, errors.New(fmt.Sprintf("Skip backend with incorrect url %s", id))
	}
	backend.Url = tmp.Url

	if tmp.ConnectTimeout != 0 {
		backend.ConnectTimeout = tmp.ConnectTimeout
	}

	return backend, nil
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

func isBackend(r etcd.Response) bool {
	return strings.Contains(r.Node.Key, "backends")
}

func isFrontend(r etcd.Response) bool {
	return strings.Contains(r.Node.Key, "frontends")
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

		if r.Action == "delete" {
			if isBackend(r) {
				collection.Applications[appId].DeleteBackend(tmpId)
				delete(collection.Backends, tmpId)
			} else if isFrontend(r) {
				collection.Applications[appId].DeleteFrontend(tmpId)
				delete(collection.Frontends, tmpId)
			} else {
				collection.Applications[appId].Stop()
				delete(collection.Applications, appId)
			}
		} else if r.Action == "set" || r.Action == "update" {
			if isBackend(r) {
				// Create / Update / Delete backend
				backend, err := NewBackendFromJson(tmpId, r.Node.Value)
				if err != nil {
					log.Printf("Skip backend due error: %s", err)
					continue
				}
				if _, ok := collection.Backends[tmpId]; ok {
					collection.Applications[appId].DeleteBackend(tmpId)
				}
				collection.Backends[tmpId] = backend
				collection.Applications[appId].AddBackend(backend)
			} else if isFrontend(r) {
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

				if _, ok := collection.Frontends[tmpId]; ok {
					collection.Applications[appId].DeleteFrontend(tmpId)
				}
				collection.Frontends[tmpId] = frontend

				if frontend.isSecure() {
					secureServer.AddFrontend(frontend)
					go secureServer.RunFrontend(frontend)
				} else {
					insecureServer.AddFrontend(frontend)
					go insecureServer.RunFrontend(frontend)
				}
			} else {
				collection.AddApplication(NewApplication(appId))
			}
		}

		log.Printf("%s %s %s", tmpId, r.Node.Key, r.Node.Value)
	}
}
