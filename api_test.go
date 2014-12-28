package main

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"testing"
)

var (
	srvAddr     string = "127.0.0.1:8080"
	httpSrvAddr string = "http://127.0.0.1:8080/v1/"
)

func readResp(resp *http.Response) (string, error) {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func makePostRequest(url, json string) (string, error) {
	tr := &http.Transport{
		// TLSClientConfig:    &tls.Config{RootCAs: pool},
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr}

	var jsonStr = []byte(json)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := readResp(resp)
	return string(body), nil
}

func runApiServer(apps map[string]*Application) {
	gin.SetMode(gin.TestMode)

	srv := &ApiServer{
		Applications:     apps,
		EnableLogging:    false,
		EnableCheckAlive: false,
	}
	srv.Run(srvAddr)
}

func checkErr(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Failed to accept new connection: %v", err)
	}
}

func TestCreateApplication(t *testing.T) {
	apps := make(map[string]*Application)
	go runApiServer(apps)

	var body string
	var err error

	body, err = makePostRequest(httpSrvAddr, `{"id":"1"}`)
	checkErr(t, err)
	assert.Equal(t, body, `{"status":true}
`)

	body, err = makePostRequest(httpSrvAddr, ``)
	checkErr(t, err)
	assert.Equal(t, body, `{"error":"missing id","status":false}
`)
}
