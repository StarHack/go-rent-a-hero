package game

import (
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/wok/rent-a-hero/internal/audio"
	"github.com/wok/rent-a-hero/internal/engine"
)

const (
	loc26StateTo98Unlocked = 0x4098
	loc26StateFirstArrival = 0x409c
	loc26StateS98Flag      = 0x40a0
	loc26StateS98FlagStage = 0x40a4
	loc26StateS98Grube     = 0x4094
	loc26StateS98FlagReady = 0x3f18
)

type LOC26Controller struct{}

func loc26SceneID(scene string) int {
	s := strings.ToUpper(strings.TrimSpace(scene))
	s = strings.TrimPrefix(s, "S")
	n, _ := strconv.Atoi(s)
	return n
}

func loc26Actor(ctx *Context) string {
	if actor := ctx.session.PlayerActor(); actor != nil {
		return actor.ID
	}
	return "RodrigoSmall"
}

func loc26SetOriginal(ctx *Context, offset, value int) engine.Task {
	return engine.Immediate(func() {
		putOriginalFlag(ctx.session.state.OriginalState, offset, value)
	})
}

func loc26Rod(ctx *Context, line string) engine.Task {
	return ctx.Say(loc26Actor(ctx), line, "["+line+"]")
}

func loc26Dragon(ctx *Context, line string, start, end int) engine.Task {
	return ctx.PlaySpeechBoundToLayer("S93_DragonBreath", line, "["+line+"]", start, end)
}

func loc26LoopLayer(ctx *Context, id string) engine.Task {
	return engine.Immediate(func() {
		if layer, ok := ctx.layer(id); ok {
			layer.Visible = true
			layer.Enabled = true
			layer.Mode = engine.AnimLoop
			layer.Playing = true
			layer.TaskDriven = false
		}
	})
}

func loc26ShowDragon(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.FreezeLayer("S93_DragonBreath", 10),
		ctx.ShowLayer("S93_DragonBreath"),
	)
}

func loc26Prepare93(ctx *Context) engine.Task {
	return engine.Sequence(
		engine.Immediate(func() {
			if layer, ok := ctx.layer("S93_DragonTakeoff"); ok {
				if len(layer.FrameEvents[0x42]) == 0 {
					layer.AddFrameEvent(0x42, func() { _ = ctx.PlaySFXVolume("Sfx_DragonWings.wav", 70).Update(0) })
				}
				if len(layer.FrameEvents[0x57]) == 0 {
					layer.AddFrameEvent(0x57, func() { _ = ctx.PlaySFXVolume("Sfx_DragonWings.wav", 100).Update(0) })
				}
			}
		}),
		ctx.HideLayer("S93_DragonLanding"),
		ctx.HideLayer("S93_DragonTakeoff"),
		ctx.HideLayer("S93_DragonBreath"),
		loc26LoopLayer(ctx, "S93_MeerLO"),
		loc26LoopLayer(ctx, "S93_MeerLU"),
		loc26LoopLayer(ctx, "S93_MeerRO"),
	)
}

func loc26LoopOcean(ctx *Context) engine.Task {
	sound, err := engine.LoadSound(ctx.session.idx, "Sfx_Ocean.wav")
	if err != nil {
		return engine.Immediate(func() {})
	}
	return engine.Immediate(func() {
		ctx.session.audioEngine.PlayLooping(sound, audio.CategorySFX)
	})
}

func (LOC26Controller) Enter(ctx *Context, scene string, from string) engine.Task {
	switch loc26SceneID(scene) {
	case 93:
		return loc26Enter93(ctx, from)
	case 98:
		return loc26Enter98(ctx, from)
	}
	return nil
}

