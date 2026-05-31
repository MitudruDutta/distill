package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/MitudruDutta/distill/internal/app"
	"github.com/MitudruDutta/distill/internal/convert"
	converters "github.com/MitudruDutta/distill/internal/converters/src"
	"github.com/MitudruDutta/distill/internal/plugin"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "distill:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) > 0 {
		switch args[0] {
		case "batch":
			return runBatch(args[1:], stdout)
		case "serve":
			return runServe(args[1:], stdout)
		case "mcp":
			return app.MCP(registryWithPlugins(envPlugins()), stdin, stdout)
		}
	}
	return runConvert(args, stdin, stdout)
}

// envPlugins reports whether DISTILL_USE_PLUGINS is set (non-empty).
func envPlugins() bool { return os.Getenv("DISTILL_USE_PLUGINS") != "" }

// registryWithPlugins returns the default registry; when enable is true it also
// loads, discovers, and registers configured plugins (ahead of built-ins).
// Plugin discovery problems are reported to stderr and are non-fatal.
func registryWithPlugins(enable bool) *convert.Registry {
	reg := converters.Default()
	if !enable {
		return reg
	}
	_, derrs, err := converters.RegisterPlugins(reg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "distill: plugin config:", err)
	}
	for _, e := range derrs {
		fmt.Fprintln(os.Stderr, "distill: plugin:", e)
	}
	return reg
}

// listConfiguredPlugins prints discovered plugins (and discovery failures).
func listConfiguredPlugins(stdout io.Writer) error {
	manifests, err := plugin.LoadManifests()
	if err != nil {
		return err
	}
	if len(manifests) == 0 {
		fmt.Fprintln(stdout, "No plugins configured. See docs/plugins.md to add one.")
		return nil
	}
	plugins, errs := plugin.Discover(manifests)
	for _, p := range plugins {
		formats := append(append([]string{}, p.Capabilities.Extensions...), p.Capabilities.Mimetypes...)
		fmt.Fprintf(stdout, "%-20s %-10s %s\n", p.Capabilities.Name, p.Capabilities.Version, strings.Join(formats, " "))
	}
	for _, e := range errs {
		fmt.Fprintln(stdout, "(unavailable)", e)
	}
	return nil
}

func runConvert(args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("distill", flag.ContinueOnError)
	out := fs.String("o", "", "output file (default: stdout)")
	ext := fs.String("x", "", "extension hint, e.g. csv (useful for stdin)")
	mimeHint := fs.String("m", "", "MIME type hint")
	charset := fs.String("c", "", "charset hint, e.g. utf-8")
	asJSON := fs.Bool("json", false, "emit a JSON document model instead of Markdown")
	userAgent := fs.String("user-agent", os.Getenv("DISTILL_USER_AGENT"),
		"HTTP User-Agent for URL fetches (env: DISTILL_USER_AGENT)")
	usePlugins := fs.Bool("use-plugins", envPlugins(), "enable third-party converter plugins (env: DISTILL_USE_PLUGINS)")
	listPlugins := fs.Bool("list-plugins", false, "list configured plugins and exit")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *listPlugins {
		return listConfiguredPlugins(stdout)
	}
	// Allow flags after the filename (the flag package stops at the first positional).
	var files []string
	for rest := fs.Args(); len(rest) > 0; rest = fs.Args() {
		files = append(files, rest[0])
		if err := fs.Parse(rest[1:]); err != nil {
			return err
		}
	}

	base := convert.StreamInfo{Mimetype: *mimeHint, Charset: *charset}
	if *ext != "" {
		e := strings.ToLower(*ext)
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		base.Extension = e
	}

	r := stdin
	if len(files) > 0 {
		if app.IsURI(files[0]) {
			data, info, err := app.FetchURI(files[0], app.FetchOptions{UserAgent: *userAgent})
			if err != nil {
				return err
			}
			r = bytes.NewReader(data)
			base = base.Merge(info)
		} else {
			f, err := os.Open(files[0])
			if err != nil {
				return err
			}
			defer f.Close()
			r = f
			base.Filename, base.LocalPath = files[0], files[0]
			if base.Extension == "" {
				base.Extension = convert.ExtensionOf(files[0])
			}
		}
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	peek := data
	if len(peek) > 512 {
		peek = peek[:512]
	}
	res, err := registryWithPlugins(*usePlugins).Convert(bytes.NewReader(data), convert.Guess(base, peek))
	if err != nil {
		return err
	}

	output := []byte(res.Markdown)
	if *asJSON {
		if output, err = json.MarshalIndent(res, "", "  "); err != nil {
			return err
		}
	}
	if *out != "" {
		return os.WriteFile(*out, append(output, '\n'), 0o644)
	}
	_, err = fmt.Fprintln(stdout, string(output))
	return err
}

func runBatch(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("distill batch", flag.ContinueOnError)
	outDir := fs.String("out-dir", "", "output directory (required)")
	asJSON := fs.Bool("json", false, "emit JSON sidecars instead of Markdown")
	workers := fs.Int("workers", 0, "concurrent workers (default: NumCPU)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var dirs []string
	for rest := fs.Args(); len(rest) > 0; rest = fs.Args() {
		dirs = append(dirs, rest[0])
		if err := fs.Parse(rest[1:]); err != nil {
			return err
		}
	}
	if len(dirs) == 0 {
		return errors.New("batch: input directory required")
	}
	if *outDir == "" {
		return errors.New("batch: --out-dir is required")
	}

	ok, bad, err := app.Batch(registryWithPlugins(envPlugins()), app.BatchOptions{
		InDir: dirs[0], OutDir: *outDir, JSON: *asJSON, Workers: *workers,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "converted %d, failed %d\n", ok, bad)
	return nil
}

func runServe(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("distill serve", flag.ContinueOnError)
	addr := fs.String("addr", "127.0.0.1:8080", "listen address (host:port)")
	token := fs.String("token", os.Getenv("DISTILL_TOKEN"), "auth token; required for non-loopback binds")
	maxBytes := fs.Int64("max-bytes", 32<<20, "maximum request body size in bytes")
	if err := fs.Parse(args); err != nil {
		return err
	}
	srv, err := app.NewServer(registryWithPlugins(envPlugins()), app.ServeOptions{Addr: *addr, Token: *token, MaxBytes: *maxBytes})
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "distill serve listening on %s\n", *addr)
	return srv.ListenAndServe()
}
