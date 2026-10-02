package game

import (
	"math/rand/v2"
	"strings"

	"github.com/wok/rent-a-hero/internal/engine"
)

const (
	loc12StateWellStage    = 0x3f74
	loc12StateWellSubstage = 0x3f78
	loc12StateForestExit   = 0x3f60
	loc12StateMoneyMissing = 0x3fa0
	loc12StateStoryDone    = 0x3fbc
	loc12StateStoryGate    = 0x409c
)

type LOC12Controller struct{}

func (LOC12Controller) LoadConditionMask(*Context, string) int { return 0 }

func (LOC12Controller) Enter(ctx *Context, scene, from string) engine.Task {
	if loc12SceneID(scene) != "S91" {
		return nil
	}
	actor := loc12ActorID(ctx)
	return engine.Sequence(
		loc12PrepareScene(ctx, from),
		ctx.PlayMusic("Loc10_ForestInDarkness.wav"),
		ctx.RunAmbient(&loc12NightSFXTask{ctx: ctx}),
		engine.Immediate(func() {
			if getOriginalFlag(ctx.session.state.OriginalState, loc12StateStoryGate) == 0 {
				putOriginalFlag(ctx.session.state.OriginalState, loc12StateWellStage, 4)
			}
		}),
		loc12EnterActor(ctx, actor, from),
		ctx.MakeLayerClickable("S91_Brunnen"),
	)
}

func (LOC12Controller) Exit(*Context, string, string) engine.Task { return nil }

func (LOC12Controller) Click(ctx *Context, area string) engine.Task {
	if loc12SceneID(ctx.session.state.Scene) != "S91" {
		return nil
	}
	actor := loc12ActorID(ctx)
	switch strings.ToLower(area) {
	case "s91_door1":
		return engine.Sequence(
			ctx.WalkToFacing(actor, 0xf6, 0x13d, 4),
			ctx.HideActor(actor),
			ctx.ShowLayer("S91_RodRuettel1"),
			ctx.PlayLayer("S91_RodRuettel1"),
			ctx.HideLayer("S91_RodRuettel1"),
			ctx.ShowActor(actor),
			ctx.WalkToFacing(actor, 0xf6, 0x140, 0),
			loc12Rod(ctx, "091_ROD_02"),
		)
	case "s91_door2":
		return engine.Sequence(
			ctx.WalkToFacing(actor, 0x1c6, 300, 6),
			ctx.HideActor(actor),
			ctx.ShowLayer("S91_RodRuettel2"),
			ctx.PlayLayer("S91_RodRuettel2"),
			ctx.HideLayer("S91_RodRuettel2"),
			ctx.ShowActor(actor),
			ctx.WalkToFacing(actor, 0x1c6, 0x131, 0),
			loc12Rod(ctx, "091_ROD_03"),
		)
	case "s91_cyndoor":
		tasks := []engine.Task{ctx.WalkToFacing(actor, 0x161, 0x11f, 4)}
		if loc14SceneID(ctx.session.state.PreviousScene) != "S90" {
			tasks = append(tasks, ctx.HideActor(actor), ctx.ShowLayer("S91_RodReinCam"), ctx.PlayLayer("S91_RodReinCam"), ctx.HideLayer("S91_RodReinCam"), ctx.ShowActor(actor))
		}
		tasks = append(tasks, ctx.ChangeLocation(14, "90"))
		return engine.Sequence(tasks...)
	case "s91_brunnen":
		return loc12Well(ctx, actor)
	case "s91_wald":
		return engine.Sequence(
			ctx.WalkToFacing(actor, 4, 0x150, 2),
			loc12SetOriginal(ctx, loc12StateForestExit, 1),
			ctx.ChangeLocation(10, "42"),
		)
	}
	return nil
}

func (LOC12Controller) UseItem(*Context, int, string) engine.Task { return nil }

func (LOC12Controller) SelectItem(ctx *Context, item int) engine.Task {
	if loc12SceneID(ctx.session.state.Scene) != "S91" || item != 7 {
		return nil
	}
	if getOriginalFlag(ctx.session.state.OriginalState, loc12StateWellStage) != 2 {
		return nil
	}
	return loc12UseMoney(ctx, loc12ActorID(ctx))
}

func loc12SceneID(scene string) string {
	switch strings.ToUpper(strings.TrimSpace(scene)) {
	case "91", "091", "S91", "S091":
		return "S91"
	default:
		return scene
	}
}

func loc12ActorID(ctx *Context) string {
	if actor := ctx.session.PlayerActor(); actor != nil {
		return actor.ID
	}
	if ctx.session.scene != nil && len(ctx.session.scene.CharacterOrder) != 0 {
		return ctx.session.scene.CharacterOrder[0]
	}
	return ""
}

