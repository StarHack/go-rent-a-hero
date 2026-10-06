package game

import (
	"fmt"
	"math/rand/v2"

	"github.com/wok/rent-a-hero/internal/engine"
)

const (
	loc17StatePiratesGone = 0x3fc8
	loc17StateBushStage   = 0x3fcc
	loc17StateEarSeen     = 0x3fd0
	loc17StateReturnFlag  = 0x3fd4
	loc17StateBushSeen    = 0x3fd8
	loc17StateBrainStage  = 0x3fdc
	loc17StateBrainUsed   = 0x3fe0
	loc17StateJawsOpened  = 0x3fe4
	loc17StateFirstBrain  = 0x3fe8
)

type LOC17Controller struct{}

func (LOC17Controller) Enter(ctx *Context, scene, from string) engine.Task {
	n := loc15NumericScene(scene)
	if n != 59 && n != 60 {
		return nil
	}
	actor := loc15ActorID(ctx)
	tasks := []engine.Task{loc17Initialize(ctx, n), ctx.PlayMusic("Loc17_UglyAndDisgusting.wav")}
	if getOriginalFlag(ctx.session.state.OriginalState, loc17StateReturnFlag) != 0 {
		tasks = append(tasks, loc15SetOriginal(ctx, loc17StateReturnFlag, 0), loc15SetOriginal(ctx, 0x3f44, 1))
	}
	if n == 59 {
		tasks = append(tasks, loc17Enter59(ctx, actor, from), ctx.RunAmbient(&loc17Ambient{ctx: ctx}))
	} else {
		tasks = append(tasks, loc17Enter60(ctx, actor, from), ctx.RunAmbient(&loc17Ambient{ctx: ctx}))
	}
	return engine.Sequence(tasks...)
}

func loc17Initialize(ctx *Context, scene int) engine.Task {
	return engine.Immediate(func() {
		if scene == 60 {
			for _, id := range []string{"S60_RodRaus", "S60_TryOpenSkullMitBusch", "S60_TryOpenSkullOhneBusch"} {
				layer, _, _ := ctx.ensureAssetLayer(id)
				if layer != nil {
					layer.Presentation = true
					layer.ColorKeyed = false
				}
			}
			loc17AddFrameSFX(ctx, "S60_RodRaus", 0x2c, "Sfx_Knock_Wet.wav", 70)
			loc17AddFrameSFX(ctx, "S60_Spritz", 5, "Sfx_TableHit.wav", 100)
			loc17AddFrameSFX(ctx, "S60_Spritz", 9, "Sfx_OOh.wav", 100)
			loc17AddFrameSFX(ctx, "S60_Spritz", 0x0c, "Sfx_Platzen.wav", 100)
		}
		ids := []string{"Aasfresser1", "Aasfresser2", "Aasfresser3", "Aasfresser4", "Aasfresser5", "Aasfresser6", "Aasfresser7", "S60_BuschWeg", "S60_Busch", "S60_BlattLU", "S60_BlattRO", "S60_Licht", "S60_Spritz", "S60_RodRaus", "S60_TryOpenSkullMitBusch", "S60_TryOpenSkullOhneBusch"}
		for i, id := range ids {
			if l, ok := ctx.layer(id); ok {
				if scene == 59 && i < 7 {
					ctx.session.state.Flags[fmt.Sprintf("loc17.aas.%d.x", i+1)] = l.X
					ctx.session.state.Flags[fmt.Sprintf("loc17.aas.%d.y", i+1)] = l.Y
				}
				l.Visible = false
				l.Playing = false
				l.TaskDriven = false
				l.Frame = 0
				l.Accumulator = 0
			}
		}
	})
}

func loc17AddFrameSFX(ctx *Context, layerID string, frame int, name string, volume int) {
	layer, ok := ctx.layer(layerID)
	if !ok || len(layer.FrameEvents[frame]) != 0 {
		return
	}
	layer.AddFrameEvent(frame, func() {
		if volume == 100 {
			_ = ctx.PlaySFX(name).Update(0)
			return
		}
		_ = ctx.PlaySFXVolume(name, volume).Update(0)
	})
}

