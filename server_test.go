package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	cfg, err := LoadConfig("rules.yaml")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(newHandler(cfg))
	t.Cleanup(srv.Close)
	return srv
}

func TestServer_Redact(t *testing.T) {
	srv := newTestServer(t)
	body := `{"headers":{},"body":{"email":"bob@example.com","password":"hunter2"}}` + "\n" +
		"not json, mail bob@example.com\n"
	resp, err := http.Post(srv.URL+"/redact", "application/x-ndjson", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	out := string(data)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if strings.Contains(out, "bob@") || strings.Contains(out, "hunter2") {
		t.Errorf("sensitive data in the response: %s", out)
	}
	if !strings.Contains(out, "unparseable_line") {
		t.Errorf("bad line was not replaced by a placeholder: %s", out)
	}
}

func TestServer_StreamsLargeBody(t *testing.T) {
	srv := newTestServer(t)
	const n = 20000 // a few MB: the response starts while the request is still being sent
	var sb strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&sb, `{"headers":{},"body":{"email":"user%d@example.com"}}`+"\n", i)
	}
	resp, err := http.Post(srv.URL+"/redact", "application/x-ndjson", strings.NewReader(sb.String()))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	out := string(data)
	if got := strings.Count(out, "\n"); got != n {
		t.Errorf("want %d lines back, got %d", n, got)
	}
	if strings.Contains(out, "@") {
		t.Error("an email survived")
	}
}

func TestServer_HealthAndMethods(t *testing.T) {
	srv := newTestServer(t)

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("healthz: status %d", resp.StatusCode)
	}

	resp, err = http.Get(srv.URL + "/redact") // only POST is allowed
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("GET /redact: want 405, got %d", resp.StatusCode)
	}
}
