package main

import "testing"

func TestLuhnValid(t *testing.T) {
	cases := map[string]bool{
		"4111111111111111":    true,
		"4111 1111 1111 1111": true,
		"5555-5555-5555-4444": true,
		"378282246310005":     true,
		"4111111111111112":    false, // fails checksum
		"1234":                false, // too short
		"":                    false,
		"4111a11111111111":    false, // contains a letter
	}
	for input, want := range cases {
		if got := luhnValid(input); got != want {
			t.Errorf("luhnValid(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestPartial(t *testing.T) {
	cases := []struct{ in, want string }{
		{"4111111111111111", "************1111"},
		{"123", "***"},
		{"", ""},
		{"************1111", "************1111"}, // already redacted: unchanged
		{"שלום עולם 1234", "**********1234"},
	}
	for _, c := range cases {
		if got := partial(c.in, 4, "*"); got != c.want {
			t.Errorf("partial(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestApplyAction(t *testing.T) {
	cfg, err := LoadConfig("rules.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.applyAction("mask", "secret"); got != "****" {
		t.Errorf("mask = %q", got)
	}
	wantHash := "ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb"
	if got := cfg.applyAction("hash", "a"); got != wantHash {
		t.Errorf("hash = %q", got)
	}
}
