-- Eval(ctx, setCodeLua, []string{key}, code).Int()
-- 获取KEYS数组中的第一个元素，其实就是获取[]string{key}中的第一个元素，即key
local key = KEYS[1]

-- Eval(ctx, setCodeLua, []string{key}, code).Int()
-- 获取ARGV数组中的第一个元素，即code
local code = ARGV[1]

-- 创建keyCount，即"用户剩余发送短信次数"
local keyCount = key..":send:count" -- ..是字符串连接符

-- 获取key的剩余过期时间
-- ttl == -1，key存在但没有设置过期时间
-- ttl == -2，key不存在
-- ttl == 正数，key存在且没有过期
local ttl = tonumber(redis.call("ttl", key))

-- 获取当前keyCount的值
local count = tonumber(redis.call("get", keyCount))
-- keyCount不存在，初始化为5
if not count then
    redis.call("set", keyCount, 5)
    redis.call("expire", keyCount, 86400) -- 设置24h的过期时间
    count = 5
end

-- 不允许发送短息
-- key存在，但没有过期时间
if ttl == -1 then
    return -1 -- 系统错误

-- 允许发送短息
-- (key不存在 or key的有效期过去1分钟了) and 用户剩余发送短信次数>0
elseif (ttl == -2 or ttl <= 540) and count > 0 then  -- 540
    -- 更新key和keyCount
    redis.call("set", key, code) -- 设置key的值为code
    redis.call("expire", key, 600) -- 设置key的过期时间为600秒
    redis.call("decr", keyCount) -- 减少剩余发送次数
    return 0 -- 成功发送短信

elseif ttl > 540 then -- 540
    return -2 -- 发送太频繁，1分钟内只能发一条

elseif count == 0 then
    return -3 -- 今日已经发送5次短信了，发送次数超过上限

end
