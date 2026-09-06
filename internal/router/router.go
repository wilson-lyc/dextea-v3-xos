package router

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"gin-quickstart/internal/config"
	"gin-quickstart/internal/handler"
	"gin-quickstart/internal/middleware"
	_ "gin-quickstart/internal/provider/s3" // 注册 s3 协议 provider
	"gin-quickstart/internal/repository"
	"gin-quickstart/internal/service"
	"gin-quickstart/pkg/response"
)

// Setup 组装依赖并注册路由。
func Setup(cfg *config.Config) (*gin.Engine, error) {
	db, err := sql.Open("mysql", cfg.Database.DSN())
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}

	r := gin.New()
	// otelgin 放在最前，保证后续中间件与 handler 都运行在请求 span 内
	r.Use(otelgin.Middleware(cfg.Telemetry.ServiceName))
	r.Use(gin.Logger(), middleware.Recovery(), middleware.RequestID())
	r.NoRoute(middleware.NoRoute())
	r.NoMethod(middleware.NoMethod())

	// 依赖注入：config -> db/provider -> repository/service -> handler
	galleryRepo := repository.NewGalleryRepository(db)
	uploadSvc, err := service.NewUploadService(cfg.Storage.Sources, galleryRepo)
	if err != nil {
		return nil, err
	}
	uploadHandler := handler.NewUploadHandler(uploadSvc)
	galleryHandler := handler.NewGalleryHandler(service.NewGalleryService(galleryRepo))

	// 健康检查
	r.GET("/ping", func(c *gin.Context) {
		response.OK(c, gin.H{"message": "pong"})
	})

	api := r.Group("/api/v1")
	{
		// :source 为存储源名称，对应 configs/config.yaml 中 storage.sources 的 key
		api.POST("/storage/:source/objects", uploadHandler.Upload)
		api.GET("/gallery", galleryHandler.ListPage)
		api.DELETE("/gallery/:id", galleryHandler.Delete)
	}

	return r, nil
}
