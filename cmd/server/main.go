package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/config"
	"gin-quickstart/internal/router"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	gin.SetMode(cfg.Server.Mode)

	r, err := router.Setup(cfg)
	if err != nil {
		log.Fatalf("setup router: %v", err)
	}

	addr := cfg.Server.Host + ":" + cfg.Server.Port
	log.Printf("listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
