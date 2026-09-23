package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfigReturnsZeroValuesWithoutDotEnv(t *testing.T) {
	clearConfigurationEnvironment(t)

	configuration, err := LoadConfig(t.TempDir())
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if *configuration != (Config{}) {
		t.Fatalf("LoadConfig() = %+v, want zero values", configuration)
	}
}

func TestLoadConfigFromDotEnv(t *testing.T) {
	clearConfigurationEnvironment(t)
	directory := t.TempDir()
	writeDotEnv(t, directory, "PORT=9090\nRATE_LIMIT_IP=3\nRATE_LIMIT_TOKEN=7\nBLOCK_TIME=2s\nREDIS_ADDR=redis.internal:6379\nREDIS_PASSWORD=secret\nREDIS_DB=2\n")

	configuration, err := LoadConfig(directory)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if configuration.Port != "9090" {
		t.Errorf("Port = %q, want %q", configuration.Port, "9090")
	}
	if configuration.RateLimitIP != 3 {
		t.Errorf("RateLimitIP = %d, want 3", configuration.RateLimitIP)
	}
	if configuration.RateLimitToken != 7 {
		t.Errorf("RateLimitToken = %d, want 7", configuration.RateLimitToken)
	}
	if configuration.BlockTime != 2*time.Second {
		t.Errorf("BlockTime = %s, want 2s", configuration.BlockTime)
	}
	if configuration.RedisAddr != "redis.internal:6379" {
		t.Errorf("RedisAddr = %q, want %q", configuration.RedisAddr, "redis.internal:6379")
	}
	if configuration.RedisPassword != "secret" {
		t.Errorf("RedisPassword = %q, want %q", configuration.RedisPassword, "secret")
	}
	if configuration.RedisDB != 2 {
		t.Errorf("RedisDB = %d, want 2", configuration.RedisDB)
	}
}

func TestLoadConfigEnvironmentOverridesDotEnv(t *testing.T) {
	clearConfigurationEnvironment(t)
	directory := t.TempDir()
	writeDotEnv(t, directory, "PORT=9090\nRATE_LIMIT_IP=3\nRATE_LIMIT_TOKEN=7\nBLOCK_TIME=2s\nREDIS_ADDR=redis.internal:6379\nREDIS_DB=2\n")

	t.Setenv("RATE_LIMIT_IP", "42")
	t.Setenv("BLOCK_TIME", "1m")

	configuration, err := LoadConfig(directory)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if configuration.RateLimitIP != 42 {
		t.Errorf("RateLimitIP = %d, want 42", configuration.RateLimitIP)
	}
	if configuration.BlockTime != time.Minute {
		t.Errorf("BlockTime = %s, want 1m", configuration.BlockTime)
	}
}

func clearConfigurationEnvironment(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"PORT",
		"RATE_LIMIT_IP",
		"RATE_LIMIT_TOKEN",
		"BLOCK_TIME",
		"REDIS_ADDR",
		"REDIS_PASSWORD",
		"REDIS_DB",
	} {
		t.Setenv(key, "")
	}
}

func writeDotEnv(t *testing.T, directory, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(directory, ".env"), []byte(contents), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
}
