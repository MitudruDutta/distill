package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/MitudruDutta/distill/internal/convert"
	"github.com/MitudruDutta/distill/internal/converters"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "distill:", err)
		os.Exit(1)
	}
}

func run() error {
	out := flag.String("o", "", "output file (default: stdout)")
	ext := flag.String("x", "", "extension hint, e.g. csv (useful for stdin)")
	mimeHint := flag.String("m", "", "MIME type hint")
	charset := flag.String("c", "", "charset hint, e.g. utf-8")
	flag.Parse()

	// Allow flags to appear after the filename (e.g. `distill file.csv -o out.md`),
	// which the stdlib flag package does not handle by default.
	var files []string
	for rest := flag.Args(); len(rest) > 0; rest = flag.Args() {
		files = append(files, rest[0])
		flag.CommandLine.Parse(rest[1:])
	}

	base := convert.StreamInfo{Mimetype: *mimeHint, Charset: *charset}
	if *ext != "" {
		e := strings.ToLower(*ext)
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		base.Extension = e
	}

	var r io.Reader = os.Stdin
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
	fmt.Println(res.Markdown)
	return nil
}
