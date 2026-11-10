package main

import (
	"strings"
	"testing"
)

func TestPairs_FormEncodedBody(t *testing.T) {
	r := newTestRedactor(t)
	out := redactJSON(t, r, `{"headers":{},"body":"email=bob%40example.com&password=hunter2&card=4111111111111111&remember=on"}`)
	for _, leaked := range []string{"bob", "hunter2", "4111111111111111"} {
		if strings.Contains(out, leaked) {
			t.Errorf("%q survived in %s", leaked, out)
		}
	}
	for _, kept := range []string{"password=****", "remember=on"} {
		if !strings.Contains(out, kept) {
			t.Errorf("%q missing from %s", kept, out)
		}
	}
}

func TestPairs_FreeText(t *testing.T) {
	r := newTestRedactor(t)
	out := redactJSON(t, r, `{"headers":{},"body":{"debug":"login attempt password=hunter2 for user=zed"}}`)
	if strings.Contains(out, "hunter2") {
		t.Errorf("password survived: %s", out)
	}
	if !strings.Contains(out, "password=****") || !strings.Contains(out, "user=zed") {
		t.Errorf("unexpected result: %s", out)
	}
}

func TestPairs_QueryStringInPath(t *testing.T) {
	r := newTestRedactor(t)
	out := redactJSON(t, r, `{"path":"/v1/reset?token=abc123def456&email=gina%40example.com","headers":{},"body":null}`)
	for _, leaked := range []string{"abc123def456", "gina"} {
		if strings.Contains(out, leaked) {
			t.Errorf("%q survived in %s", leaked, out)
		}
	}
	if !strings.Contains(out, "/v1/reset?token=****") {
		t.Errorf("path structure was damaged: %s", out)
	}
}

func TestPairs_Base64PaddingIsNotTouched(t *testing.T) {
	r := newTestRedactor(t)
	out := redactJSON(t, r, `{"headers":{},"body":"UEsDBBQAAAAIAA=="}`)
	if !strings.Contains(out, "UEsDBBQAAAAIAA==") {
		t.Errorf("base64 text was changed: %s", out)
	}
}

func TestPairs_Idempotent(t *testing.T) {
	r := newTestRedactor(t)
	in := `{"path":"/v1/reset?token=abc&email=gina%40example.com","headers":{},"body":"email=bob%40example.com&password=hunter2&card=4111111111111111&remember=on"}`
	once := redactJSON(t, r, in)
	twice := redactJSON(t, r, once)
	if once != twice {
		t.Errorf("not idempotent:\n%s\n%s", once, twice)
	}
}
