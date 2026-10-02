package cinepak

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/wok/rent-a-hero/internal/formats/avi"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	return abs
}

func readFixtureAVI(t *testing.T, rel string) *avi.VideoStream {
	t.Helper()
	path := filepath.Join(repoRoot(t), rel)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture %s not available: %v", path, err)
	}

	vs, err := avi.Parse(data)
	if err != nil {
		t.Fatalf("avi.Parse(%s): %v", path, err)
	}
	return vs
}

func decodeAll(t *testing.T, vs *avi.VideoStream) [][]byte {
	t.Helper()

	dec := NewDecoder(vs.Width, vs.Height)
	out := make([][]byte, len(vs.Frames))
	for i, chunk := range vs.Frames {
		rgba, err := dec.DecodeFrame(chunk)
		if err != nil {
			t.Fatalf("DecodeFrame(%d): %v", i, err)
		}
		out[i] = rgba
	}
	return out
}

func TestDecodeLiftVideo(t *testing.T) {
	vs := readFixtureAVI(t, "data/cd/GAME/LOC01/LIFTMITTERAUF.AVI")
	frames := decodeAll(t, vs)

	if len(frames) != 30 {
		t.Fatalf("decoded %d frames, want 30", len(frames))
	}

	want := vs.Width * vs.Height * 4
	allBlack := true
	for i, f := range frames {
		if len(f) != want {
			t.Fatalf("frame %d: len = %d, want %d", i, len(f), want)
		}
		for _, b := range f {
			if b != 0 {
				allBlack = false
			}
		}
	}
	if allBlack {
		t.Fatal("every decoded frame is entirely black; decoder likely drew nothing")
	}
}

// TestDecodeMatchesFFmpeg cross-checks this package's output against
// ffmpeg's own Cinepak decoder, byte for byte, as an independent oracle.
// ffmpeg is not a runtime dependency of this package or of the game (see
// agents/IMPLEMENTATION.md "AVI strategy"); it is only ever invoked here,
// in a test, and only if available.
func TestDecodeMatchesFFmpeg(t *testing.T) {
	ffmpegPath := findFFmpeg(t)

	for _, rel := range []string{
		"data/cd/GAME/LOC01/LIFTMITTERAUF.AVI",
		"data/cd/GAME/LOC40/S116_IntroPart3.avi",
	} {
		t.Run(rel, func(t *testing.T) {
			vs := readFixtureAVI(t, rel)
			frames := decodeAll(t, vs)

			want := decodeWithFFmpeg(t, ffmpegPath, filepath.Join(repoRoot(t), rel), vs.Width, vs.Height, len(frames))

			for i := range frames {
				got := toRGB(frames[i], vs.Width, vs.Height)
				if !bytes.Equal(got, want[i]) {
					t.Fatalf("frame %d does not match ffmpeg's decode (first mismatch of possibly more)", i)
				}
			}
		})
	}
}

func findFFmpeg(t *testing.T) string {
	t.Helper()
	for _, candidate := range []string{"/opt/homebrew/bin/ffmpeg", "/usr/local/bin/ffmpeg", "/usr/bin/ffmpeg"} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	if path, err := exec.LookPath("ffmpeg"); err == nil {
		return path
	}
	t.Skip("ffmpeg not available; skipping cross-check against an independent decoder")
	return ""
}

// decodeWithFFmpeg extracts every frame of srcPath as raw RGB24 via ffmpeg
// and splits the result into one []byte per frame.
func decodeWithFFmpeg(t *testing.T, ffmpegPath, srcPath string, width, height, wantFrames int) [][]byte {
	t.Helper()

	cmd := exec.Command(ffmpegPath, "-v", "error", "-i", srcPath, "-pix_fmt", "rgb24", "-f", "rawvideo", "-")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("ffmpeg %s: %v: %s", srcPath, err, stderr.String())
	}

	frameSize := width * height * 3
	data := stdout.Bytes()
	if len(data) != frameSize*wantFrames {
		t.Fatalf("ffmpeg produced %d bytes, want %d (%d frames of %d bytes)", len(data), frameSize*wantFrames, wantFrames, frameSize)
	}

	out := make([][]byte, wantFrames)
	for i := range out {
		out[i] = data[i*frameSize : (i+1)*frameSize]
	}
	return out
}

// toRGB drops our decoder's alpha channel so it can be compared directly
// against ffmpeg's packed RGB24 output.
func toRGB(rgba []byte, width, height int) []byte {
	out := make([]byte, width*height*3)
	for i := 0; i < width*height; i++ {
		out[i*3+0] = rgba[i*4+0]
		out[i*3+1] = rgba[i*4+1]
		out[i*3+2] = rgba[i*4+2]
	}
	return out
}
