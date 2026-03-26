package api

import (
	"encoding/json"
	"net/http"
)

// APIKeyAuth returns middleware that validates the X-API-Key header.
func APIKeyAuth(validKeys []string) func(http.Handler) http.Handler {
	keySet := make(map[string]bool, len(validKeys))
	for _, k := range validKeys {
		keySet[k] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-API-Key")
			if key == "" || !keySet[key] {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":   "unauthorized",
					"message": "valid API key required",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
