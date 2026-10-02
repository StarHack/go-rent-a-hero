package wav

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

func TestParseRod01(t *testing.T) {
	data := fixture(t, "046_ROD_01.WAV")

	s, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if s.Channels != 1 {
		t.Errorf("Channels = %d, want 1", s.Channels)
	}

	if s.BitsPerSample != 8 {
		t.Errorf("BitsPerSample = %d, want 8", s.BitsPerSample)
	}

	if s.SampleRate != 22050 {
		t.Errorf("SampleRate = %d, want 22050", s.SampleRate)
	}

	if s.Frames() != 46190 {
		t.Errorf("Frames = %d, want 46190", s.Frames())
	}

	wantDuration := 2.09478458
	if math.Abs(s.Duration()-wantDuration) > 1e-4 {
		t.Errorf("Duration = %v, want %v", s.Duration(), wantDuration)
	}
}

func TestToInt16Mono(t *testing.T) {
	data := fixture(t, "046_ROD_01.WAV")

	s, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	out := s.ToInt16Mono()

	if len(out) != s.Frames() {
		t.Fatalf("len(out) = %d, want %d", len(out), s.Frames())
	}
}

func TestParseRejectsBadMagic(t *testing.T) {
	if _, err := Parse([]byte("not a wav file at all")); err == nil {
		t.Error("expected error for non-RIFF input")
	}
}

func TestParseRejectsMissingChunks(t *testing.T) {
	// Valid RIFF/WAVE magic but no fmt/data chunks.
	data := []byte("RIFF\x04\x00\x00\x00WAVE")

	if _, err := Parse(data); err == nil {
		t.Error("expected error for missing fmt/data chunks")
	}
}
