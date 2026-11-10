package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"
)

// newHandler builds the HTTP handler. The config is read-only after loading
// (compiled regexes are safe to share), but a Redactor keeps counters, so
// every request gets its own Redactor. This avoids data races.
func newHandler(cfg *Config) http.Handler {
	mux := http.NewServeMux()

	// The server only starts after the rules loaded successfully, so a 200
	// here means: the process is up and it has valid rules.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	mux.HandleFunc("POST /redact", func(w http.ResponseWriter, req *http.Request) {
		// HTTP/1.x normally stops allowing body reads once the response starts.
		// We read the request and write the response at the same time.
		_ = http.NewResponseController(w).EnableFullDuplex()

		w.Header().Set("Content-Type", "application/x-ndjson")
		r := NewRedactor(cfg)
		if err := r.RedactStream(req.Body, w, os.Stderr); err != nil {
			// Headers are already sent, so the status cannot change any more.
			log.Printf("redact request failed: %v", err)
			return
		}
		log.Printf("redacted request: %s", countsLine(r.Counts))
	})

	return mux
}

// countsLine formats redaction counts for the log. It has rule names and
// numbers only, never data.
func countsLine(counts map[string]int) string {
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, fmt.Sprintf("%s=%d", name, counts[name]))
	}
	return strings.Join(parts, " ")
}

// runServer serves until the process receives SIGINT or SIGTERM, then lets
// running requests finish. Kubernetes stops a pod by sending SIGTERM.
func runServer(cfg *Config, addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           newHandler(cfg),
		ReadHeaderTimeout: 10 * time.Second, // protects against slow-header clients
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	log.Printf("redactor listening on %s", addr)

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Println("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
