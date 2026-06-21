package middleware

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"
)

// Auth enforces Bearer-token authentication.
//
// It fails closed: if API_TOKEN is not set, every request is rejected unless
// AUTH_DISABLED=true is explicitly provided. This prevents a misconfigured
// deployment from silently becoming an open proxy/SSRF gateway.
func Auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		expectedToken := os.Getenv("API_TOKEN")
		if expectedToken == "" {
			if os.Getenv("AUTH_DISABLED") == "true" {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"server misconfigured: API_TOKEN is not set"}`))
			return
		}

		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if !strings.HasPrefix(authHeader, "Bearer ") ||
			subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"Unauthorized"}`))
			return
		}
		next.ServeHTTP(w, r)
	}
}
