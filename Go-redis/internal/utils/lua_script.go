package utils

import (
	_ "embed"

	"github.com/redis/go-redis/v9"
)

//go:embed lua/unlock.lua
var unlockLua string

var unlockScript = redis.NewScript(unlockLua)
