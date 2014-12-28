package main

import (
	"sync"
)

func NewApplication(Id string, Frontends []*Frontend, Backends []*Backend) *Application {
	app := &Application{
		Id:        Id,
		Frontends: Frontends,
		Backends:  Backends,
		ch:        make(chan bool),
		waitGroup: &sync.WaitGroup{},
	}
	return app
}

type Application struct {
	Id        string      `json:"id"`
	Frontends []*Frontend `json:"frontends"`
	Backends  []*Backend  `json:"backends"`
	ch        chan bool
	waitGroup *sync.WaitGroup
}

func (self *Application) Create() (err error) {
	_, err = etcdClient.CreateDir("/"+config.EtcdKey+"/"+self.Id, 1)
	return err
}

func (self *Application) Start() (err error) {
	// self.waitGroup.Add(1)

	// defer s.waitGroup.Done()

	// TODO run application: frontends / backends
	// defer wg.Done()

	return nil
}

func (self *Application) Delete() (err error) {
	_, err = etcdClient.DeleteDir("/" + config.EtcdKey + "/" + self.Id)
	return err
}

func (self *Application) Stop() {
	// close(s.ch)
	// s.waitGroup.Wait()
}

type Backend struct {
	Url            string `"json:url"`
	ConnectTimeout int    `json:connect_timeout"`
}
