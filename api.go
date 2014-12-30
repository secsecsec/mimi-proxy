package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"time"
)

type ApiServer struct {
	EnableLogging    bool
	EnableCheckAlive bool
	secureServer     *Server
	insecureServer   *Server
}

func (self *ApiServer) CheckAlive() {
	ticker := time.NewTicker(1 * time.Minute)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			log.Printf("Run periodic check alive")
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

func (self *ApiServer) ListenAndServe(listen string) {
	gin.SetMode(gin.ReleaseMode)

	if self.EnableCheckAlive {
		go self.CheckAlive()
	}

	r := gin.Default()

	// Global middlewares
	if self.EnableLogging {
		r.Use(gin.Logger())
	}
	r.Use(gin.Recovery())

	v1 := r.Group("/v1")
	{
		// Applications
		v1.GET("/", func(c *gin.Context) {
			c.JSON(200, collection.Applications)
		})
		v1.GET("/:id", func(c *gin.Context) {
			id := c.Params.ByName("id")
			if _, ok := collection.Applications[id]; ok {
				c.JSON(200, collection.Applications[id])
			} else {
				c.JSON(200, gin.H{
					"status": false,
					"error":  "application not found",
				})
			}
		})
		v1.POST("/", func(c *gin.Context) {
			type AppReq struct {
				Id string `json:"id"`
			}

			appJson := &AppReq{}

			if appJson.Id == "" {
				c.JSON(200, gin.H{
					"status": false,
					"error":  "missing id",
				})
			} else {
				if _, ok := collection.Applications[appJson.Id]; ok {
					c.JSON(200, gin.H{
						"status": false,
						"error":  "application already exists",
					})
				} else {
					app := NewApplication(appJson.Id)
					if err := app.Create(); err != nil {
						c.JSON(200, gin.H{
							"status": false,
							"error":  err,
						})
					} else {
						c.JSON(200, gin.H{
							"status": true,
						})
					}
				}
			}
		})
		v1.DELETE("/:id", func(c *gin.Context) {
			id := c.Params.ByName("id")
			if _, ok := collection.Applications[id]; ok {
				err := collection.Applications[id].Delete()
				if err != nil {
					c.JSON(200, gin.H{
						"status": false,
						"error":  err,
					})
				} else {
					c.JSON(200, gin.H{
						"status": true,
					})
				}
			} else {
				c.JSON(200, gin.H{
					"status": false,
					"error":  "application not found",
				})
			}
		})

		// Frontends
		v1.GET("/:id/frontend/:fid", func(c *gin.Context) {
			id := c.Params.ByName("id")
			fid := c.Params.ByName("fid")
			if app, ok := collection.Applications[id]; ok {
				if frontend, fok := app.Frontends[fid]; fok {
					c.JSON(200, frontend)
				} else {
					c.JSON(200, gin.H{
						"status": false,
						"error":  "Frontend not found",
					})
				}
			} else {
				c.JSON(200, gin.H{
					"status": false,
					"error":  "Application not found",
				})
			}
		})
		v1.POST("/:id/frontend/:fid", func(c *gin.Context) {
			id := c.Params.ByName("id")
			fid := c.Params.ByName("fid")
			if app, ok := collection.Applications[id]; ok {
				err := app.DeleteFrontend(fid)
				if err != nil {
					c.JSON(200, gin.H{
						"status": false,
						"error":  err,
					})
				}

				frontend := NewFrontend(fid)

				var tmp FrontendTmp
				c.Bind(tmp)

				frontend.Hosts = tmp.Hosts

				if tmp.TLSCrt != "" || tmp.TLSKey != "" {
					err := frontend.SetTLS(tmp.TLSCrt, tmp.TLSCrt)
					if err != nil {
						c.JSON(200, gin.H{
							"status": false,
							"err":    err,
						})
						return
					}
				}

				if frontend.isSecure() {
					self.secureServer.AddFrontend(frontend)
					go self.secureServer.RunFrontend(frontend)
				} else {
					self.insecureServer.AddFrontend(frontend)
					go self.insecureServer.RunFrontend(frontend)
				}

				c.JSON(200, gin.H{
					"status": true,
				})
			} else {
				c.JSON(200, gin.H{
					"status": false,
					"error":  "Application not found",
				})
			}
		})
		v1.DELETE("/:id/frontend/:fid", func(c *gin.Context) {
			id := c.Params.ByName("id")
			fid := c.Params.ByName("fid")
			if app, ok := collection.Applications[id]; ok {
				err := app.DeleteFrontend(fid)
				if err != nil {
					c.JSON(200, gin.H{
						"status": false,
						"error":  err,
					})
				} else {
					c.JSON(200, gin.H{
						"status": true,
					})
				}
			} else {
				c.JSON(200, gin.H{
					"status": false,
					"error":  "Application not found",
				})
			}
		})

		// Backends
		v1.GET("/:id/backend/:bid", func(c *gin.Context) {
			id := c.Params.ByName("id")
			bid := c.Params.ByName("bid")
			if app, ok := collection.Applications[id]; ok {
				if backend, fok := app.Backends[bid]; fok {
					c.JSON(200, backend)
				} else {
					c.JSON(200, gin.H{
						"status": false,
						"error":  "Backend not found",
					})
				}
			} else {
				c.JSON(200, gin.H{
					"status": false,
					"error":  "Application not found",
				})
			}
		})
		v1.POST("/:id/backend/:bid", func(c *gin.Context) {
			id := c.Params.ByName("id")
			bid := c.Params.ByName("bid")
			if app, ok := collection.Applications[id]; ok {
				err := app.DeleteFrontend(bid)
				if err != nil {
					c.JSON(200, gin.H{
						"status": false,
						"error":  err,
					})
				}

				backend := NewBackend(bid)

				var tmp BackendTmp
				c.Bind(tmp)

				if tmp.Url == "" {
					c.JSON(200, gin.H{
						"status": false,
						"error":  fmt.Sprintf("Skip backend with incorrect url %s", id),
					})
					return
				}
				backend.Url = tmp.Url

				if tmp.ConnectTimeout != 0 {
					backend.ConnectTimeout = tmp.ConnectTimeout
				}

				app.AddBackend(backend)

				c.JSON(200, gin.H{
					"status": true,
				})
			} else {
				c.JSON(200, gin.H{
					"status": false,
					"error":  "Application not found",
				})
			}
		})
		v1.DELETE("/:id/backend/:bid", func(c *gin.Context) {
			id := c.Params.ByName("id")
			bid := c.Params.ByName("bid")
			if app, ok := collection.Applications[id]; ok {
				err := app.DeleteBackend(bid)
				if err != nil {
					c.JSON(200, gin.H{
						"status": false,
						"error":  err,
					})
				} else {
					c.JSON(200, gin.H{
						"status": true,
					})
				}
			} else {
				c.JSON(200, gin.H{
					"status": false,
					"error":  "Application not found",
				})
			}
		})
	}

	s := &http.Server{
		Addr:           listen,
		Handler:        r,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	s.ListenAndServe()
}
