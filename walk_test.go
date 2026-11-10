package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// redactJSON runs one transaction (as JSON text) through the redactor.
func redactJSON(t *testing.T, r *Redactor, in string) string {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(in))
	dec.UseNumber() // keep big numbers exactly as written
	var tx map[string]any
	if err := dec.Decode(&tx); err != nil {
		t.Fatal(err)
	}
	r.redactTransaction(tx)
	var sb strings.Builder
	enc := json.NewEncoder(&sb)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(tx); err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(sb.String())
}

func TestWalk_NestedArraysAndObjects(t *testing.T) {
	r := newTestRedactor(t)
	out := redactJSON(t, r, `{"headers":{},"body":{"orders":[{"items":[{"sku":"A1","buyer":{"email":"alice@example.com"}}]},{"items":[[{"deep":["carol@example.net"]}]]}]}}`)
	if strings.Contains(out, "@") {
		t.Errorf("an email survived: %s", out)
	}
	if r.Counts["email"] != 2 {
		t.Errorf("want 2 email redactions, got %d", r.Counts["email"])
	}
}

func TestWalk_HeaderNamesAreCaseInsensitive(t *testing.T) {
	r := newTestRedactor(t)
	out := redactJSON(t, r, `{"headers":{"authorization":"Bearer abc","AUTHORIZATION":"Basic dXNlcjpwYXNz","Content-Type":"application/json"},"body":null}`)
	for _, want := range []string{`"authorization":"****"`, `"AUTHORIZATION":"****"`, `application/json`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in %s", want, out)
		}
	}
}

func TestWalk_LookalikeKeysAreNotRedacted(t *testing.T) {
	r := newTestRedactor(t)
	out := redactJSON(t, r, `{"headers":{},"body":{"password_hint":"first pet","passwordless":true,"tokenizer":"bpe","secret_santa":"Ann"}}`)
	for _, want := range []string{"first pet", `"passwordless":true`, "bpe", "Ann"} {
		if !strings.Contains(out, want) {
			t.Errorf("over-redacted: %s missing from %s", want, out)
		}
	}
}

func TestWalk_BodyKeysAreCaseInsensitive(t *testing.T) {
	r := newTestRedactor(t)
	out := redactJSON(t, r, `{"headers":{},"body":{"Password":"a","PASSWORD":"b","ApiKey":"c","Client_Secret":"f"}}`)
	for _, want := range []string{`"Password":"****"`, `"PASSWORD":"****"`, `"Client_Secret":"****"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in %s", want, out)
		}
	}
	if strings.Contains(out, `"ApiKey":"c"`) {
		t.Errorf("ApiKey was not hashed: %s", out)
	}
}

func TestWalk_EmptyValuesStayEmpty(t *testing.T) {
	r := newTestRedactor(t)
	out := redactJSON(t, r, `{"headers":{},"body":{"password":null,"secret":"","token":false,"pwd":[],"card_number":""}}`)
	for _, want := range []string{`"password":null`, `"secret":""`, `"token":false`, `"pwd":[]`, `"card_number":""`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in %s", want, out)
		}
	}
}

func TestWalk_RemoveAndPartial(t *testing.T) {
	r := newTestRedactor(t)
	out := redactJSON(t, r, `{"headers":{},"body":{"user":{"card":{"number":"4111111111111111","cvv":"123"}}}}`)
	if strings.Contains(out, "cvv") {
		t.Errorf("cvv key was not removed: %s", out)
	}
	if !strings.Contains(out, "************1111") {
		t.Errorf("card was not partially masked: %s", out)
	}
}

func TestWalk_NumbersKeepTheirExactText(t *testing.T) {
	r := newTestRedactor(t)
	out := redactJSON(t, r, `{"headers":{},"body":{"order_id":12345678901234567890,"amount":10.50}}`)
	for _, want := range []string{"12345678901234567890", "10.50"} {
		if !strings.Contains(out, want) {
			t.Errorf("number changed: %s missing from %s", want, out)
		}
	}
}

func TestWalk_Idempotent(t *testing.T) {
	r := newTestRedactor(t)
	in := `{"headers":{"Authorization":"Bearer x","X-Api-Key":"sk_live_abc123"},"body":{"user":{"email":"a@b.com","card_number":"4111111111111111","password":"p"},"note":"card 4111111111111111 on file"}}`
	once := redactJSON(t, r, in)
	twice := redactJSON(t, r, once)
	if once != twice {
		t.Errorf("not idempotent:\n%s\n%s", once, twice)
	}
}
