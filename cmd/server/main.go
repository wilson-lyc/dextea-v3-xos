package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	pb "github.com/wilson-lyc/dextea-v3-proto/gen/go/xos/v1"
	"github.com/wilson-lyc/dextea-v3-xos/internal/cache"
	"github.com/wilson-lyc/dextea-v3-xos/internal/config"
	"github.com/wilson-lyc/dextea-v3-xos/internal/provider"
	_ "github.com/wilson-lyc/dextea-v3-xos/internal/provider/s3"
	"github.com/wilson-lyc/dextea-v3-xos/internal/repository"
	"github.com/wilson-lyc/dextea-v3-xos/internal/rpc"
	"github.com/wilson-lyc/dextea-v3-xos/internal/service"
	"github.com/wilson-lyc/dextea-v3-xos/internal/telemetry"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
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
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if cfg.Telemetry.Enable {
		shutdown, err := telemetry.Setup(ctx, cfg.Telemetry.ServiceName, cfg.Telemetry.ServiceVersion)
		if err != nil {
			return err
		}
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := shutdown(ctx); err != nil {
				log.Printf("telemetry shutdown: %v", err)
			}
		}()
	}
	db, err := sql.Open("mysql", cfg.Database.DSN())
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
	srv := grpc.NewServer(grpc.MaxRecvMsgSize(int(maxMessage)), grpc.UnaryInterceptor(rpc.UnaryInterceptor), grpc.StatsHandler(otelgrpc.NewServerHandler()))
	pb.RegisterXOSServiceServer(srv, rpc.NewServer(upload, service.NewGalleryService(repo, manager, c, cfg.Redis.TTL), cfg.Storage.MaxUploadSize))
	h := health.NewServer()
	healthpb.RegisterHealthServer(srv, h)
	h.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	h.SetServingStatus(pb.XOSService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_SERVING)
	listener, err := net.Listen("tcp", net.JoinHostPort(cfg.RPC.Host, cfg.RPC.Port))
	if err != nil {
		return err
	}
	defer listener.Close()
	errs := make(chan error, 1)
	go func() { errs <- srv.Serve(listener) }()
	log.Printf("XOS gRPC listening on %s", listener.Addr())
	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
	}
	h.Shutdown()
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
