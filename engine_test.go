package main

import (
	"strings"
	"testing"
)

func newTestRedactor(t *testing.T) *Redactor {
	t.Helper()
	cfg, err := LoadConfig("rules.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return NewRedactor(cfg)
}

func TestRedactString_Emails(t *testing.T) {
	r := newTestRedactor(t)
	out := r.redactString("mail bob@example.com,carol@example.com;dave@example.com.")
	if strings.Contains(out, "@") {
		t.Errorf("an email survived: %q", out)
	}
	if r.Counts["email"] != 3 {
		t.Errorf("want 3 email redactions, got %d", r.Counts["email"])
	}
}

func TestRedactString_CardsUseLuhn(t *testing.T) {
	r := newTestRedactor(t)
	in := "first 4111111111111111 then 4111111111111112 then 5555555555554444."
	want := "first ************1111 then 4111111111111112 then ************4444."
	if got := r.redactString(in); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRedactString_Idempotent(t *testing.T) {
	r := newTestRedactor(t)
	once := r.redactString("bob@example.com card 4111111111111111")
	twice := r.redactString(once)
	if once != twice {
		t.Errorf("not idempotent: %q vs %q", once, twice)
	}
}
