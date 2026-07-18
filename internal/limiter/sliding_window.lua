local key = KEYS[1]
local now_nano = tonumber(ARGV[1])
local window_seconds = tonumber(ARGV[2])
local requests_allowed = tonumber(ARGV[3])
local member = ARGV[4]

local window_nano = window_seconds * 1000000000
local cutoff = now_nano - window_nano

redis.call("ZREMRANGEBYSCORE", key, "-inf", cutoff)

local count = redis.call("ZCARD", key)

if count < requests_allowed then
    redis.call("ZADD", key, now_nano, member)
    redis.call("EXPIRE", key, window_seconds)
    local remaining = requests_allowed - count - 1
    return {1, remaining, 0}
else
    local oldest = redis.call("ZRANGE", key, 0, 0, "WITHSCORES")
    if #oldest >= 2 then
        local oldest_score = tonumber(oldest[2])
        local oldest_seconds = oldest_score / 1000000000
        local now_seconds = now_nano / 1000000000
        local retry_after = math.ceil(oldest_seconds + window_seconds - now_seconds)
        if retry_after < 0 then
            retry_after = 0
        end
        return {0, 0, retry_after}
    end
    return {0, 0, 0}
end
