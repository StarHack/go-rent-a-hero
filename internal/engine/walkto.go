package engine

import "math"

// DefaultWalkSpeed is the actor movement speed in scene pixels/second. Not
// derived from any original data (no confirmed source specifies it); it is
// a tunable default chosen for a visually reasonable pace.
const DefaultWalkSpeed = 120.0

// DefaultSnapRadius bounds how far NearestWalkablePixel will search when a
// requested walk target lands off the walk mask, per agents/ZBF.md
// ("Do not snap across large walls").
const DefaultSnapRadius = 60

// stepDistance is how far (in scene pixels) the actor travels between
// alternating footstep triggers. Not derived from data; chosen to
// approximate a natural stride length.
const stepDistance = 18.0

// FootstepSink receives footstep trigger events during a walk, keyed by the
// SFX filename declared on the actor (SFXLeft/SFXRight in the SZN). No
// audio backend exists yet (that lands with Milestone 4); this interface
// lets WalkTo report footstep timing now without depending on it.
type FootstepSink interface {
	PlayFootstep(sfxName string)
}

// NoopFootstepSink discards footstep events.
type NoopFootstepSink struct{}

func (NoopFootstepSink) PlayFootstep(string) {}

func UpdateActorPerspective(scene *Scene, actor *Actor) bool {
	if scene == nil || actor == nil || scene.Background == nil || scene.Background.Walk == nil {
		return false
	}
	r := scene.Background.Walk
	x := int(math.Round(actor.X))
	y := int(math.Round(actor.Y))
	if !r.InBounds(x, y) {
		return false
	}
	depth := int(r.At(x, y))
	if depth == 0 {
		return false
	}
	referenceDepth := actor.AuthoredZPos
	referenceZoom := actor.AuthoredZoom
	if referenceDepth == 0 {
		referenceDepth = actor.ZPos
	}
	if referenceZoom == 0 {
		referenceZoom = actor.Zoom
	}
	ratio := 1.0
	if r.Header.Param2 == 0 {
		ratio = float64(referenceDepth) / float64(depth)
	} else {
		span := int(r.Header.MaxValue) - int(r.Header.MinValue)
		if span == 0 {
			return false
		}
		slope := float64(r.Header.Param2-r.Header.Param1) / float64(span)
		referenceValue := float64(referenceDepth-int(r.Header.MinValue))*slope + float64(r.Header.Param1)
		currentValue := float64(depth-int(r.Header.MinValue))*slope + float64(r.Header.Param1)
		if currentValue == 0 {
			return false
		}
		ratio = referenceValue / currentValue
	}
	actor.ZPos = depth
	actor.Zoom = int(float64(referenceZoom) * ratio)
	return true
}

// WalkToTask moves an actor along a precomputed path at a fixed speed,
// updating its facing/walk-cycle animation and triggering alternating
// footstep events as it goes.

type WalkToTask struct {
	scene     *Scene
	actor     *Actor
	waypoints []Point
	index     int
	speed     float64
	footsteps FootstepSink

	distanceSinceStep float64
	nextStepIsLeft    bool
	failed            bool
}

// NewWalkTo builds a walk task for actor to (targetX, targetY) within
// scene. If the target is not walkable it is snapped to the nearest
// walkable pixel within DefaultSnapRadius; if no path exists at all (no
// walk raster, target unreachable, or too far from any walkable pixel to
// snap), the returned task completes on its very first Update without
// moving the actor, per agents/ZBF.md ("fail if none exists").
func NewStraightWalkTo(actor *Actor, targetX, targetY float64, footsteps FootstepSink) *WalkToTask {
	if footsteps == nil {
		footsteps = NoopFootstepSink{}
	}

	start := Point{int(math.Round(actor.X)), int(math.Round(actor.Y))}
	goal := Point{int(math.Round(targetX)), int(math.Round(targetY))}
	t := &WalkToTask{
		actor:          actor,
		waypoints:      []Point{start, goal},
		index:          1,
		speed:          DefaultWalkSpeed,
		footsteps:      footsteps,
		nextStepIsLeft: true,
	}
	actor.State = ActorWalking
	return t
}

