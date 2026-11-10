package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the parsed form of rules.yaml.
type Config struct {
	Version    int         `yaml:"version"`
	Settings   Settings    `yaml:"settings"`
	KeyRules   []KeyRule   `yaml:"key_rules"`
	ValueRules []ValueRule `yaml:"value_rules"`
}

type Settings struct {
	Mask struct {
		Replacement string `yaml:"replacement"`
	} `yaml:"mask"`
	Partial struct {
		KeepLast int    `yaml:"keep_last"`
		Fill     string `yaml:"fill"`
	} `yaml:"partial"`
	Hash struct {
		Algorithm string `yaml:"algorithm"`
	} `yaml:"hash"`
}

// KeyRule matches on the NAME of a JSON key or HTTP header.
type KeyRule struct {
	Name   string   `yaml:"name"`
	Scope  string   `yaml:"scope"` // headers | body | any
	Match  KeyMatch `yaml:"match"`
	Action string   `yaml:"action"`

	exact map[string]bool // lowercase names, built by compile
	re    *regexp.Regexp  // compiled Match.Regex, built by compile
}

type KeyMatch struct {
	Exact []string `yaml:"exact"`
	Regex string   `yaml:"regex"`
}

// ValueRule matches on the CONTENT of a string value.
type ValueRule struct {
	Name      string `yaml:"name"`
	Pattern   string `yaml:"pattern"`
	Validator string `yaml:"validator"` // optional: "luhn"
	Action    string `yaml:"action"`

	re *regexp.Regexp
}

var validActions = map[string]bool{"mask": true, "partial": true, "hash": true, "remove": true}
var validScopes = map[string]bool{"headers": true, "body": true, "any": true}

// LoadConfig reads, parses and validates a rules file.
// Any problem is returned as an error, so callers can refuse to run
// instead of silently passing data through unredacted.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read rules file: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse rules file: %w", err)
	}
	if err := cfg.compile(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) compile() error {
	// Defaults for settings that were left out.
	if c.Settings.Mask.Replacement == "" {
		c.Settings.Mask.Replacement = "****"
	}
	if c.Settings.Partial.KeepLast == 0 {
		c.Settings.Partial.KeepLast = 4
	}
	if c.Settings.Partial.Fill == "" {
		c.Settings.Partial.Fill = "*"
	}
	if alg := c.Settings.Hash.Algorithm; alg != "" && alg != "sha256" {
		return fmt.Errorf("unsupported hash algorithm %q", alg)
	}

	for i := range c.KeyRules {
		r := &c.KeyRules[i]
		if !validActions[r.Action] {
			return fmt.Errorf("key rule %q: unknown action %q", r.Name, r.Action)
		}
		if !validScopes[r.Scope] {
			return fmt.Errorf("key rule %q: unknown scope %q", r.Name, r.Scope)
		}
		if len(r.Match.Exact) == 0 && r.Match.Regex == "" {
			return fmt.Errorf("key rule %q: needs match.exact or match.regex", r.Name)
		}
		if len(r.Match.Exact) > 0 {
			r.exact = make(map[string]bool, len(r.Match.Exact))
			for _, n := range r.Match.Exact {
				r.exact[strings.ToLower(n)] = true // header names are case-insensitive
			}
		}
		if r.Match.Regex != "" {
			re, err := regexp.Compile(r.Match.Regex)
			if err != nil {
				return fmt.Errorf("key rule %q: bad regex: %w", r.Name, err)
			}
			r.re = re
		}
	}

	for i := range c.ValueRules {
		r := &c.ValueRules[i]
		if !validActions[r.Action] {
			return fmt.Errorf("value rule %q: unknown action %q", r.Name, r.Action)
		}
		if r.Validator != "" && r.Validator != "luhn" {
			return fmt.Errorf("value rule %q: unknown validator %q", r.Name, r.Validator)
		}
		re, err := regexp.Compile(r.Pattern)
		if err != nil {
			return fmt.Errorf("value rule %q: bad pattern: %w", r.Name, err)
		}
		r.re = re
	}
	return nil
}