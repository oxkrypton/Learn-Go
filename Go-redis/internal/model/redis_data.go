package model

import (
	"encoding/json"
	"time"
)

// RedisData逻辑过期缓存包装结构体
// Data 存储实际业务数据的 JSON（使用 json.RawMessage 保持通用性）
// ExpireTime 存储逻辑过期时间点
type RedisData struct {
	Data       json.RawMessage `json:"data"`
	ExpireTime time.Time       `json:"expireTime"`
}
