//go:build !pdfium

package converters

import "github.com/MitudruDutta/distill/internal/convert"

// registerPDFium is a no-op unless built with -tags pdfium. This keeps the
// embedded PDFium wasm (~5 MB) out of the default binary.
func registerPDFium(*convert.Registry) {}
