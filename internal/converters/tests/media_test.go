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

// Slow: downloads/loads a whisper model. Gated behind an env var so the normal
// suite stays fast. Asserts a transcript is produced (not its exact text, which
// varies with synthetic speech).
func TestMediaTranscribesSpeech(t *testing.T) {
	if os.Getenv("DISTILL_TEST_WHISPER") == "" {
		t.Skip("set DISTILL_TEST_WHISPER=1 to run the whisper transcription test")
	}
	if _, err := exec.LookPath("whisper"); err != nil {
		t.Skip("whisper not installed")
	}
	if _, err := exec.LookPath("espeak-ng"); err != nil {
		t.Skip("espeak-ng not installed")
	}
	wav := filepath.Join(t.TempDir(), "s.wav")
	if out, err := exec.Command("espeak-ng", "-w", wav, "Hello world this is distill").CombinedOutput(); err != nil {
		t.Skipf("espeak-ng failed: %s", out)
	}
	data, err := os.ReadFile(wav)
	if err != nil {
		t.Fatal(err)
	}
	res, err := (Media{}).Convert(bytes.NewReader(data), convert.StreamInfo{Extension: ".wav", Filename: "s.wav"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Markdown, "## Transcript") || len(strings.TrimSpace(res.Markdown)) < 20 {
		t.Fatalf("expected a non-empty transcript section:\n%s", res.Markdown)
	}
}
