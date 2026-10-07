package game

import (
	"math/rand/v2"
	"strings"

	"github.com/wok/rent-a-hero/internal/engine"
)

const (
	loc24StateReturnTo102 = 0x4088
	loc24StateFightSpeed  = 0x408c
	loc24FightFlag        = "Loc24FightState"
)

type LOC24Controller struct{}

func loc24SceneID(scene string) string {
	s := strings.ToUpper(strings.TrimSpace(scene))
	s = strings.TrimPrefix(s, "S")
	if s == "103" || s == "0103" {
		return "S103"
	}
	return strings.ToUpper(strings.TrimSpace(scene))
}

func loc24Original(ctx *Context, off int) int {
	return getOriginalFlag(ctx.session.state.OriginalState, off)
}

func loc24SetOriginal(ctx *Context, off, value int) engine.Task {
	return engine.Immediate(func() { putOriginalFlag(ctx.session.state.OriginalState, off, value) })
}

func loc24EnsureLayers(ctx *Context) {
	ids := []string{
		"S103_Tisch", "S103_CapComes", "S103_CapAttack1", "S103_CapAttack2", "S103_CapAttack3",
		"S103_CapCameTalk", "S103_CapDecktUnten", "S103_RodTakesHelm", "S103_Helm", "S103_SabAttack2",
		"S103_SabDeckt1", "S103_SabDeckt2", "S103_SabDrawsSword", "S103_Flasche", "S103_RodPickUp",
		"SabWalk", "Gen_SmallSteam",
	}
	for _, id := range ids {
		_, _, _ = ctx.ensureAssetLayer(id)
		if layer, ok := ctx.layer(id); ok {
			layer.Visible = false
			layer.Playing = false
			layer.TaskDriven = false
			layer.Accumulator = 0
		}
	}
}

func loc24BindFrameSFX(ctx *Context) {
	events := map[string]map[int]string{
		"S103_CapAttack1":    {0x18: "Sfx_Step_Cyber.wav", 0x14: "Sfx_Sword1.wav"},
		"S103_CapAttack2":    {7: "Sfx_Step_Cyber.wav", 10: "Sfx_Sword2.wav"},
		"S103_CapAttack3":    {2: "Sfx_Woosh.wav", 7: "Sfx_Sword1.wav", 10: "Gen_StepRight.wav", 0x0f: "Sfx_PianoStroke.wav"},
		"S103_SabAttack2":    {5: "Sfx_Sword2.wav"},
		"S103_SabDrawsSword": {5: "Sfx_SwordPull.wav"},
		"SabWalk":            {0x54: "Gen_StepLeft.wav", 0x5a: "Gen_StepRight.wav", 0x0c: "Gen_StepLeft.wav", 0x12: "Gen_StepRight.wav"},
		"S103_CapComes":      {1: "Sfx_Work_Fix.wav", 0: "Sfx_DangerStrings.wav", 3: "Sfx_Step_Cyber.wav", 8: "Gen_StepRight.wav", 0x0e: "Sfx_Step_Cyber.wav", 0x14: "Gen_StepRight.wav"},
		"S103_CapDecktUnten": {7: "Sfx_Step_Cyber.wav"},
	}
	for id, frames := range events {
		layer, ok := ctx.layer(id)
		if !ok || layer == nil {
			continue
		}
		for frame, sound := range frames {
			if len(layer.FrameEvents[frame]) != 0 {
				continue
			}
			sound := sound
			layer.AddFrameEvent(frame, func() { _ = ctx.PlaySFX(sound).Update(0) })
		}
	}
}

func loc24MakeClickables(ctx *Context) engine.Task {
	tasks := []engine.Task{
		ctx.MakeLayerClickable("S103_Tisch"),
		ctx.MakeLayerClickable("S103_CapAttack1"),
		ctx.MakeLayerClickable("S103_CapAttack2"),
		ctx.MakeLayerClickable("S103_CapAttack3"),
		ctx.MakeLayerClickable("S103_CapDecktUnten"),
		ctx.MakeLayerClickable("S103_SabAttack2"),
		ctx.MakeLayerClickable("S103_SabDeckt1"),
		ctx.MakeLayerClickable("S103_SabDeckt2"),
	}
	if !ctx.HasItem(0x18) {
		tasks = append(tasks, ctx.MakeLayerClickable("S103_Flasche"))
	}
	if !ctx.HasItem(5) {
		tasks = append(tasks, ctx.MakeLayerClickable("S103_Helm"))
	}
	return engine.Sequence(tasks...)
}

func loc24HideFight(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.HideLayer("S103_SabAttack2"),
		ctx.HideLayer("S103_SabDeckt1"),
		ctx.HideLayer("S103_SabDeckt2"),
		ctx.HideLayer("S103_SabDrawsSword"),
		ctx.HideLayer("SabWalk"),
	)
}

