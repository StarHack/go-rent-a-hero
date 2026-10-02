package game

import (
	"math/rand/v2"

	"github.com/wok/rent-a-hero/internal/engine"
)

const (
	loc03Scene = "S12"
	loc03Actor = "RodrigoSmall"

	FlagLoc03RaidersGone = "loc03.raiders_gone"
)

type LOC03Controller struct{}

func (LOC03Controller) Enter(ctx *Context, scene string, from string) engine.Task {
	if scene != loc03Scene {
		return nil
	}

	music := "Loc01_Smashville.wav"
	if from == "S0015" {
		music = "Loc09_SmashvilleAttack.wav"
		_, _, _ = ctx.ensureAssetLayer("S12_PirateShip")
	}

	tasks := []engine.Task{
		ctx.PlayMusic(music),
		loc03Initialize(ctx),
		ctx.RunAmbient(newLoc03BirdAmbient(ctx)),
		ctx.MakeLayerClickable("S12_RodGlider"),
	}
	if ctx.GetFlag(FlagLoc03RaidersGone) == 0 {
		tasks = append(tasks,
			ctx.MakeLayerClickable("S12_SabSword"),
			ctx.MakeLayerClickable("S12_SabGlider"),
			ctx.MakeLayerClickable("S12_Rab1Talk"),
			ctx.MakeLayerClickable("S12_Rab2"),
		)
	}

	if ctx.GetFlag(FlagLoc03RaidersGone) == 0 {
		tasks = append(tasks, ctx.RunAmbient(newLoc03RaiderAmbient(ctx)))
	}

	switch from {
	case "8":
		tasks = append(tasks,
			ctx.HideActor(loc03Actor),
			loc03PlayVisibleRange(ctx, "S12_Anflug", 0, -1, false),
			ctx.ShowLayer("S12_RodGlider"),
			ctx.ShowActor(loc03Actor),
			ctx.PlaceActor(loc03Actor, 0x27B, 0x14E),
			ctx.WalkTo(loc03Actor, 0x245, 0x137),
		)
	case "S0015":
		tasks = append(tasks, loc03AttackArrival(ctx))
	case "13":
		tasks = append(tasks,
			ctx.ShowLayer("S12_RodGlider"),
			ctx.PlaceActor(loc03Actor, 0x27A, 0x10E),
			ctx.WalkTo(loc03Actor, 0x238, 0x118),
		)
	default:
		tasks = append(tasks, ctx.ShowLayer("S12_RodGlider"), ctx.ShowActor(loc03Actor))
	}

	return engine.Sequence(tasks...)
}

func (LOC03Controller) Exit(ctx *Context, scene string, to string) engine.Task { return nil }

func (LOC03Controller) Click(ctx *Context, area string) engine.Task {
	if ctx.session.state.Scene != loc03Scene {
		return nil
	}

	switch area {
	case "S12_Bridge":
		return loc03SpeakAt(ctx, 0x81, 0x123, 3, "012_ROD_03")
	case "S12_Schild":
		return loc03SpeakAt(ctx, 0x1F9, 0x117, 4, "012_ROD_01")
	case "S12_To13":
		return engine.Sequence(ctx.WalkToFacing(loc03Actor, 0x27A, 0x10E, 6), ctx.ChangeLocation(2, "13"))
	case "S12_Pflanze":
		return loc03SpeakAt(ctx, 0x22A, 0x123, 4, "012_ROD_02")
	case "S12_RodGlider":
		return loc03Depart(ctx)
	case "S12_SabGlider":
		line := "012_ROD_04"
		if rand.IntN(2) != 0 {
			line = "012_ROD_05"
		}
		return loc03SpeakAt(ctx, 0x19F, 0x108, 4, line)
	case "S12_SabSword", "S12_Rab1Talk", "S12_Rab2":
		if ctx.GetFlag(FlagLoc03RaidersGone) == 0 {
			return loc03ConfrontRaiders(ctx)
		}
	}
	return nil
}

func (LOC03Controller) UseItem(ctx *Context, item int, area string) engine.Task { return nil }

func (LOC03Controller) SelectItem(ctx *Context, item int) engine.Task { return nil }

func (LOC03Controller) LoadConditionMask(ctx *Context, scene string) int {
	if scene == loc03Scene && ctx.GetFlag(FlagLoc03RaidersGone) == 0 {
		return 1
	}
	return 0
}

