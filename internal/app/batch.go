package app

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/MitudruDutta/distill/internal/convert"
)

// BatchOptions configures a batch conversion run.
type BatchOptions struct {
	InDir   string
	OutDir  string
	JSON    bool // emit a JSON document model instead of Markdown
	Workers int  // 0 => GOMAXPROCS
}

// Batch converts every regular file under opts.InDir concurrently, writing each
// result into opts.OutDir mirroring the input's relative path (".md", or
// ".json" when opts.JSON). It returns the count of converted and failed files.
func Batch(reg *convert.Registry, opts BatchOptions) (converted, failed int, err error) {
	workers := opts.Workers
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}

	var files []string
	if err = filepath.WalkDir(opts.InDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, p)
		}
		return nil
	}); err != nil {
		return 0, 0, err
	}

	var ok, bad int64
	paths := make(chan string)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range paths {
				if convertFile(reg, p, opts) == nil {
					atomic.AddInt64(&ok, 1)
				} else {
					atomic.AddInt64(&bad, 1)
				}
			}
		}()
	}
	for _, p := range files {
		paths <- p
	}
	close(paths)
	wg.Wait()
	return int(ok), int(bad), nil
}

func convertFile(reg *convert.Registry, path string, opts BatchOptions) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	peek := data
	if len(peek) > 512 {
		peek = peek[:512]
	}
	base := convert.StreamInfo{Filename: path, LocalPath: path, Extension: convert.ExtensionOf(path)}
	res, err := reg.Convert(bytes.NewReader(data), convert.Guess(base, peek))
	if err != nil {
		return err
	}

	rel, err := filepath.Rel(opts.InDir, path)
	if err != nil {
		rel = filepath.Base(path)
	}
	outExt, out := ".md", []byte(res.Markdown)
	if opts.JSON {
		if out, err = json.MarshalIndent(res, "", "  "); err != nil {
			return err
		}
		outExt = ".json"
	}
	dest := filepath.Join(opts.OutDir, strings.TrimSuffix(rel, filepath.Ext(rel))+outExt)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, out, 0o644)
}