func loc24HideCaptain(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.HideLayer("S103_CapComes"),
		ctx.HideLayer("S103_CapAttack1"),
		ctx.HideLayer("S103_CapAttack2"),
		ctx.HideLayer("S103_CapAttack3"),
		ctx.HideLayer("S103_CapCameTalk"),
		ctx.HideLayer("S103_CapDecktUnten"),
	)
}

func loc24SetFightState(ctx *Context, state int) engine.Task {
	return ctx.SetFlag(loc24FightFlag, state)
}

func loc24FightState(ctx *Context) int {
	return ctx.GetFlag(loc24FightFlag)
}

type loc24WaitFrame struct {
	ctx   *Context
	layer string
	frame int
}

func (t *loc24WaitFrame) Update(float64) bool {
	layer, ok := t.ctx.layer(t.layer)
	if !ok || layer == nil {
		return true
	}
	return layer.Frame >= t.frame
}

type loc24FightLoop struct {
	ctx     *Context
	scene   *engine.Scene
	main    engine.Task
	aux     []engine.Task
	started bool
	state   int
}

func (t *loc24FightLoop) startState(state int) {
	t.state = state
	t.started = true
	t.aux = nil
	switch state {
	case 0:
		_ = loc24HideFight(t.ctx).Update(0)
		_ = loc24HideCaptain(t.ctx).Update(0)
		t.main = t.ctx.PlayLayerFrames("S103_CapAttack1", 0x0e, 0x1b)
		t.aux = append(t.aux, engine.Sequence(
			t.ctx.PlayLayerFrames("S103_SabDeckt1", 0, 5),
			t.ctx.PlayLayerFrames("S103_SabDeckt1", 4, 0),
		))
	case 1:
		_ = loc24HideCaptain(t.ctx).Update(0)
		t.main = engine.Sequence(
			t.ctx.PlayLayerFrames("S103_CapAttack2", 0, 10),
			t.ctx.PlayLayerFrames("S103_CapAttack2", 9, 0),
		)
		t.aux = append(t.aux, engine.Sequence(
			&loc24WaitFrame{ctx: t.ctx, layer: "S103_CapAttack2", frame: 8},
			loc24HideFight(t.ctx),
			t.ctx.PlayLayerFrames("S103_SabDeckt1", 0, 5),
			t.ctx.PlayLayerFrames("S103_SabDeckt1", 4, 0),
		))
	case 2:
		_ = loc24HideCaptain(t.ctx).Update(0)
		t.main = engine.Sequence(
			t.ctx.PlayLayerFrames("S103_CapDecktUnten", 0, 10),
			t.ctx.PlayLayerFrames("S103_CapDecktUnten", 9, 0),
		)
		t.aux = append(t.aux, engine.Sequence(
			&loc24WaitFrame{ctx: t.ctx, layer: "S103_CapDecktUnten", frame: 8},
			loc24HideFight(t.ctx),
			t.ctx.PlayLayerFrames("S103_SabAttack2", 0, 5),
			t.ctx.PlayLayerFrames("S103_SabAttack2", 4, 0),
		))
	case 3:
		_ = loc24HideFight(t.ctx).Update(0)
		_ = loc24HideCaptain(t.ctx).Update(0)
		t.main = t.ctx.PlayLayerFrames("S103_CapAttack3", 0, 0x0f)
		t.aux = append(t.aux,
			engine.Sequence(t.ctx.PlayVoiceover("103_CAP_06", ""), t.ctx.PlayVoiceover("103_SAB_07", "")),
			t.ctx.PlayLayerFrames("S103_SabDeckt2", 0, 0x0b),
		)
	default:
		t.main = nil
	}
}

func (t *loc24FightLoop) Update(dt float64) bool {
	if t.ctx.session.scene != t.scene || loc24SceneID(t.ctx.session.state.Scene) != "S103" {
		return true
	}
	if !t.started {
		t.startState(loc24FightState(t.ctx))
	}
	for i := 0; i < len(t.aux); {
		if t.aux[i].Update(dt) {
			t.aux = append(t.aux[:i], t.aux[i+1:]...)
			continue
		}
		i++
	}
	if t.main == nil || !t.main.Update(dt) {
		return false
	}
	if t.state == 3 {
		putOriginalFlag(t.ctx.session.state.OriginalState, loc24StateFightSpeed, 1)
		t.ctx.session.state.Flags[loc24FightFlag] = 0
		for _, id := range []string{"S103_SabAttack2", "S103_SabDeckt1"} {
			if layer, ok := t.ctx.layer(id); ok && layer != nil {
				layer.X += 0x38
			}
		}
		for _, id := range []string{"S103_CapAttack1", "S103_CapAttack2", "S103_CapDecktUnten"} {
			if layer, ok := t.ctx.layer(id); ok && layer != nil {
				layer.X += 0x24
			}
		}
	} else if loc24FightState(t.ctx) != 3 {
		t.ctx.session.state.Flags[loc24FightFlag] = rand.IntN(3)
	}
	t.started = false
	t.main = nil
	return false
}