func loc17Enter59(ctx *Context, actor, from string) engine.Task {
	tasks := []engine.Task{ctx.EnableArea("S59_To58"), ctx.EnableArea("S59_Skeleton"), ctx.EnableArea("S59_Wall"), ctx.EnableArea("S59_Head")}
	if actor != "" && loc15NumericScene(from) == 58 {
		tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x12d, 0x163), ctx.ShowActor(actor))
		if getOriginalFlag(ctx.session.state.OriginalState, loc17StatePiratesGone) == 0 {
			tasks = append(tasks, loc17ScavengerArrival(ctx))
		}
	} else if actor != "" && loc15NumericScene(from) == 60 {
		tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x177, 0x119), ctx.ShowActor(actor))
	}
	return engine.Sequence(tasks...)
}

func loc17ScavengerArrival(ctx *Context) engine.Task {
	tasks := make([]engine.Task, 0, 24)
	for i := 1; i <= 7; i++ {
		tasks = append(tasks, ctx.ShowLayer(fmt.Sprintf("Aasfresser%d", i)))
	}
	tasks = append(tasks, ctx.PlaySFX("Sfx_Schmatzen.wav"))
	for i := 1; i <= 7; i++ {
		id := fmt.Sprintf("Aasfresser%d", i)
		tasks = append(tasks, ctx.RunAmbient(&loc17ScavengerTask{ctx: ctx, id: id, triggerFrame: 10 + rand.IntN(11)}), ctx.Wait(float64(100+rand.IntN(301))/1000.0))
	}
	tasks = append(tasks, ctx.Wait(1))
	return engine.Sequence(tasks...)
}

type loc17ScavengerTask struct {
	ctx          *Context
	id           string
	stage        int
	triggerFrame int
	advanceCount int
	accumulator  float64
	moveSteps    int
	moveStep     int
	startX       int
}

func (t *loc17ScavengerTask) Update(dt float64) bool {
	if t.ctx.session.state.Location != 17 || loc15NumericScene(t.ctx.session.state.Scene) != 59 {
		return true
	}
	layer, ok := t.ctx.layer(t.id)
	if !ok || layer.Source == nil {
		return true
	}
	if t.stage == 0 {
		layer.Visible = true
		layer.Enabled = true
		layer.Playing = false
		layer.TaskDriven = true
		layer.Frame = 0
		layer.Accumulator = 0
		t.stage = 1
	}
	switch t.stage {
	case 1:
		t.advanceRange(layer, dt, 0, 2, true, func() {
			t.advanceCount++
			if t.advanceCount >= t.triggerFrame {
				t.stage = 2
				layer.Frame = 2
				t.accumulator = 0
			}
		})
	case 2:
		t.advanceRange(layer, dt, 2, 5, false, func() {
			if layer.Frame >= 5 {
				t.stage = 3
				t.moveSteps = 20 + rand.IntN(11)
				t.startX = layer.X
				layer.Frame = 6
				t.accumulator = 0
			}
		})
	case 3:
		advanced := t.advanceRange(layer, dt, 6, 17, true, nil)
		for i := 0; i < advanced && t.moveStep < t.moveSteps; i++ {
			t.moveStep++
			layer.X = t.startX + (485-t.startX)*t.moveStep/t.moveSteps
		}
		if t.moveStep >= t.moveSteps {
			layer.X = 485
			layer.Z = 20
			layer.Visible = false
			layer.Enabled = false
			layer.TaskDriven = false
			layer.Playing = false
			t.stage = 4
		}
	case 4:
		return true
	}
	return false
}

func (t *loc17ScavengerTask) advanceRange(layer *engine.Layer, dt float64, from, to int, loop bool, onAdvance func()) int {
	fps := layer.FPS
	if fps <= 0 {
		return 0
	}
	t.accumulator += dt
	step := 1.0 / float64(fps)
	advanced := 0
	for t.accumulator >= step {
		t.accumulator -= step
		if layer.Frame < from || layer.Frame > to {
			layer.Frame = from
		}
		if layer.Frame < to {
			layer.Frame++
		} else if loop {
			layer.Frame = from
		}
		advanced++
		if onAdvance != nil {
			onAdvance()
			if t.stage == 2 && layer.Frame == 2 {
				return advanced
			}
			if t.stage == 3 && layer.Frame == 6 {
				return advanced
			}
		}
		if !loop && layer.Frame >= to {
			return advanced
		}
	}
	return advanced
}

