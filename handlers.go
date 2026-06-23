package main

import "net/http"

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writePlainText(w, "mymcp is running\n")
}

func mapHandler(w http.ResponseWriter, _ *http.Request) {
	writePlainText(w, modelGuide())
}