type loc24Ambient struct {
	ctx   *Context
	scene *engine.Scene
	wait  float64
}

func (t *loc24Ambient) Update(dt float64) bool {
	if t.ctx.session.scene != t.scene || loc24SceneID(t.ctx.session.state.Scene) != "S103" {
		return true
	}
	if t.wait <= 0 {
		t.wait = float64(10+rand.IntN(21)) / 10
	}
	t.wait -= dt
	if t.wait > 0 {
		return false
	}
	switch rand.IntN(16) + 1 {
	case 1, 2, 3, 4, 5, 6:
		_ = t.ctx.PlaySFXVolume("Sfx_Ocean.wav", 80+rand.IntN(21)).Update(0)
	case 7, 8, 9:
		_ = t.ctx.PlaySFXVolume("Sfx_Seagull.wav", 50).Update(0)
	case 10:
		_ = t.ctx.PlaySFXVolume("Sfx_Thunder.wav", 70).Update(0)
	case 11:
		_ = t.ctx.PlaySFXVolume("Sfx_Thunder2.wav", 70).Update(0)
	}
	t.wait = float64(10+rand.IntN(21)) / 10
	return false
}

func loc24Arrival(ctx *Context, actor string) engine.Task {
	return engine.Sequence(
		ctx.PlaceActorPerspective(actor, 0x8e, 0x108),
		ctx.WalkToFacingPerspective(actor, 0x11f, 0x154, 7),
		ctx.Wait(1.5),
		ctx.ShowLayer("SabWalk"),
		loc08SetLayerTransform(ctx, "SabWalk", 108, 139, 198, 90),
		engine.Parallel(
			ctx.PlayLayerFrames("SabWalk", 0x54, 0x5f),
			loc08TweenLayer(ctx, "SabWalk", 178, 0, 0, 0, 11, true, false, false, false),
		),
		engine.Parallel(
			ctx.PlayLayerFrames("SabWalk", 0x0c, 0x17),
			loc08TweenLayer(ctx, "SabWalk", 174, 133, 125, 118, 10, true, true, true, true),
		),
		engine.Parallel(
			ctx.PlayLayerFrames("SabWalk", 0x54, 0x5f),
			loc08TweenLayer(ctx, "SabWalk", 284, 133, 152, 118, 10, true, true, true, true),
		),
		ctx.FreezeLayer("SabWalk", 4),
		ctx.Wait(0.1),
		ctx.FreezeLayer("SabWalk", 0x10),
		ctx.Wait(0.1),
		ctx.FreezeLayer("SabWalk", 0x1c),
		ctx.Wait(0.1),
		ctx.HideLayer("SabWalk"),
		ctx.ShowLayer("S103_SabDrawsSword"),
		ctx.PlayLayerFrames("S103_CapComes", 0, 9),
		ctx.SetLayerZ("S103_CapComes", 0x73),
		ctx.PlayLayerFrames("S103_CapComes", 10, -1),
		ctx.SetActorDirection(actor, 4),
		ctx.PlaySpeechBoundToLayer("S103_CapCameTalk", "103_CAP_01", "", 0, 0x2a),
		loc08LoopLayer(ctx, "S103_SabDrawsSword"),
		ctx.PlayVoiceover("103_SAB_01", ""),
		ctx.HideLayer("S103_CapCameTalk"),
		ctx.PlayLayerFrames("S103_CapAttack1", 0, 0x0d),
	)
}

func (LOC24Controller) LoadConditionMask(*Context, string) int { return 0 }

func (LOC24Controller) Exit(*Context, string, string) engine.Task { return nil }

