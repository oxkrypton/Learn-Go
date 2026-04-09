package cache

import (
	"encoding/json"
	"time"
)

// RedisData 逻辑过期缓存包装结构体
type RedisData struct {
	Data       json.RawMessage `json:"data"`
	ExpireTime time.Time       `json:"expireTime"`
}
