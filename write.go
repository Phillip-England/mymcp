package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	writeModeOverwrite = "overwrite"
	writeModeAppend    = "append"
)

func writeHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if strings.TrimSpace(path) == "" {
		http.Error(w, "path query parameter must contain a file path", http.StatusBadRequest)
		return
	}

	mode := r.URL.Query().Get("mode")
	flags, err := writeFlags(mode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if info, err := os.Stat(path); err == nil && info.IsDir() {
		http.Error(w, "path is a directory", http.StatusBadRequest)
		return
	} else if err != nil && !os.IsNotExist(err) {
		http.Error(w, "cannot access path: "+err.Error(), http.StatusInternalServerError)
		return
	}

	file, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "parent directory not found", http.StatusNotFound)
			return
		}
		http.Error(w, "cannot open file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	written, writeErr := io.Copy(file, r.Body)
	closeErr := file.Close()
	if writeErr != nil {
		http.Error(w, "cannot write file: "+writeErr.Error(), http.StatusInternalServerError)
		return
	}
	if closeErr != nil {
		http.Error(w, "cannot close file: "+closeErr.Error(), http.StatusInternalServerError)
		return
	}

	writePlainText(w, fmt.Sprintf("%s %d bytes to %s\n", mode, written, path))
}

func writeFlags(mode string) (int, error) {
	switch mode {
	case writeModeOverwrite:
		return os.O_WRONLY | os.O_CREATE | os.O_TRUNC, nil
	case writeModeAppend:
		return os.O_WRONLY | os.O_CREATE | os.O_APPEND, nil
	default:
		return 0, fmt.Errorf("mode must be overwrite or append")
	}
}
