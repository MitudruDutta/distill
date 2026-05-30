package converters

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/MitudruDutta/distill/internal/convert"
)

const (
	maxArchiveEntries = 2000
	maxEntryBytes     = 16 << 20  // 16 MiB per entry
	maxTotalBytes     = 128 << 20 // 128 MiB total
)

// Archive converts the entries of a ZIP or TAR archive. Each supported entry is
// converted via Reg and emitted under a heading. Nested archives are noted but
// not recursed into, and entry count/byte totals are capped (zip-bomb guard).
type Archive struct {
	Reg *convert.Registry
}

var archiveExts = map[string]bool{".zip": true, ".tar": true}

func (Archive) Accepts(info convert.StreamInfo) bool {
	return archiveExts[info.Extension] ||
		info.Mimetype == "application/zip" || info.Mimetype == "application/x-tar"
}

func (a Archive) Convert(r io.Reader, info convert.StreamInfo) (convert.Result, error) {
	if info.Extension == ".tar" || info.Mimetype == "application/x-tar" {
		return a.convertTar(r)
	}
	return a.convertZip(r)
}

func (a Archive) convertZip(r io.Reader) (convert.Result, error) {
	zr, err := openZip(r)
	if err != nil {
		return convert.Result{}, err
	}
	names := make([]string, 0, len(zr.File))
	for _, f := range zr.File {
		if !f.FileInfo().IsDir() {
			names = append(names, f.Name)
		}
	}
	sort.Strings(names)

	var b strings.Builder
	total := 0
	for i, name := range names {
		if i >= maxArchiveEntries || total >= maxTotalBytes {
			break
		}
		data, err := zipEntry(zr, name)
		if err != nil {
			continue
		}
		total += len(data)
		a.emitEntry(&b, name, data)
	}
	return convert.Result{Markdown: strings.TrimSpace(b.String())}, nil
}

func (a Archive) convertTar(r io.Reader) (convert.Result, error) {
	tr := tar.NewReader(r)
	var b strings.Builder
	total, count := 0, 0
	for count < maxArchiveEntries && total < maxTotalBytes {
		hdr, err := tr.Next()
		if err != nil {
			break
		}
		if hdr.FileInfo().IsDir() {
			continue
		}
		count++
		data, _ := io.ReadAll(io.LimitReader(tr, maxEntryBytes))
		total += len(data)
		a.emitEntry(&b, hdr.Name, data)
	}
	return convert.Result{Markdown: strings.TrimSpace(b.String())}, nil
}

func (a Archive) emitEntry(b *strings.Builder, name string, data []byte) {
	fmt.Fprintf(b, "## %s\n\n", name)
	if archiveExts[convert.ExtensionOf(name)] {
		b.WriteString("_(nested archive skipped)_\n\n")
		return
	}
	if len(data) == 0 {
		b.WriteString("_(empty)_\n\n")
		return
	}
	if len(data) > maxEntryBytes {
		data = data[:maxEntryBytes]
	}
	guess := convert.Guess(convert.StreamInfo{Filename: name, Extension: convert.ExtensionOf(name)}, data)
	res, err := a.Reg.Convert(bytes.NewReader(data), guess)
	if err != nil {
		b.WriteString("_(unsupported)_\n\n")
		return
	}
	b.WriteString(strings.TrimSpace(res.Markdown))
	b.WriteString("\n\n")
}
