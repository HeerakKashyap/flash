package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goflash/flash/v2"
)

func TestRateLimitBlocksAfterCapacity(t *testing.T) {
	a := flash.New()
	lim := NewSimpleIPLimiter(1, 100*time.Millisecond)
	a.Use(RateLimit(lim))
	a.GET("/", func(c flash.Ctx) error { return c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("first request should pass")
	}

	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second request should be 429, got %d", rec.Code)
	}
}

func TestRateLimitSetsRetryAfter(t *testing.T) {
	a := flash.New()
	lim := NewSimpleIPLimiter(1, 200*time.Millisecond)
	a.Use(RateLimit(lim))
	a.GET("/", func(c flash.Ctx) error { return c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, req)

	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	h := rec.Header().Get("Retry-After")
	if h == "" {
		t.Fatalf("missing Retry-After header")
	}
	// For sub-second retry durations, header should be "1"
	if h != "1" {
		t.Fatalf("expected Retry-After '1' for sub-second retry, got %q", h)
	}
}

func TestRateLimitResetAllowsAfterRetry(t *testing.T) {
	a := flash.New()
	lim := NewSimpleIPLimiter(1, 20*time.Millisecond)
	a.Use(RateLimit(lim))
	a.GET("/", func(c flash.Ctx) error { return c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, req) // first allowed

	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, req) // second blocked
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429")
	}
	time.Sleep(25 * time.Millisecond)
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected allowed after reset")
	}
}

func TestRateLimitRemainingDecrementBranch(t *testing.T) {
	a := flash.New()
	lim := NewSimpleIPLimiter(2, 1*time.Second)
	a.Use(RateLimit(lim))
	a.GET("/", func(c flash.Ctx) error { return c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, req) // first allowed, remaining=1
	if rec.Code != http.StatusOK {
		t.Fatalf("first request should pass")
	}

	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, req) // second allowed, remaining=0
	if rec.Code != http.StatusOK {
		t.Fatalf("second request should pass")
	}
}

// captureLimiter captures the last key seen.
type captureLimiter struct{ last string }

func (c *captureLimiter) Allow(key string) (bool, time.Duration) { c.last = key; return true, 0 }

func TestClientIPExtraction(t *testing.T) {
	// X-Forwarded-For should win and trim spaces, pick first IP
	{
		a := flash.New()
		cap := &captureLimiter{}
		a.Use(RateLimit(cap))
		a.GET("/", func(c flash.Ctx) error { return c.String(http.StatusOK, "ok") })
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Forwarded-For", " 1.2.3.4, 5.6.7.8 ")
		rec := httptest.NewRecorder()
		a.ServeHTTP(rec, req)
		if cap.last != "1.2.3.4" {
			t.Fatalf("xff: got %q want %q", cap.last, "1.2.3.4")
		}
	}
	// X-Real-IP should be used when no XFF
	{
		a := flash.New()
		cap := &captureLimiter{}
		a.Use(RateLimit(cap))
		a.GET("/", func(c flash.Ctx) error { return c.String(http.StatusOK, "ok") })
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Real-IP", "9.8.7.6")
		rec := httptest.NewRecorder()
		a.ServeHTTP(rec, req)
		if cap.last != "9.8.7.6" {
			t.Fatalf("x-real-ip: got %q want %q", cap.last, "9.8.7.6")
		}
	}
	// RemoteAddr with host:port should use host part
	{
		a := flash.New()
		cap := &captureLimiter{}
		a.Use(RateLimit(cap))
		a.GET("/", func(c flash.Ctx) error { return c.String(http.StatusOK, "ok") })
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "127.0.0.1:9898"
		rec := httptest.NewRecorder()
		a.ServeHTTP(rec, req)
		if cap.last != "127.0.0.1" {
			t.Fatalf("remote host: got %q want %q", cap.last, "127.0.0.1")
		}
	}
	// RemoteAddr without colon should fall back to as-is value
	{
		a := flash.New()
		cap := &captureLimiter{}
		a.Use(RateLimit(cap))
		a.GET("/", func(c flash.Ctx) error { return c.String(http.StatusOK, "ok") })
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "badremote"
		rec := httptest.NewRecorder()
		a.ServeHTTP(rec, req)
		if cap.last != "badremote" {
			t.Fatalf("remote fallback: got %q want %q", cap.last, "badremote")
		}
	}
}

func TestFixedWindowLimiter(t *testing.T) {
	lim := NewFixedWindowLimiter(2, 100*time.Millisecond)

	// First two requests should pass
	allowed, _ := lim.Allow("key1")
	if !allowed {
		t.Fatal("first request should be allowed")
	}
	allowed, _ = lim.Allow("key1")
	if !allowed {
		t.Fatal("second request should be allowed")
	}

	// Third should be blocked
	allowed, retry := lim.Allow("key1")
	if allowed {
		t.Fatal("third request should be blocked")
	}
	if retry <= 0 {
		t.Fatal("retry should be positive")
	}

	// Wait for window to reset
	time.Sleep(110 * time.Millisecond)
	allowed, _ = lim.Allow("key1")
	if !allowed {
		t.Fatal("request after window reset should be allowed")
	}
}

func TestFixedWindowLimiterDifferentKeys(t *testing.T) {
	lim := NewFixedWindowLimiter(1, 100*time.Millisecond)

	// Different keys should have separate limits
	allowed1, _ := lim.Allow("key1")
	allowed2, _ := lim.Allow("key2")

	if !allowed1 || !allowed2 {
		t.Fatal("different keys should have independent limits")
	}
}

func TestSlidingWindowLimiter(t *testing.T) {
	lim := NewSlidingWindowLimiter(2, 100*time.Millisecond)

	// First two requests should pass
	allowed, _ := lim.Allow("key1")
	if !allowed {
		t.Fatal("first request should be allowed")
	}
	allowed, _ = lim.Allow("key1")
	if !allowed {
		t.Fatal("second request should be allowed")
	}

	// Third should be blocked
	allowed, retry := lim.Allow("key1")
	if allowed {
		t.Fatal("third request should be blocked")
	}
	if retry <= 0 {
		t.Fatal("retry should be positive")
	}

	// Wait for oldest request to expire
	time.Sleep(110 * time.Millisecond)
	allowed, _ = lim.Allow("key1")
	if !allowed {
		t.Fatal("request after window slide should be allowed")
	}
}

func TestSlidingWindowLimiterWithMiddleware(t *testing.T) {
	a := flash.New()
	lim := NewSlidingWindowLimiter(1, 50*time.Millisecond)
	a.Use(RateLimit(lim))
	a.GET("/", func(c flash.Ctx) error { return c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("first request should pass, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second request should be 429, got %d", rec.Code)
	}
}