type loc17ScavengerReturnTask struct {
	ctx         *Context
	id          string
	targetX     int
	step        int
	accumulator float64
}

func (t *loc17ScavengerReturnTask) Update(dt float64) bool {
	layer, ok := t.ctx.layer(t.id)
	if !ok || layer.Source == nil {
		return true
	}
	if t.step == 0 {
		layer.X = 490
		layer.Visible = true
		layer.Enabled = true
		layer.TaskDriven = true
		layer.Playing = false
		layer.Frame = 18
	}
	fps := layer.FPS
	if fps <= 0 {
		fps = 10
	}
	t.accumulator += dt
	frameStep := 1.0 / float64(fps)
	for t.accumulator >= frameStep && t.step < 30 {
		t.accumulator -= frameStep
		t.step++
		layer.Frame++
		if layer.Frame > 29 {
			layer.Frame = 18
		}
		layer.X = 490 + (t.targetX-490)*t.step/30
	}
	return t.step >= 30
}

func loc17ScavengerDeparture(ctx *Context) engine.Task {
	tasks := []engine.Task{ctx.PlaySFX("Sfx_Schmatzen.wav")}
	returns := make([]engine.Task, 0, 7)
	for i := 1; i <= 7; i++ {
		returns = append(returns, &loc17ScavengerReturnTask{
			ctx:     ctx,
			id:      fmt.Sprintf("Aasfresser%d", i),
			targetX: ctx.session.state.Flags[fmt.Sprintf("loc17.aas.%d.x", i)],
		})
	}
	tasks = append(tasks, engine.Parallel(returns...), ctx.Wait(1.5))
	return engine.Sequence(tasks...)
}

func loc17Enter60(ctx *Context, actor, from string) engine.Task {
	state := ctx.session.state.OriginalState
	bushStage := getOriginalFlag(state, loc17StateBushStage)
	jawsOpened := getOriginalFlag(state, loc17StateJawsOpened) != 0
	piratesGone := getOriginalFlag(state, loc17StatePiratesGone) != 0
	tasks := []engine.Task{ctx.EnableArea("S60_To59"), ctx.EnableArea("S60_Skull"), ctx.EnableArea("S60_Mouth")}
	for _, id := range []string{"S60_BlattLU", "S60_BlattRO", "S60_Licht"} {
		tasks = append(tasks, ctx.ShowLayer(id))
	}
	if bushStage > 3 {
		tasks = append(tasks, engine.Immediate(func() {
			if l, ok := ctx.layer("S60_Busch"); ok {
				l.X = 0x26c
				l.Y = 0x66
			}
		}))
	}
	tasks = append(tasks, ctx.ShowLayer("S60_Busch"))
	if !jawsOpened && bushStage <= 3 {
		tasks = append(tasks, ctx.MakeLayerClickable("S60_Busch"), ctx.DisableArea("S60_Ear"))
	} else {
		tasks = append(tasks, ctx.DisableArea("S60_Busch"), ctx.EnableArea("S60_Ear"))
	}
	if piratesGone {
		tasks = append(tasks, engine.Immediate(func() {
			if l, ok := ctx.layer("S60_Spritz"); ok {
				l.Frame = 0x1c
				l.Visible = true
				l.Enabled = true
				l.Playing = false
				l.TaskDriven = false
			}
		}))
	}
	if actor == "" {
		return engine.Sequence(tasks...)
	}
	if loc15NumericScene(from) == 59 {
		tasks = append(tasks, ctx.PlaceActorPerspective(actor, 8, 0x128), ctx.WalkToFacingPerspective(actor, 0x5f, 0x139, 7))
		return engine.Sequence(tasks...)
	}
	if jawsOpened {
		tasks = append(tasks, loc17SkullTrapSequence(ctx, actor))
		return engine.Sequence(tasks...)
	}
	tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x270, 0xf4), ctx.ShowActor(actor))
	return engine.Sequence(tasks...)
}

type loc17SkullTrapTask struct {
	ctx         *Context
	id          string
	startY      int
	steps       int
	step        int
	accumulator float64
	started     bool
}

