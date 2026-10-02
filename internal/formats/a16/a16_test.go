package a16

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wok/rent-a-hero/internal/formats/tcc"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "fixtures", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}

	return data
}

var wantOffsets = []int{
	0, 3830, 7431, 10608, 13289, 15226, 17014, 19348, 22009, 24808,
	27447, 29979, 32641, 35317, 37731, 39667, 41535, 44119, 47184, 50734,
}

func TestParseItem001(t *testing.T) {
	data := fixture(t, "Item001.a16")

	if len(data) != 54539 {
		t.Fatalf("fixture size = %d, want 54539", len(data))
	}

	anim, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if anim.Count() != 20 {
		t.Fatalf("frame count = %d, want 20", anim.Count())
	}

	for i, f := range anim.Frames {
		if f.Offset != wantOffsets[i] {
			t.Errorf("frame %d offset = %d, want %d", i, f.Offset, wantOffsets[i])
		}

		if f.Header.Width != 60 || f.Header.Height != 60 {
			t.Errorf("frame %d dims = %dx%d, want 60x60", i, f.Header.Width, f.Header.Height)
		}
	}

	last := anim.Frames[len(anim.Frames)-1]
	size, err := tcc.FrameSize(last.Header)
	if err != nil {
		t.Fatalf("frame size: %v", err)
	}

	if last.Offset+size != len(data) {
		t.Errorf("last frame ends at %d, want EOF %d", last.Offset+size, len(data))
	}
}

func TestFirstFrameHeader(t *testing.T) {
	data := fixture(t, "Item001.a16")

	anim, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	h := anim.Frames[0].Header

	if h.Flags != 0x20 {
		t.Errorf("Flags = %#x, want 0x20", h.Flags)
	}

	if h.StreamSize0 != 1592 || h.StreamSize1 != 1592 || h.StreamSize2 != 45 || h.StreamSize3 != 573 {
		t.Errorf("stream sizes = %d,%d,%d,%d, want 1592,1592,45,573", h.StreamSize0, h.StreamSize1, h.StreamSize2, h.StreamSize3)
	}
}

func TestDecodeFrame0AlphaSteps(t *testing.T) {
	data := fixture(t, "Item001.a16")

	anim, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	img, err := anim.Decode(0)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	want := map[byte]bool{0: false, 51: false, 102: false, 153: false, 204: false, 255: false}

	for i := 0; i < img.Width*img.Height; i++ {
		a := img.Pixels[i*4+3]
		if _, ok := want[a]; ok {
			want[a] = true
		}
	}

	for v, seen := range want {
		if !seen {
			t.Errorf("expected to see alpha value %d in frame 0, did not", v)
		}
	}
}

func TestDecodeAllFrames(t *testing.T) {
	data := fixture(t, "Item001.a16")

	anim, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	for i := 0; i < anim.Count(); i++ {
		if _, err := anim.Decode(i); err != nil {
			t.Errorf("Decode(%d): %v", i, err)
		}
	}
}

func TestParseRejectsTrailingData(t *testing.T) {
	data := fixture(t, "Item001.a16")

	if _, err := Parse(data[:len(data)-1]); err == nil {
		t.Error("expected error when truncating final frame by one byte")
	}
}
