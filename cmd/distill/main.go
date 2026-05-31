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
			return app.MCP(converters.Default(), stdin, stdout)
		}
	}
	return runConvert(args, stdin, stdout)
}

func runConvert(args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("distill", flag.ContinueOnError)
	out := fs.String("o", "", "output file (default: stdout)")
	ext := fs.String("x", "", "extension hint, e.g. csv (useful for stdin)")
	mimeHint := fs.String("m", "", "MIME type hint")
	charset := fs.String("c", "", "charset hint, e.g. utf-8")
	asJSON := fs.Bool("json", false, "emit a JSON document model instead of Markdown")
	if err := fs.Parse(args); err != nil {
		return err
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
			data, info, err := app.FetchURI(files[0], app.FetchOptions{})
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
	res, err := converters.Default().Convert(bytes.NewReader(data), convert.Guess(base, peek))
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

	ok, bad, err := app.Batch(converters.Default(), app.BatchOptions{
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
	srv, err := app.NewServer(converters.Default(), app.ServeOptions{Addr: *addr, Token: *token, MaxBytes: *maxBytes})
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "distill serve listening on %s\n", *addr)
	return srv.ListenAndServe()
}
