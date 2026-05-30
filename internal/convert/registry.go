package convert

import (
	"bytes"
	"errors"
	"io"
	"regexp"
	"sort"
	"strings"
)

// Converter turns a document stream into Markdown.
type Converter interface {
	// Accepts reports whether this converter can handle the given stream type.
	// It must not consume the stream.
	Accepts(info StreamInfo) bool
	// Convert reads the stream from the start and returns the result.
	Convert(r io.Reader, info StreamInfo) (Result, error)
}

// ErrUnsupported is returned when no registered converter accepts the input.
var ErrUnsupported = errors.New("distill: unsupported or unrecognized format")

type registration struct {
	conv     Converter
	priority int
}

// Registry holds converters and dispatches conversions by priority.
type Registry struct {
	regs []registration
}

// Register adds a converter. Lower priority values are tried first; specific
// converters should use a lower value than generic catch-alls.
func (reg *Registry) Register(c Converter, priority int) {
	reg.regs = append(reg.regs, registration{conv: c, priority: priority})
}

// Convert buffers the input, then tries each type guess against each accepting
// converter in priority order, returning the first successful result.
func (reg *Registry) Convert(r io.Reader, guesses []StreamInfo) (Result, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return Result{}, err
	}

	ordered := make([]registration, len(reg.regs))
	copy(ordered, reg.regs)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].priority < ordered[j].priority
	})

	if len(guesses) == 0 {
		guesses = []StreamInfo{{}}
	}

	var firstErr error
	for _, info := range guesses {
		for _, rg := range ordered {
			if !rg.conv.Accepts(info) {
				continue
			}
			res, err := rg.conv.Convert(bytes.NewReader(data), info)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			res.Markdown = normalize(res.Markdown)
			return res, nil
		}
	}
	if firstErr != nil {
		return Result{}, firstErr
	}
	return Result{}, ErrUnsupported
}

var blankLines = regexp.MustCompile(`\n{3,}`)

// normalize trims trailing whitespace per line, collapses 3+ blank lines into
// one, and strips leading/trailing whitespace.
func normalize(s string) string {
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		lines[i] = strings.TrimRight(ln, " \t\r")
	}
	joined := strings.Join(lines, "\n")
	return strings.TrimSpace(blankLines.ReplaceAllString(joined, "\n\n"))
}
