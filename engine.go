package main

// Redactor applies a Config to data and counts what it redacted.
type Redactor struct {
	cfg    *Config
	Counts map[string]int // number of redactions per rule name
}

func NewRedactor(cfg *Config) *Redactor {
	return &Redactor{cfg: cfg, Counts: make(map[string]int)}
}

// redactString applies every value rule to s, in config order.
func (r *Redactor) redactString(s string) string {
	for i := range r.cfg.ValueRules {
		s = r.applyValueRule(&r.cfg.ValueRules[i], s)
	}
	return s
}

func (r *Redactor) applyValueRule(rule *ValueRule, s string) string {
	return rule.re.ReplaceAllStringFunc(s, func(match string) string {
		if rule.Validator == "luhn" && !luhnValid(match) {
			return match // looks like a card but fails the checksum: leave it
		}
		r.Counts[rule.Name]++
		if rule.Action == "remove" {
			return ""
		}
		return r.cfg.applyAction(rule.Action, match)
	})
}
