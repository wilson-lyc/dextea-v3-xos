package repository

import (
	"context"
	"database/sql"

	"gin-quickstart/internal/entity"
)

// GalleryRepository gallery 表数据访问层。
type GalleryRepository interface {
	Create(ctx context.Context, g *entity.Gallery) (int64, error)
	ListPage(ctx context.Context, page, pageSize int) ([]entity.Gallery, int64, error)
}

type galleryRepository struct {
	db *sql.DB
}

func NewGalleryRepository(db *sql.DB) GalleryRepository {
	return &galleryRepository{db: db}
}

func (r *galleryRepository) Create(ctx context.Context, g *entity.Gallery) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		"INSERT INTO gallery (source, url, object_key, name) VALUES (?, ?, ?, ?)",
		g.Source, g.URL, g.ObjectKey, g.Name,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListPage 分页查询 gallery 表，按创建时间倒序，返回当前页记录与总条数。
func (r *galleryRepository) ListPage(ctx context.Context, page, pageSize int) ([]entity.Gallery, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM gallery").Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx,
		"SELECT id, source, url, object_key, name, created_at FROM gallery ORDER BY id DESC LIMIT ? OFFSET ?",
		pageSize, (page-1)*pageSize,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]entity.Gallery, 0, pageSize)
	for rows.Next() {
		var g entity.Gallery
		if err := rows.Scan(&g.ID, &g.Source, &g.URL, &g.ObjectKey, &g.Name, &g.CreatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, g)
	}
	return list, total, rows.Err()
}
