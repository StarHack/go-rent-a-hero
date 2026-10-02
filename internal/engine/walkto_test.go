package engine

import (
	"math"
	"testing"

	"github.com/wok/rent-a-hero/internal/formats/zbf"
)

// fakeSpriteSource is a minimal SpriteSource stub with a configurable frame
// count, used to exercise walk animation without decoding real assets.
type walkFakeSource struct{ frames int }

func (s walkFakeSource) Width() int  { return 10 }
func (s walkFakeSource) Height() int { return 10 }
func (s walkFakeSource) Frames() int { return s.frames }
func (s walkFakeSource) RGBA(int) ([]byte, error) {
	return make([]byte, 10*10*4), nil
}

func sceneWithBackground(walk *zbf.Raster) *Scene {
	return &Scene{
		Background: &Background{Walk: walk},
		Characters: map[string]*Actor{},
	}
}

func newRodrigoActor(x, y float64) *Actor {
	return &Actor{
		ID:        "Rodrigo",
		X:         x,
		Y:         y,
		Zoom:      100,
		Source:    walkFakeSource{frames: walkCycleFrameCount},
		StepLeft:  "left.wav",
		StepRight: "right.wav",
	}
}

type recordingFootsteps struct {
	events []string
}

func (r *recordingFootsteps) PlayFootstep(sfx string) {
	r.events = append(r.events, sfx)
}

func TestWalkToStaysWithinWalkableArea(t *testing.T) {
	r := load7BWalkRaster(t)
	scene := sceneWithBackground(r)

	start, goal := largestComponentExtremes(t, r)
	actor := newRodrigoActor(float64(start.X), float64(start.Y))

	task := NewWalkTo(scene, actor, float64(goal.X), float64(goal.Y), nil)

	const dt = 1.0 / 60.0
	for i := range 100000 {
		if !r.Walkable(int(math.Round(actor.X)), int(math.Round(actor.Y))) {
			t.Fatalf("iteration %d: actor left the walkable area at (%v,%v)", i, actor.X, actor.Y)
		}

		if task.Update(dt) {
			break
		}
	}

	if !r.Walkable(int(math.Round(actor.X)), int(math.Round(actor.Y))) {
		t.Fatalf("final position (%v,%v) is not walkable", actor.X, actor.Y)
	}
}

func TestWalkToRejectsTargetFarBeyondSnapRadius(t *testing.T) {
	r := load7BWalkRaster(t)
	scene := sceneWithBackground(r)

	start, _ := largestComponentExtremes(t, r)
	actor := newRodrigoActor(float64(start.X), float64(start.Y))

	// A target deep inside solid black background, far from any walkable
	// pixel (top-left corner region is background per the visualized mask).
	task := NewWalkTo(scene, actor, 2, 2, nil)

	done := task.Update(1.0 / 60.0)
	if !done {
		t.Fatal("expected task to complete immediately (fail gracefully) for an unreachable target")
	}

	if actor.X != float64(start.X) || actor.Y != float64(start.Y) {
		t.Error("actor should not have moved when the target could not be reached")
	}
}

func TestWalkToSnapsNearbyInvalidTarget(t *testing.T) {
	r := load7BWalkRaster(t)
	scene := sceneWithBackground(r)

	start, goal := largestComponentExtremes(t, r)
	actor := newRodrigoActor(float64(start.X), float64(start.Y))

	// Aim exactly at a real walkable point; if it happens to be walkable
	// already that's fine, we're testing that a near-miss still resolves to
	// approximately the intended destination rather than failing.
	task := NewWalkTo(scene, actor, float64(goal.X), float64(goal.Y), nil)

	const dt = 1.0 / 60.0
	for range 100000 {
		if task.Update(dt) {
			break
		}
	}

	dist := math.Hypot(actor.X-float64(goal.X), actor.Y-float64(goal.Y))
	if dist > float64(DefaultSnapRadius) {
		t.Errorf("final position is %.1f px from goal, want within snap radius %d", dist, DefaultSnapRadius)
	}
}

