package indeo5

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
	if vs.Compression != "IV50" {
		t.Fatalf("%s: compression = %q, want IV50", path, vs.Compression)
	}
	return vs
}

func TestDecodeRealAssetsWithoutError(t *testing.T) {
	for _, rel := range []string{
		"data/installation/Common/S1_RodOnGlider.avi",
		"data/installation/Common/S1_RodChecksItems.avi",
		"data/installation/Common/S1_Schild.avi",
		"data/installation/Common/S1_RodDrawsBlaster.avi",
		"data/installation/Common/S1_RodToCloseup.avi",
	} {
		t.Run(rel, func(t *testing.T) {
			vs := readFixtureAVI(t, rel)
			dec := NewDecoder(vs.Width, vs.Height)

			for i, chunk := range vs.Frames {
				rgba, err := dec.DecodeFrame(chunk)
				if err != nil {
					t.Fatalf("DecodeFrame(%d): %v", i, err)
				}
				want := vs.Width * vs.Height * 4
				if len(rgba) != want {
					t.Fatalf("DecodeFrame(%d): len = %d, want %d", i, len(rgba), want)
				}
			}
		})
	}
}

// TestDecodeMatchesFFmpeg cross-checks this package's raw Y/U/V plane
// output against ffmpeg's own Indeo5 decoder, byte for byte, across every
// frame of two real game assets (one small, one with many frames). ffmpeg
// is not a runtime dependency of this package or of the game; it is only
// ever invoked here, in a test, and only if available -- an independent
// correctness oracle, the same role it played verifying
// internal/formats/cinepak.
//
// Comparing raw planes (not RGB) deliberately keeps this decoder's
// verified correctness scoped to the actual codec (bitstream parsing,
// wavelet reconstruction, motion compensation): the RGB conversion in
// output.go is a separate, ordinary, well-known YUV->RGB formula that
// doesn't need to bit-match ffmpeg's own (different) RGB conversion path.
func TestDecodeMatchesFFmpeg(t *testing.T) {
	ffmpegPath := findFFmpeg(t)

	for _, rel := range []string{
		"data/installation/Common/S1_RodOnGlider.avi",
		"data/installation/Common/S1_RodChecksItems.avi",
		"data/installation/Common/S1_Schild.avi",
		"data/installation/Common/S1_RodDrawsBlaster.avi",
		"data/installation/Common/S1_RodToCloseup.avi",
	} {
		t.Run(rel, func(t *testing.T) {
			vs := readFixtureAVI(t, rel)
			dec := NewDecoder(vs.Width, vs.Height)

			chromaW, chromaH := (vs.Width+3)>>2, (vs.Height+3)>>2
			wantFrames := decodeYUV410PWithFFmpeg(t, ffmpegPath, filepath.Join(repoRoot(t), rel), vs.Width, vs.Height, chromaW, chromaH, len(vs.Frames))

			for i, chunk := range vs.Frames {
				if _, err := dec.DecodeFrame(chunk); err != nil {
					t.Fatalf("DecodeFrame(%d): %v", i, err)
				}

				gotY := make([]byte, vs.Width*vs.Height)
				if dec.isScalable {
					dec.recompose53(&dec.planes[0], gotY, vs.Width)
				} else {
					outputPlane(&dec.planes[0], gotY, vs.Width)
				}

				gotU := make([]byte, chromaW*chromaH)
				outputPlane(&dec.planes[2], gotU, chromaW)
				gotV := make([]byte, chromaW*chromaH)
				outputPlane(&dec.planes[1], gotV, chromaW)

				want := wantFrames[i]
				if !bytes.Equal(gotY, want.y) {
					t.Fatalf("frame %d: Y plane mismatch (first of possibly more)", i)
				}
				if !bytes.Equal(gotU, want.u) {
					t.Fatalf("frame %d: U plane mismatch (first of possibly more)", i)
				}
				if !bytes.Equal(gotV, want.v) {
					t.Fatalf("frame %d: V plane mismatch (first of possibly more)", i)
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

type yuvFrame struct{ y, u, v []byte }

func decodeYUV410PWithFFmpeg(t *testing.T, ffmpegPath, srcPath string, width, height, chromaW, chromaH, wantFrames int) []yuvFrame {
	t.Helper()

	cmd := exec.Command(ffmpegPath, "-v", "error", "-i", srcPath, "-pix_fmt", "yuv410p", "-f", "rawvideo", "-")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("ffmpeg %s: %v: %s", srcPath, err, stderr.String())
	}

	ySize := width * height
	cSize := chromaW * chromaH
	frameSize := ySize + 2*cSize

	data := stdout.Bytes()
	if len(data) != frameSize*wantFrames {
		t.Fatalf("ffmpeg produced %d bytes, want %d (%d frames of %d bytes)", len(data), frameSize*wantFrames, wantFrames, frameSize)
	}

	out := make([]yuvFrame, wantFrames)
	for i := range out {
		f := data[i*frameSize : (i+1)*frameSize]
		out[i] = yuvFrame{
			y: f[:ySize],
			u: f[ySize : ySize+cSize],
			v: f[ySize+cSize : ySize+2*cSize],
		}
	}
	return out
}
