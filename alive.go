package main

import (
	"log"
	"time"
)

func checkAlive() {
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
