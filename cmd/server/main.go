package main

import (
	"context"
	"log"
	"net/http"
	"strings"

	goredis "github.com/redis/go-redis/v9"

	"rate-limiter/internal/config"
	"rate-limiter/internal/ratelimiter"
	redisstorage "rate-limiter/internal/redis"
	"rate-limiter/internal/web"
)

func main() {
	applicationConfig, err := config.LoadConfig(".")
	if err != nil {
		log.Fatalf("carregar configuração: %v", err)
	}

	client := goredis.NewClient(&goredis.Options{
		Addr:     applicationConfig.RedisAddr,
		Password: applicationConfig.RedisPassword,
		DB:       applicationConfig.RedisDB,
	})
	defer client.Close()

	if err := client.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("conectar ao Redis: %v", err)
	}

	limiter := ratelimiter.New(redisstorage.NewStorage(client))
	rateLimitMiddleware := web.NewMiddleware(limiter, web.RateLimitConfig{
		IPLimit:    applicationConfig.RateLimitIP,
		TokenLimit: applicationConfig.RateLimitToken,
		BlockTime:  applicationConfig.BlockTime,
	})

	handler := rateLimitMiddleware.Handler(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write([]byte("ok"))
	}))

	address := applicationConfig.Port
	if !strings.HasPrefix(address, ":") {
		address = ":" + address
	}

	log.Printf("rate limiter escutando em %s", address)
	if err := http.ListenAndServe(address, handler); err != nil {
		log.Fatal(err)
	}
}
