package router

import (
	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/handler"
	"gin-quickstart/internal/middleware"
	"gin-quickstart/internal/repository"
	"gin-quickstart/internal/service"
)

// Setup 组装依赖并注册路由。
func Setup() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middleware.RequestID())

	// 依赖注入：repository -> service -> handler
	pingRepo := repository.NewPingRepository()
	pingSvc := service.NewPingService(pingRepo)
	pingHandler := handler.NewPingHandler(pingSvc)

	api := r.Group("/api/v1")
	{
		api.GET("/ping", pingHandler.Ping)
	}

	// 健康检查
	r.GET("/ping", pingHandler.Ping)

	return r
}
