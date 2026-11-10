package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func runStream(t *testing.T, r *Redactor, in string) (out, warn string) {
	t.Helper()
	var o, w bytes.Buffer
	if err := r.RedactStream(strings.NewReader(in), &o, &w); err != nil {
		t.Fatal(err)
	}
	return o.String(), w.String()
}

func TestStream_BadLinesNeverCrashOrLeak(t *testing.T) {
	r := newTestRedactor(t)
	in := "{\"email\":\"trunc@exam\n" + // truncated JSON
		"2025-01-15 WARN upstream timeout user=erin@example.com\n" + // not JSON
		"   \n" + // whitespace only
		"\n" + // blank
		"{\"ok\":true} trailing-garbage\n" // data after the JSON value
	out, warn := runStream(t, r, in)
	for _, leaked := range []string{"trunc@", "erin@", "garbage"} {
		if strings.Contains(out, leaked) || strings.Contains(warn, leaked) {
			t.Errorf("%q leaked into the output or the warnings", leaked)
		}
	}
	if got := strings.Count(out, "unparseable_line"); got != 3 {
		t.Errorf("want 3 placeholders, got %d", got)
	}
}

func TestStream_NonObjectLines(t *testing.T) {
	r := newTestRedactor(t)
	out, _ := runStream(t, r, "42\n[\"a@b.com\",\"4111111111111111\"]\n")
	if !strings.HasPrefix(out, "42\n") {
		t.Errorf("a bare number should pass through unchanged: %q", out)
	}
	if strings.Contains(out, "@") || strings.Contains(out, "4111111111111111") {
		t.Errorf("array elements were not redacted: %q", out)
	}
}

func TestStream_LastLineWithoutNewline(t *testing.T) {
	r := newTestRedactor(t)
	out, _ := runStream(t, r, `{"body":{"email":"a@b.com"}}`)
	if strings.Contains(out, "@") {
		t.Errorf("email survived: %q", out)
	}
	if !strings.HasSuffix(out, "\n") {
		t.Errorf("output line should end with a newline: %q", out)
	}
}

func TestStream_ProvidedInputIsIdempotent(t *testing.T) {
	data, err := os.ReadFile("input.ndjson")
	if err != nil {
		t.Fatal(err)
	}
	once, _ := runStream(t, newTestRedactor(t), string(data))
	twice, _ := runStream(t, newTestRedactor(t), once)
	if once != twice {
		line, a, b := firstDiff(once, twice)
		t.Errorf("output changed on the second run, first difference at output line %d:\nrun 1: %s\nrun 2: %s", line, a, b)
	}
}

// firstDiff returns the 1-based number of the first line that differs, and both versions.
func firstDiff(a, b string) (int, string, string) {
	al := strings.Split(a, "\n")
	bl := strings.Split(b, "\n")
	for i := 0; i < len(al) && i < len(bl); i++ {
		if al[i] != bl[i] {
			return i + 1, al[i], bl[i]
		}
	}
	return len(al), "(lengths differ)", "(lengths differ)"
}
