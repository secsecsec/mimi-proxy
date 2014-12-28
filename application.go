package main

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
	_, err = etcdClient.DeleteDir("/" + config.EtcdKey + "/" + self.Id)
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