func loc03Initialize(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		_, _, _ = ctx.ensureAssetLayer("S12_Wegflug")
		_, _, _ = ctx.ensureAssetLayer("S12_PirateShip")
		scene := ctx.session.scene
		for _, id := range []string{"S12_Wegflug", "S12_Anflug", "S12_Rab1Run", "S12_SabInGlider", "SabWalk"} {
			if l, ok := scene.Layers[id]; ok {
				l.Visible = false
				l.Playing = false
				l.Frame = 0
				l.Accumulator = 0
				l.TaskDriven = false
			}
		}
		if l, ok := scene.Layers["S12_RodGlider"]; ok {
			l.Visible = false
			l.Playing = false
			l.Frame = 0
			l.Accumulator = 0
		}
		if ctx.GetFlag(FlagLoc03RaidersGone) == 0 {
			for _, id := range []string{"S12_Rab1Talk", "S12_Rab2", "S12_SabSword", "S12_SabGlider"} {
				if l, ok := scene.Layers[id]; ok {
					l.Visible = true
					l.Enabled = true
					l.Playing = false
					l.Frame = 0
					l.Accumulator = 0
					l.TaskDriven = false
				}
			}
		} else {
			for _, id := range []string{"S12_Rab1Talk", "S12_Rab2", "S12_SabSword", "S12_SabGlider", "S12_Rab1Run", "S12_SabInGlider", "SabWalk"} {
				if l, ok := scene.Layers[id]; ok {
					l.Visible = false
					l.Playing = false
				}
			}
		}
	})
}

func loc03SpeakAt(ctx *Context, x, y float64, direction int, line string) engine.Task {
	return engine.Sequence(ctx.WalkToFacing(loc03Actor, x, y, direction), ctx.Say(loc03Actor, line, "["+line+"]"))
}

func loc03PlayVisibleRange(ctx *Context, id string, from, to int, hideAfter bool) engine.Task {
	tasks := []engine.Task{ctx.ShowLayer(id), ctx.PlayLayerFrames(id, from, to)}
	if hideAfter {
		tasks = append(tasks, ctx.HideLayer(id))
	}
	return engine.Sequence(tasks...)
}

func loc03LayerSpeech(ctx *Context, id, line string, start, end int) engine.Task {
	if l, ok := ctx.session.scene.Layers[id]; ok && l.Source != nil {
		last := l.Source.Frames() - 1
		if end < 0 || end > last {
			end = last
		}
		if start > last {
			start = 0
		}
		return engine.Sequence(ctx.ShowLayer(id), ctx.PlaySpeechBoundToLayer(id, line, "["+line+"]", start, end), ctx.FreezeLayer(id, 0))
	}
	return ctx.PlayVoiceover(line, "["+line+"]")
}

func loc03Depart(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.WalkToFacing(loc03Actor, 0x27B, 0x14E, 6),
		ctx.HideActor(loc03Actor),
		ctx.HideLayer("S12_RodGlider"),
		ctx.ShowLayer("S12_Wegflug"),
		ctx.PlayLayerFrames("S12_Wegflug", 0, 0x1E),
		ctx.PlaySFX("Sfx_Glider_PassingBy.wav"),
		ctx.PlayLayerFrames("S12_Wegflug", 0x1F, 0x44),
		ctx.PlaySFX("Sfx_Glider_PassingBy.wav"),
		ctx.PlayLayerFrames("S12_Wegflug", 0x45, -1),
		ctx.ChangeLocation(1, "7A"),
	)
}

