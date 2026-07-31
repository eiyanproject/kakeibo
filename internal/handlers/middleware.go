package handlers

import (
	"bytes"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status >= 400 {
		r.body.Write(b)
	}
	return r.ResponseWriter.Write(b)
}

// withLogging logs every request's method, path, status, and duration, recovers from
// panics (returning a 500 instead of dropping the connection), and logs the response
// body for 5xx responses so errors show up in `journalctl -u kakeibo` instead of only
// ever being visible in the browser.
func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			if err := recover(); err != nil {
				log.Printf("PANIC %s %s: %v\n%s", r.Method, r.URL.Path, err, debug.Stack())
				http.Error(rec, "internal server error", http.StatusInternalServerError)
			}
			dur := time.Since(start)
			if rec.status >= 400 {
				log.Printf("%s %s -> %d (%s) error=%q", r.Method, r.URL.Path, rec.status, dur, strings.TrimSpace(rec.body.String()))
			} else {
				log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, rec.status, dur)
			}
		}()
		next.ServeHTTP(rec, r)
	})
}
