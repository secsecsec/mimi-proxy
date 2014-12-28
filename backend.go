package main

type Backend struct {
	Url            string `"json:url"`
	ConnectTimeout int    `json:connect_timeout"`
}
