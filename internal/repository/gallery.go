package repository

import (
	"context"
	"database/sql"

	"gin-quickstart/internal/entity"
)

// GalleryRepository gallery 表数据访问层。
type GalleryRepository interface {
	Create(ctx context.Context, g *entity.Gallery) (int64, error)
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
