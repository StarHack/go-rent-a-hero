package game

import (
	"math/rand"

	"github.com/wok/rent-a-hero/internal/engine"
)

// LOC37Controller implements Location 37 / Scene S3 — the walkable landing
// between intro Scenes 115 and 116 (see agents/scenes/3.MD and
// agents/scenes/115_TRANSITION.MD).
type LOC37Controller struct{}

const (
	loc37SceneID = "S3"
	loc37Actor   = "RodrigoSmall"

	loc37ProgressionFlag = "Loc37Progression"
	loc37GliderBusyFlag  = "Loc37GliderBusy"

	loc37ProgressionInitial    = 1
	loc37ProgressionAfterRod02 = 2

	loc37Event3InitialDelay = 30.0
	loc37Event3SecondDelay  = 10.0
	loc37Event3BusyDefer    = 1.0

	loc37HandoffX, loc37HandoffY = 280.0, 264.0
	loc37WalkX, loc37WalkY       = 340.0, 283.0
	loc37WalkDirection           = 5
)

// Loc37InitialSceneID is the scene id for Session.LoadInitialScene at Location 37.
func Loc37InitialSceneID() string { return loc37SceneID }

var loc37BirdSFX = []string{
	"Sfx_Bird.wav",
	"Sfx_Bird2.wav",
	"Sfx_Bird3.wav",
	"Sfx_Chirp.wav",
}

func (LOC37Controller) Enter(ctx *Context, scene string, from string) engine.Task {
	if scene != loc37SceneID {
		return nil
	}

	if actor, ok := ctx.actor(loc37Actor); ok {
		actor.Visible = false
	}

	return engine.Sequence(
		ctx.HideActor(loc37Actor),
		ctx.PlayMusic("Loc36_IntroDragonAttack.wav"),
		ctx.HideLayer("S3_Zeigen"),
		ctx.HideLayer("S3_Gleiter"),
		ctx.FreezeLayer("S3_Landen", 0),
		buildLoc37FirstEntry(ctx),
		ctx.MakeLayerClickable("S3_Gleiter"),
		ctx.RunAmbient(newLoc37SceneTask(ctx)),
	)
}

func buildLoc37FirstEntry(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.PlaySFX("Sfx_Glider_PassingBy.wav"),
		ctx.PlayLayerFrames("S3_Landen", 0, -1),
		// Present the final landing frame without a live actor, then hand
		// off on the next tick (agents/scenes/3_INTRO_HANDOFF_ALPHA_FIX.MD).
		&nextTickTask{},
		ctx.HideLayer("S3_Landen"),
		ctx.ShowLayer("S3_Gleiter"),
		ctx.PlaceActor(loc37Actor, loc37HandoffX, loc37HandoffY),
		ctx.ShowActor(loc37Actor),
		&nextTickTask{},
		ctx.WalkTo(loc37Actor, loc37WalkX, loc37WalkY),
		ctx.SetActorDirection(loc37Actor, loc37WalkDirection),
		ctx.HideActor(loc37Actor),
		// 003_ROD_01 is fire-and-forget; S3_Zeigen is what we wait on.
		ctx.RunAmbient(ctx.PlayVoiceover("003_ROD_01", "[003_ROD_01]")),
		ctx.ShowLayer("S3_Zeigen"),
		ctx.PlayLayerFrames("S3_Zeigen", 0, -1),
		ctx.HideLayer("S3_Zeigen"),
		ctx.ShowActor(loc37Actor),
		ctx.SetFlag(loc37ProgressionFlag, loc37ProgressionInitial),
	)
}

// nextTickTask completes on its second Update so a Sequence can present
// the current frame (e.g. the last S3_Landen frame) before running the
// following steps.
type nextTickTask struct{ armed bool }

func (t *nextTickTask) Update(dt float64) bool {
	if !t.armed {
		t.armed = true
		return false
	}
	return true
}

func (LOC37Controller) Exit(ctx *Context, scene string, to string) engine.Task { return nil }

func (LOC37Controller) Click(ctx *Context, area string) engine.Task {
	switch area {
	case "S3_Hoehle":
		return buildLoc37CaveExit(ctx)
	case "S3_Gleiter":
		return buildLoc37GliderInspect(ctx)
	default:
		return nil
	}
}

