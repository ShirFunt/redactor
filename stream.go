package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// RedactStream reads NDJSON from in, one line at a time, and writes redacted
// NDJSON to out. Memory use is bounded by the longest single line, not by the
// size of the input. Warnings contain line numbers only, never line content.
func (r *Redactor) RedactStream(in io.Reader, out, warn io.Writer) error {
	reader := bufio.NewReader(in)
	writer := bufio.NewWriter(out)
	lineNo := 0
	for {
		line, readErr := reader.ReadBytes('\n')
		if len(line) > 0 { // also true for a last line with no trailing newline
			lineNo++
			if err := r.processLine(line, lineNo, writer, warn); err != nil {
				return err
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	return writer.Flush()
}

func (r *Redactor) processLine(line []byte, lineNo int, w, warn io.Writer) error {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return nil // blank line: skip
	}

	dec := json.NewDecoder(bytes.NewReader(line))
	dec.UseNumber() // keep numbers exactly as written
	var v any
	err := dec.Decode(&v)
	if err == nil {
		// Anything after the first JSON value means the line is malformed.
		if _, extra := dec.Token(); extra != io.EOF {
			err = errors.New("trailing data")
		}
	}
	if err != nil {
		return r.writePlaceholder(lineNo, w, warn)
	}

	if tx, ok := v.(map[string]any); ok {
		r.redactTransaction(tx)
	} else {
		v = r.redactValue(v, false) // valid JSON that is not an object
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return r.writePlaceholder(lineNo, w, warn)
	}
	_, err = w.Write(buf.Bytes())
	return err
}

func (r *Redactor) writePlaceholder(lineNo int, w, warn io.Writer) error {
	r.Counts["(unparseable line)"]++
	fmt.Fprintf(warn, "warning: line %d is not valid JSON; replaced with a placeholder\n", lineNo)
	// Keys are written in alphabetical order, the same order the JSON encoder
	// uses, so running the tool on its own output leaves this line unchanged.
	_, err := fmt.Fprintf(w, "{\"line\":%d,\"redactor_error\":\"unparseable_line\"}\n", lineNo)
	return err
}
