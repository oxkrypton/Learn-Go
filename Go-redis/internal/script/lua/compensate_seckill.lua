local stockKey = KEYS[1]
local orderKey = KEYS[2]
local userId = ARGV[1]

redis.call('INCRBY', stockKey, 1)
redis.call('SREM', orderKey, userId)
return 0
