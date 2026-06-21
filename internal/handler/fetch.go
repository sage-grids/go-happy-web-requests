package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/sage-grids/go-happy-web-requests/internal/models"
	"github.com/sage-grids/go-happy-web-requests/internal/racing"
)

const (
	// defaultMaxRequestBytes caps the inbound request body. Override with MAX_REQUEST_BYTES.
	defaultMaxRequestBytes = 1 << 20 // 1 MiB
	// defaultMaxProxies caps how many proxies a single request may race. Override with MAX_PROXIES.
	defaultMaxProxies = 50
)

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

func Fetch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(models.FetchResponse{Status: "error", Message: "Method not allowed"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, int64(envInt("MAX_REQUEST_BYTES", defaultMaxRequestBytes)))

	var reqBody models.FetchRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.FetchResponse{Status: "error", Message: "Invalid JSON"})
		return
	}

	if reqBody.URL == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.FetchResponse{Status: "error", Message: "url is required"})
		return
	}

	if len(reqBody.Proxies) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.FetchResponse{Status: "error", Message: "at least one proxy is required"})
		return
	}

	if maxProxies := envInt("MAX_PROXIES", defaultMaxProxies); len(reqBody.Proxies) > maxProxies {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.FetchResponse{Status: "error", Message: fmt.Sprintf("too many proxies: %d (max %d)", len(reqBody.Proxies), maxProxies)})
		return
	}

	timeout := 15
	if reqBody.TimeoutSeconds > 0 {
		timeout = reqBody.TimeoutSeconds
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeout)*time.Second)
	defer cancel()

	start := time.Now()
	var res models.RaceResult
	var err error

	switch reqBody.Mode {
	case "playwright":
		err = fmt.Errorf("playwright mode not yet implemented")
	default:
		res, err = racing.RaceHTTP(ctx, reqBody.URL, reqBody.Proxies)
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.FetchResponse{Status: "error", Message: err.Error()})
		return
	}

	json.NewEncoder(w).Encode(models.FetchResponse{
		Status:       "success",
		WinningProxy: res.Proxy,
		Content:      res.Content,
		TimeTakenMs:  time.Since(start).Milliseconds(),
	})
}
