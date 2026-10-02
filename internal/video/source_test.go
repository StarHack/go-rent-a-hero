package video

import (
	"path/filepath"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	return abs
}

func liftAVI(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "data", "cd", "GAME", "LOC01", "LIFTMITTERAUF.AVI")
}

func rodOnGliderAVI(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "data", "installation", "Common", "S1_RodOnGlider.avi")
}

func TestLoadAndDecodeRealLiftVideo(t *testing.T) {
	src, err := Load(liftAVI(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if src.Width() != 640 || src.Height() != 360 {
		t.Errorf("dims = %dx%d, want 640x360", src.Width(), src.Height())
	}
	if src.Frames() != 30 {
		t.Fatalf("Frames() = %d, want 30", src.Frames())
	}
	if src.FPS() != 10 {
		t.Errorf("FPS() = %v, want 10", src.FPS())
	}

	want := 640 * 360 * 4
	for i := range src.Frames() {
		rgba, err := src.RGBA(i)
		if err != nil {
			t.Fatalf("RGBA(%d): %v", i, err)
		}
		if len(rgba) != want {
			t.Fatalf("RGBA(%d) len = %d, want %d", i, len(rgba), want)
		}
		for p := 3; p < len(rgba); p += 4 {
			if rgba[p] != 255 {
				t.Fatalf("RGBA(%d) alpha byte %d = %d, want 255 (opaque AVI pixel)", i, p, rgba[p])
			}
		}
	}

	last := src.Frames() - 1
	first, err := src.RGBA(last)
	if err != nil {
		t.Fatalf("RGBA(%d): %v", last, err)
	}
	second, err := src.RGBA(last)
	if err != nil {
		t.Fatalf("RGBA(%d) again: %v", last, err)
	}
	if !bytesEqual(first, second) {
		t.Fatalf("repeated RGBA(%d) differs", last)
	}

	if _, err := src.RGBA(0); err != nil {
		t.Fatalf("RGBA(0) after decoding forward: %v", err)
	}
}

func TestAVILayerKeepsAuthoredLeadingFrame(t *testing.T) {
	src, err := Load(rodOnGliderAVI(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// The original AVI object exposes AVIStreamLength frames starting at
	// AVIStreamStart; it does not delete a uniform leading frame for layer
	// playback. This asset has 31 raw frames, so all 31 remain addressable.
	if got := src.Frames(); got != 31 {
		t.Fatalf("Frames() = %d, want raw authored count 31", got)
	}

	rgba, err := src.RGBA(0)
	if err != nil {
		t.Fatalf("RGBA(0): %v", err)
	}
	for p := 3; p < len(rgba); p += 4 {
		if rgba[p] != 255 {
			t.Fatalf("frame 0 alpha byte %d = %d, want 255", p, rgba[p])
		}
	}
}

func TestLoadRejectsMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(repoRoot(t), "data", "does-not-exist.avi")); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestVideoColorKeyIsDirectDrawBlack(t *testing.T) {
	r, g, b, ok := (&Source{}).ColorKey()
	if !ok || r != 0 || g != 0 || b != 0 {
		t.Fatalf("ColorKey() = (%d,%d,%d,%v), want (0,0,0,true)", r, g, b, ok)
	}
}
