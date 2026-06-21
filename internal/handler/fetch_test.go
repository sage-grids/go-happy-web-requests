package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sage-grids/go-happy-web-requests/internal/models"
)

func newTestTargetServer(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, body)
	}))
}

func newTestProxyServer(targetURL string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp, err := http.Get(targetURL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}))
}

func newTestFailingProxyServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusBadGateway)
	}))
}

func doFetch(t *testing.T, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fetch", &buf)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	Fetch(rr, req)
	return rr
}

func decodeResponse(t *testing.T, rr *httptest.ResponseRecorder) models.FetchResponse {
	t.Helper()
	var resp models.FetchResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return resp
}

func TestFetch_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fetch", nil)
	rr := httptest.NewRecorder()
	Fetch(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}

	resp := decodeResponse(t, rr)
	if resp.Status != "error" {
		t.Errorf("expected status 'error', got %q", resp.Status)
	}
}

func TestFetch_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/fetch", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	Fetch(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}

	resp := decodeResponse(t, rr)
	if resp.Status != "error" {
		t.Errorf("expected status 'error', got %q", resp.Status)
	}
}

func TestFetch_MissingURL(t *testing.T) {
	rr := doFetch(t, models.FetchRequest{
		Proxies: []string{"http://proxy:8080"},
	})

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}

	resp := decodeResponse(t, rr)
	if resp.Message != "url is required" {
		t.Errorf("expected 'url is required', got %q", resp.Message)
	}
}

func TestFetch_MissingProxies(t *testing.T) {
	rr := doFetch(t, models.FetchRequest{
		URL: "http://example.com",
	})

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}

	resp := decodeResponse(t, rr)
	if resp.Message != "at least one proxy is required" {
		t.Errorf("expected 'at least one proxy is required', got %q", resp.Message)
	}
}

func TestFetch_PlaywrightNotImplemented(t *testing.T) {
	target := newTestTargetServer("test")
	defer target.Close()
	proxy := newTestProxyServer(target.URL)
	defer proxy.Close()

	rr := doFetch(t, models.FetchRequest{
		URL:     target.URL,
		Proxies: []string{proxy.URL},
		Mode:    "playwright",
	})

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}

	resp := decodeResponse(t, rr)
	if resp.Status != "error" {
		t.Errorf("expected status 'error', got %q", resp.Status)
	}
	if resp.Message != "playwright mode not yet implemented" {
		t.Errorf("unexpected message: %q", resp.Message)
	}
}

func TestFetch_Success(t *testing.T) {
	target := newTestTargetServer("<html>hello</html>")
	defer target.Close()

	proxy := newTestProxyServer(target.URL)
	defer proxy.Close()

	rr := doFetch(t, models.FetchRequest{
		URL:     target.URL,
		Proxies: []string{proxy.URL},
		Mode:    "http",
	})

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	resp := decodeResponse(t, rr)
	if resp.Status != "success" {
		t.Errorf("expected status 'success', got %q", resp.Status)
	}
	if resp.Content != "<html>hello</html>" {
		t.Errorf("expected content '<html>hello</html>', got %q", resp.Content)
	}
	if resp.WinningProxy != proxy.URL {
		t.Errorf("expected winning proxy %q, got %q", proxy.URL, resp.WinningProxy)
	}
	if resp.TimeTakenMs < 0 {
		t.Error("expected non-negative time_taken_ms")
	}
}

func TestFetch_AllProxiesFail(t *testing.T) {
	target := newTestTargetServer("unreachable")
	defer target.Close()

	fail1 := newTestFailingProxyServer()
	defer fail1.Close()

	fail2 := newTestFailingProxyServer()
	defer fail2.Close()

	rr := doFetch(t, models.FetchRequest{
		URL:     target.URL,
		Proxies: []string{fail1.URL, fail2.URL},
		Mode:    "http",
	})

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}

	resp := decodeResponse(t, rr)
	if resp.Status != "error" {
		t.Errorf("expected status 'error', got %q", resp.Status)
	}
}

func TestFetch_DefaultModeIsHTTP(t *testing.T) {
	target := newTestTargetServer("default mode")
	defer target.Close()

	proxy := newTestProxyServer(target.URL)
	defer proxy.Close()

	rr := doFetch(t, models.FetchRequest{
		URL:     target.URL,
		Proxies: []string{proxy.URL},
	})

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	resp := decodeResponse(t, rr)
	if resp.Status != "success" {
		t.Errorf("expected success with default mode, got %q", resp.Status)
	}
}

func TestFetch_ContentTypeIsJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/fetch", nil)
	rr := httptest.NewRecorder()
	Fetch(rr, req)

	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", ct)
	}
}

func TestFetch_TooManyProxies(t *testing.T) {
	t.Setenv("MAX_PROXIES", "2")

	rr := doFetch(t, models.FetchRequest{
		URL:     "http://example.com",
		Proxies: []string{"http://p1:8080", "http://p2:8080", "http://p3:8080"},
	})

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}

	resp := decodeResponse(t, rr)
	if resp.Status != "error" {
		t.Errorf("expected status 'error', got %q", resp.Status)
	}
}

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	Health(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", ct)
	}
}

func TestFetch_CustomTimeout(t *testing.T) {
	target := newTestTargetServer("timeout test")
	defer target.Close()

	proxy := newTestProxyServer(target.URL)
	defer proxy.Close()

	rr := doFetch(t, models.FetchRequest{
		URL:            target.URL,
		Proxies:        []string{proxy.URL},
		TimeoutSeconds: 30,
	})

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}
