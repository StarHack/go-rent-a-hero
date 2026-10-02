package audio

import (
	"math"
	"testing"
)

func TestSoundDuration(t *testing.T) {
	s := &Sound{SampleRate: 22050, Mono: make([]int16, 22050)}
	if d := s.Duration(); math.Abs(d-1.0) > 1e-9 {
		t.Errorf("Duration = %v, want 1.0", d)
	}
}

func TestResampleLinearUpsampleLength(t *testing.T) {
	input := make([]int16, 100)
	out := resampleLinear(input, 22050, 44100)

	wantLen := 200
	if len(out) != wantLen {
		t.Errorf("len(out) = %d, want %d", len(out), wantLen)
	}
}

func TestResampleLinearSameRateIsNoop(t *testing.T) {
	input := []int16{1, 2, 3}
	out := resampleLinear(input, 22050, 22050)

	if len(out) != len(input) {
		t.Fatalf("len = %d, want %d", len(out), len(input))
	}
	for i := range input {
		if out[i] != input[i] {
			t.Errorf("out[%d] = %d, want %d", i, out[i], input[i])
		}
	}
}

func TestResampleLinearInterpolatesRamp(t *testing.T) {
	// A linear ramp resampled should stay linear (interpolation exactness).
	input := []int16{0, 100, 200, 300}
	out := resampleLinear(input, 1, 2)

	if len(out) != 8 {
		t.Fatalf("len(out) = %d, want 8", len(out))
	}

	// out[0] should equal input[0]; out[2] should equal input[1]; etc.
	if out[0] != 0 {
		t.Errorf("out[0] = %d, want 0", out[0])
	}
}

func TestToStereoBytesInterleaves(t *testing.T) {
	mono := []int16{1, -1}
	b := toStereoBytes(mono)

	if len(b) != 8 {
		t.Fatalf("len = %d, want 8", len(b))
	}

	// Sample 0: value 1 (0x0001 LE = 01 00), duplicated across L/R.
	want := []byte{0x01, 0x00, 0x01, 0x00, 0xff, 0xff, 0xff, 0xff}
	for i := range want {
		if b[i] != want[i] {
			t.Errorf("b[%d] = %#x, want %#x", i, b[i], want[i])
		}
	}
}

func TestNullEngineHandleIsImmediatelyDone(t *testing.T) {
	e := NewNullEngine()
	h := e.Play(&Sound{SampleRate: 22050, Mono: make([]int16, 22050)}, CategorySpeech)

	if h.IsPlaying() {
		t.Error("NullEngine handle should never report playing")
	}
	if h.Elapsed() != 0 {
		t.Error("NullEngine handle elapsed should be 0")
	}

	h.Stop() // must not panic
}

func TestFakeEngineTracksPlaysAndAdvances(t *testing.T) {
	e := NewFakeEngine()
	sound := &Sound{SampleRate: 20, Mono: make([]int16, 40)} // 2 second clip

	handle := e.Play(sound, CategorySpeech)

	if len(e.Plays) != 1 {
		t.Fatalf("Plays = %d, want 1", len(e.Plays))
	}
	if e.Plays[0].Category != CategorySpeech {
		t.Errorf("Category = %v, want CategorySpeech", e.Plays[0].Category)
	}

	if !handle.IsPlaying() {
		t.Fatal("expected handle to be playing immediately after Play")
	}

	fake := e.Plays[0].Handle
	fake.Advance(1.0)

	if !handle.IsPlaying() {
		t.Error("expected still playing at 1.0s of a 2.0s clip")
	}
	if math.Abs(handle.Elapsed()-1.0) > 1e-9 {
		t.Errorf("Elapsed = %v, want 1.0", handle.Elapsed())
	}

	fake.Advance(1.5)

	if handle.IsPlaying() {
		t.Error("expected done after advancing past clip duration")
	}
}

func TestFakeEngineStop(t *testing.T) {
	e := NewFakeEngine()
	handle := e.Play(&Sound{SampleRate: 20, Mono: make([]int16, 200)}, CategorySFX)

	handle.Stop()

	if handle.IsPlaying() {
		t.Error("expected Stop to end playback immediately")
	}
}

func TestFakeHandleVolume(t *testing.T) {
	e := NewFakeEngine()
	h := e.PlayLoopingWithVolume(&Sound{SampleRate: 1, Mono: []int16{0}}, CategorySFX, 0.25)
	if got := e.Plays[0].Handle.Volume(); got != 0.25 {
		t.Fatalf("initial Volume = %v, want 0.25", got)
	}
	h.SetVolume(0.75)
	if got := e.Plays[0].Handle.Volume(); got != 0.75 {
		t.Fatalf("Volume = %v, want 0.75", got)
	}
}
