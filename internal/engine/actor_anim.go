package engine

import "math"

type FacingDirection int

const (
	FacingDown FacingDirection = iota
	FacingDownLeft
	FacingLeft
	FacingUpLeft
	FacingUp
	FacingUpRight
	FacingRight
	FacingDownRight
)

const (
	DirectionRight = FacingRight
	DirectionLeft  = FacingLeft
)

const walkCycleFrameCount = 105
const talkCycleFrameCount = 96

type walkCycleBand struct {
	start int
	end   int
	idle  int
}

var walkCycleBands = map[FacingDirection]walkCycleBand{
	FacingDown:      {12, 23, 98},
	FacingDownLeft:  {24, 35, 99},
	FacingLeft:      {36, 47, 100},
	FacingUpLeft:    {48, 59, 101},
	FacingUp:        {60, 71, 102},
	FacingUpRight:   {72, 83, 103},
	FacingRight:     {84, 95, 104},
	FacingDownRight: {0, 11, 97},
}

var type10IdleFrames = [8]int{0x16, 0x22, 0x2e, 0x34, 0x46, 0x52, 0x5e, 4}
var type11IdleFrames = [8]int{0x0d, 0x19, 0x25, 0x31, 0x3d, 0x49, 0x55, 1}

func actorIdleFrame(a *Actor) int {
	if a == nil {
		return 0
	}
	d := int(a.Facing) & 7
	switch a.Type {
	case 10:
		return type10IdleFrames[d]
	case 11, 12:
		return type11IdleFrames[d]
	default:
		return walkCycleBands[a.Facing].idle
	}
}

const DefaultWalkAnimFPS = 12.0

func FacingFromDirectionValue(v int) FacingDirection {
	if v < 0 || v > 7 {
		return FacingDown
	}
	return FacingDirection(v)
}

func ClassifyFacing(dx, dy float64) FacingDirection {
	ax := math.Abs(dx)
	ay := math.Abs(dy)
	if ax == 0 && ay == 0 {
		return FacingDown
	}
	ratio := ay * 90.0 / (ay + ax)
	if ratio < 23.0 || ratio > 67.0 {
		if ay < ax {
			dy = 0
		} else {
			dx = 0
		}
	}
	switch {
	case dx > 0 && dy > 0:
		return FacingDownRight
	case dx < 0 && dy > 0:
		return FacingDownLeft
	case dx > 0 && dy < 0:
		return FacingUpRight
	case dx < 0 && dy < 0:
		return FacingUpLeft
	case dx > 0:
		return FacingRight
	case dx < 0:
		return FacingLeft
	case dy < 0:
		return FacingUp
	default:
		return FacingDown
	}
}

func StepFacingToward(current, target FacingDirection) FacingDirection {
	c := int(current) & 7
	t := int(target) & 7
	d := (t - c + 8) & 7
	if d == 0 {
		return FacingDirection(c)
	}
	if d <= 4 {
		return FacingDirection((c + 1) & 7)
	}
	return FacingDirection((c + 7) & 7)
}

func (a *Actor) SetFacingImmediate(facing FacingDirection) {
	a.Facing = FacingDirection(int(facing) & 7)
	a.WalkPhase = -1
	a.WalkAccumulator = 0
	if a.Source == nil {
		return
	}
	frame := actorIdleFrame(a)
	if frame >= 0 && frame < a.Source.Frames() {
		a.Frame = frame
	}
}

func (a *Actor) StepFacing(facing FacingDirection) bool {
	target := FacingDirection(int(facing) & 7)
	if a.Facing == target {
		a.SetFacingImmediate(target)
		return true
	}
	a.Facing = StepFacingToward(a.Facing, target)
	a.WalkPhase = -1
	a.WalkAccumulator = 0
	if a.Source != nil {
		frame := actorIdleFrame(a)
		if frame >= 0 && frame < a.Source.Frames() {
			a.Frame = frame
		}
	}
	return a.Facing == target
}

func (a *Actor) UpdateWalkAnimation(dt float64, moving bool, dx, dy float64) {
	if a.Source == nil || a.Source.Frames() < talkCycleFrameCount {
		return
	}
	if !moving {
		a.WalkPhase = -1
		a.WalkAccumulator = 0
		frame := actorIdleFrame(a)
		if frame >= 0 && frame < a.Source.Frames() {
			a.Frame = frame
		}
		return
	}
	if dx != 0 || dy != 0 {
		target := ClassifyFacing(dx, dy)
		a.Direction = int(target)
		if a.Facing != target {
			a.Facing = StepFacingToward(a.Facing, target)
		}
	}
	band := walkCycleBands[a.Facing]
	count := band.end - band.start + 1
	if a.WalkPhase < 0 || a.WalkPhase >= count {
		a.WalkPhase = 0
	}
	a.Frame = band.start + a.WalkPhase
	frameDuration := 1.0 / DefaultWalkAnimFPS
	a.WalkAccumulator += dt
	for a.WalkAccumulator >= frameDuration {
		a.WalkAccumulator -= frameDuration
		a.WalkPhase = (a.WalkPhase + 1) % count
		a.Frame = band.start + a.WalkPhase
	}
}
