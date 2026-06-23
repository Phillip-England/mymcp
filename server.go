package main

import (
	"log"
	"net/http"
	"net/url"

	"github.com/phillip-england/mymcp/internal/protocol"
)

func newHandler(logger *log.Logger) http.Handler {
	return requestLogger(logger)(newRouter())
}

func newRouter() http.Handler {
	mux := http.NewServeMux()
	handlers := map[string]http.HandlerFunc{
		protocol.HealthName:  healthHandler,
		protocol.MapName:     mapHandler,
		protocol.ErrorName:   terminalHandler(protocol.TerminalError),
		protocol.SuccessName: terminalHandler(protocol.TerminalSuccess),
		protocol.LSName:      lsHandler,
		protocol.ReadName:    readHandler,
		protocol.TreeName:    treeHandler,
		protocol.WriteName:   writeHandler,
	}
	for _, endpoint := range protocol.Endpoints() {
		handler, ok := handlers[endpoint.Name]
		if !ok {
			panic("missing handler for endpoint " + endpoint.Name)
		}
		mux.HandleFunc(endpoint.Pattern(), validateEndpointQuery(endpoint, handler))
	}
	if len(handlers) != len(protocol.Endpoints()) {
		panic("handler registered without protocol endpoint")
	}
	return mux
}

func validateEndpointQuery(endpoint protocol.Endpoint, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			http.Error(w, "invalid query string: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := endpoint.ValidateQuery(query); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		next(w, r)
	}
}
