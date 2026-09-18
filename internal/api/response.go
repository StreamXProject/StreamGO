package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// ErrorResponse defines standard API error envelope.
type ErrorResponse struct {
	OK      bool   `json:"ok"`
	Detail  string `json:"detail"`
	Status  int    `json:"status"`
}

// RespondJSON writes a JSON payload with standard status code.
func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// RespondError writes a standardized JSON error message.
func RespondError(w http.ResponseWriter, status int, message string) {
	RespondJSON(w, status, ErrorResponse{
		OK:     false,
		Detail: message,
		Status: status,
	})
}

// ParseQueryInt parses an integer query parameter with a fallback default.
func ParseQueryInt(r *http.Request, key string, defaultVal int) int {
	val := strings.TrimSpace(r.URL.Query().Get(key))
	if val == "" {
		return defaultVal
	}
	if n, err := strconv.Atoi(val); err == nil {
		return n
	}
	return defaultVal
}

// ParseQueryInt64 parses an int64 query parameter with a fallback default.
func ParseQueryInt64(r *http.Request, key string, defaultVal int64) int64 {
	val := strings.TrimSpace(r.URL.Query().Get(key))
	if val == "" {
		return defaultVal
	}
	if n, err := strconv.ParseInt(val, 10, 64); err == nil {
		return n
	}
	return defaultVal
}
