package main

type Collection struct {
	Applications map[string]*Application
	Backends     map[string]Backend
	Frontends    map[string]*Frontend
}

func NewApplication(Id string, Frontends []*Frontend, Backends []*Backend) *Application {
	app := &Application{
		Id:        Id,
		Frontends: Frontends,
		Backends:  Backends,
	}
	return app
}

type Application struct {
	Id        string      `json:"id"`
	Frontends []*Frontend `json:"frontends"`
	Backends  []*Backend  `json:"backends"`
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

type Backend struct {
	Url            string `"json:url"`
	ConnectTimeout int    `json:connect_timeout"`
}
