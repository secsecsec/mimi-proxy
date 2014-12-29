package main

type Backend struct {
	Id             string `json:"id"`
	Url            string `"json:url"`
	ConnectTimeout int    `json:connect_timeout"`
}

func NewBackend(id string) Backend {
	backend := Backend{
		Id:             id,
		ConnectTimeout: defaultConnectTimeout,
	}
	return backend
}
