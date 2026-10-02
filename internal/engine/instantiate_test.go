package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wok/rent-a-hero/internal/assets"
	"github.com/wok/rent-a-hero/internal/formats/szn"
)

// loc01Index builds an asset index scoped exactly like the real game would
// see it for LOC01: the location's own directory (bare filenames, no
// subdirectory) plus the shared Common directory for cross-location assets
// like RodrigoSmall.a16.
func loc01Index(t *testing.T) *assets.Index {
	t.Helper()

	repoRoot := repoRoot(t)

	idx, err := assets.NewIndex(
		filepath.Join(repoRoot, "data", "cd", "GAME", "LOC01"),
		filepath.Join(repoRoot, "data", "installation", "Common"),
	)
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}

	return idx
}

func repoRoot(t *testing.T) string {
	t.Helper()

	// internal/engine -> repo root
	abs, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}

	return abs
}

func load7B(t *testing.T) *szn.Definition {
	t.Helper()

	path := filepath.Join(repoRoot(t), "data", "cd", "GAME", "LOC01", "7B.SZN")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read 7B.SZN: %v", err)
	}

	def, err := szn.Load(path, data)
	if err != nil {
		t.Fatalf("szn.Load: %v", err)
	}

	return def
}

func TestInstantiateScene7B(t *testing.T) {
	def := load7B(t)
	idx := loc01Index(t)

	scene, err := InstantiateScene(def, idx, nil, 0)
	if err != nil {
		t.Fatalf("InstantiateScene: %v", err)
	}

	if scene.Background == nil {
		t.Fatal("missing background")
	}

	if scene.Background.Source.Width() != 640 || scene.Background.Source.Height() != 360 {
		t.Errorf("background dims = %dx%d, want 640x360", scene.Background.Source.Width(), scene.Background.Source.Height())
	}

	if scene.Background.Depth == nil {
		t.Error("missing background depth raster")
	}
	if scene.Background.Walk == nil {
		t.Error("missing background walk raster")
	}

	actor, ok := scene.Characters["RodrigoSmall"]
	if !ok {
		t.Fatal("missing RodrigoSmall actor")
	}

	if actor.X != 353 || actor.Y != 165 {
		t.Errorf("actor pos = %v,%v, want 353,165", actor.X, actor.Y)
	}

	if actor.Source.Frames() < 1 {
		t.Error("actor source has no frames")
	}

	if len(scene.LayerOrder) != 2 {
		t.Fatalf("layers = %d, want 2", len(scene.LayerOrder))
	}

	hebel, ok := scene.Layers["Hebel"]
	if !ok {
		t.Fatal("missing Hebel layer")
	}
	if hebel.X != 320 || hebel.Y != 101 || hebel.Z != 214 {
		t.Errorf("Hebel placement = %d,%d,z=%d, want 320,101,z=214", hebel.X, hebel.Y, hebel.Z)
	}
	if !hebel.Enabled {
		t.Error("Hebel should be enabled (no LoadCond)")
	}

	if len(scene.AreaOrder) != 6 {
		t.Fatalf("areas = %d, want 6", len(scene.AreaOrder))
	}

	area, ok := scene.Areas["007b_Treppe"]
	if !ok {
		t.Fatal("missing 007b_Treppe area")
	}
	if area.CursorType != 1 {
		t.Errorf("CursorType = %d, want 1", area.CursorType)
	}
}

func TestInstantiateSceneLoadCondDefaultsHidden(t *testing.T) {
	path := filepath.Join(repoRoot(t), "data", "cd", "GAME", "LOC01", "46.SZN")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read 46.SZN: %v", err)
	}

	def, err := szn.Load(path, data)
	if err != nil {
		t.Fatalf("szn.Load: %v", err)
	}

	idx := loc01Index(t)

	scene, err := InstantiateScene(def, idx, nil, 0)
	if err != nil {
		t.Fatalf("InstantiateScene: %v", err)
	}

	layer, ok := scene.Layers["KanalTalk"]
	if !ok {
		t.Fatal("missing KanalTalk layer")
	}

	// KanalTalk has LoadCond=1, and mask=0 doesn't set that bit, so per the
	// documented conservative default it must be hidden, not shown --
	// LoadCond is a bitmask requirement, never a "controller ID" reference
	// (agents/scenes/7A-46.MD sections 1/18/26).
	if layer.Enabled {
		t.Error("KanalTalk should default to hidden when mask=0 doesn't satisfy LoadCond=1")
	}

	if len(scene.Warnings) == 0 {
		t.Error("expected a warning about LoadCond not being satisfied by the mask")
	}
}

