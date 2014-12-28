package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"time"
)

type ApiServer struct {
	Applications     map[string]*Application
	EnableLogging    bool
	EnableCheckAlive bool
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
			c.JSON(200, self.Applications)
		})
		v1.GET("/:id", func(c *gin.Context) {

		})
		v1.GET("/:id/stats", func(c *gin.Context) {

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
				if _, ok := self.Applications[appJson.Id]; ok {
					c.JSON(200, gin.H{
						"status": false,
						"error":  "application already exists",
					})
				} else {
					app := Application{
						Id: appJson.Id,
					}
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

		})

		// Frontends
		v1.GET("/:id/frontend/:fid", func(c *gin.Context) {

		})
		v1.POST("/:id/frontend", func(c *gin.Context) {

		})
		v1.PUT("/:id/frontend/:fid", func(c *gin.Context) {

		})
		v1.DELETE("/:id/frontend/:fid", func(c *gin.Context) {

		})

		// Backends
		v1.GET("/:id/backend/:bid", func(c *gin.Context) {

		})
		v1.POST("/:id/backend", func(c *gin.Context) {

		})
		v1.PUT("/:id/backend/:bid", func(c *gin.Context) {

		})
		v1.DELETE("/:id/backend/:bid", func(c *gin.Context) {

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
