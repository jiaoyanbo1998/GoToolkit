-- 使用 ZADD 记录每个请求的时间戳。
-- 使用 ZCOUNT 计算当前时间窗口内的请求数量
-- 使用 ZREMRANGEBYSCORE 删除过期的请求记录
-- 使用 PEXPIRE 设置有序集合的过期时间，保证到期后有序集合会被删除

-- 限流对象，有序集合
local key = KEYS[1]

-- 窗口大小（时间间隔的毫秒数）
local window = tonumber(ARGV[1])

-- 阈值（系统最高处理请求的数量）
local threshold = tonumber( ARGV[2])

-- 当前时间戳
local now = tonumber(ARGV[3])

-- 窗口的起始时间
local min = now - window

-- 删除过期请求
--     时间窗口：[-无穷,now-window] ∪ [now-window,now+window]
--     过期窗口：[-无穷,now-window]
redis.call('ZREMRANGEBYSCORE', key, '-inf', min) -- -inf <==> -无穷

-- 统计当前窗口内的请求数量
--    统计[min,+无穷]之间的请求个数，将请求个数存储在变量cnt中
local cnt = redis.call('ZCOUNT', key, min, '+inf') -- '-inf', '+inf'

-- 判断是否触发限流
if cnt >= threshold then
    -- 执行限流
    return "true"
else
    -- 将当前请求的时间戳当作score和member插入到redis的有序集合中
    --    参数一：命令('ZADD')
    --    参数二：key为有序集合，表示限流的对象
    --    参数三：score，排序分数，redis会根据score的值对集合进行排序
    --    参数四：member，有序集合中的实际值
    redis.call('ZADD', key, now, now)
    -- 设置有序集合的过期时间 == 窗口大小，到期后有序集合会被删除
    redis.call('PEXPIRE', key, window)
    return "false"
end