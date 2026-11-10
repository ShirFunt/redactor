package main

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
)

// redactTransaction redacts one decoded HTTP transaction in place.
// The top-level "headers" and "response.headers" objects are header context;
// everything else is body context. This decides which key rules apply (scope).
func (r *Redactor) redactTransaction(tx map[string]any) {
	for key, val := range tx {
		switch key {
		case "headers":
			tx[key] = r.redactValue(val, true)
		case "response":
			if resp, ok := val.(map[string]any); ok {
				for k, v := range resp {
					resp[k] = r.redactValue(v, k == "headers")
				}
			} else {
				tx[key] = r.redactValue(val, false)
			}
		default:
			tx[key] = r.redactValue(val, false)
		}
	}
}

// redactValue walks any decoded JSON value. It calls itself on every nested
// object and array, so depth does not matter.
func (r *Redactor) redactValue(v any, inHeaders bool) any {
	switch t := v.(type) {
	case map[string]any:
		r.redactObject(t, inHeaders)
		return t
	case []any:
		for i := range t {
			t[i] = r.redactValue(t[i], inHeaders)
		}
		return t
	case string:
		if embedded, ok := r.redactEmbeddedJSON(t, inHeaders); ok {
			return embedded
		}
		return r.redactString(t)
	default: // json.Number, bool, nil: value rules only look at text
		return t
	}
}

// redactEmbeddedJSON handles strings that themselves contain a JSON object or
// array, for example a response body stored as an escaped string. If the text
// is not valid JSON, ok is false and the caller treats it as plain text.
func (r *Redactor) redactEmbeddedJSON(s string, inHeaders bool) (string, bool) {
	trimmed := strings.TrimSpace(s)
	if len(trimmed) < 2 || (trimmed[0] != '{' && trimmed[0] != '[') {
		return "", false
	}
	dec := json.NewDecoder(strings.NewReader(trimmed))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return "", false
	}
	if _, err := dec.Token(); err != io.EOF {
		return "", false // data after the JSON value: treat as text
	}
	v = r.redactValue(v, inHeaders)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", false
	}
	return strings.TrimRight(buf.String(), "\n"), true
}

func (r *Redactor) redactObject(m map[string]any, inHeaders bool) {
	for key, val := range m {
		if rule := r.matchKeyRule(key, inHeaders); rule != nil {
			if newVal, keep := r.applyKeyRule(rule, val); keep {
				m[key] = newVal
			} else {
				delete(m, key)
			}
			continue // the whole value is handled by the key rule
		}
		m[key] = r.redactValue(val, inHeaders)
	}
}

// matchKeyRule returns the first key rule that matches this key name, or nil.
func (r *Redactor) matchKeyRule(key string, inHeaders bool) *KeyRule {
	for i := range r.cfg.KeyRules {
		rule := &r.cfg.KeyRules[i]
		if rule.Scope == "headers" && !inHeaders {
			continue
		}
		if rule.Scope == "body" && inHeaders {
			continue
		}
		if rule.exact[strings.ToLower(key)] || (rule.re != nil && rule.re.MatchString(key)) {
			return rule
		}
	}
	return nil
}

// applyKeyRule redacts the value of a matched key.
// The bool result is false when the key itself must be dropped.
func (r *Redactor) applyKeyRule(rule *KeyRule, val any) (any, bool) {
	if rule.Action == "remove" {
		r.Counts[rule.Name]++
		return nil, false
	}
	if isEmpty(val) {
		return val, true // nothing sensitive to hide
	}
	r.Counts[rule.Name]++
	return r.cfg.applyAction(rule.Action, valueText(val)), true
}

// isEmpty reports values that carry no secret: null, booleans, "", [] and {}.
func isEmpty(v any) bool {
	switch t := v.(type) {
	case nil:
		return true
	case bool:
		return true
	case string:
		return t == ""
	case []any:
		return len(t) == 0
	case map[string]any:
		return len(t) == 0
	}
	return false
}

// valueText turns any value into the text a rule action works on.
// Objects and arrays are serialized, so a secret hidden in a container is still covered.
func valueText(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}
