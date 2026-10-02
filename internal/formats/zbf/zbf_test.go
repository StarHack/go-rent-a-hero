package zbf

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

func assertCommonHeader(t *testing.T, name string) *Raster {
	t.Helper()

	data := fixture(t, name)

	if len(data) != 230420 {
		t.Fatalf("%s: file size = %d, want 230420", name, len(data))
	}

	r, err := Parse(data)
	if err != nil {
		t.Fatalf("%s: Parse: %v", name, err)
	}

	if r.Header.HeaderSize != 20 {
		t.Errorf("%s: HeaderSize = %d, want 20", name, r.Header.HeaderSize)
	}

	if r.Header.Param1 != 13.0 {
		t.Errorf("%s: Param1 = %v, want 13", name, r.Header.Param1)
	}

	if r.Header.MinValue != 5 {
		t.Errorf("%s: MinValue = %d, want 5", name, r.Header.MinValue)
	}

	if r.Header.Param2 != 25.0 {
		t.Errorf("%s: Param2 = %v, want 25", name, r.Header.Param2)
	}

	if r.Header.MaxValue != 250 {
		t.Errorf("%s: MaxValue = %d, want 250", name, r.Header.MaxValue)
	}

	if r.Width != 640 || r.Height != 360 {
		t.Errorf("%s: dims = %dx%d, want 640x360", name, r.Width, r.Height)
	}

	if len(r.Pixels) != 230400 {
		t.Errorf("%s: payload = %d, want 230400", name, len(r.Pixels))
	}

	return r
}

func TestParseDepth(t *testing.T) {
	assertCommonHeader(t, "BACK7A_DEPTH.ZBF")
}

func TestParsePathStatistics(t *testing.T) {
	r := assertCommonHeader(t, "BACK7A_PATH.ZBF")

	nonZero := 0
	uniqueVals := map[byte]bool{}
	var maxVal byte

	for _, v := range r.Pixels {
		uniqueVals[v] = true
		if v != 0 {
			nonZero++
		}
		if v > maxVal {
			maxVal = v
		}
	}

	if nonZero != 10374 {
		t.Errorf("non-zero count = %d, want 10374", nonZero)
	}

	if len(uniqueVals) != 195 {
		t.Errorf("unique values = %d, want 195", len(uniqueVals))
	}

	if maxVal != 212 {
		t.Errorf("max value = %d, want 212", maxVal)
	}

	if nonZero >= len(r.Pixels)/2 {
		t.Errorf("expected zero to dominate the path raster, got %d non-zero of %d", nonZero, len(r.Pixels))
	}
}

func TestWalkableMatchesValue(t *testing.T) {
	data := fixture(t, "BACK7A_PATH.ZBF")

	r, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	found := false

	for y := 0; y < r.Height && !found; y++ {
		for x := 0; x < r.Width; x++ {
			v := r.Value(x, y)
			if r.Walkable(x, y) != (v != 0) {
				t.Fatalf("Walkable/Value mismatch at (%d,%d): v=%d", x, y, v)
			}
			if v != 0 {
				found = true
				break
			}
		}
	}

	if !found {
		t.Fatal("expected at least one walkable pixel")
	}
}

func TestParseRejectsShortHeader(t *testing.T) {
	if _, err := Parse(make([]byte, 10)); err == nil {
		t.Error("expected error for too-short input")
	}
}

func TestParseRejectsWrongDimensions(t *testing.T) {
	data := make([]byte, MinHeaderSize+100)
	// HeaderSize field
	data[0] = 20

	if _, err := Parse(data); err == nil {
		t.Error("expected error for wrong payload size")
	}
}

func TestParseExtendedHeader(t *testing.T) {
	data := fixture(t, "S0013_DEPTH.ZBF")

	if len(data) != 230448 {
		t.Fatalf("fixture size = %d, want 230448", len(data))
	}

	r, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if !r.Header.Extended {
		t.Fatal("expected Extended header")
	}

	if r.Header.HeaderSize != 48 {
		t.Errorf("HeaderSize = %d, want 48", r.Header.HeaderSize)
	}

	if r.Header.MinValue != 5 || r.Header.MaxValue != 250 {
		t.Errorf("min/max = %d/%d, want 5/250", r.Header.MinValue, r.Header.MaxValue)
	}

	if r.Header.Width != 640 || r.Header.Height != 360 {
		t.Errorf("declared dims = %dx%d, want 640x360", r.Header.Width, r.Header.Height)
	}

	if r.Width != 640 || r.Height != 360 {
		t.Errorf("Raster dims = %dx%d, want 640x360", r.Width, r.Height)
	}

	if len(r.Pixels) != 640*360 {
		t.Errorf("payload = %d, want %d", len(r.Pixels), 640*360)
	}

	if len(r.Header.ExtraRaw) != ExtendedHeaderSize-28 {
		t.Errorf("ExtraRaw len = %d, want %d", len(r.Header.ExtraRaw), ExtendedHeaderSize-28)
	}
}
