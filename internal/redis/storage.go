package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	keyPrefix   = "rate_limiter"
	window      = time.Second
	countSuffix = "count"
	blockSuffix = "block"
)

var rateLimitScript = goredis.NewScript(`
local blocked = redis.call("EXISTS", KEYS[2])
if blocked == 1 then
    return 0
end

local count = redis.call("INCR", KEYS[1])
if count == 1 then
    redis.call("PEXPIRE", KEYS[1], ARGV[1])
end

if count > tonumber(ARGV[2]) then
    redis.call("SET", KEYS[2], "1", "PX", ARGV[3])
    return 0
end

return 1
`)

type Storage struct {
	client *goredis.Client
}

func NewStorage(client *goredis.Client) *Storage {
	return &Storage{client: client}
}

func (storage *Storage) Allow(ctx context.Context, key string, limit int, blockTime time.Duration) (bool, error) {
	if storage == nil || storage.client == nil {
		return false, errors.New("cliente Redis indisponível")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	blockMilliseconds := blockTime.Milliseconds()
	if blockMilliseconds < 1 {
		blockMilliseconds = 1
	}

	result, err := rateLimitScript.Run(
		ctx,
		storage.client,
		[]string{storage.countKey(key), storage.blockKey(key)},
		window.Milliseconds(),
		limit,
		blockMilliseconds,
	).Result()
	if err != nil {
		return false, fmt.Errorf("executar script do rate limiter: %w", err)
	}

	allowed, ok := result.(int64)
	if !ok {
		return false, fmt.Errorf("tipo de resultado inesperado do rate limiter Redis: %T", result)
	}
	return allowed == 1, nil
}

func (storage *Storage) countKey(key string) string {
	return fmt.Sprintf("%s:%s:%s", keyPrefix, countSuffix, key)
}

func (storage *Storage) blockKey(key string) string {
	return fmt.Sprintf("%s:%s:%s", keyPrefix, blockSuffix, key)
}
