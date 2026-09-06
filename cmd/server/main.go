package main

import (
	"log"

	"gin-quickstart/internal/config"
	"gin-quickstart/internal/router"
)

func main() {
	cfg := config.Load()

	r := router.Setup()

	addr := cfg.Host + ":" + cfg.Port
	if err := r.Run(addr); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
