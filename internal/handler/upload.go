package handler

import (
	"github.com/gin-gonic/gin"

	"gin-quickstart/internal/ecode"
	"gin-quickstart/internal/provider"
	"gin-quickstart/internal/service"
	"gin-quickstart/pkg/response"
)

// UploadHandler 上传接口层。
type UploadHandler struct {
	svc service.UploadService
}

func NewUploadHandler(svc service.UploadService) *UploadHandler {
	return &UploadHandler{svc: svc}
}

// Upload 处理 multipart 文件上传。
// 路径参数 :source 指定对象存储源（yml 中 storage.sources 的 key）；
// 表单字段：file（必填）、bucket（可选，缺省用源的 default-bucket）、objectKey（可选）。
func (h *UploadHandler) Upload(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, ecode.InvalidParam.WithCause(err))
		return
	}

	f, err := fh.Open()
	if err != nil {
		response.Fail(c, ecode.InvalidParam.WithCause(err))
		return
	}
	defer f.Close()

	resp, err := h.svc.Upload(c.Request.Context(), c.Param("source"), c.PostForm("bucket"), c.PostForm("objectKey"), provider.UploadInput{
		FileName:    fh.Filename,
		Reader:      f,
		Size:        fh.Size,
		ContentType: fh.Header.Get("Content-Type"),
	})
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, resp)
}
