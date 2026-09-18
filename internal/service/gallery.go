package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/wilson-lyc/dextea-v3-xos/internal/cache"
	"github.com/wilson-lyc/dextea-v3-xos/internal/ecode"
	"github.com/wilson-lyc/dextea-v3-xos/internal/entity"
	"github.com/wilson-lyc/dextea-v3-xos/internal/provider"
	"github.com/wilson-lyc/dextea-v3-xos/internal/repository"
)

// GalleryPageResult 分页查询结果。
type GalleryPageResult struct {
	List       []entity.Gallery `json:"list"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"pageSize"`
	TotalPages int64            `json:"totalPages"`
}

// galleryInfo 单条图库记录的缓存载荷，同时服务于存在性校验与 url 查询。
type galleryInfo struct {
	Exists bool   `json:"exists"`
	URL    string `json:"url"`
}

// GalleryService gallery 表业务接口。
type GalleryService interface {
	ListPage(ctx context.Context, page, pageSize int) (*GalleryPageResult, error)
	Delete(ctx context.Context, id int64) error
	ValidateID(ctx context.Context, id int64) (bool, error)
	GetURLsByIDs(ctx context.Context, ids []int64) (map[int64]string, error)
	UpdateName(ctx context.Context, id int64, name string) (*entity.Gallery, bool, error)
}

func (s *galleryService) UpdateName(ctx context.Context, id int64, name string) (*entity.Gallery, bool, error) {
	g, ok, e := s.repo.UpdateName(ctx, id, name)
	if e != nil || !ok {
		return g, ok, e
	}
	_ = s.cache.Del(ctx, listVerKey)
	_ = s.cache.Del(ctx, infoKey(id))
	return g, true, nil
}

type galleryService struct {
	repo    repository.GalleryRepository
	cache   *cache.Cache
	manager *provider.Manager // 删除记录时同步清理对象存储中的文件
	infoTTL time.Duration
	listTTL time.Duration
}

// 缓存 key 约定：单条记录 gallery:info:{id}；列表通过版本号 key 失效，
// 写操作删除版本 key，下次读自增出新版本，旧列表 key 自然过期。
const (
	infoKeyPrefix = "gallery:info:"
	listVerKey    = "gallery:list:ver"
	listKeyFmt    = "gallery:list:%d:%d:%d" // ver:page:page_size
)

func NewGalleryService(repo repository.GalleryRepository, manager *provider.Manager, c *cache.Cache, ttl time.Duration) GalleryService {
	return &galleryService{
		repo:    repo,
		cache:   c,
		manager: manager,
		infoTTL: ttl,
		listTTL: ttl,
	}
}

func infoKey(id int64) string {
	return infoKeyPrefix + strconv.FormatInt(id, 10)
}

func (s *galleryService) listKey(ver int64, page, pageSize int) string {
	return fmt.Sprintf(listKeyFmt, ver, page, pageSize)
}

// ListPage 分页查询 gallery 表，旁路缓存：读版本号 -> 命中直接返回，未命中查库回填。
func (s *galleryService) ListPage(ctx context.Context, page, pageSize int) (*GalleryPageResult, error) {
	key := ""
	ver, err := s.cache.Incr(ctx, listVerKey)
	if err == nil {
		key = s.listKey(ver, page, pageSize)
		var cached GalleryPageResult
		if hit, err := s.cache.Get(ctx, key, &cached); err == nil && hit {
			return &cached, nil
		}
	} else {
		// Redis 不可用时降级查库，版本号留待下次读取再补
		s.logCacheErr("incr list version", err)
	}

	list, total, err := s.repo.ListPage(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}
	totalPages := (total + int64(pageSize) - 1) / int64(pageSize)
	result := &GalleryPageResult{
		List:       list,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
	if key != "" {
		if err := s.cache.Set(ctx, key, result, s.listTTL); err != nil {
			s.logCacheErr("set list cache", err)
		}
	}
	return result, nil
}

// Delete 按 id 删除 gallery 记录，并同步删除对象存储中的文件。
// 记录不存在时返回 NotFound；对象删除失败时不删记录，保证可重试。
func (s *galleryService) Delete(ctx context.Context, id int64) error {
	g, found, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !found {
		return ecode.NotFound
	}

	// 先删对象再删记录：对象删除失败则整体失败，客户端重试时记录仍在
	if p, _, ok := s.manager.Get(g.Source); ok {
		if err := p.Delete(ctx, "", g.ObjectKey); err != nil {
			return ErrStorageDeleteFailed(g.Source, g.ObjectKey, err)
		}
	} else {
		// 存储源已下线等场景：记录仍可删除，孤儿对象留给对账清理
		fmt.Printf("[WARN] source %q not found, skip object delete, key=%s\n", g.Source, g.ObjectKey)
	}

	affected, err := s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ecode.NotFound
	}
	if err := s.cache.Del(ctx, infoKey(id), listVerKey); err != nil {
		s.logCacheErr("invalidate after delete", err)
	}
	return nil
}