func loc26Enter93(ctx *Context, from string) engine.Task {
	actor := loc26Actor(ctx)
	tasks := []engine.Task{
		ctx.PlayMusic("Loc26_IslandOfMegophias.wav"),
		loc26LoopOcean(ctx),
		ctx.HideActor(actor),
		loc26Prepare93(ctx),
		ctx.EnableArea("S93_Huette"),
		ctx.EnableArea("S93_Meer"),
		ctx.EnableArea("S93_Dragon"),
	}
	if getOriginalFlag(ctx.session.state.OriginalState, loc26StateTo98Unlocked) == 0 {
		tasks = append(tasks, ctx.EnableArea("S93_Wald"), ctx.DisableArea("S93_To98"))
	} else {
		tasks = append(tasks, ctx.DisableArea("S93_Wald"), ctx.EnableArea("S93_To98"))
	}
	if actor != "" {
		switch loc26SceneID(from) {
		case 58:
			tasks = append(tasks,
				ctx.ShowLayer("S93_DragonLanding"),
				ctx.PlayLayerFrames("S93_DragonLanding", 1, -1),
				ctx.HideLayer("S93_DragonLanding"),
				loc26ShowDragon(ctx),
				ctx.ShowActor(actor),
				ctx.PlaceActorPerspective(actor, 0x229, 0x15e),
				ctx.WalkToFacingPerspective(actor, 0x200, 0x163, 2),
			)
		case 95:
			tasks = append(tasks,
				loc26ShowDragon(ctx),
				ctx.ShowActor(actor),
				ctx.PlaceActorPerspective(actor, 0x4f, 0xe7),
				ctx.WalkToFacingPerspective(actor, 0x5d, 0x104, 7),
			)
		case 98:
			tasks = append(tasks,
				loc26ShowDragon(ctx),
				ctx.ShowActor(actor),
				ctx.PlaceActorPerspective(actor, 0x182, 0xa0),
				ctx.WalkToFacingPerspective(actor, 0x184, 0xaa, 0),
			)
		default:
			tasks = append(tasks, loc26ShowDragon(ctx), ctx.ShowActor(actor))
		}
	}
	if getOriginalFlag(ctx.session.state.OriginalState, loc26StateFirstArrival) != 0 {
		tasks = append(tasks,
			loc26SetOriginal(ctx, loc26StateFirstArrival, 0),
			loc26Rod(ctx, "093_ROD_01"),
			loc26Dragon(ctx, "093_DRA_01", 10, 10),
		)
	}
	tasks = append(tasks,
		ctx.RunAmbient(&loc26IslandAmbient{ctx: ctx, wait: float64(5 + rand.IntN(6))}),
		ctx.RunAmbient(&loc26DragonAmbient{ctx: ctx, wait: float64(10 + rand.IntN(11))}),
	)
	return engine.Sequence(tasks...)
}

func loc26Prepare98(ctx *Context) engine.Task {
	return engine.Sequence(
		engine.Immediate(func() {
			layer, _, err := ctx.ensureAssetLayer("Gen_RodPickUp")
			if err == nil && layer != nil {
				layer.Visible = false
				layer.Playing = false
				layer.TaskDriven = false
				layer.Frame = 0
				layer.Accumulator = 0
			}
		}),
		loc26LoopLayer(ctx, "S98_MeerL"),
		loc26LoopLayer(ctx, "S98_MeerR"),
	)
}

