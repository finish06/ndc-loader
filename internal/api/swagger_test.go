package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/calebdunn/ndc-loader/docs/swagger"
)

func TestSwaggerUI_Serves(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	req := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for /swagger/index.html, got %d", rec.Code)
	}

	body := rec.Body.String()
	if len(body) < 100 {
		t.Error("expected Swagger UI HTML content")
	}
}

func TestSwaggerUI_NoAuth(t *testing.T) {
	mockCS := &mockCheckpointStoreProvider{
		mockCheckpointQuerier:    newMockCheckpointQuerier(),
		mockLastLoadInfoProvider: &mockLastLoadInfoProvider{},
	}
	router := NewRouter(nil, []string{"secret-key"}, nil, mockCS, nil)

	// Swagger should work without API key.
	req := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for swagger without auth, got %d", rec.Code)
	}
}
