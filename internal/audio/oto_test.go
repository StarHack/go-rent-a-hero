package audio

import (
	"runtime"
	"testing"
	"time"
)

// TestOtoEngineRealPlayback exercises the real backend end-to-end. It skips
// itself if no audio device is available (e.g. a headless CI runner)
// rather than failing the suite, since agents/*.md audio guidance is about
// interface-level testability, not requiring real hardware in CI.
func TestOtoEngineRealPlayback(t *testing.T) {
	engine, err := NewOtoEngine(44100)
	if err != nil {
		t.Skipf("no audio device available: %v", err)
	}
	defer engine.Close()

	// Half a second of silence is enough to prove Play/IsPlaying/Elapsed
	// wiring works without making noise during test runs.
	sound := &Sound{SampleRate: 22050, Mono: make([]int16, 22050/2)}

	handle := engine.Play(sound, CategorySFX)

	if !handle.IsPlaying() {
		t.Error("expected playback to have started")
	}

	deadline := time.Now().Add(5 * time.Second)
	for handle.IsPlaying() && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}

	if handle.IsPlaying() {
		t.Error("playback did not finish within the deadline")
	}
}

// TestOtoEngineFireAndForgetSurvivesGC guards against a real regression:
// oto.Player's docs say the underlying player "is called by the finalizer"
// once its wrapper is unreachable, so a caller that discards Play's
// returned Handle (as PlaySFX/PlayMusic do -- fire-and-forget is the whole
// point) could have its audio silently cut short by an ordinary GC cycle.
// OtoEngine must keep every in-flight player referenced itself.
func TestOtoEngineFireAndForgetSurvivesGC(t *testing.T) {
	engine, err := NewOtoEngine(44100)
	if err != nil {
		t.Skipf("no audio device available: %v", err)
	}
	defer engine.Close()

	// 1.5s of silence: long enough to still be playing after forced GC.
	sound := &Sound{SampleRate: 22050, Mono: make([]int16, 22050*3/2)}

	_ = engine.Play(sound, CategorySFX) // fire-and-forget: discard the Handle

	if len(engine.active) != 1 {
		t.Fatalf("OtoEngine.active = %d players, want 1", len(engine.active))
	}
	player := engine.active[0]

	for range 5 {
		runtime.GC()
	}
	time.Sleep(50 * time.Millisecond)

	if !player.IsPlaying() {
		t.Error("fire-and-forget playback stopped early, likely finalized by GC")
	}
}

// TestOtoEnginePlayLoopingRestartsAutomatically guards the "ambient music
// shall not play ONCE but LOOP instead" fix: a Handle from PlayLooping must
// still report IsPlaying() == true well after the clip's own natural
// duration has elapsed for real, since the underlying reader wraps back to
// the start instead of ever reaching end of stream.
func TestOtoEnginePlayLoopingRestartsAutomatically(t *testing.T) {
	engine, err := NewOtoEngine(44100)
	if err != nil {
		t.Skipf("no audio device available: %v", err)
	}
	defer engine.Close()

	// A very short clip (0.1s) so waiting a few multiples of it past its own
	// length is fast, but long enough to be a real, non-degenerate buffer.
	sound := &Sound{SampleRate: 22050, Mono: make([]int16, 22050/10)}

	handle := engine.PlayLooping(sound, CategoryMusic)
	defer handle.Stop()

	if !handle.IsPlaying() {
		t.Fatal("expected playback to have started")
	}

	time.Sleep(500 * time.Millisecond) // ~5x the clip's own 0.1s length

	if !handle.IsPlaying() {
		t.Error("expected a looping handle to still be playing well past its own clip duration")
	}
}

// TestOtoEngineStopAllStopsEveryTrackedPlayer guards the fix for "whenever
// changing locations, the background audio (and voiceover) both have to
// stop properly": every fire-and-forget PlayMusic/PlaySFX call is tracked in
// e.active (see TestOtoEngineFireAndForgetSurvivesGC), and StopAll must
// pause all of them at once, regardless of category, and clear the tracking
// list.
func TestOtoEngineStopAllStopsEveryTrackedPlayer(t *testing.T) {
	engine, err := NewOtoEngine(44100)
	if err != nil {
		t.Skipf("no audio device available: %v", err)
	}
	defer engine.Close()

	sound := &Sound{SampleRate: 22050, Mono: make([]int16, 22050*3/2)}

	music := engine.Play(sound, CategoryMusic)
	sfx := engine.Play(sound, CategorySFX)
	speech := engine.Play(sound, CategorySpeech)

	if !music.IsPlaying() || !sfx.IsPlaying() || !speech.IsPlaying() {
		t.Fatal("expected all three handles to start out playing")
	}

	engine.StopAll()

	if music.IsPlaying() || sfx.IsPlaying() || speech.IsPlaying() {
		t.Error("expected StopAll to stop every category's in-flight playback")
	}
	if len(engine.active) != 0 {
		t.Errorf("engine.active = %d after StopAll, want 0", len(engine.active))
	}
}