func loc26Enter98(ctx *Context, from string) engine.Task {
	actor := loc26Actor(ctx)
	tasks := []engine.Task{
		ctx.PlayMusic("Loc26_IslandOfMegophias.wav"),
		loc26LoopOcean(ctx),
		loc26Prepare98(ctx),
		ctx.EnableArea("S98_To93"),
		ctx.EnableArea("S98_Grube"),
	}
	if getOriginalFlag(ctx.session.state.OriginalState, loc26StateS98Flag) != 0 &&
		getOriginalFlag(ctx.session.state.OriginalState, loc26StateS98FlagReady) != 0 {
		tasks = append(tasks,
			ctx.ShowLayer("S98_Fahne"),
			ctx.MakeLayerClickable("S98_Fahne"),
		)
	} else {
		tasks = append(tasks, ctx.HideLayer("S98_Fahne"), ctx.DisableArea("S98_Fahne"))
	}
	if loc26SceneID(from) == 93 {
		tasks = append(tasks,
			ctx.ShowActor(actor),
			ctx.PlaceActorPerspective(actor, 0x181, 0x9c),
			ctx.WalkToFacingPerspective(actor, 0x184, 0xaa, 0),
		)
	} else {
		tasks = append(tasks, ctx.ShowActor(actor))
	}
	tasks = append(tasks, ctx.RunAmbient(&loc26IslandAmbient{ctx: ctx, wait: float64(5 + rand.IntN(6))}))
	return engine.Sequence(tasks...)
}

func (LOC26Controller) Exit(*Context, string, string) engine.Task { return nil }

func (LOC26Controller) Click(ctx *Context, area string) engine.Task {
	actor := loc26Actor(ctx)
	switch loc26SceneID(ctx.session.state.Scene) {
	case 93:
		return loc26Click93(ctx, actor, area)
	case 98:
		return loc26Click98(ctx, actor, area)
	}
	return nil
}

func loc26Click93(ctx *Context, actor, area string) engine.Task {
	switch area {
	case "S93_Huette":
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0x4f, 0xe7, 5),
			ctx.ChangeLocation(27, "95"),
		)
	case "S93_Meer":
		return loc26Rod(ctx, "093_ROD_03")
	case "S93_Wald":
		if getOriginalFlag(ctx.session.state.OriginalState, loc26StateTo98Unlocked) != 0 {
			return nil
		}
		return loc26Rod(ctx, "093_ROD_04")
	case "S93_To98":
		if getOriginalFlag(ctx.session.state.OriginalState, loc26StateTo98Unlocked) == 0 {
			return nil
		}
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0x182, 0xa0, 4),
			ctx.ChangeLocation(26, "98"),
		)
	case "S93_Dragon":
		return loc26DragonClick(ctx, actor)
	}
	return nil
}

func loc26Click98(ctx *Context, actor, area string) engine.Task {
	switch area {
	case "S98_To93":
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0x181, 0x9c, 4),
			ctx.ChangeLocation(26, "93"),
		)
	case "S98_Grube":
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0xa8, 0x12e, 5),
			loc26Rod(ctx, "098_ROD_01b"),
			loc26SetOriginal(ctx, loc26StateS98Grube, 1),
		)
	case "S98_Fahne":
		if getOriginalFlag(ctx.session.state.OriginalState, loc26StateS98Flag) == 0 ||
			getOriginalFlag(ctx.session.state.OriginalState, loc26StateS98FlagReady) == 0 {
			return nil
		}
		switch getOriginalFlag(ctx.session.state.OriginalState, loc26StateS98FlagStage) {
		case 1:
			return engine.Sequence(
				loc26SetOriginal(ctx, loc26StateS98FlagStage, 2),
				ctx.WalkToFacingPerspective(actor, 0x56, 0x100, 4),
				loc26Rod(ctx, "098_ROD_01"),
			)
		case 2:
			return engine.Sequence(
				ctx.WalkToFacingPerspective(actor, 0x56, 0x100, 4),
				ctx.HideActor(actor),
				ctx.ShowLayer("Gen_RodPickUp"),
				ctx.PlayLayerFrames("Gen_RodPickUp", 0, 7),
				ctx.HideLayer("S98_Fahne"),
				ctx.DisableArea("S98_Fahne"),
				ctx.AddItem(2),
				loc26SetOriginal(ctx, loc26StateS98Flag, 0),
				ctx.PlayLayerFrames("Gen_RodPickUp", 8, -1),
				ctx.HideLayer("Gen_RodPickUp"),
				ctx.ShowActor(actor),
			)
		}
	}
	return nil
}

