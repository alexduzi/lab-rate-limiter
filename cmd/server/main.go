package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/alexduzi/labratelimiter/internal/config"
	"github.com/alexduzi/labratelimiter/internal/dto"
	"github.com/alexduzi/labratelimiter/internal/limiter"
	"github.com/alexduzi/labratelimiter/internal/middleware"
	"github.com/alexduzi/labratelimiter/internal/storage"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	store, err := storage.NewRedisStorage(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer store.Close()

	rl := limiter.NewRateLimiter(store, cfg)

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		response := dto.ResponseMessage{
			Message: "OK",
		}
		json.NewEncoder(w).Encode(response)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		response := dto.ResponseHealth{
			Status: "healthy",
		}
		json.NewEncoder(w).Encode(response)
	})

	// Aplica middleware
	handler := middleware.RateLimiter(rl)(mux)

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("Server starting on %s", addr)
	log.Printf("IP Rate Limit: %d req/s", cfg.IpLimitRps)
	log.Printf("Token Rate Limit: %d req/s", cfg.TokenLimitRps)

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
