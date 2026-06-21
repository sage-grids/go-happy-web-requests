package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func dummyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func TestAuth_NoTokenSet_FailsClosed(t *testing.T) {
	os.Unsetenv("API_TOKEN")
	os.Unsetenv("AUTH_DISABLED")

	handler := Auth(dummyHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when API_TOKEN is unset, got %d", rr.Code)
	}
}

func TestAuth_AuthDisabled_AllowsRequest(t *testing.T) {
	os.Unsetenv("API_TOKEN")
	t.Setenv("AUTH_DISABLED", "true")

	handler := Auth(dummyHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 with AUTH_DISABLED=true, got %d", rr.Code)
	}
	if rr.Body.String() != "ok" {
		t.Errorf("expected body 'ok', got %q", rr.Body.String())
	}
}

func TestAuth_ValidToken_AllowsRequest(t *testing.T) {
	os.Setenv("API_TOKEN", "test-secret-123")
	defer os.Unsetenv("API_TOKEN")

	handler := Auth(dummyHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer test-secret-123")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestAuth_InvalidToken_Rejects(t *testing.T) {
	os.Setenv("API_TOKEN", "test-secret-123")
	defer os.Unsetenv("API_TOKEN")

	handler := Auth(dummyHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuth_MissingHeader_Rejects(t *testing.T) {
	os.Setenv("API_TOKEN", "test-secret-123")
	defer os.Unsetenv("API_TOKEN")

	handler := Auth(dummyHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuth_MalformedHeader_NoBearerPrefix_Rejects(t *testing.T) {
	os.Setenv("API_TOKEN", "test-secret-123")
	defer os.Unsetenv("API_TOKEN")

	handler := Auth(dummyHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic test-secret-123")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuth_EmptyBearerToken_Rejects(t *testing.T) {
	os.Setenv("API_TOKEN", "test-secret-123")
	defer os.Unsetenv("API_TOKEN")

	handler := Auth(dummyHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer ")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuth_ContentTypeIsJSON(t *testing.T) {
	os.Setenv("API_TOKEN", "test-secret-123")
	defer os.Unsetenv("API_TOKEN")

	handler := Auth(dummyHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", ct)
	}
}
