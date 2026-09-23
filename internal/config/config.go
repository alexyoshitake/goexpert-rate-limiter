package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Port           string
	RateLimitIP    int
	RateLimitToken int
	BlockTime      time.Duration
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
}

func LoadConfig(path string) (*Config, error) {
	settings := viper.New()
	settings.SetConfigFile(arquivoDeConfiguracao(path))
	settings.SetConfigType("env")
	settings.AutomaticEnv()

	if err := settings.ReadInConfig(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return &Config{
		Port:           settings.GetString("PORT"),
		RateLimitIP:    settings.GetInt("RATE_LIMIT_IP"),
		RateLimitToken: settings.GetInt("RATE_LIMIT_TOKEN"),
		BlockTime:      settings.GetDuration("BLOCK_TIME"),
		RedisAddr:      settings.GetString("REDIS_ADDR"),
		RedisPassword:  settings.GetString("REDIS_PASSWORD"),
		RedisDB:        settings.GetInt("REDIS_DB"),
	}, nil
}

func arquivoDeConfiguracao(path string) string {
	diretorio, err := filepath.Abs(path)
	if err != nil {
		return filepath.Join(path, ".env")
	}

	for {
		arquivo := filepath.Join(diretorio, ".env")
		if _, err := os.Stat(arquivo); err == nil {
			return arquivo
		}

		pai := filepath.Dir(diretorio)
		if pai == diretorio {
			return arquivo
		}
		diretorio = pai
	}
}