// ValidateID 校验 id 是否合法，即 gallery 表中是否存在该记录。
func (s *galleryService) ValidateID(ctx context.Context, id int64) (bool, error) {
	info, ok, err := s.loadInfo(ctx, id)
	if err != nil {
		return false, err
	}
	if ok {
		return info.Exists, nil
	}
	// 未命中：查库并回填，顺带把 url 一并写入，供 url 查询接口复用
	exists, err := s.repo.Exists(ctx, id)
	if err != nil {
		return false, err
	}
	url := ""
	if exists {
		urls, err := s.repo.GetURLsByIDs(ctx, []int64{id})
		if err != nil {
			return false, err
		}
		url = urls[id]
	}
	s.setInfo(ctx, id, galleryInfo{Exists: exists, URL: url})
	return exists, nil
}

// GetURLsByIDs 按 id 批量查询 url，不存在的 id 不在返回的映射中。
// 旁路缓存：逐 id 读缓存，未命中的走库并回填。
func (s *galleryService) GetURLsByIDs(ctx context.Context, ids []int64) (map[int64]string, error) {
	result := make(map[int64]string, len(ids))
	misses := make([]int64, 0, len(ids))
	seen := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		info, ok, err := s.loadInfo(ctx, id)
		if err != nil {
			return nil, err
		}
		if ok && info.Exists && info.URL != "" {
			result[id] = info.URL
			continue
		}
		if !ok {
			misses = append(misses, id)
			continue
		}
		// 命中缓存但记录不存在，跳过
	}

	if len(misses) > 0 {
		urls, err := s.repo.GetURLsByIDs(ctx, misses)
		if err != nil {
			return nil, err
		}
		for _, id := range misses {
			url, found := urls[id]
			s.setInfo(ctx, id, galleryInfo{Exists: found, URL: url})
			if found {
				result[id] = url
			}
		}
	}
	return result, nil
}

// loadInfo 读取单条记录缓存，miss 返回 (nil, false, nil)，Redis 故障降级为 miss。
func (s *galleryService) loadInfo(ctx context.Context, id int64) (*galleryInfo, bool, error) {
	var info galleryInfo
	hit, err := s.cache.Get(ctx, infoKey(id), &info)
	if err != nil {
		s.logCacheErr("get "+infoKey(id), err)
		return nil, false, nil
	}
	if !hit {
		return nil, false, nil
	}
	return &info, true, nil
}

func (s *galleryService) setInfo(ctx context.Context, id int64, info galleryInfo) {
	if err := s.cache.Set(ctx, infoKey(id), info, s.infoTTL); err != nil {
		s.logCacheErr("set "+infoKey(id), err)
	}
}

// logCacheErr 缓存读写失败仅记录日志，不影响业务（旁路缓存可整体降级）。
func (s *galleryService) logCacheErr(action string, err error) {
	fmt.Printf("[WARN] gallery cache %s: %v\n", action, err)
}
