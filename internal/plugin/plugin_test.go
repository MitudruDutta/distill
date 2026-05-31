package plugin

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// writePlugin writes an executable shell-script plugin and returns its path.
func writePlugin(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell-script plugins are POSIX-only in tests")
	}
	p := filepath.Join(t.TempDir(), "plugin.sh")
	if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

const fooPlugin = `#!/bin/sh
if [ "$1" = "--distill-capabilities" ]; then
  echo '{"name":"foo","version":"1.2.3","extensions":["foo",".BAR"],"mimetypes":["application/x-foo"]}'
  exit 0
fi
printf '# foo plugin\n\n'
cat
`

func TestDiscoverHappyPathAndNormalizesExtensions(t *testing.T) {
	cmd := writePlugin(t, fooPlugin)
	plugins, errs := Discover([]Manifest{{Name: "foo", Command: cmd}})
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %v", errs)
	}
	if len(plugins) != 1 {
		t.Fatalf("want 1 plugin, got %d", len(plugins))
	}
	c := plugins[0].Capabilities
	if c.Name != "foo" || c.Version != "1.2.3" {
		t.Errorf("caps name/version = %q/%q", c.Name, c.Version)
	}
	// "foo" → ".foo", ".BAR" → ".bar"
	if len(c.Extensions) != 2 || c.Extensions[0] != ".foo" || c.Extensions[1] != ".bar" {
		t.Errorf("extensions not normalized: %v", c.Extensions)
	}
}

func TestDiscoverMissingCommand(t *testing.T) {
	_, errs := Discover([]Manifest{{Name: "ghost", Command: "/no/such/executable-xyz"}})
	if len(errs) != 1 {
		t.Fatalf("want 1 error, got %d: %v", len(errs), errs)
	}
}

func TestDiscoverEmptyCommand(t *testing.T) {
	_, errs := Discover([]Manifest{{Name: "blank", Command: ""}})
	if len(errs) != 1 {
		t.Fatalf("want 1 error for empty command, got %v", errs)
	}
}

func TestDiscoverInvalidJSON(t *testing.T) {
	cmd := writePlugin(t, "#!/bin/sh\necho 'not json'\n")
	_, errs := Discover([]Manifest{{Name: "bad", Command: cmd}})
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "invalid capabilities JSON") {
		t.Fatalf("want invalid-JSON error, got %v", errs)
	}
}

func TestDiscoverNoExtensionsOrMimetypes(t *testing.T) {
	cmd := writePlugin(t, "#!/bin/sh\necho '{\"name\":\"x\"}'\n")
	_, errs := Discover([]Manifest{{Name: "x", Command: cmd}})
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "no extensions or mimetypes") {
		t.Fatalf("want no-formats error, got %v", errs)
	}
}

func TestDiscoverTimeout(t *testing.T) {
	cmd := writePlugin(t, "#!/bin/sh\nsleep 5\n")
	oldT, oldG := discoveryTimeout, ioGrace
	discoveryTimeout = 50 * time.Millisecond
	ioGrace = 50 * time.Millisecond
	defer func() { discoveryTimeout, ioGrace = oldT, oldG }()
	start := time.Now()
	_, errs := Discover([]Manifest{{Name: "slow", Command: cmd}})
	if len(errs) != 1 {
		t.Fatalf("want timeout error, got %v", errs)
	}
	// WaitDelay must bound the wait well under the script's 5s sleep.
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("discovery did not abort promptly: took %s", elapsed)
	}
}

func TestPluginConvert(t *testing.T) {
	cmd := writePlugin(t, fooPlugin)
	plugins, _ := Discover([]Manifest{{Name: "foo", Command: cmd}})
	out, err := plugins[0].Convert([]byte("payload"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	if !strings.Contains(got, "# foo plugin") || !strings.Contains(got, "payload") {
		t.Fatalf("unexpected conversion output:\n%s", got)
	}
}

func TestPluginConvertFailureSurfacesStderr(t *testing.T) {
	cmd := writePlugin(t, `#!/bin/sh
if [ "$1" = "--distill-capabilities" ]; then echo '{"name":"f","extensions":[".f"]}'; exit 0; fi
echo "boom" >&2
exit 3
`)
	plugins, errs := Discover([]Manifest{{Name: "f", Command: cmd}})
	if len(errs) != 0 {
		t.Fatalf("discovery should succeed: %v", errs)
	}
	_, err := plugins[0].Convert([]byte("x"))
	if err == nil {
		t.Fatal("expected conversion error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error should include stderr: %v", err)
	}
}

func TestLoadManifestsWorkspaceFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.MkdirAll(".distill", 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"plugins":[{"name":"a","command":"/bin/echo","args":["hi"]}]}`
	if err := os.WriteFile(filepath.Join(".distill", "plugins.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadManifests()
	if err != nil {
		t.Fatal(err)
	}
	// global config may or may not exist; ensure our workspace entry is present.
	found := false
	for _, m := range got {
		if m.Name == "a" && m.Command == "/bin/echo" {
			found = true
		}
	}
	if !found {
		t.Fatalf("workspace manifest not loaded: %v", got)
	}
}

func TestLoadManifestsMalformedFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	os.MkdirAll(".distill", 0o755)
	os.WriteFile(filepath.Join(".distill", "plugins.json"), []byte("{ this is not json"), 0o644)
	if _, err := LoadManifests(); err == nil {
		t.Fatal("expected error for malformed plugins.json")
	}
}
