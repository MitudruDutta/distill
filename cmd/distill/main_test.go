package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
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

// setupPluginWorkspace chdir's into a temp dir holding a .distill/plugins.json
// that points at an executable upcasing plugin, plus a sample .up file.
func setupPluginWorkspace(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell-script plugins are POSIX-only in tests")
	}
	dir := t.TempDir()
	t.Chdir(dir)
	script := filepath.Join(dir, "up.sh")
	body := "#!/bin/sh\n" +
		"if [ \"$1\" = \"--distill-capabilities\" ]; then\n" +
		"  echo '{\"name\":\"upcase\",\"version\":\"0.1.0\",\"extensions\":[\".up\"]}'\n" +
		"  exit 0\n" +
		"fi\n" +
		"printf '# upcased\\n\\n'\n" +
		"tr '[:lower:]' '[:upper:]'\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(".distill", 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"plugins":[{"name":"upcase","command":"` + script + `"}]}`
	if err := os.WriteFile(filepath.Join(".distill", "plugins.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("sample.up", []byte("hello world\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunListPluginsShowsConfigured(t *testing.T) {
	setupPluginWorkspace(t)
	var out bytes.Buffer
	if err := run([]string{"--list-plugins"}, nil, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "upcase") || !strings.Contains(out.String(), ".up") {
		t.Fatalf("list output: %q", out.String())
	}
}

func TestRunUsePluginsConvertsViaPlugin(t *testing.T) {
	setupPluginWorkspace(t)
	var out bytes.Buffer
	if err := run([]string{"--use-plugins", "sample.up"}, nil, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "# upcased") || !strings.Contains(out.String(), "HELLO WORLD") {
		t.Fatalf("plugin output: %q", out.String())
	}
}

func TestRunWithoutUsePluginsIgnoresPlugin(t *testing.T) {
	setupPluginWorkspace(t)
	t.Setenv("DISTILL_USE_PLUGINS", "") // ensure env doesn't enable it
	var out bytes.Buffer
	if err := run([]string{"sample.up"}, nil, &out); err != nil {
		t.Fatal(err)
	}
	// Plain-text catch-all leaves content untouched (not upcased).
	if strings.Contains(out.String(), "HELLO WORLD") || strings.Contains(out.String(), "# upcased") {
		t.Fatalf("plugin should be OFF by default, got: %q", out.String())
	}
	if !strings.Contains(out.String(), "hello world") {
		t.Fatalf("expected plain passthrough, got: %q", out.String())
	}
}
