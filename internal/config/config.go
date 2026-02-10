package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	IpLimitRps         int
	IpBlockDuration    time.Duration
	TokenLimitRps      int
	TokenBlockDuration time.Duration
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
	ServerPort         string
}

func Load() (*Config, error) {
	ipLimit, err := strconv.Atoi(getEnv("IP_LIMIT_RPS", "10"))
	if err != nil {
		return nil, err
	}

	ipBlockDuration, err := time.ParseDuration(getEnv("IP_BLOCK_DURATION", "300s"))
	if err != nil {
		return nil, err
	}

	tokenLimit, err := strconv.Atoi(getEnv("TOKEN_LIMIT_RPS", "100"))
	if err != nil {
		return nil, err
	}

	tokenBlockDuration, err := time.ParseDuration(getEnv("TOKEN_BLOCK_DURATION", "300s"))
	if err != nil {
		return nil, err
	}

	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return nil, err
	}

	return &Config{
		IpLimitRps:         ipLimit,
		IpBlockDuration:    ipBlockDuration,
		TokenLimitRps:      tokenLimit,
		TokenBlockDuration: tokenBlockDuration,
		RedisAddr:          getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		RedisDB:            redisDB,
		ServerPort:         getEnv("SERVER_PORT", "8080"),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
