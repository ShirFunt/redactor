package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
)

func main() {
	os.Exit(run())
}

func run() int {
	rulesPath := flag.String("rules", "rules.yaml", "path to the rules file")
	flag.Parse()

	cfg, err := LoadConfig(*rulesPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	var in io.Reader = os.Stdin // no file given: read stdin
	if flag.NArg() > 0 {
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: cannot open input file")
			return 1
		}
		defer f.Close()
		in = f
	}

	r := NewRedactor(cfg)
	if err := r.RedactStream(in, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	printSummary(os.Stderr, r.Counts)
	return 0
}

// printSummary writes redaction counts per rule, sorted by name.
func printSummary(w io.Writer, counts map[string]int) {
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)
	fmt.Fprintln(w, "--- redaction summary ---")
	for _, name := range names {
		fmt.Fprintf(w, "%-22s %d\n", name, counts[name])
	}
}
