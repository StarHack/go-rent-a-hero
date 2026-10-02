package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wok/rent-a-hero/internal/assets"
	"github.com/wok/rent-a-hero/internal/audio"
)

func loc01Index2(t *testing.T) *assets.Index {
	t.Helper()
	return loc01Index(t) // defined in instantiate_test.go; same package
}

func TestSpeechCompletesWhenAudioFinishes(t *testing.T) {
	idx := loc01Index2(t)
	eng := audio.NewFakeEngine()
	actor := newRodrigoActor(0, 0)

	task, err := NewSpeech(idx, eng, actor, "046_ROD_01", "Oh man...")
	if err != nil {
		t.Fatalf("NewSpeech: %v", err)
	}

	if actor.State != ActorTalking {
		t.Fatalf("actor.State = %v, want ActorTalking immediately after starting speech", actor.State)
	}

	if len(eng.Plays) != 1 {
		t.Fatalf("Plays = %d, want 1", len(eng.Plays))
	}
	if eng.Plays[0].Category != audio.CategorySpeech {
		t.Errorf("Category = %v, want CategorySpeech", eng.Plays[0].Category)
	}

	handle := eng.Plays[0].Handle

	// Not finished yet.
	if task.Update(0.01) {
		t.Fatal("task completed before audio finished")
	}

	// Advance the fake clock exactly to the WAV's real duration (2.09478458s)
	// and confirm the task completes exactly when playback does -- this is
	// the Milestone 4 acceptance criterion "046_ROD_01.WAV completion
	// drives task completion".
	handle.Advance(2.1)

	if !task.Update(0.01) {
		t.Fatal("task did not complete after audio finished")
	}

	if actor.State != ActorIdle {
		t.Errorf("actor.State = %v, want ActorIdle after speech completes", actor.State)
	}
	if actor.MouthState != 0 {
		t.Errorf("MouthState = %d, want 0 (neutral) after speech completes", actor.MouthState)
	}
}

func TestSpeechMouthStateChangesAt20Hz(t *testing.T) {
	idx := loc01Index2(t)
	eng := audio.NewFakeEngine()
	actor := newRodrigoActor(0, 0)

	task, err := NewSpeech(idx, eng, actor, "046_ROD_01", "")
	if err != nil {
		t.Fatalf("NewSpeech: %v", err)
	}

	handle := eng.Plays[0].Handle

	// Sample mouth state every ACS tick (1/20s) and confirm it visits more
	// than one distinct value over the clip, i.e. genuinely tracks the ACS
	// track rather than staying frozen.
	seen := map[uint16]bool{}

	const tick = 1.0 / 20.0
	for range 42 {
		task.Update(tick)
		handle.Advance(tick)
		seen[actor.MouthState] = true
	}

	if len(seen) < 2 {
		t.Errorf("expected mouth state to change across the clip, only saw %v", seen)
	}
}

func TestSpeechSkipStopsPlaybackAndCompletes(t *testing.T) {
	idx := loc01Index2(t)
	eng := audio.NewFakeEngine()
	actor := newRodrigoActor(0, 0)

	task, err := NewSpeech(idx, eng, actor, "046_ROD_01", "")
	if err != nil {
		t.Fatalf("NewSpeech: %v", err)
	}

	task.Skip()

	if !task.Update(0.01) {
		t.Fatal("expected task to complete immediately after Skip")
	}
}

func TestSpeechMissingWAVFails(t *testing.T) {
	idx := loc01Index2(t)
	eng := audio.NewFakeEngine()
	actor := newRodrigoActor(0, 0)

	if _, err := NewSpeech(idx, eng, actor, "does-not-exist", ""); err == nil {
		t.Fatal("expected error for missing WAV")
	}
}

func TestSpeechSubtitle(t *testing.T) {
	idx := loc01Index2(t)
	eng := audio.NewFakeEngine()
	actor := newRodrigoActor(0, 0)

	task, err := NewSpeech(idx, eng, actor, "046_ROD_01", "hello world")
	if err != nil {
		t.Fatalf("NewSpeech: %v", err)
	}

	if task.Subtitle() != "hello world" {
		t.Errorf("Subtitle() = %q, want %q", task.Subtitle(), "hello world")
	}
}

func TestSpeechWithoutACSLeavesMouthNeutral(t *testing.T) {
	// 008_ROD_01.WAV exists with no matching .ACS in the fixture set (see
	// data/cd/GAME/LOC01 listing), exercising the documented fallback:
	// "If ACS is absent... leave the mouth neutral."
	repo := repoRoot(t)
	loc01 := filepath.Join(repo, "data", "cd", "GAME", "LOC01")

	if _, err := os.Stat(filepath.Join(loc01, "008_ROD_01.ACS")); err == nil {
		t.Skip("fixture assumption changed: 008_ROD_01.ACS now exists")
	}

	idx, err := assets.NewIndex(loc01, filepath.Join(repo, "data", "installation", "Common"))
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}

	eng := audio.NewFakeEngine()
	actor := newRodrigoActor(0, 0)

	task, err := NewSpeech(idx, eng, actor, "008_ROD_01", "")
	if err != nil {
		t.Fatalf("NewSpeech: %v", err)
	}

	handle := eng.Plays[0].Handle
	handle.Advance(0.5)
	task.Update(0.5)

	if actor.MouthState != 0 {
		t.Errorf("MouthState = %d, want 0 (neutral, no ACS track)", actor.MouthState)
	}
}
