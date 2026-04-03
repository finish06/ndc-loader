package api

import (
	"net/http"
	"strings"
)

// rootRedirectHandler returns a handler that redirects GET / to the configured
// landing URL, or to /swagger/ if no landing URL is set.
//
//	@Summary		Root redirect
//	@Description	Redirects to the landing page (if LANDING_URL is set) or Swagger UI
//	@Tags			operations
//	@Produce		html
//	@Success		302	"Redirect to landing page or /swagger/"
//	@Router			/ [get]
func rootRedirectHandler(landingURL string) http.HandlerFunc {
	target := "/swagger/"
	if trimmed := strings.TrimSpace(landingURL); trimmed != "" {
		target = trimmed
	}
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target, http.StatusFound)
	}
}
