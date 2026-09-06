package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/gabriel-vasile/mimetype"

	"gin-quickstart/internal/cache"
	"gin-quickstart/internal/config"
	"gin-quickstart/internal/dto"
	"gin-quickstart/internal/entity"
	"gin-quickstart/internal/provider"
	"gin-quickstart/internal/repository"
)

// allowedImageMimes 允许上传的图片类型白名单，基于文件头嗅探结果判断。
var allowedImageMimes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
	"image/bmp":  true,
}

// sniffSize MIME 嗅探所需读取的字节数（mimetype 官方建议值）。
const sniffSize = 512

// UploadService 上传业务逻辑层，屏蔽存储源差异。
type UploadService interface {
	Upload(ctx context.Context, source, bucket, objectKey string, in provider.UploadInput) (*dto.UploadResp, error)
}

type uploadService struct {
	manager     *provider.Manager
	galleryRepo repository.GalleryRepository
	cache       *cache.Cache // 用于上传落库后失效图库列表缓存
	maxSize     int64
}

// NewUploadService 初始化上传服务并绑定图库落库。
func NewUploadService(manager *provider.Manager, galleryRepo repository.GalleryRepository, c *cache.Cache, maxSize int64) (UploadService, error) {
	return &uploadService{
		manager:     manager,
		galleryRepo: galleryRepo,
		cache:       c,
		maxSize:     maxSize,
	}, nil
}

func (s *uploadService) Upload(ctx context.Context, source, bucket, objectKey string, in provider.UploadInput) (*dto.UploadResp, error) {
	p, spec, ok := s.manager.Get(source)
	if !ok {
		return nil, ErrUnknownSource(source)
	}

	if err := s.validate(&in); err != nil {
		return nil, err
	}

	if objectKey == "" {
		objectKey = generateObjectKey(in.FileName)
	}
	in.ObjectKey = objectKey

	res, err := p.Upload(ctx, bucket, in)
	if err != nil {
		return nil, ErrUploadFailed(source, err)
	}

	// 上传成功后写入图库表；落库失败不影响上传结果，仅记录日志
	url := buildObjectURL(spec, res.Bucket, res.ObjectKey)
	if _, err := s.galleryRepo.Create(ctx, &entity.Gallery{
		Source:    source,
		URL:       url,
		ObjectKey: res.ObjectKey,
		Name:      path.Base(in.FileName),
	}); err != nil {
		fmt.Printf("[WARN] gallery insert failed, bucket=%s key=%s: %v\n", res.Bucket, res.ObjectKey, err)
	} else {
		// 旁路缓存的写后失效：新增记录会使分页列表（尤其是首页）过期
		if err := s.cache.Del(ctx, listVerKey); err != nil {
			fmt.Printf("[WARN] invalidate gallery list cache failed: %v\n", err)
		}
	}

	return &dto.UploadResp{
		Bucket:    res.Bucket,
		ObjectKey: res.ObjectKey,
		Size:      res.Size,
		ETag:      res.ETag,
	}, nil
}

// validate 校验文件大小与图片类型，嗅探文件头而非信任客户端 Content-Type。
// 校验通过后把嗅探读出的字节拼回流头部，并回填真实 Content-Type。
func (s *uploadService) validate(in *provider.UploadInput) error {
	if s.maxSize > 0 && in.Size > s.maxSize {
		return ErrFileTooLarge(in.Size, s.maxSize)
	}

	head := make([]byte, sniffSize)
	n, err := io.ReadFull(in.Reader, head)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return ErrFileTypeDenied("unreadable")
	}
	head = head[:n]

	mtype := mimetype.Detect(head)
	if !allowedImageMimes[mtype.String()] {
		return ErrFileTypeDenied(mtype.String())
	}

	in.Reader = io.MultiReader(bytes.NewReader(head), in.Reader)
	in.ContentType = mtype.String()
	return nil
}

// buildObjectURL 根据存储源配置拼接对象的访问地址。
func buildObjectURL(spec config.SourceSpec, bucket, objectKey string) string {
	scheme := "http"
	if spec.UseSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/%s/%s", scheme, spec.Endpoint, bucket, objectKey)
}

// generateObjectKey 未指定 objectKey 时，按日期 + 时间戳 + 文件名生成，避免覆盖。
func generateObjectKey(fileName string) string {
	name := path.Base(fileName)
	now := time.Now()
	return fmt.Sprintf("%s/%d_%s", now.Format("2006/01/02"), now.UnixNano(), name)
}
