package game

import (
	"math/rand/v2"

	"github.com/wok/rent-a-hero/internal/audio"
	"github.com/wok/rent-a-hero/internal/engine"
)

const loc31Scene = "S113"

type LOC31Controller struct{}

func (LOC31Controller) Enter(ctx *Context, scene string, from string) engine.Task {
	if scene != loc31Scene && scene != "113" {
		return nil
	}

	for _, id := range []string{"S113_To30", "S113_To37", "S113_To57", "S113_To8"} {
		_, _, _ = ctx.ensureAssetLayer(id)
	}

	tasks := []engine.Task{
		engine.Immediate(func() {
			if layer, ok := ctx.layer("S113_BackLoop"); ok {
				layer.Visible = true
				layer.Enabled = true
				layer.Playing = true
				layer.Mode = engine.AnimLoop
				layer.Frame = 0
				layer.Accumulator = 0
				layer.TaskDriven = false
			}
			for _, id := range []string{"S113_To30", "S113_To37", "S113_To57", "S113_To8"} {
				if layer, ok := ctx.layer(id); ok {
					layer.Visible = false
					layer.Playing = false
					layer.Frame = 0
					layer.Accumulator = 0
					layer.TaskDriven = false
				}
			}
		}),
		ctx.EnableArea("S113_To8"),
		ctx.EnableArea("S113_To37"),
		loc31LoopOcean(ctx),
		ctx.RunAmbient(&loc31AmbientTask{ctx: ctx}),
	}

	if getOriginalFlag(ctx.session.state.OriginalState, 0x3f98) != 0 {
		tasks = append(tasks, ctx.EnableArea("S113_To57"))
	} else {
		tasks = append(tasks, ctx.DisableArea("S113_To57"))
	}
	if getOriginalFlag(ctx.session.state.OriginalState, 0x3e88) != 0 {
		tasks = append(tasks, ctx.EnableArea("S113_To30"))
	} else {
		tasks = append(tasks, ctx.DisableArea("S113_To30"))
	}

	return engine.Sequence(tasks...)
}

func (LOC31Controller) Exit(ctx *Context, scene string, to string) engine.Task { return nil }

func (LOC31Controller) Click(ctx *Context, area string) engine.Task {
	switch area {
	case "S113_To8":
		return loc31Depart(ctx, area, 1, "7A")
	case "S113_To37":
		return loc31Depart(ctx, area, 10, "37")
	case "S113_To57":
		if getOriginalFlag(ctx.session.state.OriginalState, 0x3f98) != 0 {
			return loc31Depart(ctx, area, 15, "57")
		}
	case "S113_To30":
		if getOriginalFlag(ctx.session.state.OriginalState, 0x3e88) != 0 {
			return loc31Depart(ctx, area, 28, "30")
		}
	}
	return nil
}

func (LOC31Controller) UseItem(ctx *Context, item int, area string) engine.Task { return nil }
func (LOC31Controller) SelectItem(ctx *Context, item int) engine.Task           { return nil }
func (LOC31Controller) LoadConditionMask(ctx *Context, scene string) int        { return 0 }

func loc31Depart(ctx *Context, layerID string, location int, scene string) engine.Task {
	return engine.Sequence(
		ctx.PlaySFX("Sfx_Door_Opened.wav"),
		&loc31FinishBackLoopTask{ctx: ctx},
		engine.Immediate(func() {
			if l, ok := ctx.layer(layerID); ok {
				l.Visible = true
				l.Enabled = true
				l.Playing = false
				l.Frame = 0
				l.Accumulator = 0
			}
		}),
		ctx.PlaySFX("Sfx_Glider_PassingBy.wav"),
		loc31PlayLayerDeferred(ctx, layerID),
		ctx.ChangeLocation(location, scene),
	)
}

type loc31FinishBackLoopTask struct {
	ctx     *Context
	started bool
}

func (t *loc31FinishBackLoopTask) Update(dt float64) bool {
	back, ok := t.ctx.layer("S113_BackLoop")
	if !ok || back.Source == nil {
		return true
	}
	frames := back.Source.Frames()
	if frames <= 1 {
		return true
	}
	if !t.started {
		t.started = true
		back.Mode = engine.AnimOnce
		back.Playing = true
		back.TaskDriven = true
	}
	back.AdvanceScripted(dt)
	if back.Frame < frames-1 {
		return false
	}
	back.Playing = false
	back.TaskDriven = false
	return true
}

func loc31PlayLayerDeferred(ctx *Context, layerID string) engine.Task {
	var task engine.Task
	return engine.TaskFunc(func(dt float64) bool {
		if task == nil {
			task = ctx.PlayLayer(layerID)
		}
		return task.Update(dt)
	})
}

func loc31LoopOcean(ctx *Context) engine.Task {
	sound, err := engine.LoadSound(ctx.session.idx, "Sfx_Ocean.wav")
	if err != nil {
		return engine.Immediate(func() {})
	}
	return engine.Immediate(func() {
		ctx.session.audioEngine.PlayLooping(sound, audio.CategorySFX)
	})
}

type loc31AmbientTask struct {
	ctx  *Context
	wait float64
}

func (t *loc31AmbientTask) Update(dt float64) bool {
	if t.ctx.session.state.Location != 31 || t.ctx.session.state.Scene != loc31Scene {
		return true
	}
	if t.wait <= 0 {
		t.wait = float64(3 + rand.IntN(3))
	}
	t.wait -= dt
	if t.wait > 0 {
		return false
	}
	if rand.IntN(2) == 0 {
		_ = t.ctx.PlaySFX("Sfx_Seagull.wav").Update(0)
	} else {
		_ = t.ctx.PlaySFX("Sfx_Wind.wav").Update(0)
	}
	t.wait = float64(3 + rand.IntN(3))
	return false
}