func loc03ConfrontRaiders(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.WalkToFacing(loc03Actor, 0x1F1, 0x15B, 3),
		ctx.Say(loc03Actor, "012_ROD_06", "[012_ROD_06]"),
		loc03LayerSpeech(ctx, "S12_Rab1Talk", "012_RA1_03", 0, 4),
		loc03LayerSpeech(ctx, "S12_Rab2", "012_RA2_03", 0, 4),
		loc03LayerSpeech(ctx, "S12_Rab1Talk", "012_RA1_04", 0, 4),
		ctx.PlayVoiceover("012_RA2_04", "[012_RA2_04]"),
		loc03LayerSpeech(ctx, "S12_Rab1Talk", "012_RA1_05", 0, 4),
		ctx.FreezeLayer("S12_Rab2", 0),
		loc03LayerSpeech(ctx, "S12_Rab1Talk", "012_RA1_06", 0, 4),
		loc03LayerSpeech(ctx, "S12_Rab2", "012_RA2_05", 0, 4),
		ctx.HideLayer("S12_Rab1Talk"),
		loc03PlayVisibleRange(ctx, "S12_Rab1Run", 0, 3, false),
		engine.Parallel(
			engine.Parallel(ctx.PlayLayerFrames("S12_Rab1Run", 4, 8), loc03MoveLayer(ctx, "S12_Rab1Run", 625, 215, 1.5)),
			engine.Sequence(
				ctx.PlayLayerFrames("S12_Rab2", 5, 7),
				engine.Parallel(ctx.PlayLayerFrames("S12_Rab2", 8, 0x0E), loc03MoveLayer(ctx, "S12_Rab2", 615, 240, 1.5)),
			),
		),
		ctx.HideLayer("S12_Rab1Run"),
		ctx.HideLayer("S12_Rab2"),
		ctx.Say(loc03Actor, "012_ROD_07", "[012_ROD_07]"),
		ctx.PlayVoiceover("012_SAB_01", "[012_SAB_01]"),
		ctx.PlayLayerFrames("S12_SabSword", 0, 8),
		ctx.PlaySFX("Sfx_SwordPull.wav"),
		ctx.PlayLayerFrames("S12_SabSword", 9, 0x1E),
		ctx.PlayLayerFrames("S12_SabSword", 0x1D, 0x12),
		ctx.PlaySFX("Sfx_SwordPushShort.wav"),
		ctx.PlayLayerFrames("S12_SabSword", 0x11, 0),
		ctx.HideLayer("S12_SabSword"),
		ctx.ShowLayer("SabWalk"),
		ctx.RunAmbient(ctx.Say(loc03Actor, "012_ROD_08", "[012_ROD_08]")),
		ctx.SetFlag(FlagLoc03RaidersGone, 1),
		loc03SabineWalkToGlider(ctx),
		ctx.HideLayer("SabWalk"),
		ctx.HideLayer("S12_SabGlider"),
		ctx.ShowLayer("S12_SabInGlider"),
		ctx.PlayLayerFrames("S12_SabInGlider", 0, 0x39),
		ctx.PlaySFX("Sfx_Glider_PassingBy.wav"),
		ctx.PlayLayerFrames("S12_SabInGlider", 0x3A, -1),
		ctx.HideLayer("S12_SabInGlider"),
	)
}

func loc03MoveLayer(ctx *Context, id string, x, y int, seconds float64) engine.Task {
	return &loc03LayerMoveTask{ctx: ctx, id: id, targetX: x, targetY: y, duration: seconds}
}

func loc03SabineWalkToGlider(ctx *Context) engine.Task {
	return &loc03SabineWalkTask{ctx: ctx, steps: 52}
}

type loc03SabineWalkTask struct {
	ctx                    *Context
	steps, elapsed         int
	startX, startY, startZ int
	startZoom              int
	started                bool
}

func (t *loc03SabineWalkTask) Update(dt float64) bool {
	l, ok := t.ctx.session.scene.Layers["SabWalk"]
	if !ok {
		return true
	}
	if !t.started {
		t.started = true
		t.startX = l.X
		t.startY = l.Y
		t.startZ = l.Z
		t.startZoom = l.Zoom
		l.Mode = engine.AnimOnce
		l.Frame = 0x30
		l.Accumulator = 0
		l.Playing = true
		l.TaskDriven = true
	}
	if t.steps <= 0 {
		l.X = 411
		l.Y = 177
		l.Z = 164
		l.Zoom = 52
		l.Playing = false
		l.TaskDriven = false
		return true
	}
	l.Advance(dt)
	t.elapsed++
	p := float64(t.elapsed) / float64(t.steps)
	if p > 1 {
		p = 1
	}
	l.X = t.startX + int(float64(411-t.startX)*p)
	l.Y = t.startY + int(float64(177-t.startY)*p)
	l.Z = t.startZ + int(float64(164-t.startZ)*p)
	l.Zoom = t.startZoom + int(float64(52-t.startZoom)*p)
	if t.elapsed < t.steps {
		return false
	}
	l.X = 411
	l.Y = 177
	l.Z = 164
	l.Zoom = 52
	l.Playing = false
	l.TaskDriven = false
	return true
}

type loc03LayerMoveTask struct {
	ctx               *Context
	id                string
	targetX, targetY  int
	duration, elapsed float64
	startX, startY    int
	started           bool
}