func loc12PrepareScene(ctx *Context, from string) engine.Task {
	return engine.Immediate(func() {
		for _, id := range []string{"S91_RodReinCam", "S91_RodRausCam"} {
			_, _, _ = ctx.ensureAssetLayer(id)
		}
		if loc14SceneID(from) == "S90" {
			_, _, _ = ctx.ensureAssetLayer("S91_Back2")
		}
		for _, id := range []string{"S91_RodReinCam", "S91_RodRausCam", "S91_RodRuettel1", "S91_RodRuettel2", "S91_RodBrunnenHin", "S91_RodBrunnenWeg"} {
			if layer, ok := ctx.layer(id); ok {
				layer.Visible = false
				layer.Playing = false
				layer.TaskDriven = false
				layer.Mode = engine.AnimOnce
				layer.Frame = 0
				layer.Accumulator = 0
			}
		}
		if layer, ok := ctx.layer("S91_Brunnen"); ok {
			layer.Visible = true
			layer.Enabled = true
			layer.Playing = false
			layer.TaskDriven = false
			layer.Frame = 0
			layer.Accumulator = 0
		}
		if layer, ok := ctx.layer("S91_Top"); ok {
			layer.Visible = true
			layer.Enabled = true
		}
		if layer, ok := ctx.layer("S91_Heaven"); ok {
			layer.Visible = true
			layer.Enabled = true
			layer.Playing = true
			layer.TaskDriven = false
			layer.Mode = engine.AnimLoop
			layer.Frame = 0
			layer.Accumulator = 0
		}
	})
}

func loc12EnterActor(ctx *Context, actor, from string) engine.Task {
	if actor == "" {
		return nil
	}
	if loc12SceneID(from) == "S42" || strings.EqualFold(strings.TrimSpace(from), "42") {
		return engine.Sequence(ctx.PlaceActor(actor, 4, 0x150), ctx.WalkToFacing(actor, 0x3c, 0x150, 7))
	}
	return engine.Sequence(
		ctx.HideActor(actor),
		ctx.ShowLayer("S91_RodRausCam"),
		ctx.PlayLayer("S91_RodRausCam"),
		ctx.HideLayer("S91_RodRausCam"),
		ctx.ShowActor(actor),
		ctx.PlaceActor(actor, 0x161, 0x11f),
		ctx.WalkToFacing(actor, 0x159, 0x136, 1),
	)
}

func loc12Well(ctx *Context, actor string) engine.Task {
	stage := getOriginalFlag(ctx.session.state.OriginalState, loc12StateWellStage)
	switch stage {
	case 1:
		return engine.Sequence(
			ctx.WalkToFacing(actor, 0x171, 0x151, 0),
			ctx.HideActor(actor), ctx.HideLayer("S91_Brunnen"),
			ctx.ShowLayer("S91_RodBrunnenHin"), ctx.PlayLayer("S91_RodBrunnenHin"),
			loc12RodDetached(ctx, "091_ROD_04"), loc12RodDetached(ctx, "091_ROD_05"),
			ctx.HideLayer("S91_RodBrunnenHin"),
			ctx.ShowLayer("S91_RodBrunnenWeg"), ctx.PlayLayer("S91_RodBrunnenWeg"), ctx.HideLayer("S91_RodBrunnenWeg"),
			ctx.ShowActor(actor), ctx.ShowLayer("S91_Brunnen"), ctx.WalkToFacing(actor, 0x16d, 0x151, 0),
			loc12Rod(ctx, "091_ROD_06"),
		)
	case 2:
		tasks := []engine.Task{ctx.WalkToFacing(actor, 0x171, 0x151, 0), loc12Rod(ctx, "091_ROD_07")}
		if !ctx.HasItem(7) {
			tasks = append(tasks, loc12SetOriginal(ctx, loc12StateMoneyMissing, 1))
		}
		return engine.Sequence(tasks...)
	case 3:
		return loc12WellStory(ctx, actor)
	case 4:
		return engine.Sequence(
			ctx.WalkToFacing(actor, 0x171, 0x151, 0),
			ctx.HideActor(actor), ctx.HideLayer("S91_Brunnen"),
			ctx.ShowLayer("S91_RodBrunnenHin"), ctx.PlayLayer("S91_RodBrunnenHin"),
			loc12RodDetached(ctx, "091_ROD_04"), ctx.Wait(1.5),
			ctx.HideLayer("S91_RodBrunnenHin"),
			ctx.ShowLayer("S91_RodBrunnenWeg"), ctx.PlayLayer("S91_RodBrunnenWeg"), ctx.HideLayer("S91_RodBrunnenWeg"),
			ctx.ShowActor(actor), ctx.ShowLayer("S91_Brunnen"), ctx.WalkToFacing(actor, 0x16d, 0x151, 0),
			loc12Rod(ctx, "091_ROD_19"),
		)
	}
	return nil
}

