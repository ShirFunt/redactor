package main

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// applyAction transforms a matched value according to the rule's action.
// "remove" is handled by the caller (drop the key / delete the text), so it returns "".
func (c *Config) applyAction(action, value string) string {
	switch action {
	case "mask":
		return c.Settings.Mask.Replacement
	case "partial":
		return partial(value, c.Settings.Partial.KeepLast, c.Settings.Partial.Fill)
	case "hash":
		if isSHA256Hex(value) {
			return value // already hashed: keeps the tool idempotent
		}
		sum := sha256.Sum256([]byte(value))
		return hex.EncodeToString(sum[:])
	default:
		return ""
	}
}

// partial keeps the last `keep` characters and fills the rest.
// It works on runes, so non-ASCII text is not cut in the middle of a character.
func partial(value string, keep int, fill string) string {
	runes := []rune(value)
	if len(runes) <= keep {
		return strings.Repeat(fill, len(runes))
	}
	return strings.Repeat(fill, len(runes)-keep) + string(runes[len(runes)-keep:])
}

// luhnValid reports whether s is a plausible card number: 13-19 digits
// (spaces and dashes allowed) that pass the Luhn checksum.
func luhnValid(s string) bool {
	sum, digits, double := 0, 0, false
	for i := len(s) - 1; i >= 0; i-- {
		ch := s[i]
		if ch == ' ' || ch == '-' {
			continue
		}
		if ch < '0' || ch > '9' {
			return false
		}
		d := int(ch - '0')
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
		digits++
	}
	return digits >= 13 && digits <= 19 && sum%10 == 0
}

// isSHA256Hex reports whether s looks like a hex-encoded SHA-256 digest.
func isSHA256Hex(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
