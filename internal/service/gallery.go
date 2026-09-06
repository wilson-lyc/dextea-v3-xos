package service

import (
	"context"

	"gin-quickstart/internal/ecode"
	"gin-quickstart/internal/entity"
	"gin-quickstart/internal/repository"
)

// GalleryPageResult 分页查询结果。
type GalleryPageResult struct {
	List       []entity.Gallery `json:"list"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int64            `json:"total_pages"`
}

// GalleryService gallery 表业务接口。
type GalleryService interface {
	ListPage(ctx context.Context, page, pageSize int) (*GalleryPageResult, error)
	Delete(ctx context.Context, id int64) error
	ValidateID(ctx context.Context, id int64) (bool, error)
}

type galleryService struct {
	repo repository.GalleryRepository
}

func NewGalleryService(repo repository.GalleryRepository) GalleryService {
	return &galleryService{repo: repo}
}

// ListPage 分页查询 gallery 表。
func (s *galleryService) ListPage(ctx context.Context, page, pageSize int) (*GalleryPageResult, error) {
	list, total, err := s.repo.ListPage(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}
	totalPages := (total + int64(pageSize) - 1) / int64(pageSize)
	return &GalleryPageResult{
		List:       list,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// Delete 按 id 单删 gallery 记录，记录不存在时返回 NotFound 业务错误。
func (s *galleryService) Delete(ctx context.Context, id int64) error {
	affected, err := s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ecode.NotFound
	}
	return nil
}

// ValidateID 校验 id 是否合法，即 gallery 表中是否存在该记录。
func (s *galleryService) ValidateID(ctx context.Context, id int64) (bool, error) {
	return s.repo.Exists(ctx, id)
}
