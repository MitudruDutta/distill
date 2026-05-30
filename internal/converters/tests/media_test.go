package converters_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MitudruDutta/distill/internal/convert"
	. "github.com/MitudruDutta/distill/internal/converters/src"
)

func TestMediaProbesAudioMetadata(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not installed")
	}
	wav := filepath.Join(t.TempDir(), "a.wav")
	if out, err := exec.Command("ffmpeg", "-v", "quiet", "-f", "lavfi",
		"-i", "anullsrc=r=8000:cl=mono", "-t", "1", wav).CombinedOutput(); err != nil {
		t.Skipf("could not synthesize fixture: %s", out)
	}
	data, err := os.ReadFile(wav)
	if err != nil {
		t.Fatal(err)
	}

	res, err := (Media{}).Convert(bytes.NewReader(data), convert.StreamInfo{Extension: ".wav", Filename: "a.wav"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(res.Markdown), "audio") {
		t.Fatalf("expected an audio stream in metadata:\n%s", res.Markdown)
	}
}
