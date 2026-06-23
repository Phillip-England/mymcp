package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func readHandler(w http.ResponseWriter, r *http.Request) {
	recursive, err := parseRecursiveFlag(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	root, ok := pathFromBody(
		w,
		r,
		"request body must contain a path of at most 4096 bytes",
		"request body must contain a file or directory path",
	)
	if !ok {
		return
	}

	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "path not found", http.StatusNotFound)
			return
		}
		http.Error(w, "cannot access path: "+err.Error(), http.StatusInternalServerError)
		return
	}

	paths, err := readablePaths(root, info, recursive)
	if err != nil {
		http.Error(w, "cannot read directory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var output strings.Builder
	for _, path := range paths {
		contents, err := os.ReadFile(path)
		if err != nil {
			http.Error(w, "cannot read file "+path+": "+err.Error(), http.StatusInternalServerError)
			return
		}
		writeFileSection(&output, path, contents)
	}

	writePlainText(w, output.String())
}

func parseRecursiveFlag(r *http.Request) (bool, error) {
	value := r.URL.Query().Get("recursive")
	if value == "" {
		return false, nil
	}

	recursive, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("recursive must be true or false")
	}
	return recursive, nil
}

func readablePaths(root string, info os.FileInfo, recursive bool) ([]string, error) {
	if !info.IsDir() {
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("path is not a regular file or directory")
		}
		return []string{root}, nil
	}

	if recursive {
		var paths []string
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type().IsRegular() {
				paths = append(paths, path)
			}
			return nil
		})
		return paths, err
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Type().IsRegular() {
			paths = append(paths, filepath.Join(root, entry.Name()))
		}
	}
	return paths, nil
}

func writeFileSection(output *strings.Builder, path string, contents []byte) {
	fmt.Fprintf(output, "===== FILE: %s =====\n", path)
	_, _ = output.Write(contents)
	if len(contents) > 0 && contents[len(contents)-1] != '\n' {
		_ = output.WriteByte('\n')
	}
	fmt.Fprintf(output, "===== END FILE: %s =====\n\n", path)
}
