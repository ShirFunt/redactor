package main

import (
	"encoding/json"
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
		return r.redactString(t)
	default: // json.Number, bool, nil: value rules only look at text
		return t
	}
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