// TestInstantiateSceneLoadCondBitmask locks in agents/scenes/46.MD section
// 16's fix: LoadCond is a bitmask test against the scene's own condition
// mask (shouldLoad = cond==0 || (cond&mask)==cond), not a reference to a
// "controller ID" -- passing the mask Scene 46's own constructor would
// compute when its story flag is active (mask=1) must load KanalTalk
// (LoadCond=1), matching game.LOC01Controller.LoadConditionMask.
func TestInstantiateSceneLoadCondBitmask(t *testing.T) {
	path := filepath.Join(repoRoot(t), "data", "cd", "GAME", "LOC01", "46.SZN")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read 46.SZN: %v", err)
	}

	def, err := szn.Load(path, data)
	if err != nil {
		t.Fatalf("szn.Load: %v", err)
	}

	idx := loc01Index(t)

	scene, err := InstantiateScene(def, idx, nil, 1)
	if err != nil {
		t.Fatalf("InstantiateScene: %v", err)
	}

	layer, ok := scene.Layers["KanalTalk"]
	if !ok {
		t.Fatal("missing KanalTalk layer")
	}

	if !layer.Enabled || !layer.Visible {
		t.Error("expected KanalTalk (LoadCond=1) to load and be visible when mask=1")
	}
	if len(scene.Warnings) != 0 {
		t.Errorf("expected no warnings once the condition is satisfied by the mask, got %v", scene.Warnings)
	}
}

func TestLayerAdvanceLoop(t *testing.T) {
	l := &Layer{FPS: 10, Mode: AnimLoop, Playing: true, Source: fakeSource{frames: 4}}

	l.Advance(0.05) // half a frame at 10fps
	if l.Frame != 0 {
		t.Errorf("Frame = %d, want 0 (not yet a full tick)", l.Frame)
	}

	l.Advance(0.05) // completes 1 frame
	if l.Frame != 1 {
		t.Errorf("Frame = %d, want 1", l.Frame)
	}

	l.Advance(0.35) // comfortably 3 more ticks, wraps 1->2->3->0
	if l.Frame != 0 {
		t.Errorf("Frame = %d, want 0 (looped)", l.Frame)
	}
}

func TestLayerAdvanceOnceStopsAtLastFrame(t *testing.T) {
	l := &Layer{FPS: 10, Mode: AnimOnce, Playing: true, Source: fakeSource{frames: 3}}

	l.Advance(1.0) // far more than enough to reach the end
	if l.Frame != 2 {
		t.Errorf("Frame = %d, want 2 (clamped at last frame)", l.Frame)
	}

	l.Advance(1.0)
	if l.Frame != 2 {
		t.Errorf("Frame = %d, want 2 (still clamped)", l.Frame)
	}
}

// TestLayerAdvanceOnceReverseCountsDownToZero locks in
// agents/scenes/S1_CLOSEUP_ALPHA_FLICKER_FIXES.MD sections 4/5/26: a reverse
// range must count down through the normal per-tick FPS-timed scheduler
// (not jump instantly), never go negative, and never wrap back around.
func TestLayerAdvanceOnceReverseCountsDownToZero(t *testing.T) {
	l := &Layer{FPS: 10, Mode: AnimOnceReverse, Frame: 4, Playing: true, Source: fakeSource{frames: 5}}

	// One tick at a time (0.1s per frame at 10fps) should count down
	// smoothly: 4, 3, 2, 1, 0 -- never skipping or clamping early.
	want := []int{3, 2, 1, 0}
	for i, w := range want {
		l.Advance(0.1)
		if l.Frame != w {
			t.Fatalf("after tick %d: Frame = %d, want %d", i, l.Frame, w)
		}
	}

	// Must stop at 0, not go negative or wrap.
	l.Advance(1.0)
	if l.Frame != 0 {
		t.Errorf("Frame = %d, want 0 (clamped, not negative or wrapped)", l.Frame)
	}
}

func TestLayerAdvanceStaticSourceNoop(t *testing.T) {
	l := &Layer{FPS: 10, Mode: AnimLoop, Playing: true, Source: fakeSource{frames: 1}}

	l.Advance(10.0)
	if l.Frame != 0 {
		t.Errorf("Frame = %d, want 0 (single-frame source never advances)", l.Frame)
	}
}

func TestInstantiateLayerDefaultsToPlaying(t *testing.T) {
	// A freshly-instantiated SZN layer must animate immediately (the
	// documented "Default scene-layer mode should be Loop"); only a
	// controller explicitly freezing a "special action" prop should stop
	// that.
	def := load7B(t)
	idx := loc01Index(t)

	scene, err := InstantiateScene(def, idx, nil, 0)
	if err != nil {
		t.Fatalf("InstantiateScene: %v", err)
	}

	layer := scene.Layers["PressLeverMitteLayer"]
	if !layer.Playing {
		t.Error("expected a freshly-instantiated layer to default to Playing=true")
	}
}