func NewWalkTo(scene *Scene, actor *Actor, targetX, targetY float64, footsteps FootstepSink) *WalkToTask {
	if footsteps == nil {
		footsteps = NoopFootstepSink{}
	}

	t := &WalkToTask{scene: scene, actor: actor, speed: DefaultWalkSpeed, footsteps: footsteps, nextStepIsLeft: true}

	if scene == nil || scene.Background == nil || scene.Background.Walk == nil {
		t.failed = true
		return t
	}

	walk := scene.Background.Walk

	start := Point{int(math.Round(actor.X)), int(math.Round(actor.Y))}
	requestedGoal := Point{int(math.Round(targetX)), int(math.Round(targetY))}
	if !walk.InBounds(requestedGoal.X, requestedGoal.Y) {
		t.failed = true
		return t
	}

	grid := scene.Background.NavGrid()
	goal, ok := grid.ResolveTarget(requestedGoal.X, requestedGoal.Y)
	if !ok {
		t.failed = true
		return t
	}

	path, ok := grid.FindPath(start, goal)
	if !ok {
		t.failed = true
		return t
	}

	t.waypoints = path
	if len(path) > 1 {
		t.index = 1
	}
	actor.State = ActorWalking

	return t
}

// Path returns the remaining waypoints, for debug visualization.
func (t *WalkToTask) Path() []Point {
	return t.waypoints[t.index:]
}

// FullPath returns every waypoint of the originally computed path,
// regardless of progress, for post-hoc debug visualization.
func (t *WalkToTask) FullPath() []Point {
	return t.waypoints
}

func (t *WalkToTask) Update(dt float64) bool {
	if t.failed || t.index >= len(t.waypoints) {
		t.actor.State = ActorIdle
		t.actor.UpdateWalkAnimation(dt, false, 0, 0)
		UpdateActorPerspective(t.scene, t.actor)
		return true
	}

	for t.index < len(t.waypoints) {
		target := t.waypoints[t.index]
		dx := float64(target.X) - t.actor.X
		dy := float64(target.Y) - t.actor.Y
		dist := math.Hypot(dx, dy)
		if dist != 0 {
			break
		}
		t.index++
	}

	if t.index >= len(t.waypoints) {
		t.actor.State = ActorIdle
		t.actor.UpdateWalkAnimation(dt, false, 0, 0)
		UpdateActorPerspective(t.scene, t.actor)
		return true
	}

	target := t.waypoints[t.index]
	dx := float64(target.X) - t.actor.X
	dy := float64(target.Y) - t.actor.Y
	dist := math.Hypot(dx, dy)
	startX, startY := t.actor.X, t.actor.Y
	step := t.speed * dt

	if step >= dist {
		t.actor.X = float64(target.X)
		t.actor.Y = float64(target.Y)
	} else {
		ratio := step / dist
		t.actor.X += dx * ratio
		t.actor.Y += dy * ratio
	}

	movedDist := math.Hypot(t.actor.X-startX, t.actor.Y-startY)
	UpdateActorPerspective(t.scene, t.actor)
	t.actor.UpdateWalkAnimation(dt, true, dx, dy)
	t.actor.State = ActorWalking
	t.advanceFootsteps(movedDist)

	if step >= dist {
		t.index++
		if t.index >= len(t.waypoints) {
			t.actor.State = ActorIdle
			t.actor.UpdateWalkAnimation(0, false, 0, 0)
			UpdateActorPerspective(t.scene, t.actor)
			return true
		}
	}

	return false
}

func (t *WalkToTask) advanceFootsteps(movedDist float64) {
	t.distanceSinceStep += movedDist

	for t.distanceSinceStep >= stepDistance {
		t.distanceSinceStep -= stepDistance

		sfx := t.actor.StepRight
		if t.nextStepIsLeft {
			sfx = t.actor.StepLeft
		}
		t.nextStepIsLeft = !t.nextStepIsLeft

		if sfx != "" {
			t.footsteps.PlayFootstep(sfx)
		}
	}
}

// faceTask immediately sets an actor's facing and matching walk-cycle rest
// frame, completing on its first Update.
type faceTask struct {
	actor  *Actor
	facing FacingDirection
}

// Face returns a Task that orients actor to face the given direction.
func Face(actor *Actor, facing FacingDirection) Task {
	return &faceTask{actor: actor, facing: facing}
}

func (f *faceTask) Update(dt float64) bool {
	f.actor.Facing = f.facing
	f.actor.UpdateWalkAnimation(0, false, 0, 0)
	return true
}
