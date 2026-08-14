package server

import (
	"net/http"
	"strings"
)

func (a *app) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' blob:; style-src 'self' 'unsafe-inline'; font-src 'self'; connect-src 'self'; form-action 'self'; base-uri 'self'; frame-ancestors 'none'")
		next.ServeHTTP(&securityResponseWriter{
			ResponseWriter: w,
			app:            a,
			request:        r,
		}, r)
	})
}

type securityResponseWriter struct {
	http.ResponseWriter
	app     *app
	request *http.Request
	wrote   bool
}

func (w *securityResponseWriter) WriteHeader(status int) {
	if w.wrote {
		return
	}
	w.wrote = true
	if w.shouldPreventCaching(status) {
		w.Header().Set("Cache-Control", "no-store")
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *securityResponseWriter) Write(body []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func (w *securityResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *securityResponseWriter) Flush() {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *securityResponseWriter) shouldPreventCaching(status int) bool {
	if status >= http.StatusBadRequest || w.request == nil || w.request.URL == nil {
		return status >= http.StatusBadRequest
	}
	path := w.request.URL.Path
	if w.app != nil {
		path = stripTenantPath(path, w.app.tenantForRequest(w.request).Slug)
	}
	if strings.HasPrefix(path, "/auth/") || strings.HasPrefix(path, "/handover/") || path == "/start" || strings.HasPrefix(path, "/start/") {
		return true
	}
	if path == "/" && w.app != nil && !w.app.isMarketingHost(w.request) {
		return true
	}
	if path == "/app" || strings.HasPrefix(path, "/app/") {
		return true
	}
	return false
}