func TestLayerAdvanceNoopWhenNotPlaying(t *testing.T) {
	l := &Layer{FPS: 10, Mode: AnimLoop, Playing: false, Source: fakeSource{frames: 4}}

	l.Advance(10.0)
	if l.Frame != 0 {
		t.Errorf("Frame = %d, want 0: a frozen (Playing=false) layer must never advance", l.Frame)
	}
}

func TestFreezeLayerStopsAnimationAtGivenFrame(t *testing.T) {
	l := &Layer{FPS: 10, Mode: AnimLoop, Playing: true, Frame: 2, Source: fakeSource{frames: 4}}

	task := FreezeLayer(l, 0)
	if !task.Update(0) {
		t.Fatal("FreezeLayer task should complete on first Update")
	}

	if l.Playing {
		t.Error("expected Playing = false after FreezeLayer")
	}
	if l.Frame != 0 {
		t.Errorf("Frame = %d, want 0", l.Frame)
	}

	l.Advance(10.0)
	if l.Frame != 0 {
		t.Errorf("Frame = %d after Advance, want still 0 (frozen)", l.Frame)
	}
}

func TestPlayLayerOnceReanimatesAFrozenLayerThenRefreezes(t *testing.T) {
	l := &Layer{FPS: 10, Playing: false, Frame: 0, Source: fakeSource{frames: 3}}

	task := PlayLayerOnce(l)

	if !l.Playing {
		t.Fatal("expected PlayLayerOnce to set Playing = true immediately")
	}
	if !l.TaskDriven {
		t.Fatal("expected PlayLayerOnce to set TaskDriven = true immediately, so a caller's own per-tick Advance loop skips this layer while the task owns it")
	}

	if task.Update(1.0) != true {
		t.Fatal("expected the 3-frame animation to complete well within 1 simulated second at 10fps")
	}

	if l.Frame != 2 {
		t.Errorf("Frame = %d, want 2 (last frame)", l.Frame)
	}
	if l.Playing {
		t.Error("expected PlayLayerOnce to refreeze (Playing = false) once it reaches the last frame")
	}
	if l.TaskDriven {
		t.Error("expected PlayLayerOnce to clear TaskDriven once it reaches the last frame, handing Advance back to the caller's generic loop")
	}

	// Confirm it actually stays put afterward instead of looping.
	l.Advance(10.0)
	if l.Frame != 2 {
		t.Errorf("Frame = %d after further Advance, want still 2 (refrozen, not looping)", l.Frame)
	}
}

// TestPlayLayerOnceRestartLoopNeverProducesAGapSample locks in
// agents/scenes/S1_FLICKER_AND_INTERNAL_ALPHA.MD section 11's "no-flicker
// loop test": a short (4-frame) animation whose completion is immediately
// followed by re-arming it via a fresh PlayLayerOnce call -- exactly what a
// scripted "completion event -> restart" script step does -- must never
// produce a render sample where the layer has gone invisible or holds an
// out-of-range frame. PlayLayerOnce/layer.Advance never touch Visible at
// all, so the property under test is really that Frame always stays a
// valid, presentable index across the handoff, sampled far more often than
// the animation's own frame rate (matching the doc's "sampled at a higher
// presentation rate than the animation update rate").
func TestPlayLayerOnceRestartLoopNeverProducesAGapSample(t *testing.T) {
	l := &Layer{FPS: 4, Visible: true, Source: fakeSource{frames: 4}}
	task := PlayLayerOnce(l)

	const dt = 1.0 / 60.0 // sampled well above the 4fps animation rate
	for tick := range 600 {
		if task.Update(dt) {
			task = PlayLayerOnce(l) // completion callback immediately re-arms the same range
		}

		if !l.Visible {
			t.Fatalf("tick %d: layer went invisible across a completion/restart handoff", tick)
		}
		if l.Frame < 0 || l.Frame >= l.Source.Frames() {
			t.Fatalf("tick %d: Frame = %d out of range [0,%d)", tick, l.Frame, l.Source.Frames())
		}
	}
}

type fakeSource struct{ frames int }

func (f fakeSource) Width() int  { return 10 }
func (f fakeSource) Height() int { return 10 }
func (f fakeSource) Frames() int { return f.frames }
func (f fakeSource) RGBA(int) ([]byte, error) {
	return make([]byte, 10*10*4), nil
}