func (t *loc17SkullTrapTask) Update(dt float64) bool {
	layer, ok := t.ctx.layer(t.id)
	if !ok || layer.Source == nil {
		return true
	}
	if !t.started {
		t.started = true
		layer.X = -92
		layer.Y = t.startY
		layer.Visible = true
		layer.Enabled = true
		layer.TaskDriven = true
		layer.Playing = false
		layer.Frame = 6
		layer.Accumulator = 0
	}
	fps := layer.FPS
	if fps <= 0 {
		fps = 10
	}
	t.accumulator += dt
	frameStep := 1.0 / float64(fps)
	for t.accumulator >= frameStep && t.step < t.steps {
		t.accumulator -= frameStep
		t.step++
		layer.Frame++
		if layer.Frame > 17 {
			layer.Frame = 6
		}
		layer.X = -92 + (350+92)*t.step/t.steps
		layer.Y = t.startY + (256-t.startY)*t.step/t.steps
	}
	if t.step < t.steps {
		return false
	}
	layer.X = 350
	layer.Y = 256
	layer.Visible = false
	layer.Enabled = false
	layer.TaskDriven = false
	layer.Playing = false
	return true
}

func loc17SkullTrapSequence(ctx *Context, actor string) engine.Task {
	tasks := []engine.Task{
		ctx.HideActor(actor),
		ctx.ShowLayer("S60_RodRaus"),
		ctx.PlayLayer("S60_RodRaus"),
		ctx.HideLayer("S60_RodRaus"),
		ctx.ShowActor(actor),
		ctx.ShowLayer("S60_Spritz"),
		ctx.ShowLayer("S60_BlattLU"),
		ctx.ShowLayer("S60_BlattRO"),
		ctx.ShowLayer("S60_Licht"),
		ctx.ShowLayer("S60_Busch"),
		ctx.PlaceActorPerspective(actor, 0x140, 0x14e),
		ctx.WalkToFacingPerspective(actor, 0x140, 0x154, 4),
		ctx.PlaySFX("Sfx_Schmatzen.wav"),
	}
	for i := 1; i <= 7; i++ {
		tasks = append(tasks,
			ctx.RunAmbient(&loc17SkullTrapTask{
				ctx:    ctx,
				id:     fmt.Sprintf("Aasfresser%d", i),
				startY: 200 + rand.IntN(31),
				steps:  20 + rand.IntN(11),
			}),
			ctx.Wait(float64(100+rand.IntN(301))/1000.0),
		)
	}
	tasks = append(tasks,
		ctx.Wait(3),
		ctx.PlayLayer("S60_Spritz"),
		loc15SetOriginal(ctx, loc17StatePiratesGone, 1),
		loc15SetOriginal(ctx, loc16StatePirateBurn, 1),
		ctx.WalkToFacingPerspective(actor, 0x140, 0x15e, 0),
		ctx.PlayVoiceover("060_ROD_01", ""),
	)
	return engine.Sequence(tasks...)
}

func (LOC17Controller) Exit(*Context, string, string) engine.Task { return nil }

