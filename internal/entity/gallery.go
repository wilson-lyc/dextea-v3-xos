package entity

import "time"

// Gallery 图库表实体，与 gallery 表字段一一对应。
type Gallery struct {
	ID        int64     `json:"id"`
	Source    string    `json:"source"`     // 存储源名称，对应配置中的存储源 key
	URL       string    `json:"url"`
	ObjectKey string    `json:"objectKey"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}
