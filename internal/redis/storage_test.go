package redis

import (
	"context"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

func TestStorageAllowAndBlockWithRedis(t *testing.T) {
	client := testRedisClient(t)
	storage := NewStorage(client)
	key := "integration-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	ctx := context.Background()

	allowed, err := storage.Allow(ctx, key, 2, 3*time.Second)
	if err != nil || !allowed {
		t.Fatalf("first Allow() = (%v, %v), want (true, nil)", allowed, err)
	}
	allowed, err = storage.Allow(ctx, key, 2, 3*time.Second)
	if err != nil || !allowed {
		t.Fatalf("second Allow() = (%v, %v), want (true, nil)", allowed, err)
	}

	countTTL, err := client.PTTL(ctx, storage.countKey(key)).Result()
	if err != nil {
		t.Fatalf("count PTTL() error = %v", err)
	}
	if countTTL <= 0 || countTTL > window {
		t.Fatalf("count TTL = %s, want greater than zero and at most %s", countTTL, window)
	}

	allowed, err = storage.Allow(ctx, key, 2, 3*time.Second)
	if err != nil {
		t.Fatalf("third Allow() error = %v", err)
	}
	if allowed {
		t.Fatal("third Allow() = true, want false")
	}

	blockTTL, err := client.PTTL(ctx, storage.blockKey(key)).Result()
	if err != nil {
		t.Fatalf("block PTTL() error = %v", err)
	}
	if blockTTL <= 0 || blockTTL > 3*time.Second {
		t.Fatalf("block TTL = %s, want greater than zero and at most 3s", blockTTL)
	}

	allowed, err = storage.Allow(ctx, key, 2, 3*time.Second)
	if err != nil {
		t.Fatalf("blocked Allow() error = %v", err)
	}
	if allowed {
		t.Fatal("blocked Allow() = true, want false")
	}
}

func TestStorageWindowExpires(t *testing.T) {
	client := testRedisClient(t)
	storage := NewStorage(client)
	key := "window-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	ctx := context.Background()

	allowed, err := storage.Allow(ctx, key, 1, time.Second)
	if err != nil || !allowed {
		t.Fatalf("first Allow() = (%v, %v), want (true, nil)", allowed, err)
	}

	time.Sleep(1100 * time.Millisecond)

	allowed, err = storage.Allow(ctx, key, 1, time.Second)
	if err != nil || !allowed {
		t.Fatalf("Allow() after window = (%v, %v), want (true, nil)", allowed, err)
	}
}

func TestStorageIsAtomicUnderConcurrency(t *testing.T) {
	client := testRedisClient(t)
	storage := NewStorage(client)
	key := "concurrent-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	const (
		limit       = 20
		totalCalls  = 100
		blockPeriod = 3 * time.Second
	)

	type result struct {
		allowed bool
		err     error
	}
	results := make(chan result, totalCalls)
	var waitGroup sync.WaitGroup
	waitGroup.Add(totalCalls)
	for index := 0; index < totalCalls; index++ {
		go func() {
			defer waitGroup.Done()
			allowed, err := storage.Allow(context.Background(), key, limit, blockPeriod)
			results <- result{allowed: allowed, err: err}
		}()
	}
	waitGroup.Wait()
	close(results)

	allowedCalls := 0
	for callResult := range results {
		if callResult.err != nil {
			t.Fatalf("concurrent Allow() error = %v", callResult.err)
		}
		if callResult.allowed {
			allowedCalls++
		}
	}
	if allowedCalls != limit {
		t.Fatalf("allowed concurrent calls = %d, want %d", allowedCalls, limit)
	}
}

func TestStorageReturnsRedisError(t *testing.T) {
	client := testRedisClient(t)
	if err := client.Close(); err != nil {
		t.Fatalf("close Redis client: %v", err)
	}

	storage := NewStorage(client)
	if _, err := storage.Allow(context.Background(), "closed", 1, time.Second); err == nil {
		t.Fatal("Allow() error = nil, want Redis error")
	}
}

func testRedisClient(t *testing.T) *goredis.Client {
	t.Helper()
	address := os.Getenv("REDIS_TEST_ADDR")
	if address == "" {
		t.Skip("REDIS_TEST_ADDR is not set; Redis integration tests are skipped")
	}

	database := 0
	if rawDatabase := os.Getenv("REDIS_TEST_DB"); rawDatabase != "" {
		parsedDatabase, err := strconv.Atoi(rawDatabase)
		if err != nil {
			t.Fatalf("parse REDIS_TEST_DB: %v", err)
		}
		database = parsedDatabase
	}

	client := goredis.NewClient(&goredis.Options{Addr: address, DB: database})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		t.Fatalf("connect to Redis at %s: %v", address, err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}
