package bmp

import (
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "fixtures", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}

	return data
}

func TestDecode24bpp(t *testing.T) {
	data := fixture(t, "100_PIRATESHIP.BMP")

	img, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if img.Width != 74 || img.Height != 81 {
		t.Errorf("dims = %dx%d, want 74x81", img.Width, img.Height)
	}

	if len(img.Pixels) != 74*81*4 {
		t.Errorf("pixel buffer len = %d, want %d", len(img.Pixels), 74*81*4)
	}

	// All BMP pixels are fully opaque.
	for i := 3; i < len(img.Pixels); i += 4 {
		if img.Pixels[i] != 255 {
			t.Fatalf("alpha at pixel %d = %d, want 255", i/4, img.Pixels[i])
		}
	}
}

func TestDecodeRejectsBadMagic(t *testing.T) {
	if _, err := Decode([]byte("not a bmp")); err == nil {
		t.Error("expected error for non-BMP input")
	}
}

func TestRowStridePadding(t *testing.T) {
	// width=1, bpp=24 -> 3 bytes/row, padded to 4
	if s := rowStride(1, 24); s != 4 {
		t.Errorf("rowStride(1,24) = %d, want 4", s)
	}

	// width=4, bpp=8 -> 4 bytes/row, already aligned
	if s := rowStride(4, 8); s != 4 {
		t.Errorf("rowStride(4,8) = %d, want 4", s)
	}
}
