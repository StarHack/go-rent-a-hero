package avi

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	return abs
}

func readFixture(t *testing.T, rel string) []byte {
	t.Helper()
	path := filepath.Join(repoRoot(t), rel)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture %s not available: %v", path, err)
	}
	return data
}

func TestParseLiftVideo(t *testing.T) {
	data := readFixture(t, "data/cd/GAME/LOC01/LIFTMITTERAUF.AVI")

	vs, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if vs.Width != 640 || vs.Height != 360 {
		t.Errorf("dims = %dx%d, want 640x360", vs.Width, vs.Height)
	}
	if vs.Compression != "cvid" {
		t.Errorf("Compression = %q, want \"cvid\"", vs.Compression)
	}
	if vs.BitCount != 24 {
		t.Errorf("BitCount = %d, want 24", vs.BitCount)
	}
	if math.Abs(vs.FPS-10) > 0.01 {
		t.Errorf("FPS = %v, want 10", vs.FPS)
	}
	if len(vs.Frames) != 30 {
		t.Fatalf("len(Frames) = %d, want 30", len(vs.Frames))
	}

	// Every frame chunk should at least contain a 10-byte Cinepak frame
	// header, and its own embedded width/height (big-endian, bytes 4-7)
	// should agree with the container's.
	for i, f := range vs.Frames {
		if len(f) < 10 {
			t.Fatalf("frame %d: only %d bytes", i, len(f))
		}
		w := int(f[4])<<8 | int(f[5])
		h := int(f[6])<<8 | int(f[7])
		if w != vs.Width || h != vs.Height {
			t.Errorf("frame %d: embedded dims %dx%d, want %dx%d", i, w, h, vs.Width, vs.Height)
		}
	}
}

func TestParseRejectsNonRIFF(t *testing.T) {
	if _, err := Parse([]byte("not an avi file at all")); err == nil {
		t.Fatal("expected an error for non-RIFF data")
	}
}
