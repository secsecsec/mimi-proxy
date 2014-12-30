package main

import (
	"errors"
	"fmt"
	"log"
)

func NewApplication(Id string) *Application {
	return &Application{
		Id:        Id,
		Frontends: make(map[string]*Frontend),
		Backends:  make(map[string]Backend),
	}
}

type Application struct {
	Id        string               `json:"id"`
	Frontends map[string]*Frontend `json:"frontends"`
	Backends  map[string]Backend   `json:"backends"`
}

func (self *Application) Create() (err error) {
	_, err = etcdClient.CreateDir("/"+config.EtcdKey+"/"+self.Id, 1)
	return err
}

func (self *Application) Delete() (err error) {
	_, err = etcdClient.Delete("/"+config.EtcdKey+"/"+self.Id, true)
	return err
}

func (s *Application) Stop() {
	for _, frontend := range s.Frontends {
		frontend.Stop()
	}
}

func (s *Application) Start() {
	for _, frontend := range s.Frontends {
		frontend.Start()
	}
}

func (s *Application) AddBackend(backend Backend) {
	for _, frontend := range s.Frontends {
		frontend.AddBackend(backend)
	}
}

func (s *Application) DeleteBackend(id string) error {
	log.Printf("Application: delete backend %s: %s", id, s.Backends)
	if _, ok := s.Backends[id]; ok {
		delete(s.Backends, id)

		log.Printf("%s", s.Frontends)
		for _, frontend := range s.Frontends {
			frontend.DeleteBackend(id)
		}

		return nil
	}

	return errors.New(fmt.Sprintf("Unknown backend id: %s", id))
}

func (s *Application) DeleteFrontend(id string) error {
	if frontend, ok := s.Frontends[id]; ok {
		frontend.Stop()
		delete(s.Frontends, id)
		return nil
	}

	return errors.New(fmt.Sprintf("Unknown backend id: %s", id))
}
