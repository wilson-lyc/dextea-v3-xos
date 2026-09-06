package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/ecode"
	"gin-quickstart/internal/service"
	"gin-quickstart/pkg/response"
)

// maxURLQueryIDs 批量查询 url 时允许的最大 id 数量。
const maxURLQueryIDs = 100

// GalleryHandler gallery 接口层。
type GalleryHandler struct {
	svc service.GalleryService
}

func NewGalleryHandler(svc service.GalleryService) *GalleryHandler {
	return &GalleryHandler{svc: svc}
}

// ListPage 分页查询 gallery 表。
// 入参：page（页码，必填，从 1 开始）、pageSize（页大小，必填，正整数）。
func (h *GalleryHandler) ListPage(c *gin.Context) {
	page, err := strconv.Atoi(c.Query("page"))
	if err != nil || page < 1 {
		response.Fail(c, ecode.InvalidParam)
		return
	}
	pageSize, err := strconv.Atoi(c.Query("pageSize"))
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

// GetURLs 批量根据 id 查询 url，请求体为 {ids: [int64]}，最多 100 个。
// 返回 id -> url 映射，不存在的 id 不在结果中。
func (h *GalleryHandler) GetURLs(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 || len(req.IDs) > maxURLQueryIDs {
		response.Fail(c, ecode.InvalidParam)
		return
	}
	for _, id := range req.IDs {
		if id < 1 {
			response.Fail(c, ecode.InvalidParam)
			return
		}
	}

	result, err := h.svc.GetURLsByIDs(c.Request.Context(), req.IDs)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, gin.H{"urls": result})
}

// ValidateID 校验 id 是否合法，路径参数 :id 为待校验的主键。
// 合法（记录存在）返回 data: {valid: true}，不存在返回 {valid: false}。
func (h *GalleryHandler) ValidateID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		response.Fail(c, ecode.InvalidParam)
		return
	}

	valid, err := h.svc.ValidateID(c.Request.Context(), id)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, gin.H{"id": id, "valid": valid})
}
