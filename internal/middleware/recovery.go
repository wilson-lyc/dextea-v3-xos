package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/ecode"
	"gin-quickstart/pkg/response"
)

// Recovery 替代 gin.Recovery：panic 时以统一响应格式返回 500，
// 底层堆栈仅写日志，不暴露给调用方。
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[panic] request_id=%s err=%v\n%s",
					requestID(c), r, debug.Stack())
				response.Fail(c, ecode.Internal)
				c.Abort()
			}
		}()
		c.Next()
	}
}

// NoRoute / NoMethod 将 404、405 也纳入统一响应格式。
func NoRoute() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotFound, response.Response{
			Code: ecode.NotFound.Code, Message: ecode.NotFound.Message,
		})
	}
}

func NoMethod() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, response.Response{
			Code: ecode.MethodNotAllow.Code, Message: ecode.MethodNotAllow.Message,
		})
	}
}

func requestID(c *gin.Context) string {
	if id, ok := c.Get("request_id"); ok {
		if s, ok := id.(string); ok {
			return s
		}
	}
	return "-"
}
