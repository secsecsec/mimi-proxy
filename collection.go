package main

type Collection struct {
	Applications map[string]*Application
	Backends     map[string]Backend
	Frontends    map[string]*Frontend
}

func NewCollection() *Collection {
	return &Collection{
		Applications: make(map[string]*Application),
		Backends:     make(map[string]Backend),
		Frontends:    make(map[string]*Frontend),
	}
}

func (c *Collection) AddApplication(app *Application) {
	c.Applications[app.Id] = app
}
