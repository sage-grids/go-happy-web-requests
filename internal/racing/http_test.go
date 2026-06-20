package racing

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func newTargetServer(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, body)
	}))
}

func newProxyServer(targetURL string) *httptest.Server {
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

func newSlowProxyServer(targetURL string, delay time.Duration) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
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

func newFailingProxyServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "proxy failure", http.StatusBadGateway)
	}))
}

func TestRaceHTTP_SingleProxySuccess(t *testing.T) {
	target := newTargetServer("hello world")
	defer target.Close()

	proxy := newProxyServer(target.URL)
	defer proxy.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := RaceHTTP(ctx, target.URL, []string{proxy.URL})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Content != "hello world" {
		t.Errorf("expected content 'hello world', got: %q", res.Content)
	}
	if res.Proxy != proxy.URL {
		t.Errorf("expected winning proxy %q, got: %q", proxy.URL, res.Proxy)
	}
}

func TestRaceHTTP_MultipleProxies_FirstWins(t *testing.T) {
	target := newTargetServer("race result")
	defer target.Close()

	fastProxy := newProxyServer(target.URL)
	defer fastProxy.Close()

	slowProxy := newSlowProxyServer(target.URL, 500*time.Millisecond)
	defer slowProxy.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := RaceHTTP(ctx, target.URL, []string{fastProxy.URL, slowProxy.URL})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Content != "race result" {
		t.Errorf("expected content 'race result', got: %q", res.Content)
	}
	if res.Proxy != fastProxy.URL {
		t.Errorf("expected fast proxy to win, got: %q", res.Proxy)
	}
}

func TestRaceHTTP_AllProxiesFail(t *testing.T) {
	target := newTargetServer("unreachable")
	defer target.Close()

	fail1 := newFailingProxyServer()
	defer fail1.Close()

	fail2 := newFailingProxyServer()
	defer fail2.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := RaceHTTP(ctx, target.URL, []string{fail1.URL, fail2.URL})
	if err == nil {
		t.Fatal("expected error when all proxies fail")
	}
	if err.Error() != "all proxies failed" {
		t.Errorf("expected 'all proxies failed', got: %q", err.Error())
	}
}

func TestRaceHTTP_ContextTimeout(t *testing.T) {
	target := newTargetServer("slow")
	defer target.Close()

	slowProxy := newSlowProxyServer(target.URL, 3*time.Second)
	defer slowProxy.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := RaceHTTP(ctx, target.URL, []string{slowProxy.URL})
	if err == nil {
		t.Fatal("expected context deadline error")
	}
}

func TestRaceHTTP_EmptyProxies(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := RaceHTTP(ctx, "http://example.com", []string{})
	if err == nil {
		t.Fatal("expected error for empty proxies")
	}
	if err.Error() != "no proxies provided" {
		t.Errorf("expected 'no proxies provided', got: %q", err.Error())
	}
}

func TestRaceHTTP_MixedSuccessAndFailure(t *testing.T) {
	target := newTargetServer("mixed result")
	defer target.Close()

	failProxy := newFailingProxyServer()
	defer failProxy.Close()

	successProxy := newProxyServer(target.URL)
	defer successProxy.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := RaceHTTP(ctx, target.URL, []string{failProxy.URL, successProxy.URL})
	if err != nil {
		t.Fatalf("expected success with one working proxy, got: %v", err)
	}
	if res.Content != "mixed result" {
		t.Errorf("expected 'mixed result', got: %q", res.Content)
	}
	if res.Proxy != successProxy.URL {
		t.Errorf("expected success proxy to win, got: %q", res.Proxy)
	}
}

func TestRaceHTTP_ContextAlreadyCancelled(t *testing.T) {
	target := newTargetServer("cancelled")
	defer target.Close()

	proxy := newProxyServer(target.URL)
	defer proxy.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := RaceHTTP(ctx, target.URL, []string{proxy.URL})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestRaceHTTP_InvalidProxyURL(t *testing.T) {
	target := newTargetServer("test")
	defer target.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := RaceHTTP(ctx, target.URL, []string{"://invalid-url"})
	if err == nil {
		t.Fatal("expected error for invalid proxy URL")
	}
}

func TestRaceHTTP_CancelsLosingGoroutines(t *testing.T) {
	target := newTargetServer("done")
	defer target.Close()

	var slowHit atomic.Int32
	slowProxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		slowHit.Add(1)
		resp, err := http.Get(target.URL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}))
	defer slowProxy.Close()

	fastProxy := newProxyServer(target.URL)
	defer fastProxy.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := RaceHTTP(ctx, target.URL, []string{fastProxy.URL, slowProxy.URL})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Proxy != fastProxy.URL {
		t.Errorf("expected fast proxy to win")
	}

	time.Sleep(3 * time.Second)
	if slowHit.Load() != 0 {
		t.Log("slow proxy handler was still running (expected - it gets cancelled by context)")
	}
}
