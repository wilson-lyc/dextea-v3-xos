package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"gin-quickstart/internal/config"
	"gin-quickstart/internal/router"
	"gin-quickstart/internal/telemetry"
)

func main() {
	// 加载本地 .env（不存在时忽略，部署环境由容器/编排层注入环境变量）
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("load .env: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	gin.SetMode(cfg.Server.Mode)

	// 初始化 OpenTelemetry SDK，退出前 flush 未导出的 span
	ctx := context.Background()
	if cfg.Telemetry.Enable {
		otelShutdown, err := telemetry.Setup(ctx, cfg.Telemetry.ServiceName, cfg.Telemetry.ServiceVersion)
		if err != nil {
			log.Fatalf("setup otel sdk: %v", err)
		}
		defer func() {
			if err := otelShutdown(context.Background()); err != nil {
				log.Printf("otel shutdown: %v", err)
			}
		}()
	}

	r, err := router.Setup(cfg)
	if err != nil {
		log.Fatalf("setup router: %v", err)
	}

	srv := &http.Server{
		Addr:    cfg.Server.Host + ":" + cfg.Server.Port,
		Handler: r,
	}

	go func() {
		log.Printf("listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down server ...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	log.Println("server exiting")
}
