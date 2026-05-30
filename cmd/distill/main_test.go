package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunConvertsStdinCSVToStdout(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"-x", "csv"}, strings.NewReader("a,b\n1,2\n"), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "| a | b |") {
		t.Fatalf("stdout:\n%s", out.String())
	}
}

// Regression: flags placed after the filename must still be honored.
func TestRunHonorsFlagsAfterFilename(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "t.tsv")
	outPath := filepath.Join(dir, "t.md")
	if err := os.WriteFile(in, []byte("a\tb\n1\t2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{in, "-o", outPath}, nil, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("output file not written: %v", err)
	}
	if !strings.Contains(string(got), "| a | b |") {
		t.Fatalf("file contents:\n%s", got)
	}
}

func TestRunEmitsJSONModel(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"-x", "csv", "-json"}, strings.NewReader("a\n1\n"), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"markdown"`) {
		t.Fatalf("expected JSON model, got:\n%s", out.String())
	}
}

func TestRunBatchSubcommand(t *testing.T) {
	in, out := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(in, "a.csv"), []byte("x,y\n1,2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := run([]string{"batch", in, "--out-dir", out}, nil, &buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "converted 1") {
		t.Fatalf("summary: %q", buf.String())
	}
	if _, err := os.Stat(filepath.Join(out, "a.md")); err != nil {
		t.Fatalf("output missing: %v", err)
	}
}
