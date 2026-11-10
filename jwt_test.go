package main

import "testing"

// Regression test for the reported bug: real JWTs use base64url ('-' and '_').
func TestRedactString_JWTBase64URL(t *testing.T) {
	r := newTestRedactor(t)
	jwts := []string{
		// signature contains '_'
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
		// payload contains '_', signature contains '-' and '_'
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhIjoiPz8_In0.dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
	}
	for _, jwt := range jwts {
		if got := r.redactString("token " + jwt + " end"); got != "token **** end" {
			t.Errorf("JWT not fully redacted: %q", got)
		}
	}
}
