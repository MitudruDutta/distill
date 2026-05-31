package plugin

import (
	"bytes"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// These tests verify the plugin runtime stays resilient when a plugin
// misbehaves: hangs, forks, floods stdout, ignores stdin, emits binary/invalid
// UTF-8, runs concurrently, or exits non-zero.

// capsLine builds the discovery branch shared by the plugins below.
func capsLine(name, ext string) string {
	return "#!/bin/sh\n" +
		"if [ \"$1\" = \"--distill-capabilities\" ]; then\n" +
		"  echo '{\"name\":\"" + name + "\",\"extensions\":[\"" + ext + "\"]}'\n" +
		"  exit 0\n" +
		"fi\n"
}

func shrinkTimeouts(t *testing.T, conv, disc, grace time.Duration) {
	t.Helper()
	oc, od, og := conversionTimeout, discoveryTimeout, ioGrace
	conversionTimeout, discoveryTimeout, ioGrace = conv, disc, grace
	t.Cleanup(func() { conversionTimeout, discoveryTimeout, ioGrace = oc, od, og })
}

// A plugin that hangs during CONVERSION must be aborted by the timeout,
// promptly (not after the script's own 5s sleep).
func TestConvertTimeoutFires(t *testing.T) {
	cmd := writePlugin(t, capsLine("slow", ".s")+"sleep 5\n")
	shrinkTimeouts(t, 50*time.Millisecond, 5*time.Second, 50*time.Millisecond)
	p := Plugin{Manifest: Manifest{Command: cmd}, Capabilities: Capabilities{Name: "slow"}}
	start := time.Now()
	if _, err := p.Convert([]byte("x")); err == nil {
		t.Fatal("expected conversion timeout error")
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("conversion did not abort promptly: %s", d)
	}
}

// A plugin that exits 0 but leaves a forked child holding the stdout pipe must
// not wedge distill open for the child's lifetime.
func TestConvertForkingChildReturnsPromptly(t *testing.T) {
	cmd := writePlugin(t, capsLine("fork", ".f")+"sleep 5 &\necho done\n")
	shrinkTimeouts(t, 30*time.Second, 5*time.Second, 100*time.Millisecond)
	p := Plugin{Manifest: Manifest{Command: cmd}, Capabilities: Capabilities{Name: "fork"}}
	start := time.Now()
	_, _ = p.Convert([]byte("x"))
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("forking plugin wedged the call open: %s", d)
	}
}

// THE DoS GUARD: a plugin emitting more than the cap must be killed and
// errored, in bounded time and bounded memory.
func TestConvertOutputCapEnforced(t *testing.T) {
	cmd := writePlugin(t, capsLine("big", ".b")+"head -c 5242880 /dev/zero\n") // 5 MiB
	shrinkTimeouts(t, 30*time.Second, 5*time.Second, 100*time.Millisecond)
	old := maxOutputBytes
	maxOutputBytes = 4096
	t.Cleanup(func() { maxOutputBytes = old })
	p := Plugin{Manifest: Manifest{Command: cmd}, Capabilities: Capabilities{Name: "big"}}
	start := time.Now()
	_, err := p.Convert([]byte("x"))
	if !errors.Is(err, errOutputTooLarge) {
		t.Fatalf("expected errOutputTooLarge, got %v", err)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("output cap did not bound time: %s", d)
	}
}

// A plugin that ignores stdin and exits immediately must not deadlock or error
// on a large input (common: sniff first bytes, then exit).
func TestConvertIgnoresStdinNoDeadlock(t *testing.T) {
	cmd := writePlugin(t, capsLine("nostdin", ".n")+"echo ignored\nexit 0\n")
	shrinkTimeouts(t, 30*time.Second, 5*time.Second, 2*time.Second)
	p := Plugin{Manifest: Manifest{Command: cmd}, Capabilities: Capabilities{Name: "nostdin"}}
	big := bytes.Repeat([]byte("A"), 5<<20) // 5 MiB stdin the plugin never reads
	start := time.Now()
	out, err := p.Convert(big)
	if err != nil {
		t.Fatalf("ignoring stdin should not error: %v", err)
	}
	if !strings.Contains(string(out), "ignored") {
		t.Fatalf("output: %q", out)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("ignoring stdin deadlocked: %s", d)
	}
}

