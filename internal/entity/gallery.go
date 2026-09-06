package entity

import "time"

// Gallery 图库表实体，与 gallery 表字段一一对应。
type Gallery struct {
	ID        int64
	Source    string // 存储源名称，对应配置中的存储源 key
	URL       string
	ObjectKey string
	Name      string
	CreatedAt time.Time
}
