package service

import (
	"context"
	"fmt"
	"path"
	"time"

	"gin-quickstart/internal/config"
	"gin-quickstart/internal/dto"
	"gin-quickstart/internal/entity"
	"gin-quickstart/internal/provider"
	"gin-quickstart/internal/repository"
)

// UploadService 上传业务逻辑层，屏蔽存储源差异。
type UploadService interface {
	Upload(ctx context.Context, source, bucket, objectKey string, in provider.UploadInput) (*dto.UploadResp, error)
}

type uploadService struct {
	providers   map[string]provider.ObjectProvider // key 为存储源名称
	specs       map[string]config.SourceSpec
	galleryRepo repository.GalleryRepository
}

// NewUploadService 初始化所有存储源的 provider 并绑定图库落库。
// 协议固定为 s3；后续接入非 S3 协议厂商时，可依据 spec.Vendor/新增协议字段路由到不同 provider。
func NewUploadService(sources map[string]config.SourceSpec, galleryRepo repository.GalleryRepository) (UploadService, error) {
	providers := make(map[string]provider.ObjectProvider, len(sources))
	for name, spec := range sources {
		// 当前所有源统一走 S3 协议；扩展点：按 spec 声明的协议选择 provider
		p, err := provider.New("s3", spec)
		if err != nil {
			return nil, fmt.Errorf("init storage source %q: %w", name, err)
		}
		providers[name] = p
	}
	return &uploadService{
		providers:   providers,
		specs:       sources,
		galleryRepo: galleryRepo,
	}, nil
}

func (s *uploadService) Upload(ctx context.Context, source, bucket, objectKey string, in provider.UploadInput) (*dto.UploadResp, error) {
	p, ok := s.providers[source]
	if !ok {
		return nil, ErrUnknownSource(source)
	}
	spec := s.specs[source]

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
	}

	return &dto.UploadResp{
		Bucket:    res.Bucket,
		ObjectKey: res.ObjectKey,
		Size:      res.Size,
		ETag:      res.ETag,
	}, nil
}

// buildObjectURL 根据存储源配置拼接对象的访问地址。
func buildObjectURL(spec config.SourceSpec, bucket, objectKey string) string {
	scheme := "http"
	if spec.UseSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/%s/%s", scheme, spec.Endpoint, bucket, objectKey)
}

// generateObjectKey 未指定 object_key 时，按日期 + 时间戳 + 文件名生成，避免覆盖。
func generateObjectKey(fileName string) string {
	name := path.Base(fileName)
	now := time.Now()
	return fmt.Sprintf("%s/%d_%s", now.Format("2006/01/02"), now.UnixNano(), name)
}
