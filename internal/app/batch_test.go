package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	converters "github.com/MitudruDutta/distill/internal/converters/src"
)

func TestBatchConvertsTreeMirroringLayout(t *testing.T) {
	in, out := t.TempDir(), t.TempDir()
	mustWrite(t, filepath.Join(in, "a.csv"), "x,y\n1,2\n")
	mustWrite(t, filepath.Join(in, "sub", "b.txt"), "hello")

	ok, bad, err := Batch(converters.Default(), BatchOptions{InDir: in, OutDir: out})
	if err != nil {
		t.Fatal(err)
	}
	if ok != 2 || bad != 0 {
		t.Fatalf("converted=%d failed=%d, want 2/0", ok, bad)
	}

	md, err := os.ReadFile(filepath.Join(out, "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(md), "| x | y |") {
		t.Fatalf("a.md:\n%s", md)
	}
	if _, err := os.Stat(filepath.Join(out, "sub", "b.md")); err != nil {
		t.Fatalf("sub/b.md missing: %v", err)
	}
}

func TestBatchWritesJSONSidecars(t *testing.T) {
	in, out := t.TempDir(), t.TempDir()
	mustWrite(t, filepath.Join(in, "a.csv"), "x\n1\n")

	if _, _, err := Batch(converters.Default(), BatchOptions{InDir: in, OutDir: out, JSON: true}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "a.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"markdown"`) {
		t.Fatalf("a.json:\n%s", data)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
