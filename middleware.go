package main

import (
	"log"
	"net/http"
	"time"
)

type responseMetrics struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *responseMetrics) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseMetrics) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(body)
	w.bytes += n
	return n, err
}

func (w *responseMetrics) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func requestLogger(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			metrics := &responseMetrics{ResponseWriter: w}

			next.ServeHTTP(metrics, r)
			if metrics.status == 0 {
				metrics.status = http.StatusOK
			}

			logger.Printf(
				"request method=%s path=%q status=%d bytes=%d duration=%s remote=%q",
				r.Method,
				r.URL.RequestURI(),
				metrics.status,
				metrics.bytes,
				time.Since(started).Round(time.Microsecond),
				r.RemoteAddr,
			)
		})
	}
}
