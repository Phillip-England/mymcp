package main

import (
	"log"
	"net/http"
	"os"
)

const defaultPort = "8765"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	addr := ":" + port
	log.Printf("mymcp listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, newHandler(log.Default())); err != nil {
		log.Fatal(err)
	}
}
