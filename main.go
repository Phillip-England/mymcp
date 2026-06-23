package main

import (
	_ "embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const defaultPort = "8765"

//go:embed SKILL.md
var embeddedSkill []byte

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) > 0 {
		switch args[0] {
		case "skill":
			return writeSkill(args[1:], stdout)
		default:
			return fmt.Errorf("unknown command %q", args[0])
		}
	}

	return serve()
}

func writeSkill(args []string, stdout io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: mymcp skill <path>")
	}
	if args[0] == "" {
		return fmt.Errorf("skill path cannot be empty")
	}
	if err := os.WriteFile(args[0], embeddedSkill, 0o644); err != nil {
		return fmt.Errorf("write skill file: %w", err)
	}
	fmt.Fprintf(stdout, "wrote %s\n", args[0])
	return nil
}

func serve() error {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	addr := ":" + port
	log.Printf("mymcp listening on http://localhost%s", addr)
	server := &http.Server{
		Addr:              addr,
		Handler:           newHandler(log.Default()),
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		return err
	}
	return nil
}
