package api

import (
	"log"
	"net/http"
)

func maskPassword(p string) string {
	if p == "" {
		return ""
	}
	return "********"
}

func jsonInternal(w http.ResponseWriter, err error) {
	if err != nil {
		log.Printf("api: %v", err)
	}
	jsonError(w, http.StatusInternalServerError, "internal error")
}

func cookieSecure(r *http.Request, dashboardURL string) bool {
	if r.TLS != nil {
		return true
	}
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		return true
	}
	return len(dashboardURL) >= 8 && (dashboardURL[:8] == "https://")
}
