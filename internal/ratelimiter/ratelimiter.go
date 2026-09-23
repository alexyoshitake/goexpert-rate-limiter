package ratelimiter

import (
	"context"
	"time"
)

type RateLimiterStorage interface {
	Allow(ctx context.Context, key string, limit int, blockTime time.Duration) (bool, error)
}

type RateLimiter struct {
	storage RateLimiterStorage
}

func New(storage RateLimiterStorage) *RateLimiter {
	return &RateLimiter{storage: storage}
}

func (limiter *RateLimiter) Allow(ctx context.Context, key string, limit int, blockTime time.Duration) (bool, error) {
	if limiter == nil || limiter.storage == nil {
		return false, ErrStorageUnavailable
	}
	if limit <= 0 {
		return false, ErrInvalidLimit
	}
	if blockTime <= 0 {
		return false, ErrInvalidBlockTime
	}
	if ctx == nil {
		ctx = context.Background()
	}

	return limiter.storage.Allow(ctx, key, limit, blockTime)
}