// The stdin→stdout channel must be byte-exact (NUL, high bytes, no mangling).
func TestConvertBinaryRoundTrip(t *testing.T) {
	cmd := writePlugin(t, capsLine("cat", ".c")+"cat\n")
	p := Plugin{Manifest: Manifest{Command: cmd}, Capabilities: Capabilities{Name: "cat"}}
	in := []byte{0x00, 0x01, 0xFF, 0xFE, 'h', 'i', 0x00, 0x80}
	out, err := p.Convert(in)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(in, out) {
		t.Fatalf("binary not preserved: in=%v out=%v", in, out)
	}
}

// Empty output with exit 0 is valid (empty Markdown, no error).
func TestConvertEmptyOutputOK(t *testing.T) {
	cmd := writePlugin(t, capsLine("empty", ".e")+"exit 0\n")
	p := Plugin{Manifest: Manifest{Command: cmd}, Capabilities: Capabilities{Name: "empty"}}
	out, err := p.Convert([]byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Fatalf("expected empty output, got %q", out)
	}
}

// Invalid UTF-8 from a plugin must flow through without panic or mangling.
func TestConvertInvalidUTF8FlowsThrough(t *testing.T) {
	cmd := writePlugin(t, capsLine("u", ".u")+"printf '\\377\\376bad'\n")
	p := Plugin{Manifest: Manifest{Command: cmd}, Capabilities: Capabilities{Name: "u"}}
	out, err := p.Convert([]byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out, []byte{0xFF, 0xFE}) {
		t.Fatalf("invalid UTF-8 not preserved: %v", out)
	}
}

// Concurrent conversions must be race-free (batch runs plugins in parallel).
func TestConvertConcurrent(t *testing.T) {
	cmd := writePlugin(t, capsLine("cc", ".cc")+"cat\n")
	p := Plugin{Manifest: Manifest{Command: cmd}, Capabilities: Capabilities{Name: "cc"}}
	const n = 20
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = p.Convert([]byte("payload"))
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("concurrent convert %d failed: %v", i, err)
		}
	}
}

// One plugin that hangs during DISCOVERY (never answers --distill-capabilities)
// must not prevent discovering the rest, and discovery must stay time-bounded.
func TestDiscoverHangingDoesNotBlockOthers(t *testing.T) {
	hang := writePlugin(t, "#!/bin/sh\nsleep 5\n") // never prints caps
	good := writePlugin(t, capsLine("good", ".g")+"cat\n")
	shrinkTimeouts(t, 30*time.Second, 100*time.Millisecond, 50*time.Millisecond)
	start := time.Now()
	plugins, errs := Discover([]Manifest{
		{Name: "hang", Command: hang},
		{Name: "good", Command: good},
	})
	if len(plugins) != 1 || plugins[0].Capabilities.Name != "good" {
		t.Fatalf("expected only 'good' discovered, got %+v", plugins)
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 discovery error (hang), got %v", errs)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("hanging plugin blocked discovery: %s", d)
	}
}

// Partial output followed by a non-zero exit is an error (output discarded).
func TestConvertPartialThenCrash(t *testing.T) {
	cmd := writePlugin(t, capsLine("crash", ".x")+"printf 'partial'\nexit 7\n")
	p := Plugin{Manifest: Manifest{Command: cmd}, Capabilities: Capabilities{Name: "crash"}}
	if _, err := p.Convert([]byte("x")); err == nil {
		t.Fatal("expected error on non-zero exit")
	}
}
