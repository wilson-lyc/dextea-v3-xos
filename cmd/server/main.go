package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	pb "github.com/wilson-lyc/dextea-v3-proto/gen/go/xos/v1"
	"github.com/wilson-lyc/dextea-v3-xos/internal/cache"
	"github.com/wilson-lyc/dextea-v3-xos/internal/config"
	"github.com/wilson-lyc/dextea-v3-xos/internal/provider"
	_ "github.com/wilson-lyc/dextea-v3-xos/internal/provider/s3"
	"github.com/wilson-lyc/dextea-v3-xos/internal/registry"
	"github.com/wilson-lyc/dextea-v3-xos/internal/repository"
	"github.com/wilson-lyc/dextea-v3-xos/internal/rpc"
	"github.com/wilson-lyc/dextea-v3-xos/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
func run() error {
	configPath := flag.String("config", "configs/config.yaml", "配置文件路径")
	flag.Parse()
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	db, err := sql.Open("mysql", cfg.MySQL.DSN())
	if err != nil {
		return err
	}
	defer db.Close()
	startup, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.PingContext(startup); err != nil {
		return err
	}
	c := cache.NewRedis(cfg.Redis.Addr(), cfg.Redis.Password, cfg.Redis.DB)
	defer c.Close()
	if err := c.Ping(startup); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}
	manager, err := provider.NewManager(cfg.Storage.Sources)
	if err != nil {
		return err
	}
	repo := repository.NewGalleryRepository(db)
	upload, err := service.NewUploadService(manager, repo, c, cfg.Storage.MaxUploadSize)
	if err != nil {
		return err
	}
	// Allow protobuf metadata in addition to image bytes; enforce exact image limit in the handler.
	maxMessage := cfg.Storage.MaxUploadSize + (64 << 10)
	if maxMessage <= 0 || maxMessage > int64(^uint(0)>>1) {
		return fmt.Errorf("invalid max upload size")
	}
	srv := grpc.NewServer(grpc.MaxRecvMsgSize(int(maxMessage)), grpc.UnaryInterceptor(rpc.UnaryInterceptor))
	pb.RegisterXOSServiceServer(srv, rpc.NewServer(upload, service.NewGalleryService(repo, manager, c, cfg.Redis.TTL), cfg.Storage.MaxUploadSize))
	h := health.NewServer()
	healthpb.RegisterHealthServer(srv, h)
	h.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	h.SetServingStatus(pb.XOSService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_SERVING)
	listener, err := net.Listen("tcp", cfg.Server.Addr)
	if err != nil {
		return err
	}
	defer listener.Close()
	errs := make(chan error, 1)
	go func() { errs <- srv.Serve(listener) }()
	log.Printf("dextea-xos gRPC listening on %s", cfg.Server.Addr)
	var reg *registry.Registrar
	if cfg.Nacos.Enabled {
		reg, err = registry.Register(cfg.Nacos, cfg.Server.Addr)
		if err != nil {
			log.Printf("[warn] register dextea-xos to nacos failed; service remains available: %v", err)
		}
	}
	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
	}
	h.Shutdown()
	if reg != nil {
		reg.Deregister()
	}
	done := make(chan struct{})
	go func() { srv.GracefulStop(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		srv.Stop()
		<-done
	}
	return nil
}
