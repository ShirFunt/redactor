package main

import (
	"strings"
	"testing"
)

func TestCards_WithSeparators(t *testing.T) {
	r := newTestRedactor(t)
	for _, in := range []string{
		"card 4111 1111 1111 1111 on file",
		"please charge 5555-5555-5555-4444 and send",
		"amex 3782 822463 10005.",
	} {
		out := r.redactString(in)
		if out == in {
			t.Errorf("card was not redacted: %q", in)
		}
		if strings.Contains(out, "4111 1111") || strings.Contains(out, "5555-5555") || strings.Contains(out, "822463") {
			t.Errorf("card digits survived: %q", out)
		}
	}
}

func TestCards_SeparatedNumberFailingLuhnIsKept(t *testing.T) {
	r := newTestRedactor(t)
	in := "ref 4111 1111 1111 1112 ok"
	if got := r.redactString(in); got != in {
		t.Errorf("a non-card number was redacted: %q", got)
	}
}
