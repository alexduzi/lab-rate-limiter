package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/alexduzi/labratelimiter/internal/config"
	"github.com/alexduzi/labratelimiter/internal/storage"
)

type RateLimiter struct {
	storage storage.Storage
	cfg     *config.Config
}

func NewRateLimiter(storage storage.Storage, cfg *config.Config) *RateLimiter {
	return &RateLimiter{
		storage: storage,
		cfg:     cfg,
	}
}

func (rl *RateLimiter) AllowIP(ctx context.Context, ip string) (bool, error) {
	key := fmt.Sprintf("ip:%s", ip)

	return rl.allow(ctx, key, rl.cfg.IpLimitRps, rl.cfg.IpBlockDuration)
}

func (rl *RateLimiter) AllowToken(ctx context.Context, token string) (bool, error) {
	key := fmt.Sprintf("token:%s", token)

	return rl.allow(ctx, key, rl.cfg.TokenLimitRps, rl.cfg.TokenBlockDuration)
}

// allow é a lógica central do rate limiting
func (rl *RateLimiter) allow(ctx context.Context, key string, limit int, blockDuration time.Duration) (bool, error) {
	// 1. Verifica se está bloqueado
	blocked, err := rl.storage.IsBlocked(ctx, key)
	if err != nil {
		return false, fmt.Errorf("failed to check if blocked: %w", err)
	}

	if blocked {
		return false, nil
	}

	// 2. Incrementa o contador
	count, err := rl.storage.Increment(ctx, key, time.Second)
	if err != nil {
		return false, fmt.Errorf("failed to increment counter: %w", err)
	}

	// 3. Verifica se excedeu o limite
	if int(count) > limit {
		// Bloqueia por X tempo
		if err := rl.storage.Block(ctx, key, blockDuration); err != nil {
			return false, fmt.Errorf("failed to block key: %w", err)
		}
		return false, nil
	}

	return true, nil
}

// Allow verifica IP ou Token (token tem precedência)
func (rl *RateLimiter) Allow(ctx context.Context, ip, token string) (bool, error) {
	if token != "" {
		return rl.AllowToken(ctx, token)
	}

	return rl.AllowIP(ctx, ip)
}
