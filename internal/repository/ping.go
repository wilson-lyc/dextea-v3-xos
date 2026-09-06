package repository

import "context"

// PingRepository 数据访问层接口，后续可替换为数据库实现。
type PingRepository interface {
	Ping(ctx context.Context) error
}

type pingRepository struct{}

func NewPingRepository() PingRepository {
	return &pingRepository{}
}

func (r *pingRepository) Ping(ctx context.Context) error {
	return nil
}
