package main

import (
	"os"
	"path/filepath"
)

func existingTestDirectory(path string) string {
	if path == "" {
		return os.TempDir()
	}
	candidate := path
	if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
		candidate = filepath.Dir(candidate)
	}
	for {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			return os.TempDir()
		}
		candidate = parent
	}
}
