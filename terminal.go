package main

import (
	"io"
	"net/http"
	"strings"

	"github.com/phillip-england/mymcp/internal/protocol"
)

const maxTerminalMessageSize = 64 * 1024

func terminalHandler(outcome string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxTerminalMessageSize))
		if err != nil {
			http.Error(w, "terminal message must be at most 65536 bytes", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(string(body)) == "" {
			http.Error(w, "terminal message must not be empty", http.StatusBadRequest)
			return
		}

		w.Header().Set(protocol.TerminalHeader, outcome)
		writePlainText(w, string(body))
	}
}