func loc12WellStory(ctx *Context, actor string) engine.Task {
	sub := getOriginalFlag(ctx.session.state.OriginalState, loc12StateWellSubstage)
	lines := [][2]string{{"091_ROD_11", "091_ROD_12"}, {"091_ROD_13", "091_ROD_14"}, {"091_ROD_15", "091_ROD_16"}, {"091_ROD_17", "091_ROD_18"}}
	if sub < 1 || sub > 4 {
		return nil
	}
	tasks := []engine.Task{
		ctx.WalkToFacing(actor, 0x171, 0x151, 0),
		ctx.HideActor(actor), ctx.HideLayer("S91_Brunnen"),
		ctx.ShowLayer("S91_RodBrunnenHin"), ctx.PlayLayer("S91_RodBrunnenHin"),
		loc12RodDetached(ctx, lines[sub-1][0]), loc12RodDetached(ctx, lines[sub-1][1]),
		ctx.HideLayer("S91_RodBrunnenHin"),
		ctx.ShowLayer("S91_RodBrunnenWeg"), ctx.PlayLayer("S91_RodBrunnenWeg"), ctx.HideLayer("S91_RodBrunnenWeg"),
	}
	if sub == 4 {
		tasks = append(tasks,
			loc12SetOriginal(ctx, loc12StateWellSubstage, 3),
			loc12SetOriginal(ctx, loc12StateStoryDone, 1),
			ctx.ShowActor(actor), ctx.ShowLayer("S91_Brunnen"), ctx.WalkToFacing(actor, 0x16d, 0x151, 0),
			loc12Rod(ctx, "091_ROD_20"),
		)
		return engine.Sequence(tasks...)
	}
	tasks = append(tasks,
		loc12SetOriginal(ctx, loc12StateWellSubstage, sub+1),
		ctx.ShowActor(actor), ctx.ShowLayer("S91_Brunnen"),
	)
	return engine.Sequence(tasks...)
}

func loc12UseMoney(ctx *Context, actor string) engine.Task {
	return engine.Sequence(
		ctx.WalkToFacing(actor, 0x171, 0x151, 0),
		ctx.HideActor(actor), ctx.HideLayer("S91_Brunnen"),
		ctx.ShowLayer("S91_RodBrunnenHin"), ctx.PlayLayer("S91_RodBrunnenHin"),
		ctx.PlaySFX("Sfx_MoneyBag.wav"), ctx.RemoveItem(7),
		loc12SetOriginal(ctx, loc12StateWellStage, 3),
		ctx.Wait(3), ctx.PlaySFX("Sfx_Bubble.wav"),
		loc12RodDetached(ctx, "091_ROD_08"), loc12RodDetached(ctx, "091_ROD_09"),
		ctx.HideLayer("S91_RodBrunnenHin"),
		ctx.ShowLayer("S91_RodBrunnenWeg"), ctx.PlayLayer("S91_RodBrunnenWeg"), ctx.HideLayer("S91_RodBrunnenWeg"),
		ctx.ShowActor(actor), ctx.ShowLayer("S91_Brunnen"), ctx.WalkToFacing(actor, 0x16d, 0x151, 0),
		loc12Rod(ctx, "091_ROD_10"),
	)
}

func loc12RodDetached(ctx *Context, line string) engine.Task {
	return &loc12DetachedSpeechTask{ctx: ctx, line: line}
}

type loc12DetachedSpeechTask struct {
	ctx     *Context
	line    string
	inner   engine.Task
	started bool
}

func (t *loc12DetachedSpeechTask) Update(dt float64) bool {
	if !t.started {
		t.started = true
		t.inner = t.ctx.PlayVoiceover(t.line, "["+t.line+"]")
	}
	if t.inner == nil {
		return true
	}
	return t.inner.Update(dt)
}

func loc12Rod(ctx *Context, line string) engine.Task {
	actor := loc12ActorID(ctx)
	if actor == "" {
		return ctx.PlayVoiceover(line, "["+line+"]")
	}
	return ctx.Say(actor, line, "["+line+"]")
}

func loc12SetOriginal(ctx *Context, offset, value int) engine.Task {
	return engine.Immediate(func() { putOriginalFlag(ctx.session.state.OriginalState, offset, value) })
}

type loc12NightSFXTask struct {
	ctx       *Context
	remaining float64
}

func (t *loc12NightSFXTask) Update(dt float64) bool {
	t.remaining -= dt
	if t.remaining > 0 {
		return false
	}
	switch rand.IntN(4) + 1 {
	case 1, 2:
		_ = t.ctx.PlaySFX("Sfx_Owl.wav").Update(0)
	case 3:
		_ = t.ctx.PlaySFX("Sfx_Thunder.wav").Update(0)
	case 4:
		_ = t.ctx.PlaySFX("Sfx_Thunder2.wav").Update(0)
	}
	t.remaining = float64(rand.IntN(6) + 5)
	return false
}
