package main

import (
	"fmt"
	"net/http"
	"strings"
)

func lsHandler(w http.ResponseWriter, r *http.Request) {
	root, ok := directoryRequestPath(w, r)
	if !ok {
		return
	}

	children, err := listDirectoryEntries(root)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var output strings.Builder
	_, _ = fmt.Fprintln(&output, "TYPE\tSIZE_BYTES\tNAME")
	for _, child := range children {
		_, _ = fmt.Fprintf(&output, "%s\t%d\t%s\n", child.kind, child.size, child.name)
	}

	writePlainText(w, output.String())
}
