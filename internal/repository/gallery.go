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
	Delete(ctx context.Context, id int64) (int64, error)
	Exists(ctx context.Context, id int64) (bool, error)
	GetByID(ctx context.Context, id int64) (*entity.Gallery, bool, error)
	GetURLsByIDs(ctx context.Context, ids []int64) (map[int64]string, error)
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

// Delete 按 id 删除单条 gallery 记录，返回受影响行数（0 表示记录不存在）。
func (r *galleryRepository) Delete(ctx context.Context, id int64) (int64, error) {
	res, err := r.db.ExecContext(ctx, "DELETE FROM gallery WHERE id = ?", id)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Exists 检查指定 id 的 gallery 记录是否存在。
func (r *galleryRepository) Exists(ctx context.Context, id int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM gallery WHERE id = ?)", id,
	).Scan(&exists)
	return exists, err
}

// GetByID 按 id 查询单条记录，found 为 false 表示记录不存在。
func (r *galleryRepository) GetByID(ctx context.Context, id int64) (*entity.Gallery, bool, error) {
	var g entity.Gallery
	err := r.db.QueryRowContext(ctx,
		"SELECT id, source, url, object_key, name, created_at FROM gallery WHERE id = ?", id,
	).Scan(&g.ID, &g.Source, &g.URL, &g.ObjectKey, &g.Name, &g.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &g, true, nil
}

// GetURLsByIDs 按 id 批量查询 url，返回 id -> url 映射，不存在的 id 不在结果中。
func (r *galleryRepository) GetURLsByIDs(ctx context.Context, ids []int64) (map[int64]string, error) {
	if len(ids) == 0 {
		return map[int64]string{}, nil
	}

	query := "SELECT id, url FROM gallery WHERE id IN (" + placeholders(len(ids)) + ")"
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64]string, len(ids))
	for rows.Next() {
		var id int64
		var url string
		if err := rows.Scan(&id, &url); err != nil {
			return nil, err
		}
		result[id] = url
	}
	return result, rows.Err()
}

// placeholders 生成 "?, ?, ..." 占位符，调用方需保证 n > 0。
func placeholders(n int) string {
	s := make([]byte, 0, n*2)
	for i := 0; i < n; i++ {
		if i > 0 {
			s = append(s, ',', ' ')
		}
		s = append(s, '?')
	}
	return string(s)
}