func (LOC37Controller) UseItem(ctx *Context, item int, area string) engine.Task { return nil }

func (LOC37Controller) SelectItem(ctx *Context, item int) engine.Task { return nil }

func (LOC37Controller) LoadConditionMask(ctx *Context, scene string) int { return 0 }

func buildLoc37CaveExit(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.WalkTo(loc37Actor, 482, 207),
		ctx.SetActorDirection(loc37Actor, 6),
		ctx.CompleteLocation(),
	)
}

func buildLoc37GliderInspect(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.SetFlag(loc37GliderBusyFlag, 1),
		ctx.WalkTo(loc37Actor, 289, 266),
		ctx.SetActorDirection(loc37Actor, 5),
		ctx.PlayVoiceover("003_ROD_04", "[003_ROD_04]"),
		ctx.SetFlag(loc37GliderBusyFlag, 0),
	)
}

func buildLoc37AutoCaveExit(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.WalkTo(loc37Actor, 482, 207),
		ctx.SetActorDirection(loc37Actor, 6),
		ctx.CompleteLocation(),
	)
}

// loc37SceneTask drives event 3 (delayed dialogue / auto exit) and event 4
// (ambient birds), per agents/scenes/3.MD sections 7–8.
type loc37SceneTask struct {
	ctx *Context

	event3Armed bool
	event3Timer float64

	event4Armed bool
	event4Timer float64

	voice   engine.Task
	birdSFX engine.Task
}

func newLoc37SceneTask(ctx *Context) *loc37SceneTask {
	return &loc37SceneTask{
		ctx:         ctx,
		event3Armed: true,
		event3Timer: loc37Event3InitialDelay,
		event4Armed: true,
		event4Timer: loc37RandomDelay(),
	}
}

func loc37RandomDelay() float64 {
	return 3.0 + rand.Float64()*7.0
}

func (t *loc37SceneTask) Update(dt float64) bool {
	if t.voice != nil && t.voice.Update(dt) {
		t.voice = nil
	}

	if t.birdSFX != nil && t.birdSFX.Update(dt) {
		t.birdSFX = nil
	}

	if t.event3Armed {
		t.event3Timer -= dt
		if t.event3Timer <= 0 {
			t.event3Armed = false
			t.handleEvent3()
		}
	}

	if t.event4Armed {
		t.event4Timer -= dt
		if t.event4Timer <= 0 {
			t.fireBird()
			t.event4Armed = true
			t.event4Timer = loc37RandomDelay()
		}
	}

	return false
}

func (t *loc37SceneTask) handleEvent3() {
	if t.ctx.GetFlag(loc37GliderBusyFlag) != 0 {
		t.armEvent3(loc37Event3BusyDefer)
		return
	}

	switch t.ctx.GetFlag(loc37ProgressionFlag) {
	case loc37ProgressionInitial:
		t.voice = engine.Sequence(
			t.ctx.PlayVoiceover("003_ROD_02", "[003_ROD_02]"),
			engine.Immediate(func() {
				if t.ctx.session.state.Flags == nil {
					t.ctx.session.state.Flags = map[string]int{}
				}
				t.ctx.session.state.Flags[loc37ProgressionFlag] = loc37ProgressionAfterRod02
			}),
		)
		t.armEvent3(loc37Event3SecondDelay)
	case loc37ProgressionAfterRod02:
		t.ctx.session.startControllerTask(engine.Sequence(
			t.ctx.PlayVoiceover("003_ROD_03", "[003_ROD_03]"),
			buildLoc37AutoCaveExit(t.ctx),
		))
	default:
		// First-entry sequence sets progression before the timer fires.
	}
}

func (t *loc37SceneTask) armEvent3(seconds float64) {
	t.event3Armed = true
	t.event3Timer = seconds
}

func (t *loc37SceneTask) fireBird() {
	if len(loc37BirdSFX) == 0 {
		return
	}
	clip := loc37BirdSFX[rand.Intn(len(loc37BirdSFX))]
	t.birdSFX = t.ctx.PlaySFX(clip)
}
