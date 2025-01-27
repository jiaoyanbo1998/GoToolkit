local task = redis.call('ZRANGEBYSCORE', KEYS[1], '-inf', ARGV[1], 'LIMIT', 0, 1)
if #task == 0 then return nil end
redis.call('ZREM', KEYS[1], task[1])
return task[1]