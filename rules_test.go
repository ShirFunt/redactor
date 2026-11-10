package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_ProvidedRules(t *testing.T) {
	cfg, err := LoadConfig("rules.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if len(cfg.KeyRules) != 5 {
		t.Errorf("want 5 key rules, got %d", len(cfg.KeyRules))
	}
	if len(cfg.ValueRules) != 4 {
		t.Errorf("want 4 value rules, got %d", len(cfg.ValueRules))
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	if _, err := LoadConfig("does-not-exist.yaml"); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}

func TestLoadConfig_BadRegex(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.yaml")
	content := "version: 1\nvalue_rules:\n  - name: bad\n    pattern: '('\n    action: mask\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("expected an error for an invalid regex")
	}
}