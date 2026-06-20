package middleware

import (
	"net/http"
	"os"
	"strings"
)

func Auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		expectedToken := os.Getenv("API_TOKEN")
		if expectedToken == "" {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") || strings.TrimPrefix(authHeader, "Bearer ") != expectedToken {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"Unauthorized"}`))
			return
		}
		next.ServeHTTP(w, r)
	}
}