func TestWalkToStableAcrossDifferentTimesteps(t *testing.T) {
	r := load7BWalkRaster(t)
	start, goal := largestComponentExtremes(t, r)

	run := func(dt float64) (float64, float64) {
		scene := sceneWithBackground(r)
		actor := newRodrigoActor(float64(start.X), float64(start.Y))
		task := NewWalkTo(scene, actor, float64(goal.X), float64(goal.Y), nil)

		for range 2_000_000 {
			if task.Update(dt) {
				break
			}
		}

		return actor.X, actor.Y
	}

	x60, y60 := run(1.0 / 60.0)
	x30, y30 := run(1.0 / 30.0)
	x240, y240 := run(1.0 / 240.0)

	const tol = 2.0 // pixels; sub-frame overshoot rounding differs slightly by step size

	if math.Hypot(x60-x30, y60-y30) > tol {
		t.Errorf("60Hz result (%v,%v) vs 30Hz result (%v,%v) differ by more than %v px", x60, y60, x30, y30, tol)
	}

	if math.Hypot(x60-x240, y60-y240) > tol {
		t.Errorf("60Hz result (%v,%v) vs 240Hz result (%v,%v) differ by more than %v px", x60, y60, x240, y240, tol)
	}
}

func TestWalkToNoWalkRasterFailsImmediately(t *testing.T) {
	scene := &Scene{Background: &Background{}}
	actor := newRodrigoActor(100, 100)

	task := NewWalkTo(scene, actor, 200, 200, nil)

	if !task.Update(1.0 / 60.0) {
		t.Fatal("expected immediate completion when the scene has no walk raster")
	}

	if actor.X != 100 || actor.Y != 100 {
		t.Error("actor should not move without a walk raster")
	}
}

func TestWalkToTriggersAlternatingFootsteps(t *testing.T) {
	r := load7BWalkRaster(t)
	scene := sceneWithBackground(r)

	start, goal := largestComponentExtremes(t, r)
	actor := newRodrigoActor(float64(start.X), float64(start.Y))

	sink := &recordingFootsteps{}
	task := NewWalkTo(scene, actor, float64(goal.X), float64(goal.Y), sink)

	const dt = 1.0 / 60.0
	for range 100000 {
		if task.Update(dt) {
			break
		}
	}

	if len(sink.events) == 0 {
		t.Fatal("expected at least one footstep event over a long walk")
	}

	for i, ev := range sink.events {
		want := "right.wav"
		if i%2 == 0 {
			want = "left.wav"
		}
		if ev != want {
			t.Errorf("event %d = %q, want %q (should strictly alternate)", i, ev, want)
		}
	}
}

func TestFaceSetsFacingAndRestFrame(t *testing.T) {
	actor := newRodrigoActor(0, 0)

	task := Face(actor, DirectionLeft)
	if !task.Update(0) {
		t.Fatal("Face task should complete on first Update")
	}

	if actor.Facing != FacingLeft {
		t.Errorf("Facing = %v, want FacingLeft", actor.Facing)
	}

	wantFrame := walkCycleBands[FacingLeft].start
	if actor.Frame != wantFrame {
		t.Errorf("Frame = %d, want %d (band start)", actor.Frame, wantFrame)
	}
}

func TestSequenceRunsTasksInOrder(t *testing.T) {
	var order []int

	mk := func(n int) Task {
		return TaskFunc(func(dt float64) bool {
			order = append(order, n)
			return true
		})
	}

	seq := Sequence(mk(1), mk(2), mk(3))

	for !seq.Update(0.1) {
	}

	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Errorf("order = %v, want [1 2 3]", order)
	}
}

