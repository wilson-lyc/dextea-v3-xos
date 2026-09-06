package service

import (
	"context"

	"gin-quickstart/internal/dto"
	"gin-quickstart/internal/repository"
)

// PingService 业务逻辑层。
type PingService interface {
	Ping(ctx context.Context) (*dto.PingResp, error)
}

type pingService struct {
	repo repository.PingRepository
}

func NewPingService(repo repository.PingRepository) PingService {
	return &pingService{repo: repo}
}

func (s *pingService) Ping(ctx context.Context) (*dto.PingResp, error) {
	if err := s.repo.Ping(ctx); err != nil {
		return nil, err
	}
	// 实际场景中此处应将 entity 转换为 dto，避免内部实体直接暴露给接口层
	return &dto.PingResp{Message: "pong"}, nil
}
