package handler

import (
"log/slog"
"net/http"
"time"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rw, r)

		msg := r.RemoteAddr + " - \"" + r.Method + " " + r.URL.Path + " " + r.Proto + "\""
		attrs := []any{
			"status", rw.status,
			"duration", time.Since(start).String(),
		}

		switch {
		case rw.status >= 500:
			slog.Error(msg, attrs...)
		case rw.status >= 400:
			slog.Warn(msg, attrs...)
		default:
			slog.Info(msg, attrs...)
		}
	})
}
