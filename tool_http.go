package main

import (
	"io"
	"net/http"
	"os"
	"strings"
)

const maxPathRequestSize = 4096

func directoryRequestPath(w http.ResponseWriter, r *http.Request) (string, bool) {
	path, ok := sandboxedPathFromBody(
		w,
		r,
		"request body must contain a directory path of at most 4096 bytes",
		"request body must contain a directory path",
	)
	if !ok {
		return "", false
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "directory not found", http.StatusNotFound)
		} else {
			http.Error(w, "cannot access directory: "+err.Error(), http.StatusInternalServerError)
		}
		return "", false
	}
	if !info.IsDir() {
		http.Error(w, "path is not a directory", http.StatusBadRequest)
		return "", false
	}
	return path, true
}

func sandboxedPathFromBody(w http.ResponseWriter, r *http.Request, oversizedMessage, emptyMessage string) (string, bool) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxPathRequestSize))
	if err != nil {
		http.Error(w, oversizedMessage, http.StatusBadRequest)
		return "", false
	}

	path := strings.TrimSpace(string(body))
	if path == "" {
		http.Error(w, emptyMessage, http.StatusBadRequest)
		return "", false
	}
	path, err = resolveSandboxPath(r, path)
	if handlePathResolutionError(w, err) {
		return "", false
	}
	return path, true
}

func writePlainText(w http.ResponseWriter, text string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, text)
}
