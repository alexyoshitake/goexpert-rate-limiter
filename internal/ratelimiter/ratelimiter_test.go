package ratelimiter

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeStorage struct {
	allowed   bool
	err       error
	key       string
	limit     int
	blockTime time.Duration
}

func (storage *fakeStorage) Allow(_ context.Context, key string, limit int, blockTime time.Duration) (bool, error) {
	storage.key = key
	storage.limit = limit
	storage.blockTime = blockTime
	return storage.allowed, storage.err
}

func TestRateLimiterDelegatesToStorage(t *testing.T) {
	storage := &fakeStorage{allowed: true}
	limiter := New(storage)

	allowed, err := limiter.Allow(context.Background(), "token:abc", 4, 3*time.Second)
	if err != nil {
		t.Fatalf("Allow() error = %v", err)
	}
	if !allowed {
		t.Fatal("Allow() = false, want true")
	}
	if storage.key != "token:abc" || storage.limit != 4 || storage.blockTime != 3*time.Second {
		t.Fatalf("storage received key=%q limit=%d blockTime=%s", storage.key, storage.limit, storage.blockTime)
	}
}

func TestRateLimiterReturnsStorageError(t *testing.T) {
	wantErr := errors.New("falha no armazenamento")
	limiter := New(&fakeStorage{err: wantErr})

	_, err := limiter.Allow(context.Background(), "ip:127.0.0.1", 1, time.Second)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Allow() error = %v, want %v", err, wantErr)
	}
}

func TestRateLimiterRejectsInvalidArguments(t *testing.T) {
	limiter := New(&fakeStorage{allowed: true})

	if _, err := limiter.Allow(context.Background(), "key", 0, time.Second); !errors.Is(err, ErrInvalidLimit) {
		t.Fatalf("invalid limit error = %v, want %v", err, ErrInvalidLimit)
	}
	if _, err := limiter.Allow(context.Background(), "key", 1, 0); !errors.Is(err, ErrInvalidBlockTime) {
		t.Fatalf("invalid block time error = %v, want %v", err, ErrInvalidBlockTime)
	}
	if _, err := New(nil).Allow(context.Background(), "key", 1, time.Second); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("nil storage error = %v, want %v", err, ErrStorageUnavailable)
	}
}