func (LOC24Controller) Enter(ctx *Context, scene, from string) engine.Task {
	if loc24SceneID(scene) != "S103" {
		return nil
	}
	actor := loc10ActorID(ctx)
	loc24EnsureLayers(ctx)
	loc24BindFrameSFX(ctx)
	if ctx.session.state.Flags == nil {
		ctx.session.state.Flags = map[string]int{}
	}
	ctx.session.state.Flags[loc24FightFlag] = 0
	tasks := []engine.Task{
		ctx.PlayMusic("Loc23_PirateShip.wav"),
		ctx.ShowLayer("S103_Tisch"),
		loc08LoopLayer(ctx, "Gen_SmallSteam"),
		// The original does not assign event 4 to S103_To102 until the
		// opening S103 sequence (CAP_01 / SAB_01 / first attack) has finished.
		ctx.DisableArea("S103_To102"),
	}
	if !ctx.HasItem(0x18) {
		tasks = append(tasks, ctx.ShowLayer("S103_Flasche"))
	}
	if !ctx.HasItem(5) {
		tasks = append(tasks, ctx.ShowLayer("S103_Helm"))
	}
	if actor != "" {
		if !ctx.session.loadedSave {
			tasks = append(tasks, loc24Arrival(ctx, actor))
		} else {
			tasks = append(tasks, ctx.ShowActor(actor))
		}
	}
	// Match the original event wiring: S103_To102 becomes clickable only
	// after the initial arrival/dialogue sequence has completed. On a loaded
	// save the intro is skipped, so this is reached immediately, just as in
	// the original.
	tasks = append(tasks, ctx.EnableArea("S103_To102"))

	if loc24Original(ctx, loc24StateFightSpeed) != 0 {
		tasks = append(tasks, engine.Immediate(func() {
			for _, id := range []string{"S103_SabAttack2", "S103_SabDeckt1"} {
				if layer, ok := ctx.layer(id); ok && layer != nil {
					layer.X += 0x38
				}
			}
			for _, id := range []string{"S103_CapAttack1", "S103_CapAttack2", "S103_CapDecktUnten"} {
				if layer, ok := ctx.layer(id); ok && layer != nil {
					layer.X += 0x24
				}
			}
		}))
	}
	tasks = append(tasks,
		loc24MakeClickables(ctx),
		ctx.RunAmbient(&loc24FightLoop{ctx: ctx, scene: ctx.session.scene}),
		ctx.RunAmbient(&loc24Ambient{ctx: ctx, scene: ctx.session.scene}),
	)
	return engine.Sequence(tasks...)
}

func (LOC24Controller) Click(ctx *Context, area string) engine.Task {
	if loc24SceneID(ctx.session.state.Scene) != "S103" {
		return nil
	}
	actor := loc10ActorID(ctx)
	switch area {
	case "S103_Ofen":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xe6, 0x15e, 2), ctx.PlayVoiceover("103_ROD_08", ""))
	case "S103_Hebel":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xd9, 0x147, 3), ctx.PlayVoiceover("103_ROD_10", ""))
	case "S103_Instrumente":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xd9, 0x147, 3), ctx.PlayVoiceover("103_ROD_09", ""))
	case "S103_To102":
		if !ctx.HasItem(5) {
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xe4, 0x14b, 4), ctx.PlayVoiceover("103_ROD_05", ""))
		}
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0x8e, 0x108, 2),
			loc24SetOriginal(ctx, loc24StateReturnTo102, 1),
			ctx.ChangeLocation(23, "102"),
		)
	case "S103_Swords":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x1a8, 0x165, 4), ctx.PlayVoiceover("103_ROD_11", ""))
	case "S103_Tisch":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x1a8, 0x165, 4), ctx.PlayVoiceover("103_ROD_04", ""))
	case "S103_Flasche":
		if ctx.HasItem(0x18) {
			return nil
		}
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0x1d5, 0x162, 4),
			ctx.HideActor(actor),
			ctx.PlayLayerFrames("S103_RodPickUp", 0, 9),
			ctx.HideLayer("S103_Flasche"),
			ctx.DisableArea("S103_Flasche"),
			ctx.AddItem(0x18),
			ctx.PlayLayerFrames("S103_RodPickUp", 10, -1),
			ctx.ShowActor(actor),
		)
	case "S103_Helm":
		if ctx.HasItem(5) {
			return nil
		}
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0x20e, 0x144, 2),
			ctx.HideActor(actor),
			ctx.HideLayer("S103_Helm"),
			ctx.DisableArea("S103_Helm"),
			ctx.PlayLayer("S103_RodTakesHelm"),
			ctx.AddItem(5),
			loc24SetFightState(ctx, 3),
			ctx.ShowActor(actor),
			ctx.WalkToFacingPerspective(actor, 0x1d7, 0x15d, 1),
		)
	case "S103_CapAttack1", "S103_CapAttack2", "S103_CapAttack3", "S103_CapDecktUnten":
		return ctx.PlayVoiceover("103_SAB_02", "")
	case "S103_SabAttack2", "S103_SabDeckt1", "S103_SabDeckt2":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x178, 0x162, 4), ctx.PlayVoiceover("103_ROD_03", ""))
	}
	return nil
}

func (LOC24Controller) UseItem(*Context, int, string) engine.Task { return nil }

func (LOC24Controller) SelectItem(*Context, int) engine.Task { return nil }
