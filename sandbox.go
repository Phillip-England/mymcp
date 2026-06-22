package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/phillip-england/mymcp/internal/protocol"
)

const sandboxHeader = protocol.SandboxHeader

type sandboxPathError struct {
	status  int
	message string
}

func (e *sandboxPathError) Error() string {
	return e.message
}

func resolveSandboxPath(r *http.Request, requested string) (string, error) {
	rootValue := strings.TrimSpace(r.Header.Get(sandboxHeader))
	if rootValue == "" {
		return "", sandboxError(http.StatusForbidden, sandboxHeader+" header is required")
	}

	root, err := filepath.Abs(rootValue)
	if err != nil {
		return "", sandboxError(http.StatusBadRequest, "invalid sandbox path: "+err.Error())
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", sandboxError(http.StatusBadRequest, "cannot access sandbox: "+err.Error())
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", sandboxError(http.StatusBadRequest, "cannot access sandbox: "+err.Error())
	}
	if !info.IsDir() {
		return "", sandboxError(http.StatusBadRequest, "sandbox path is not a directory")
	}

	target := requested
	if !filepath.IsAbs(target) {
		target = filepath.Join(root, target)
	}
	target = filepath.Clean(target)

	resolved, err := resolveTargetPath(target)
	if err != nil {
		return "", err
	}
	if !pathWithin(root, resolved) {
		return "", sandboxError(http.StatusForbidden, "requested path is outside the sandbox")
	}
	return resolved, nil
}

func resolveTargetPath(target string) (string, error) {
	resolved, err := filepath.EvalSymlinks(target)
	if err == nil {
		return resolved, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	if _, lstatErr := os.Lstat(target); lstatErr == nil {
		return "", sandboxError(http.StatusForbidden, "requested path contains an unresolved symlink")
	}

	// Resolving the parent both validates missing final paths and prevents a
	// symlinked directory from redirecting a newly created file.
	parent, parentErr := filepath.EvalSymlinks(filepath.Dir(target))
	if parentErr != nil {
		return "", parentErr
	}
	return filepath.Join(parent, filepath.Base(target)), nil
}

func pathWithin(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

func handlePathResolutionError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if pathErr, ok := err.(*sandboxPathError); ok {
		http.Error(w, pathErr.message, pathErr.status)
		return true
	}
	if os.IsNotExist(err) {
		http.Error(w, "path not found", http.StatusNotFound)
		return true
	}
	http.Error(w, "cannot resolve path: "+err.Error(), http.StatusInternalServerError)
	return true
}

func sandboxError(status int, message string) error {
	return &sandboxPathError{status: status, message: message}
}
