package converters

import (
	"archive/zip"
	"bytes"
	"io"
)

// openZip buffers r and returns a reader over the zip archive.
func openZip(r io.Reader) (*zip.Reader, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return zip.NewReader(bytes.NewReader(data), int64(len(data)))
}

// zipEntry returns the bytes of the named archive entry.
func zipEntry(zr *zip.Reader, name string) ([]byte, error) {
	f, err := zr.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}
