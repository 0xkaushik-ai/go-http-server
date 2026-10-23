// Package middleware contains reusable HTTP middleware.
//
// A middleware is a function that takes an http.Handler and returns a new
// http.Handler that adds behaviour (logging, recovery, auth, ...) before
// and/or after calling the wrapped one.
package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// Middleware wraps an http.Handler with extra behaviour.
type Middleware func(http.Handler) http.Handler

// Chain applies middlewares so that the first one listed is the outermost,
// i.e. Chain(h, A, B) handles a request as A -> B -> h.
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// statusRecorder captures the status code written by a handler so the
// logger can report it.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Logger logs the method, path, status and duration of every request.
func Logger(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r)

			log.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration", time.Since(start),
				"remote", r.RemoteAddr,
			)
		})
	}
}

// Recover turns a panic inside a handler into a 500 response instead of
// letting it crash the whole server.
func Recover(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Error("panic recovered", "error", err, "stack", string(debug.Stack()))
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
