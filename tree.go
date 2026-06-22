package main

import (
	"fmt"
	"net/http"
	"strings"
)

func treeHandler(w http.ResponseWriter, r *http.Request) {
	root, ok := directoryRequestPath(w, r)
	if !ok {
		return
	}

	entries, err := treeEntries(root)
	if err != nil {
		http.Error(w, "cannot read directory tree: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var output strings.Builder
	_, _ = fmt.Fprintln(&output, "TYPE\tSIZE_BYTES\tPATH")
	for _, entry := range entries {
		_, _ = fmt.Fprintf(&output, "%s\t%d\t%s\n", entry.kind, entry.size, entry.path)
	}

	writePlainText(w, output.String())
}
