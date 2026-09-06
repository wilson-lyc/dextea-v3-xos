package entity

// Ping 数据库实体，字段与存储层结构一一对应。
type Ping struct {
	ID      int64
	Message string
}
