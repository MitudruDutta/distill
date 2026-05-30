package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

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
	fs := flag.NewFlagSet("distill", flag.ContinueOnError)
	out := fs.String("o", "", "output file (default: stdout)")
	ext := fs.String("x", "", "extension hint, e.g. csv (useful for stdin)")
	mimeHint := fs.String("m", "", "MIME type hint")
	charset := fs.String("c", "", "charset hint, e.g. utf-8")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Allow flags to appear after the filename (e.g. `distill file.csv -o out.md`);
	// the flag package stops at the first positional argument otherwise.
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
	if *out != "" {
		return os.WriteFile(*out, []byte(res.Markdown+"\n"), 0o644)
	}
	_, err = fmt.Fprintln(stdout, res.Markdown)
	return err
}
