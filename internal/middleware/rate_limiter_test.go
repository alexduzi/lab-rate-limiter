package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/alexduzi/labratelimiter/internal/config"
	"github.com/alexduzi/labratelimiter/internal/dto"
	"github.com/alexduzi/labratelimiter/internal/limiter"
	"github.com/alexduzi/labratelimiter/internal/storage"
)

func setEnvs(t *testing.T) {
	t.Helper()
	os.Setenv("IP_LIMIT_RPS", "3")
	os.Setenv("IP_BLOCK_DURATION", "3s")
	os.Setenv("TOKEN_LIMIT_RPS", "4")
	os.Setenv("TOKEN_BLOCK_DURATION", "4s")
	os.Setenv("SERVER_PORT", "8080")
}

func unsetEnvs(t *testing.T) {
	t.Helper()
	os.Unsetenv("IP_LIMIT_RPS")
	os.Unsetenv("IP_BLOCK_DURATION")
	os.Unsetenv("TOKEN_LIMIT_RPS")
	os.Unsetenv("TOKEN_BLOCK_DURATION")
	os.Unsetenv("SERVER_PORT")
}

func setupRouter(t *testing.T) *http.ServeMux {
	t.Helper()

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

	return mux
}

func setupServer(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()

	setEnvs(t)
	t.Cleanup(func() { unsetEnvs(t) })

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	store := storage.NewMemoryStorage()
	rl := limiter.NewRateLimiter(store, cfg)
	mux := setupRouter(t)
	handler := RateLimiter(rl)(mux)

	server := httptest.NewServer(handler)
	t.Cleanup(func() { server.Close() })

	return server, server.Client()
}

func TestRateLimiterMiddleware_BlocksIPAfterLimit(t *testing.T) {
	server, client := setupServer(t)

	for i := 1; i <= 3; i++ {
		req, err := http.NewRequest(http.MethodGet, server.URL+"/", nil)
		if err != nil {
			t.Fatalf("request %d: failed to create request: %v", i, err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request %d: failed to execute request: %v", i, err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("request %d: expected status %d, got %d", i, http.StatusOK, resp.StatusCode)
		}
	}

	req, err := http.NewRequest(http.MethodGet, server.URL+"/", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, resp.StatusCode)
	}

	var body dto.ResponseMessage
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body.Message == "" {
		t.Error("expected non-empty error message in response body")
	}
}

func TestRateLimiterMiddleware_BlocksTokenAfterLimit(t *testing.T) {
	server, client := setupServer(t)

	token := "my-api-token-123"

	for i := 1; i <= 4; i++ {
		req, err := http.NewRequest(http.MethodGet, server.URL+"/", nil)
		if err != nil {
			t.Fatalf("request %d: failed to create request: %v", i, err)
		}
		req.Header.Set("API_KEY", token)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request %d: failed to execute request: %v", i, err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("request %d: expected status %d, got %d", i, http.StatusOK, resp.StatusCode)
		}
	}

	req, err := http.NewRequest(http.MethodGet, server.URL+"/", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("API_KEY", token)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, resp.StatusCode)
	}

	var body dto.ResponseMessage
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body.Message == "" {
		t.Error("expected non-empty error message in response body")
	}
}

func TestRateLimiterMiddleware_TokenLimitIsIndependentFromIP(t *testing.T) {
	server, client := setupServer(t)

	token := "independent-token"

	for i := 1; i <= 3; i++ {
		req, err := http.NewRequest(http.MethodGet, server.URL+"/", nil)
		if err != nil {
			t.Fatalf("request %d: failed to create request: %v", i, err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request %d: failed to execute request: %v", i, err)
		}
		resp.Body.Close()
	}

	req, err := http.NewRequest(http.MethodGet, server.URL+"/", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("API_KEY", token)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestRateLimiterMiddleware_DifferentTokensHaveSeparateLimits(t *testing.T) {
	server, client := setupServer(t)

	for i := 1; i <= 4; i++ {
		req, err := http.NewRequest(http.MethodGet, server.URL+"/", nil)
		if err != nil {
			t.Fatalf("token-a request %d: failed to create request: %v", i, err)
		}
		req.Header.Set("API_KEY", "token-a")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("token-a request %d: failed to execute request: %v", i, err)
		}
		resp.Body.Close()
	}

	req, err := http.NewRequest(http.MethodGet, server.URL+"/", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("API_KEY", "token-a")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("token-a: expected status %d, got %d", http.StatusTooManyRequests, resp.StatusCode)
	}

	req, err = http.NewRequest(http.MethodGet, server.URL+"/", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("API_KEY", "token-b")

	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("token-b: expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestRateLimiterMiddleware_BlockedIPStaysBlocked(t *testing.T) {
	server, client := setupServer(t)

	for i := 1; i <= 4; i++ {
		req, err := http.NewRequest(http.MethodGet, server.URL+"/", nil)
		if err != nil {
			t.Fatalf("request %d: failed to create request: %v", i, err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request %d: failed to execute request: %v", i, err)
		}
		resp.Body.Close()
	}

	for i := 1; i <= 3; i++ {
		req, err := http.NewRequest(http.MethodGet, server.URL+"/", nil)
		if err != nil {
			t.Fatalf("blocked request %d: failed to create request: %v", i, err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("blocked request %d: failed to execute request: %v", i, err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusTooManyRequests {
			t.Errorf("blocked request %d: expected status %d, got %d", i, http.StatusTooManyRequests, resp.StatusCode)
		}
	}
}

func TestRateLimiterMiddleware_RequestsWithinLimitSucceed(t *testing.T) {
	server, client := setupServer(t)

	for i := 1; i <= 3; i++ {
		req, err := http.NewRequest(http.MethodGet, server.URL+"/", nil)
		if err != nil {
			t.Fatalf("request %d: failed to create request: %v", i, err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request %d: failed to execute request: %v", i, err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("request %d: expected status %d, got %d", i, http.StatusOK, resp.StatusCode)
		}
	}
}

func TestRateLimiterMiddleware_HealthEndpointAlsoRateLimited(t *testing.T) {
	server, client := setupServer(t)

	for i := 1; i <= 3; i++ {
		req, err := http.NewRequest(http.MethodGet, server.URL+"/health", nil)
		if err != nil {
			t.Fatalf("request %d: failed to create request: %v", i, err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request %d: failed to execute request: %v", i, err)
		}
		resp.Body.Close()
	}

	req, err := http.NewRequest(http.MethodGet, server.URL+"/health", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, resp.StatusCode)
	}
}
