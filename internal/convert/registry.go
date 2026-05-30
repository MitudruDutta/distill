package convert

import (
	"bytes"
	"errors"
	"io"
	"regexp"
	"sort"
	"strings"
	"unicode"
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

// bulletPrefix matches a leading bullet-like glyph (with optional indent) so we
// can rewrite it as "- ". Covers common ASCII/Unicode bullets plus the Private
// Use Area, which most resume/CV PDFs use as their bullet codepoint.
var bulletPrefix = regexp.MustCompile(`^(\s*)[\x{2022}\x{2023}\x{2043}\x{2219}\x{25AA}\x{25AB}\x{25CB}\x{25CF}\x{25E6}\x{25C6}\x{25C7}\x{25BA}\x{25B8}\x{00B7}\x{E000}-\x{F8FF}][\s\x{00A0}]+`)

// exoticSpaces are Unicode whitespace codepoints that look identical to ASCII
// space but tokenize as separate (rarer) tokens for LLMs.
var exoticSpaces = map[rune]bool{
	'\u00A0': true, '\u1680': true, '\u2000': true, '\u2001': true,
	'\u2002': true, '\u2003': true, '\u2004': true, '\u2005': true,
	'\u2006': true, '\u2007': true, '\u2008': true, '\u2009': true,
	'\u200A': true, '\u202F': true, '\u205F': true, '\u3000': true,
}

// invisibleZW are zero-width / BOM codepoints worth dropping; they render
// identically to nothing but cost tokens.
var invisibleZW = map[rune]bool{
	'\u200B': true, '\u200C': true, '\uFEFF': true,
}

// longLineThreshold is the character count beyond which we try to break a line
// at sentence boundaries. Most real-world paragraphs are well under this, but
// some sources (HTML/EPUB with one giant <p>, ipynb cells, etc.) emit
// thousand-character lines that are useless for LLM chunking.
const longLineThreshold = 800

// wrapLongLine splits an overly long line at sentence boundaries (". " / "! "
// / "? " followed by an uppercase letter). It only fires above the threshold,
// keeps each chunk substantial, and is a no-op when no clean split exists.
func wrapLongLine(line string) string {
	if len(line) <= longLineThreshold {
		return line
	}
	const minChunk = 200
	runes := []rune(line)
	var out []string
	var cur strings.Builder
	for i := 0; i < len(runes); i++ {
		cur.WriteRune(runes[i])
		if cur.Len() < minChunk || i+2 >= len(runes) {
			continue
		}
		r := runes[i]
		if (r == '.' || r == '!' || r == '?') && runes[i+1] == ' ' && unicode.IsUpper(runes[i+2]) {
			out = append(out, cur.String())
			cur.Reset()
			i++ // skip the space
		}
	}
	if cur.Len() > 0 {
		out = append(out, strings.TrimSpace(cur.String()))
	}
	if len(out) <= 1 {
		return line
	}
	return strings.Join(out, "\n")
}

// normalizeSpaces folds exotic whitespace to ASCII space and drops invisibles.
func normalizeSpaces(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case exoticSpaces[r]:
			return ' '
		case invisibleZW[r]:
			return -1 // drop
		}
		return r
	}, s)
}

// normalize trims trailing whitespace per line, normalizes exotic spaces and
// bullet glyphs to their ASCII equivalents (skipping fenced code blocks),
// collapses 3+ blank lines into one, and strips leading/trailing whitespace.
func normalize(s string) string {
	lines := strings.Split(s, "\n")
	inCode := false
	for i, ln := range lines {
		ln = strings.TrimRight(ln, " \t\r")
		if strings.HasPrefix(strings.TrimSpace(ln), "```") {
			inCode = !inCode
			lines[i] = ln
			continue
		}
		if !inCode {
			ln = normalizeSpaces(ln)
			ln = bulletPrefix.ReplaceAllString(ln, "$1- ")
			ln = wrapLongLine(ln)
		}
		lines[i] = ln
	}
	joined := strings.Join(lines, "\n")
	joined = blankLines.ReplaceAllString(joined, "\n\n")
	return stripBlankLines(joined)
}

// stripBlankLines removes leading and trailing empty/whitespace-only lines but
// preserves the content lines' own indentation (so nested bullets stay nested).
func stripBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	start, end := 0, len(lines)
	for start < end && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return strings.Join(lines[start:end], "\n")
}
