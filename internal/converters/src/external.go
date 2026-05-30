package converters

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

const toolTimeout = 120 * time.Second

// toolAvailable reports whether an external command is on PATH.
func toolAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// runToolStdin runs name with args, feeding stdin, and returns stdout.
func runToolStdin(name string, stdin []byte, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), toolTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = bytes.NewReader(stdin)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s: %w: %s", name, err, bytes.TrimSpace(errb.Bytes()))
	}
	return out.Bytes(), nil
}

// runTool runs name with args (no stdin) and returns stdout.
func runTool(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), toolTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s: %w: %s", name, err, bytes.TrimSpace(errb.Bytes()))
	}
	return out.Bytes(), nil
}

// writeTemp writes data to a temp file with the given extension and returns its
// path and a cleanup func.
func writeTemp(ext string, data []byte) (string, func(), error) {
	f, err := os.CreateTemp("", "distill-*"+ext)
	if err != nil {
		return "", func() {}, err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", func() {}, err
	}
	f.Close()
	return f.Name(), func() { os.Remove(f.Name()) }, nil
}
