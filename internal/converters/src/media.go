package converters

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/MitudruDutta/distill/internal/convert"
)

// Media converts audio/video files to a Markdown metadata block via ffprobe,
// optionally appending a transcript when the `whisper` CLI is available.
type Media struct{}

var mediaExts = map[string]bool{
	".mp3": true, ".wav": true, ".flac": true, ".m4a": true, ".aac": true, ".ogg": true, ".opus": true,
	".mp4": true, ".mov": true, ".mkv": true, ".webm": true, ".avi": true, ".m4v": true,
}

func (Media) Accepts(info convert.StreamInfo) bool {
	return mediaExts[info.Extension] ||
		strings.HasPrefix(info.Mimetype, "audio/") || strings.HasPrefix(info.Mimetype, "video/")
}

func (Media) Convert(r io.Reader, info convert.StreamInfo) (convert.Result, error) {
	if !toolAvailable("ffprobe") {
		return convert.Result{}, fmt.Errorf("distill: media support requires ffprobe (install ffmpeg)")
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return convert.Result{}, err
	}
	ext := info.Extension
	if ext == "" {
		ext = ".bin"
	}
	path, cleanup, err := writeTemp(ext, data)
	if err != nil {
		return convert.Result{}, err
	}
	defer cleanup()

	out, err := runTool("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", path)
	if err != nil {
		return convert.Result{}, err
	}
	var probe struct {
		Format struct {
			FormatLongName string `json:"format_long_name"`
			Duration       string `json:"duration"`
		} `json:"format"`
		Streams []struct {
			CodecType string `json:"codec_type"`
			CodecName string `json:"codec_name"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(out, &probe); err != nil {
		return convert.Result{}, err
	}

	title := info.Filename
	if title == "" {
		title = "media"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", title)
	if probe.Format.FormatLongName != "" {
		fmt.Fprintf(&b, "- Format: %s\n", probe.Format.FormatLongName)
	}
	if probe.Format.Duration != "" {
		fmt.Fprintf(&b, "- Duration: %s s\n", probe.Format.Duration)
	}
	for i, s := range probe.Streams {
		desc := s.CodecType + " (" + s.CodecName + ")"
		if s.Width > 0 {
			desc += fmt.Sprintf(", %d×%d", s.Width, s.Height)
		}
		fmt.Fprintf(&b, "- Stream %d: %s\n", i, desc)
	}

	if toolAvailable("whisper") {
		if t, e := transcribe(path); e == nil && t != "" {
			b.WriteString("\n## Transcript\n\n" + t + "\n")
		}
	}
	return convert.Result{Markdown: strings.TrimSpace(b.String()), Title: title}, nil
}

// transcribe runs the openai-whisper CLI and reads back the produced .txt.
func transcribe(srcPath string) (string, error) {
	dir, err := os.MkdirTemp("", "distill-whisper")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	if _, err := runTool("whisper", srcPath, "--model", "base", "--output_format", "txt", "--output_dir", dir); err != nil {
		return "", err
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "*.txt"))
	if len(matches) == 0 {
		return "", nil
	}
	out, err := os.ReadFile(matches[0])
	return strings.TrimSpace(string(out)), err
}
