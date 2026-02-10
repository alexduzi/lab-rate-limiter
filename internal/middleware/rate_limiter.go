package middleware

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/alexduzi/labratelimiter/internal/dto"
	"github.com/alexduzi/labratelimiter/internal/limiter"
)

func RateLimiter(rl *limiter.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.Background()

			// Extrai IP real
			ip := getIP(r)

			// Extrai token do header
			token := r.Header.Get("API_KEY")

			// Verifica rate limit
			allowed, err := rl.Allow(ctx, ip, token)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				response := dto.ResponseMessage{
					Message: "you have reached the maximum number of requests or actions allowed within a certain time frame",
				}
				json.NewEncoder(w).Encode(response)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getIP extrai o IP real da requisição
func getIP(r *http.Request) string {
	// Tenta X-Forwarded-For primeiro (para proxies/load balancers)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	// Tenta X-Real-IP
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Usa RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return ip
}
