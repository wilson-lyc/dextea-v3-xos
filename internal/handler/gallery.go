package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/ecode"
	"gin-quickstart/internal/service"
	"gin-quickstart/pkg/response"
)

// GalleryHandler gallery 接口层。
type GalleryHandler struct {
	svc service.GalleryService
}

func NewGalleryHandler(svc service.GalleryService) *GalleryHandler {
	return &GalleryHandler{svc: svc}
}

// ListPage 分页查询 gallery 表。
// 入参：page（页码，必填，从 1 开始）、page_size（页大小，必填，正整数）。
func (h *GalleryHandler) ListPage(c *gin.Context) {
	page, err := strconv.Atoi(c.Query("page"))
	if err != nil || page < 1 {
		response.Fail(c, ecode.InvalidParam)
		return
	}
	pageSize, err := strconv.Atoi(c.Query("page_size"))
	if err != nil || pageSize < 1 {
		response.Fail(c, ecode.InvalidParam)
		return
	}

	result, err := h.svc.ListPage(c.Request.Context(), page, pageSize)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, result)
}

// Delete 按 id 单删 gallery 记录，路径参数 :id 为记录主键。
func (h *GalleryHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		response.Fail(c, ecode.InvalidParam)
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, nil)
}
