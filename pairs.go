package main

import (
	"net/url"
	"regexp"
	"strings"
)

// pairRe finds name=value pairs in form bodies, query strings and free text.
// The value stops at whitespace, '&', ';', ',', quotes and angle brackets.
var pairRe = regexp.MustCompile(`([A-Za-z0-9_.-]+)=([^&\s;,"'<>]*)`)

// redactPairs redacts name=value pairs. The name is checked against the key
// rules (body scope). Otherwise the value is URL-decoded and checked against
// the value rules, so "bob%40example.com" is recognised as an email.
// A value that needs no redaction is left exactly as it was.
func (r *Redactor) redactPairs(s string) string {
	if !strings.Contains(s, "=") {
		return s
	}
	return pairRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := pairRe.FindStringSubmatch(m)
		key, val := sub[1], sub[2]
		if val == "" {
			return m
		}

		if rule := r.matchKeyRule(key, false); rule != nil {
			newVal, keep := r.applyKeyRule(rule, val)
			if !keep {
				return "" // "remove" action: drop the whole pair
			}
			if text, ok := newVal.(string); ok {
				return key + "=" + text
			}
			return m
		}

		decoded, err := url.PathUnescape(val)
		if err != nil {
			decoded = val
		}
		redacted := r.redactString(decoded)
		if redacted == decoded {
			return m
		}
		// '*' is kept as it is, so masked values read naturally.
		escaped := strings.ReplaceAll(url.PathEscape(redacted), "%2A", "*")
		return key + "=" + escaped
	})
}
