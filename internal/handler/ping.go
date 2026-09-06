package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/service"
)

// PingHandler HTTP 接口层，只负责参数处理和响应，不含业务逻辑。
type PingHandler struct {
	svc service.PingService
}

func NewPingHandler(svc service.PingService) *PingHandler {
	return &PingHandler{svc: svc}
}

// Ping godoc
// GET /ping
func (h *PingHandler) Ping(c *gin.Context) {
	resp, err := h.svc.Ping(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}
