package main

import (
	"log"
	"net/http"
	"time"
)

type responseMetrics struct {
	http.ResponseWriter
	status int
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
	return w.ResponseWriter.Write(body)
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
				"[%s][%s][%d][%s]",
				r.Method,
				r.URL.Path,
				metrics.status,
				time.Since(started).Round(time.Microsecond),
			)
		})
	}
}