func (t *loc03LayerMoveTask) Update(dt float64) bool {
	l, ok := t.ctx.session.scene.Layers[t.id]
	if !ok {
		return true
	}
	if !t.started {
		t.started = true
		t.startX = l.X
		t.startY = l.Y
	}
	if t.duration <= 0 {
		l.X = t.targetX
		l.Y = t.targetY
		return true
	}
	t.elapsed += dt
	p := t.elapsed / t.duration
	if p > 1 {
		p = 1
	}
	l.X = t.startX + int(float64(t.targetX-t.startX)*p)
	l.Y = t.startY + int(float64(t.targetY-t.startY)*p)
	return p >= 1
}

func loc03AttackArrival(ctx *Context) engine.Task {
	return engine.Sequence(
		engine.Immediate(func() {
			ctx.session.scene.InventoryHidden = true
			ctx.session.scene.InventoryHoverIndex = -1
			ctx.session.scene.InventoryClickIndex = -1
		}),
		ctx.HideActor(loc03Actor),
		ctx.ShowLayer("S12_PirateShip"),
		ctx.PlayLayerFrames("S12_PirateShip", 0, 0x3A),
		engine.Parallel(
			ctx.PlayVoiceover("012_GA1_01", "[012_GA1_01]"),
			ctx.PlayLayerFrames("S12_PirateShip", 0x3B, 0x5E),
		),
		engine.Parallel(
			ctx.PlayVoiceover("012_GA2_01", "[012_GA2_01]"),
			ctx.PlayLayerFrames("S12_PirateShip", 0x5F, -1),
		),
		ctx.SetFlag(FlagLoc05PirateAttackSeen, 1),
		ctx.SetFlag(flagLoc05QuestB, 0),
		ctx.HideLayer("S12_PirateShip"),
		engine.Immediate(func() {
			ctx.session.scene.InventoryHidden = false
			ctx.session.audioEngine.StopAll()
		}),
		ctx.PlayMusic("Loc01_Smashville.wav"),
		ctx.ShowLayer("S12_RodGlider"),
		ctx.ShowActor(loc03Actor),
		ctx.PlaceActor(loc03Actor, 0xEE, 0x126),
		ctx.WalkTo(loc03Actor, 0x13B, 0x15C),
	)
}

type loc03Ambient struct {
	ctx       *Context
	remaining float64
	active    engine.Task
	last      int
	raiders   bool
}

func newLoc03RaiderAmbient(ctx *Context) engine.Task {
	return &loc03Ambient{ctx: ctx, remaining: loc03RaiderDelay(), last: -1, raiders: true}
}

func newLoc03BirdAmbient(ctx *Context) engine.Task {
	return &loc03Ambient{ctx: ctx, remaining: float64(3 + rand.IntN(4))}
}

func loc03RaiderDelay() float64 {
	return float64(4+rand.IntN(6)) * 0.333
}

func (t *loc03Ambient) Update(dt float64) bool {
	if t.raiders && t.ctx.GetFlag(FlagLoc03RaidersGone) != 0 {
		return true
	}
	if t.active != nil {
		if !t.active.Update(dt) {
			return false
		}
		t.active = nil
		if t.raiders {
			t.remaining = loc03RaiderDelay()
		} else {
			t.remaining = float64(3 + rand.IntN(4))
		}
		return false
	}
	if t.ctx.session.Locked() {
		return false
	}
	t.remaining -= dt
	if t.remaining > 0 {
		return false
	}
	if t.raiders {
		choice := rand.IntN(4)
		for choice == t.last {
			choice = rand.IntN(4)
		}
		t.last = choice
		switch choice {
		case 0:
			t.active = loc03LayerSpeech(t.ctx, "S12_Rab1Talk", "012_RA1_01", 0, 4)
		case 1:
			t.active = loc03LayerSpeech(t.ctx, "S12_Rab2", "012_RA2_01", 0, 4)
		case 2:
			t.active = loc03LayerSpeech(t.ctx, "S12_Rab1Talk", "012_RA1_02", 0, 4)
		case 3:
			t.active = loc03LayerSpeech(t.ctx, "S12_Rab2", "012_RA2_02", 0, 4)
		}
	} else {
		switch rand.IntN(9) {
		case 0:
			t.active = t.ctx.PlaySFX("Sfx_Bird.wav")
		case 1:
			t.active = t.ctx.PlaySFX("Sfx_Bird3.wav")
		case 2, 3:
			t.active = t.ctx.PlaySFX("Sfx_Glider_PassingBy3.wav")
		case 4, 5, 6:
			t.active = t.ctx.PlaySFX("Sfx_Wind2.wav")
		default:
			t.active = t.ctx.PlaySFX("Sfx_Work_Hack2.wav")
		}
	}
	return false
}
