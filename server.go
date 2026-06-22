package main

import (
	"log"
	"net/http"

	"github.com/phillip-england/mymcp/internal/protocol"
)

func newHandler(logger *log.Logger) http.Handler {
	return requestLogger(logger)(newRouter())
}

func newRouter() http.Handler {
	mux := http.NewServeMux()
	handlers := map[string]http.HandlerFunc{
		"/":           healthHandler,
		"/map":        mapHandler,
		"/tool/ls":    lsHandler,
		"/tool/read":  readHandler,
		"/tool/tree":  treeHandler,
		"/tool/write": writeHandler,
	}
	for _, endpoint := range protocol.Endpoints() {
		handler, ok := handlers[endpoint.Path]
		if !ok {
			panic("missing handler for " + endpoint.Path)
		}
		mux.HandleFunc(endpoint.Pattern(), handler)
	}
	if len(handlers) != len(protocol.Endpoints()) {
		panic("handler registered without protocol endpoint")
	}
	return mux
}
