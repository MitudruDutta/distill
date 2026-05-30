package converters

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ocrAvailable reports whether Tesseract OCR is usable.
func ocrAvailable() bool { return toolAvailable("tesseract") }

// ocrImageText runs Tesseract over an in-memory image and returns the text.
func ocrImageText(data []byte) (string, error) {
	out, err := runToolStdin("tesseract", data, "stdin", "stdout")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ocrPDFText rasterizes a PDF with pdftoppm and OCRs each page with Tesseract.
func ocrPDFText(data []byte) (string, error) {
	pdfPath, cleanup, err := writeTemp(".pdf", data)
	if err != nil {
		return "", err
	}
	defer cleanup()

	dir, err := os.MkdirTemp("", "distill-ocr")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)

	prefix := filepath.Join(dir, "page")
	if _, err := runTool("pdftoppm", "-png", "-r", "150", pdfPath, prefix); err != nil {
		return "", err
	}
	pages, _ := filepath.Glob(prefix + "*.png")
	sort.Strings(pages)

	var b strings.Builder
	for _, p := range pages {
		out, err := runTool("tesseract", p, "stdout")
		if err != nil {
			continue
		}
		if s := strings.TrimSpace(string(out)); s != "" {
			b.WriteString(s)
			b.WriteString("\n\n")
		}
	}
	return strings.TrimSpace(b.String()), nil
}
