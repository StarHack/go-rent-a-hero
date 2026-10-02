package tcc

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

func TestParseHeaderHebel(t *testing.T) {
	data := fixture(t, "HEBEL.TCC")

	h, err := ParseHeader(data)
	if err != nil {
		t.Fatalf("ParseHeader: %v", err)
	}

	if h.Flags != 0 {
		t.Errorf("Flags = %d, want 0", h.Flags)
	}

	wantSizes := [4]uint32{320, 320, 302, 159}
	gotSizes := [4]uint32{h.StreamSize0, h.StreamSize1, h.StreamSize2, h.StreamSize3}
	if gotSizes != wantSizes {
		t.Errorf("stream sizes = %v, want %v", gotSizes, wantSizes)
	}

	if h.Width != 20 || h.Height != 20 {
		t.Errorf("dims = %dx%d, want 20x20", h.Width, h.Height)
	}

	size, err := FrameSize(h)
	if err != nil {
		t.Fatalf("FrameSize: %v", err)
	}

	if size != len(data) {
		t.Errorf("computed size %d != file size %d", size, len(data))
	}
}

func TestDecodeHebelIsBrown(t *testing.T) {
	data := fixture(t, "HEBEL.TCC")

	img, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if img.Width != 20 || img.Height != 20 {
		t.Fatalf("dims = %dx%d, want 20x20", img.Width, img.Height)
	}

	// Sample opaque pixels and confirm a brown/orange bias (R > G > B),
	// not blue, proving plane order is R,G,B,A and not reversed.
	var brownVotes, opaqueCount int

	for i := 0; i < img.Width*img.Height; i++ {
		r := img.Pixels[i*4+0]
		g := img.Pixels[i*4+1]
		b := img.Pixels[i*4+2]
		a := img.Pixels[i*4+3]

		if a == 0 {
			continue
		}

		opaqueCount++

		if r >= g && g >= b && r > b {
			brownVotes++
		}
	}

	if opaqueCount == 0 {
		t.Fatal("no opaque pixels decoded")
	}

	if float64(brownVotes)/float64(opaqueCount) < 0.5 {
		t.Errorf("expected majority brown/orange pixels (R>=G>=B), got %d/%d", brownVotes, opaqueCount)
	}
}

func TestParseHeaderHelpMask(t *testing.T) {
	data := fixture(t, "HelpMask.TCC")

	h, err := ParseHeader(data)
	if err != nil {
		t.Fatalf("ParseHeader: %v", err)
	}

	if h.Width != 320 || h.Height != 360 {
		t.Errorf("dims = %dx%d, want 320x360", h.Width, h.Height)
	}

	wantSizes := [4]uint32{1356, 1356, 1356, 14042}
	gotSizes := [4]uint32{h.StreamSize0, h.StreamSize1, h.StreamSize2, h.StreamSize3}
	if gotSizes != wantSizes {
		t.Errorf("stream sizes = %v, want %v", gotSizes, wantSizes)
	}
}

func TestDecodeHelpMaskAlphaVaries(t *testing.T) {
	data := fixture(t, "HelpMask.TCC")

	img, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	seenIntermediate := false

	for i := 0; i < img.Width*img.Height; i++ {
		r := img.Pixels[i*4+0]
		g := img.Pixels[i*4+1]
		b := img.Pixels[i*4+2]
		a := img.Pixels[i*4+3]

		if r != 0 || g != 0 || b != 0 {
			t.Fatalf("expected all-zero RGB planes, found (%d,%d,%d) at pixel %d", r, g, b, i)
		}

		if a != 0 && a != 255 {
			seenIntermediate = true
		}
	}

	if !seenIntermediate {
		t.Error("expected at least one intermediate (non-0, non-255) alpha value")
	}
}

func TestFrameSizeRejectsOversizedDimensions(t *testing.T) {
	h := Header{Width: MaxDimension + 1, Height: 1}

	if _, err := FrameSize(h); err == nil {
		t.Error("expected error for oversized width")
	}
}

func TestDecodeZeroSizeAlphaStreamDefaultsToOpaque(t *testing.T) {
	// A real shipped asset (099_PIRATESHIP.TCC) has StreamSize3 == 0: no
	// alpha plane data at all, not even an empty RLE stream. Per agents/TCC.md,
	// this must decode as fully opaque rather than erroring.
	data := fixture(t, "099_PIRATESHIP.TCC")

	h, err := ParseHeader(data)
	if err != nil {
		t.Fatalf("ParseHeader: %v", err)
	}

	if h.StreamSize3 != 0 {
		t.Fatalf("fixture StreamSize3 = %d, want 0 (fixture assumption changed)", h.StreamSize3)
	}

	img, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	for i := 3; i < len(img.Pixels); i += 4 {
		if img.Pixels[i] != 255 {
			t.Fatalf("alpha at pixel %d = %d, want 255", i/4, img.Pixels[i])
		}
	}
}

func TestDecodePlaneToleratesSmallOverrun(t *testing.T) {
	// A single repeat command emitting one byte more than pixelCount must
	// still succeed, per the documented padding tolerance.
	pixelCount := 4
	src := []byte{0x01, byte(pixelCount + 1), 0x7F} // repeat op, count, value

	out, err := decodePlane(src, pixelCount)
	if err != nil {
		t.Fatalf("decodePlane: %v", err)
	}

	if len(out) != pixelCount {
		t.Fatalf("len(out) = %d, want %d", len(out), pixelCount)
	}
}
