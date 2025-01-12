-- Eval(ctx, setCodeLua, []string{key}, code).Int()
-- 获取KEYS数组中的第一个元素，key
local key = KEYS[1]

-- 获取ARGV数组中的第一个元素，code
local inputCode = ARGV[1]

-- 获取key的值，也就是redis中存储的code
local code = redis.call("get", key)

-- 创建发送次数的key，即key:count
local keyCount = key..":verify:count"

-- 获取还可以验证的次数
local count = tonumber(redis.call("get", keyCount))

-- 如果没有获取到count，并且code不为nil，则初始化为3（假设最多尝试次数为3）
if count == nil and code ~= nil then
    count = 3 -- 初始化count
    redis.call("set", keyCount, count)
end

-- 用户一直输错
if count == 0 then
    -- 返回"输入错误次数太多，稍后重试"
    redis.call("del", key)
    redis.call("del", keyCount)
    return -2
-- 用户输入正确
elseif inputCode == code then
    -- 删除验证码和次数限制key
    redis.call("del", key)
    redis.call("del", keyCount)
    return 0
-- 用户输入错误
else
    -- 还可以尝试次数-1
    redis.call("decr", keyCount)
    -- 用户输入错误
    return -1
end