func (LOC17Controller) Click(ctx *Context, area string) engine.Task {
	actor := loc15ActorID(ctx)
	switch area {
	case "S59_To58":
		tasks := []engine.Task{ctx.WalkToFacingPerspective(actor, 0x12d, 0x163, 0)}
		if getOriginalFlag(ctx.session.state.OriginalState, loc17StatePiratesGone) == 0 {
			tasks = append(tasks, loc17ScavengerDeparture(ctx))
		}
		tasks = append(tasks, ctx.ChangeLocation(16, "58"))
		return engine.Sequence(tasks...)
	case "S59_Skeleton":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x68, 0x14c, 4), ctx.PlayVoiceover(fmt.Sprintf("059_ROD_%02d", 1+rand.IntN(6)), ""))
	case "S59_Wall":
		line := "059_ROD_07"
		if getOriginalFlag(ctx.session.state.OriginalState, loc17StatePiratesGone) != 0 {
			line = "059_ROD_08"
		}
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x17f, 0x150, 6), ctx.PlayVoiceover(line, ""))
	case "S59_Head":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x177, 0x118, 4), ctx.ChangeScene("60"))
	case "S60_To59":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 8, 0x128, 2), ctx.ChangeScene("59"))
	case "S60_Skull":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 399, 0x15e, 4), ctx.PlayVoiceover("060_ROD_02", ""))
	case "S60_Mouth":
		if getOriginalFlag(ctx.session.state.OriginalState, loc17StatePiratesGone) != 0 {
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x150, 0x155, 4), ctx.PlayVoiceover("060_ROD_09", ""))
		}
		layer := "S60_TryOpenSkullMitBusch"
		if getOriginalFlag(ctx.session.state.OriginalState, loc17StateBushStage) >= 4 {
			layer = "S60_TryOpenSkullOhneBusch"
		}
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x150, 0x155, 4), ctx.HideActor(actor), ctx.ShowLayer(layer), ctx.PlayLayer(layer), ctx.HideLayer(layer), ctx.ShowActor(actor), ctx.WalkToFacingPerspective(actor, 0x154, 0x155, 0), ctx.PlayVoiceover("060_ROD_03", ""))
	case "S60_Ear":
		tasks := []engine.Task{ctx.WalkToFacingPerspective(actor, 0x270, 0xf4, 3)}
		if getOriginalFlag(ctx.session.state.OriginalState, loc17StatePiratesGone) != 0 {
			return engine.Sequence(append(tasks, ctx.PlayVoiceover("060_ROD_09", ""))...)
		}
		if getOriginalFlag(ctx.session.state.OriginalState, loc17StateEarSeen) == 0 {
			tasks = append(tasks, ctx.PlayVoiceover("060_ROD_08", ""), loc15SetOriginal(ctx, loc17StateEarSeen, 1))
		}
		tasks = append(tasks, ctx.ChangeLocation(18, "61"))
		return engine.Sequence(tasks...)
	case "S60_Busch":
		return loc17BushClick(ctx, actor)
	}
	return nil
}

func loc17BushClick(ctx *Context, actor string) engine.Task {
	stage := getOriginalFlag(ctx.session.state.OriginalState, loc17StateBushStage)
	base := []engine.Task{ctx.WalkToFacingPerspective(actor, 0x270, 0xf4, 3)}
	switch stage {
	case 1:
		base = append(base, ctx.PlayVoiceover("060_ROD_04", ""), loc15SetOriginal(ctx, loc17StateBushStage, 2))
	case 2:
		base = append(base, ctx.PlayVoiceover("060_ROD_05", ""), engine.Immediate(func() {
			if getOriginalFlag(ctx.session.state.OriginalState, loc17StateBushSeen) == 2 {
				putOriginalFlag(ctx.session.state.OriginalState, loc17StateBushStage, 3)
			}
			putOriginalFlag(ctx.session.state.OriginalState, loc17StateBushSeen, 2)
		}))
	case 3:
		base = append(base, ctx.PlayVoiceover("060_ROD_06", ""), ctx.ShowLayer("S60_BuschWeg"), ctx.PlayLayerFrames("S60_BuschWeg", 0, 9), loc15TweenLayerPosition(ctx, "S60_Busch", 0, 0, 0x26c, 0x66, 10), ctx.PlayLayerFrames("S60_BuschWeg", 10, -1), ctx.HideLayer("S60_BuschWeg"), ctx.PlayVoiceover("060_ROD_07", ""), loc15SetOriginal(ctx, loc17StateBushStage, 4), ctx.DisableArea("S60_Busch"), ctx.EnableArea("S60_Ear"))
	}
	return engine.Sequence(base...)
}

func (LOC17Controller) UseItem(*Context, int, string) engine.Task { return nil }
func (LOC17Controller) SelectItem(*Context, int) engine.Task      { return nil }
func (LOC17Controller) LoadConditionMask(*Context, string) int    { return 0 }

type loc17Ambient struct {
	ctx  *Context
	wait float64
}

func (t *loc17Ambient) Update(dt float64) bool {
	if t.ctx.session.state.Location != 17 {
		return true
	}
	if t.wait <= 0 {
		t.wait = float64(5 + rand.IntN(11))
	}
	t.wait -= dt
	if t.wait > 0 {
		return false
	}
	sounds := []string{"Sfx_Bird.wav", "Sfx_Bird2.wav", "Sfx_Bird3.wav", "Sfx_Chirp.wav"}
	_ = t.ctx.PlaySFX(sounds[rand.IntN(len(sounds))]).Update(0)
	t.wait = float64(5 + rand.IntN(6))
	return false
}
