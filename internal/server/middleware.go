package server

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// recoverAndLog is the outermost middleware. It recovers panics (so a nil-map or
// index bug in one of the ~71 handlers returns a 500 instead of killing the
// request with a bare stack trace) and emits one structured log line per
// request with a request id, so production is observable at all (HAUSV-141).
func recoverAndLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := newRequestID()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		defer func() {
			if rec := recover(); rec != nil {
				// If nothing has been written yet, send a clean 500. If a partial
				// response already went out, we can only log.
				if !sw.wrote {
					http.Error(sw, "Internal Server Error", http.StatusInternalServerError)
				}
				slog.Error("panic recovered",
					"request_id", reqID,
					"method", r.Method,
					"path", r.URL.Path,
					"host", r.Host,
					"panic", rec,
					"stack", string(debug.Stack()),
				)
			}
			slog.Info("request",
				"request_id", reqID,
				"method", r.Method,
				"path", r.URL.Path,
				"host", r.Host,
				"status", sw.status,
				"bytes", sw.bytes,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		}()

		next.ServeHTTP(sw, r)
	})
}

func newRequestID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "req-unknown"
	}
	return hex.EncodeToString(b[:])
}

// statusWriter records the status code and byte count without altering the
// response. It stays transparent for streaming (Flush) and for anything using
// http.ResponseController (Unwrap) — asset serving and file downloads rely on
// the underlying writer's behaviour.
type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
	wrote  bool
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.wrote = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	w.wrote = true
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// Unwrap lets http.ResponseController reach the real writer (SetReadDeadline,
// Flush, etc.) used by streaming responses.
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
