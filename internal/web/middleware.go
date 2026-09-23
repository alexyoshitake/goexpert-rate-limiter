package web

import (
	"net"
	"net/http"
	"strings"
	"time"

	"rate-limiter/internal/ratelimiter"
)

const (
	TokenHeader = "API_KEY"
	BlockedBody = "you have reached the maximum number of requests or actions allowed within a certain time frame"
)

type RateLimitConfig struct {
	IPLimit    int
	TokenLimit int
	BlockTime  time.Duration
}

type Middleware struct {
	limiter *ratelimiter.RateLimiter
	config  RateLimitConfig
}

func NewMiddleware(limiter *ratelimiter.RateLimiter, config RateLimitConfig) *Middleware {
	return &Middleware{limiter: limiter, config: config}
}

func (middleware *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if middleware == nil || middleware.limiter == nil {
			writeInternalError(response)
			return
		}

		key, limit := requestKeyAndLimit(request, middleware.config)
		allowed, err := middleware.limiter.Allow(request.Context(), key, limit, middleware.config.BlockTime)
		if err != nil {
			writeInternalError(response)
			return
		}
		if !allowed {
			writeBlocked(response)
			return
		}

		next.ServeHTTP(response, request)
	})
}

func requestKeyAndLimit(request *http.Request, config RateLimitConfig) (string, int) {
	if token := request.Header.Get(TokenHeader); token != "" {
		return "token:" + token, config.TokenLimit
	}
	return "ip:" + remoteIP(request.RemoteAddr), config.IPLimit
}

func remoteIP(remoteAddr string) string {
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	if strings.HasPrefix(remoteAddr, "[") && strings.HasSuffix(remoteAddr, "]") {
		return strings.TrimSuffix(strings.TrimPrefix(remoteAddr, "["), "]")
	}
	if remoteAddr == "" {
		return "unknown"
	}
	return remoteAddr
}

func writeBlocked(response http.ResponseWriter) {
	response.WriteHeader(http.StatusTooManyRequests)
	_, _ = response.Write([]byte(BlockedBody))
}

func writeInternalError(response http.ResponseWriter) {
	response.WriteHeader(http.StatusInternalServerError)
	_, _ = response.Write([]byte("erro interno do servidor"))
}
