package main

import (
	"fmt"
	"os"
)

func main() {
	cfg, err := LoadConfig("rules.yaml")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Printf("loaded %d key rules and %d value rules\n", len(cfg.KeyRules), len(cfg.ValueRules))
}
