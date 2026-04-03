package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRootRedirect_WithLandingURL(t *testing.T) {
	handler := rootRedirectHandler("https://rx-dag.example.com")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "https://rx-dag.example.com" {
		t.Errorf("expected Location https://rx-dag.example.com, got %s", loc)
	}
}

func TestRootRedirect_WithoutLandingURL(t *testing.T) {
	handler := rootRedirectHandler("")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/swagger/" {
		t.Errorf("expected Location /swagger/, got %s", loc)
	}
}

func TestRootRedirect_EmptyStringLandingURL(t *testing.T) {
	handler := rootRedirectHandler("   ")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/swagger/" {
		t.Errorf("expected Location /swagger/ for whitespace-only URL, got %s", loc)
	}
}

func TestRootRedirect_NoAuth(t *testing.T) {
	// Verify the redirect works through the full router without an API key.
	router := NewRouter(nil, []string{"secret-key"}, nil, nil, nil, nil, "https://rx-dag.example.com")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("expected 302 for / without auth, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "https://rx-dag.example.com" {
		t.Errorf("expected Location https://rx-dag.example.com, got %s", loc)
	}
}
