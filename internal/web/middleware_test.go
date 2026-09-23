package web

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"rate-limiter/internal/ratelimiter"
)

type recordingStorage struct {
	allowed bool
	err     error
	key     string
	limit   int
}

func (storage *recordingStorage) Allow(_ context.Context, key string, limit int, _ time.Duration) (bool, error) {
	storage.key = key
	storage.limit = limit
	return storage.allowed, storage.err
}

func TestMiddlewareUsesIPWhenAPIKeyIsAbsent(t *testing.T) {
	storage := &recordingStorage{allowed: true}
	middleware := NewMiddleware(ratelimiter.New(storage), RateLimitConfig{
		IPLimit:    10,
		TokenLimit: 100,
		BlockTime:  time.Minute,
	})
	nextCalled := false
	handler := middleware.Handler(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		response.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "203.0.113.10:4321"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if !nextCalled {
		t.Fatal("next handler was not called")
	}
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if storage.key != "ip:203.0.113.10" {
		t.Errorf("storage key = %q, want %q", storage.key, "ip:203.0.113.10")
	}
	if storage.limit != 10 {
		t.Errorf("storage limit = %d, want 10", storage.limit)
	}
}

func TestMiddlewareUsesTokenAndTokenLimitTakesPrecedence(t *testing.T) {
	storage := &recordingStorage{allowed: true}
	middleware := NewMiddleware(ratelimiter.New(storage), RateLimitConfig{
		IPLimit:    1,
		TokenLimit: 100,
		BlockTime:  time.Minute,
	})
	handler := middleware.Handler(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "203.0.113.10:4321"
	request.Header.Set(TokenHeader, "abc")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if storage.key != "token:abc" {
		t.Errorf("storage key = %q, want %q", storage.key, "token:abc")
	}
	if storage.limit != 100 {
		t.Errorf("storage limit = %d, want 100", storage.limit)
	}
}

func TestMiddlewareReturnsExact429BodyAndDoesNotCallNext(t *testing.T) {
	storage := &recordingStorage{allowed: false}
	middleware := NewMiddleware(ratelimiter.New(storage), RateLimitConfig{
		IPLimit:   1,
		BlockTime: time.Minute,
	})
	nextCalled := false
	handler := middleware.Handler(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		response.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "203.0.113.10:4321"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusTooManyRequests)
	}
	if response.Body.String() != BlockedBody {
		t.Fatalf("body = %q, want exact %q", response.Body.String(), BlockedBody)
	}
	if nextCalled {
		t.Fatal("next handler was called for a blocked request")
	}
}

func TestMiddlewareReturns500WhenStorageFails(t *testing.T) {
	storage := &recordingStorage{err: errors.New("Redis indisponível")}
	middleware := NewMiddleware(ratelimiter.New(storage), RateLimitConfig{
		IPLimit:   1,
		BlockTime: time.Minute,
	})
	nextCalled := false
	handler := middleware.Handler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if nextCalled {
		t.Fatal("next handler was called after a storage failure")
	}
}

func TestRemoteIPSupportsIPv6WithoutPort(t *testing.T) {
	if got := remoteIP("[2001:db8::1]"); got != "2001:db8::1" {
		t.Fatalf("remoteIP() = %q, want %q", got, "2001:db8::1")
	}
	if got := remoteIP("[2001:db8::1]:8080"); got != "2001:db8::1" {
		t.Fatalf("remoteIP() = %q, want %q", got, "2001:db8::1")
	}
}
