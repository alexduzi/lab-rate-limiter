package handler

import (
	"encoding/json"
	"net/http"
)

type LimiterHandler struct {
}

func NewLimiterHandler() *LimiterHandler {
	return &LimiterHandler{}
}

func (lh *LimiterHandler) HandleLimiter(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Hello world",
	})
}
