package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/ecode"
)

// Response 统一响应结构。
type Response struct {
	Code    int         `json:"code"` // 0 成功，非 0 为业务错误码
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: 0, Message: "success", Data: data})
}

// Fail 按 BizError 中定义的 HTTP 状态码与业务错误码返回。
func Fail(c *gin.Context, e *ecode.BizError) {
	if c.Writer.Written() {
		return
	}
	c.JSON(e.HTTPStatus, Response{Code: e.Code, Message: e.Message})
}

// Err handler 内的统一错误出口：service 返回的 error（含包装链）在此
// 转换为统一响应格式，未知错误归一化为 50000 Internal。
func Err(c *gin.Context, err error) {
	Fail(c, ecode.From(err))
}