func loc26DragonClick(ctx *Context, actor string) engine.Task {
	return &loc26DragonClickTask{ctx: ctx, actor: actor}
}

type loc26DragonClickTask struct {
	ctx     *Context
	actor   string
	inner   engine.Task
	started bool
}

func (t *loc26DragonClickTask) Update(dt float64) bool {
	if t.inner != nil {
		return t.inner.Update(dt)
	}
	if t.started {
		return true
	}
	t.started = true
	a, ok := t.ctx.session.scene.Characters[t.actor]
	if !ok {
		return true
	}
	dx := a.X - 0x22a
	if dx < 0 {
		dx = -dx
	}
	dy := a.Y - 0x15d
	if dy < 0 {
		dy = -dy
	}
	if dx < 10 && dy < 10 {
		t.inner = engine.Sequence(
			t.ctx.WalkToFacingPerspective(t.actor, 0x229, 0x15e, 3),
			t.ctx.HideLayer("S93_DragonBreath"),
			t.ctx.HideActor(t.actor),
			t.ctx.ShowLayer("S93_DragonTakeoff"),
			t.ctx.PlayLayerFrames("S93_DragonTakeoff", 1, -1),
			t.ctx.ChangeLocation(16, "58"),
		)
	} else {
		t.inner = engine.Sequence(
			t.ctx.WalkToFacingPerspective(t.actor, 0x229, 0x15e, 3),
			loc26Rod(t.ctx, "093_ROD_02"),
			loc26Dragon(t.ctx, "093_DRA_02", 10, 10),
		)
	}
	return t.inner.Update(dt)
}

func (LOC26Controller) UseItem(*Context, int, string) engine.Task { return nil }
func (LOC26Controller) SelectItem(ctx *Context, item int) engine.Task {
	if loc26SceneID(ctx.session.state.Scene) == 98 && (item == 9 || item == 0xb) {
		return loc26Rod(ctx, "098_ROD_02")
	}
	return nil
}
func (LOC26Controller) LoadConditionMask(*Context, string) int { return 0 }

type loc26IslandAmbient struct {
	ctx  *Context
	wait float64
}

func (t *loc26IslandAmbient) Update(dt float64) bool {
	scene := loc26SceneID(t.ctx.session.state.Scene)
	if t.ctx.session.state.Location != 26 || (scene != 93 && scene != 98) {
		return true
	}
	t.wait -= dt
	if t.wait > 0 {
		return false
	}
	switch rand.IntN(3) {
	case 0:
		_ = t.ctx.PlaySFXVolume("Sfx_Seagull.wav", 50+rand.IntN(51)).Update(0)
	case 1:
		_ = t.ctx.PlaySFX("Sfx_Wind.wav").Update(0)
	case 2:
		_ = t.ctx.PlaySFX("Sfx_Wind2.wav").Update(0)
	}
	t.wait = float64(5 + rand.IntN(6))
	return false
}

type loc26DragonAmbient struct {
	ctx   *Context
	wait  float64
	inner engine.Task
}

func (t *loc26DragonAmbient) Update(dt float64) bool {
	if t.ctx.session.state.Location != 26 || loc26SceneID(t.ctx.session.state.Scene) != 93 {
		return true
	}
	if t.inner != nil {
		t.wait -= dt
		if t.inner.Update(dt) {
			t.inner = nil
		}
		return false
	}
	t.wait -= dt
	if t.wait > 0 {
		return false
	}
	if layer, ok := t.ctx.layer("S93_DragonBreath"); ok && !layer.TaskDriven {
		t.wait = float64(10 + rand.IntN(11))
		t.inner = engine.Sequence(
			t.ctx.PlayLayerFrames("S93_DragonBreath", 1, 10),
			t.ctx.FreezeLayer("S93_DragonBreath", 10),
		)
		return t.inner.Update(dt)
	}
	t.wait = float64(10 + rand.IntN(11))
	return false
}
