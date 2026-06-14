if redis.call('HEXISTS', KEYS[1], ARGV[1]) == 0 then
    return 0
end

redis.call('PEXPIRE', KEYS[1], ARGV[2])
return 1