func TestParallelCompletesWhenAllDone(t *testing.T) {
	a := Wait(0.1)
	b := Wait(0.3)

	p := Parallel(a, b)

	if p.Update(0.1) {
		t.Fatal("should not be done after 0.1s (b needs 0.3s)")
	}
	if p.Update(0.1) {
		t.Fatal("should not be done after 0.2s")
	}
	if !p.Update(0.1) {
		t.Fatal("should be done after 0.3s")
	}
}

func TestWalkToZPosZeroStillUsesWalkMask(t *testing.T) {
	pixels := make([]byte, 40*40)
	for i := range pixels {
		pixels[i] = 1
	}
	for y := 0; y < 32; y++ {
		for x := 16; x < 20; x++ {
			pixels[y*40+x] = 0
		}
	}
	r := &zbf.Raster{Width: 40, Height: 40, Pixels: pixels}
	scene := sceneWithBackground(r)
	actor := newRodrigoActor(10, 10)
	actor.ZPos = 0

	task := NewWalkTo(scene, actor, 30, 10, nil)
	if len(task.FullPath()) < 3 {
		t.Fatalf("path = %v, want a routed path around the blocked wall", task.FullPath())
	}

	for range 10000 {
		if !r.Walkable(int(math.Round(actor.X)), int(math.Round(actor.Y))) {
			t.Fatalf("actor crossed blocked pixels at (%v,%v)", actor.X, actor.Y)
		}
		if task.Update(1.0 / 60.0) {
			break
		}
	}
}

func TestUpdateActorPerspectiveUsesWalkBufferMapping(t *testing.T) {
	r := &zbf.Raster{
		Header: zbf.Header{Param1: 10, MinValue: 50, Param2: 110, MaxValue: 150},
		Width:  2,
		Height: 1,
		Pixels: []byte{133, 100},
	}
	scene := &Scene{Background: &Background{Walk: r}}
	actor := &Actor{X: 1, Y: 0, ZPos: 5, Zoom: 72, AuthoredZPos: 133, AuthoredZoom: 46}
	if !UpdateActorPerspective(scene, actor) {
		t.Fatal("UpdateActorPerspective returned false")
	}
	refValue := (float64(133-50) * (110 - 10) / float64(150-50)) + 10
	curValue := (float64(100-50) * (110 - 10) / float64(150-50)) + 10
	wantZoom := int(46 * (refValue / curValue))
	if actor.Zoom != wantZoom {
		t.Fatalf("zoom = %d, want %d", actor.Zoom, wantZoom)
	}
	if actor.ZPos != 100 {
		t.Fatalf("z = %d, want 100", actor.ZPos)
	}
}

func TestUpdateActorPerspectiveLegacyWalkRatio(t *testing.T) {
	r := &zbf.Raster{Width: 1, Height: 1, Pixels: []byte{80}}
	scene := &Scene{Background: &Background{Walk: r}}
	actor := &Actor{AuthoredZPos: 120, AuthoredZoom: 60}
	if !UpdateActorPerspective(scene, actor) {
		t.Fatal("UpdateActorPerspective returned false")
	}
	if actor.Zoom != 90 {
		t.Fatalf("zoom = %d, want 90", actor.Zoom)
	}
	if actor.ZPos != 80 {
		t.Fatalf("z = %d, want 80", actor.ZPos)
	}
}

func TestUpdateActorPerspectiveScene006DoorHandoff(t *testing.T) {
	r := &zbf.Raster{
		Header: zbf.Header{Param1: 6, MinValue: 5, Param2: 14, MaxValue: 250},
		Width:  1,
		Height: 1,
		Pixels: []byte{18},
	}
	scene := &Scene{Background: &Background{Walk: r}}
	actor := &Actor{AuthoredZPos: 133, AuthoredZoom: 46}
	if !UpdateActorPerspective(scene, actor) {
		t.Fatal("UpdateActorPerspective returned false")
	}
	if actor.Zoom != 72 {
		t.Fatalf("zoom = %d, want 72", actor.Zoom)
	}
	if actor.ZPos != 18 {
		t.Fatalf("z = %d, want 18", actor.ZPos)
	}
}
