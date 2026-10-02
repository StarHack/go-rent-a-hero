package acs

import (
	"math"
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

var wantSamples = []uint16{
	0, 0, 0, 1, 1, 1, 5, 5, 5, 0, 0, 0, 2, 2, 1, 0, 0, 0, 2, 1,
	0, 0, 0, 5, 5, 5, 5, 5, 5, 5, 5, 4, 0, 0, 0, 5, 5, 4, 2, 2, 2, 0,
}

func TestParseRod01(t *testing.T) {
	data := fixture(t, "046_ROD_01.ACS")

	if len(data) != 84 {
		t.Fatalf("fixture size = %d, want 84", len(data))
	}

	track, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if track.Rate != 20 {
		t.Errorf("Rate = %v, want 20", track.Rate)
	}

	if len(track.Samples) != 42 {
		t.Fatalf("sample count = %d, want 42", len(track.Samples))
	}

	for i, want := range wantSamples {
		if track.Samples[i] != want {
			t.Errorf("sample[%d] = %d, want %d", i, track.Samples[i], want)
		}
	}

	wantDuration := 2.1
	if math.Abs(track.Duration()-wantDuration) > 0.05 {
		t.Errorf("Duration = %v, want ~%v", track.Duration(), wantDuration)
	}
}

func TestMouthStateAt(t *testing.T) {
	data := fixture(t, "046_ROD_01.ACS")

	track, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if got := track.MouthStateAt(0); got != 0 {
		t.Errorf("MouthStateAt(0) = %d, want 0", got)
	}

	// sample index 6 (0.3s) is 5 per wantSamples
	if got := track.MouthStateAt(0.3); got != 5 {
		t.Errorf("MouthStateAt(0.3) = %d, want 5", got)
	}

	// past the end of the track -> neutral
	if got := track.MouthStateAt(100); got != 0 {
		t.Errorf("MouthStateAt(100) = %d, want 0", got)
	}
}

func TestParseRejectsOddPayload(t *testing.T) {
	if _, err := Parse([]byte{0x01, 0x02, 0x03}); err == nil {
		t.Error("expected error for odd-length payload")
	}
}

func TestParseExtendedHeader(t *testing.T) {
	// marker=0xFFFF, offset=6, rate=10, then one sample word.
	data := []byte{0xFF, 0xFF, 0x06, 0x00, 0x0A, 0x00, 0x03, 0x00}

	track, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if track.Rate != 10 {
		t.Errorf("Rate = %v, want 10", track.Rate)
	}

	if len(track.Samples) != 1 || track.Samples[0] != 3 {
		t.Errorf("Samples = %v, want [3]", track.Samples)
	}
}
