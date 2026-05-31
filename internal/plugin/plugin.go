// Package plugin implements distill's out-of-process plugin protocol.
//
// A plugin is any executable that supports two invocations:
//
//  1. Discovery: distill runs it with "--distill-capabilities" as the first
//     argument. The plugin prints a single line of Capabilities JSON to stdout
//     and exits 0.
//  2. Conversion: distill runs it with the manifest args (no capability flag),
//     pipes the document bytes to stdin, and reads Markdown from stdout.
//     A non-zero exit is treated as a conversion error.
//
// This keeps plugins language-agnostic (a five-line shell script works) and
// cross-platform (no Go plugin/cgo constraints). Plugins run with the
// privileges of the distill process and are OFF by default — callers opt in.
package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CapabilityFlag is the first argument passed to a plugin during discovery.
const CapabilityFlag = "--distill-capabilities"

// Timeouts are package vars so tests can shorten them.
var (
	discoveryTimeout  = 5 * time.Second
	conversionTimeout = 120 * time.Second
	// ioGrace bounds how long Run waits after the context is cancelled when a
	// child process keeps the stdout/stderr pipes open (e.g. a plugin that
	// forks). Without this, a forking plugin hangs distill past its timeout.
	ioGrace = 2 * time.Second
)

// Manifest is one entry in a plugins.json config file.
type Manifest struct {
	Name    string   `json:"name"`
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
}

// Capabilities is what a plugin reports during discovery.
type Capabilities struct {
	Name       string   `json:"name"`
	Version    string   `json:"version"`
	Extensions []string `json:"extensions"`
	Mimetypes  []string `json:"mimetypes"`
}

// Plugin is a configured plugin plus its discovered capabilities.
type Plugin struct {
	Manifest     Manifest
	Capabilities Capabilities
}

type config struct {
	Plugins []Manifest `json:"plugins"`
}

// ConfigPaths returns the manifest file locations, global first then workspace.
func ConfigPaths() []string {
	var paths []string
	if dir, err := os.UserConfigDir(); err == nil {
		paths = append(paths, filepath.Join(dir, "distill", "plugins.json"))
	}
	paths = append(paths, filepath.Join(".distill", "plugins.json"))
	return paths
}

// LoadManifests reads manifests from the global and workspace config files.
// Missing files are skipped; a malformed file is an error.
func LoadManifests() ([]Manifest, error) {
	var out []Manifest
	for _, p := range ConfigPaths() {
		data, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		var c config
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		out = append(out, c.Plugins...)
	}
	return out, nil
}

// Discover invokes each manifest's command for capability discovery and returns
// the plugins that responded validly. Plugins that fail discovery are skipped
// and reported via errs (non-fatal).
func Discover(manifests []Manifest) (plugins []Plugin, errs []error) {
	for _, m := range manifests {
		caps, err := discoverOne(m)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", m.Name, err))
			continue
		}
		plugins = append(plugins, Plugin{Manifest: m, Capabilities: caps})
	}
	return plugins, errs
}

func discoverOne(m Manifest) (Capabilities, error) {
	if m.Command == "" {
		return Capabilities{}, errors.New("manifest has empty command")
	}
	ctx, cancel := context.WithTimeout(context.Background(), discoveryTimeout)
	defer cancel()

	args := append([]string{CapabilityFlag}, m.Args...)
	cmd := exec.CommandContext(ctx, m.Command, args...)
	cmd.WaitDelay = ioGrace
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return Capabilities{}, fmt.Errorf("capability discovery failed: %w: %s", err, strings.TrimSpace(errb.String()))
	}

	var caps Capabilities
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &caps); err != nil {
		return Capabilities{}, fmt.Errorf("invalid capabilities JSON: %w", err)
	}
	if caps.Name == "" {
		caps.Name = m.Name
	}
	// Normalize extensions to lowercase, dot-prefixed.
	for i, e := range caps.Extensions {
		e = strings.ToLower(strings.TrimSpace(e))
		if e != "" && !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		caps.Extensions[i] = e
	}
	if len(caps.Extensions) == 0 && len(caps.Mimetypes) == 0 {
		return Capabilities{}, errors.New("plugin declares no extensions or mimetypes")
	}
	return caps, nil
}

// Convert pipes data to the plugin's stdin and returns its stdout (Markdown).
func (p Plugin) Convert(data []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), conversionTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, p.Manifest.Command, p.Manifest.Args...)
	cmd.WaitDelay = ioGrace
	cmd.Stdin = bytes.NewReader(data)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("plugin %s: %w: %s", p.Capabilities.Name, err, strings.TrimSpace(errb.String()))
	}
	return out.Bytes(), nil
}
