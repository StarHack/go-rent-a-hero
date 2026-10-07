package game

import (
	"math/rand/v2"

	"github.com/wok/rent-a-hero/internal/audio"
	"github.com/wok/rent-a-hero/internal/engine"
)

const (
	loc08StateBarrelFirst  = 0x3f0c
	loc08StateSabTalk      = 0x3f10
	loc08StateBoatFlag     = 0x3f14
	loc08StateHarbour      = 0x3f18
	loc08StateForestSeen   = 0x3f1c
	loc08StateBoatReturned = 0x3f20
	loc08StateForestGate   = 0x4088
)

type LOC08Controller struct{}

func (LOC08Controller) Enter(ctx *Context, scene, from string) engine.Task {
	switch loc08SceneID(scene) {
	case "S99":
		return loc08Enter99(ctx, from)
	case "S100":
		return loc08Enter100(ctx, from)
	default:
		return nil
	}
}

func loc08Enter99(ctx *Context, from string) engine.Task {
	actor := loc15ActorID(ctx)
	stage := getOriginalFlag(ctx.session.state.OriginalState, loc08StateHarbour)
	tasks := []engine.Task{
		loc08Prepare99(ctx),
		ctx.PlayMusic("Loc08_Harbour.wav"),
		loc08LoopOcean(ctx, 30),
		ctx.EnableArea("099_AnlegePlatz"),
		ctx.EnableArea("099_Stadt"),
		ctx.EnableArea("099_Fass"),
		ctx.EnableArea("099_Wald"),
	}
	if stage >= 1 {
		tasks = append(tasks, ctx.EnableArea("099_Gang"))
	} else {
		tasks = append(tasks, ctx.DisableArea("099_Gang"))
	}
	if stage < 4 {
		tasks = append(tasks, ctx.ShowLayer("099_PirateShip"), ctx.MakeLayerClickable("099_PirateShip"))
	} else {
		tasks = append(tasks, ctx.HideLayer("099_PirateShip"))
	}
	if loc15NumericScene(from) == 100 {
		if actor != "" {
			tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x167, 0xdd))
			if stage > 3 {
				tasks = append(tasks, ctx.WalkToFacingPerspective(actor, 0x14a, 0xdd, 0))
			} else {
				tasks = append(tasks,
					ctx.WalkToFacingPerspective(actor, 0x12f, 0xe8, 6),
					loc08PlayAction(ctx, "SabClimbOut"),
					loc08SetLayerTransform(ctx, "SabWalk", 347, 172, 205, 33),
					ctx.ShowLayer("SabWalk"),
					engine.Parallel(
						loc08TweenLayer(ctx, "SabWalk", 306, 173, 205, 30, 0x14, true, true, true, true),
						ctx.PlayLayerFrames("SabWalk", 0x53, 0x5f),
					),
					ctx.HideLayer("SabWalk"),
					loc08SetOriginal(ctx, loc08StateHarbour, 3),
					loc08PoseSab(ctx, 3),
				)
			}
		}
	} else {
		switch stage {
		case 0:
			if actor != "" {
				tasks = append(tasks,
					ctx.HideActor(actor),
					loc08PlayAction(ctx, "RodReinCam"),
					ctx.ShowActor(actor),
				)
			}
		case 1, 2, 3:
			tasks = append(tasks, loc08PoseSab(ctx, stage))
		}
		// Original S99 entry applies this spawn/walk to every entry except S100,
		// independent of the harbour/Sab state.
		if actor != "" {
			tasks = append(tasks,
				ctx.PlaceActorPerspective(actor, 0x274, 0xfc),
				ctx.WalkToFacingPerspective(actor, 0x21a, 0x126, 2),
			)
		}
	}
	tasks = append(tasks, ctx.RunAmbient(&loc08Ambient{ctx: ctx}))
	return engine.Sequence(tasks...)
}

func loc08Enter100(ctx *Context, from string) engine.Task {
	actor := loc15ActorID(ctx)
	state := ctx.session.state.OriginalState
	stage := getOriginalFlag(state, loc08StateHarbour)
	tasks := []engine.Task{
		loc08Prepare100(ctx),
		ctx.PlayMusic("Loc08_Harbour.wav"),
		loc08LoopOcean(ctx, 70),
		ctx.EnableArea("100_WayBack"),
		loc08LoopLayer(ctx, "Brandung"),
	}
	if stage < 4 {
		tasks = append(tasks, ctx.ShowLayer("100_PirateShip"), ctx.MakeLayerClickable("100_PirateShip"))
	} else {
		tasks = append(tasks, ctx.HideLayer("100_PirateShip"))
	}
	from99 := loc15NumericScene(from) == 99
	if from99 {
		if stage == 4 {
			if actor != "" {
				tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x1d3, 0xd0), ctx.WalkToFacingPerspective(actor, 0x149, 0xe6, 1))
			}
			tasks = append(tasks, loc08ShowClickableLayer(ctx, "BoatBack"))
		} else {
			tasks = append(tasks, loc08LoopBoat(ctx), loc08MakeBoatClickable(ctx))
			if actor != "" {
				tasks = append(tasks,
					ctx.PlaceActorPerspective(actor, 0x1d3, 0xd0),
					ctx.WalkToFacingPerspective(actor, 0x149, 0xe6, 1),
				)
			}
			tasks = append(tasks,
				loc08SetLayerTransform(ctx, "SabWalk", 600, 151, 221, 44),
				ctx.ShowLayer("SabWalk"),
				// Original S100 entry: mode 0x0D keeps SabWalk frames 0x24..0x2F
				// cycling for the entire 0x3C-step position/scale transform. Playing
				// the range once in parallel makes the layer finish animating early and
				// visibly glide for the remainder of the movement.
				loc08WalkTween(ctx, "SabWalk", 341, 149, 167, 50, 0x3c, 0x24, 0x2f),
				ctx.HideLayer("SabWalk"),
				loc08PoseSab100(ctx),
			)
		}
	} else {
		tasks = append(tasks, loc08SetOriginal(ctx, loc08StateHarbour, 4))
		if actor != "" {
			tasks = append(tasks,
				ctx.ShowLayer("BoatReturns"),
				ctx.PlayLayerFrames("BoatReturns", 0, 6),
				ctx.PlaySFXVolume("Sfx_Paddling.wav", 50),
				ctx.PlayLayerFrames("BoatReturns", 7, 0x13),
				ctx.PlaySFXVolume("Sfx_Paddling.wav", 50),
				ctx.PlayLayerFrames("BoatReturns", 0x14, -1),
				ctx.HideLayer("BoatReturns"),
				loc08SetOriginal(ctx, loc08StateBoatReturned, 1),
				loc08ShowClickableLayer(ctx, "BoatBack"),
				ctx.PlaceActorPerspective(actor, 0x12e, 0xe2),
				ctx.WalkToFacingPerspective(actor, 0x12e, 0xe6, 0),
			)
		} else {
			tasks = append(tasks, loc08ShowClickableLayer(ctx, "BoatBack"))
		}
	}
	tasks = append(tasks, ctx.RunAmbient(&loc08Ambient{ctx: ctx}))
	return engine.Sequence(tasks...)
}

func (LOC08Controller) Exit(*Context, string, string) engine.Task { return nil }

func (LOC08Controller) Click(ctx *Context, area string) engine.Task {
	switch loc08SceneID(ctx.session.state.Scene) {
	case "S99":
		return loc08Click99(ctx, area)
	case "S100":
		return loc08Click100(ctx, area)
	default:
		return nil
	}
}

func loc08Click99(ctx *Context, area string) engine.Task {
	actor := loc15ActorID(ctx)
	switch area {
	case "099_PirateShip":
		if actor == "" {
			return nil
		}
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x72, 0xf6, 3), ctx.Say(actor, "099_ROD_10", ""))
	case "099_AnlegePlatz":
		if actor == "" {
			return nil
		}
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x1aa, 0x102, 1), ctx.Say(actor, "099_ROD_11", ""))
	case "099_Stadt":
		if actor == "" {
			return nil
		}
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x122, 0xde, 4), ctx.Say(actor, "099_ROD_09", ""))
	case "099_Fass":
		return loc08Barrel(ctx, actor)
	case "099_Wald":
		if actor == "" {
			return nil
		}
		prefix := []engine.Task{}
		if getOriginalFlag(ctx.session.state.OriginalState, loc08StateHarbour) == 1 && getOriginalFlag(ctx.session.state.OriginalState, loc08StateSabTalk) < 4 {
			prefix = append(prefix, loc08Sab(ctx, "099_SAB_08"))
		}
		prefix = append(prefix, ctx.WalkToFacingPerspective(actor, 0x26a, 0x102, 5))
		if getOriginalFlag(ctx.session.state.OriginalState, loc08StateForestGate) != 0 && getOriginalFlag(ctx.session.state.OriginalState, loc08StateForestSeen) == 0 {
			prefix = append(prefix, loc08SetOriginal(ctx, loc08StateForestSeen, 1), ctx.ChangeLocation(16, "58"))
		} else {
			prefix = append(prefix, ctx.ChangeLocation(10, "39"))
		}
		return engine.Sequence(prefix...)
	case "099_Gang":
		if actor == "" {
			return nil
		}
		return loc08Gang(ctx, actor)
	case "SabStand":
		return loc08SabConversation(ctx, actor)
	}
	return nil
}

func loc08Click100(ctx *Context, area string) engine.Task {
	actor := loc15ActorID(ctx)
	switch area {
	case "100_PirateShip":
		if actor == "" {
			return nil
		}
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x136, 0xeb, 3), ctx.Say(actor, "100_ROD_02", ""))
	case "SabStand":
		if actor == "" {
			return nil
		}
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x153, 0xe8, 5), ctx.Say(actor, "100_ROD_03", ""), loc08Sab100(ctx, "100_SAB_02"))
	case "100_WayBack":
		if actor == "" {
			return nil
		}
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x1d3, 0xd0, 6), ctx.ChangeLocation(8, "99"))
	case "Boat", "Boot", "BootFahne", "BoatBack":
		return loc08BoatInteraction(ctx, actor)
	}
	return nil
}

func (LOC08Controller) UseItem(*Context, int, string) engine.Task { return nil }

func (LOC08Controller) SelectItem(ctx *Context, item int) engine.Task {
	if loc08SceneID(ctx.session.state.Scene) != "S100" || item != 2 || getOriginalFlag(ctx.session.state.OriginalState, loc08StateBoatFlag) != 0 {
		return nil
	}
	actor := loc15ActorID(ctx)
	if actor == "" {
		return nil
	}
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0x174, 0xe6, 7),
		ctx.HideActor(actor),
		ctx.ShowLayer("RodFahne"),
		ctx.PlayLayerFrames("RodFahne", 0, 0x19),
		loc08HideBoat(ctx),
		loc08LoopLayer(ctx, "BootFahne"),
		ctx.PlayLayerFrames("RodFahne", 0x1a, -1),
		ctx.HideLayer("RodFahne"),
		loc08SetOriginal(ctx, loc08StateBoatFlag, 1),
		ctx.RemoveItem(2),
		ctx.MakeLayerClickable("BootFahne"),
		ctx.ShowActor(actor),
		ctx.Say(actor, "100_ROD_04", ""),
		loc08Sab100(ctx, "100_SAB_03"),
	)
}

func (LOC08Controller) LoadConditionMask(ctx *Context, scene string) int {
	scene = loc08SceneID(scene)
	if scene != "S99" && scene != "S100" {
		return 0
	}
	stage := getOriginalFlag(ctx.session.state.OriginalState, loc08StateHarbour)
	mask := 0
	if stage < 2 {
		mask |= 1
	}
	if stage < 4 {
		mask |= 2
	}
	if scene == "S100" && getOriginalFlag(ctx.session.state.OriginalState, loc08StateBoatReturned) == 0 {
		mask |= 4
	}
	return mask
}

func loc08SceneID(scene string) string {
	switch scene {
	case "99", "099", "S99", "S099":
		return "S99"
	case "100", "S100":
		return "S100"
	default:
		return scene
	}
}

func loc08LoopOcean(ctx *Context, volume int) engine.Task {
	sound, err := engine.LoadSound(ctx.session.idx, "Sfx_Ocean.wav")
	if err != nil {
		return engine.Immediate(func() {})
	}
	return engine.Immediate(func() {
		ctx.session.audioEngine.PlayLoopingWithVolume(sound, audio.CategorySFX, loc09DirectSoundGain(volume))
	})
}

func loc08Prepare100(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		for _, id := range []string{"Brandung", "BoatAway", "BoatReturns", "BoatBack", "BootFahne", "RodFahne", "SabWalk", "SabStand", "100_PirateShip"} {
			_, _, _ = ctx.ensureAssetLayer(id)
			if layer, ok := ctx.layer(id); ok {
				layer.Playing = false
				layer.TaskDriven = false
				layer.Accumulator = 0
				layer.Visible = false
			}
		}
		if _, ok := ctx.layer("Boat"); !ok {
			_, _, _ = ctx.ensureAssetLayer("Boat")
		}
		if layer, ok := ctx.layer("Boat"); ok {
			layer.Playing = false
			layer.TaskDriven = false
			layer.Accumulator = 0
			layer.Visible = false
		}
	})
}

func loc08LoopLayer(ctx *Context, id string) engine.Task {
	return engine.Immediate(func() {
		layer, ok := ctx.layer(id)
		if !ok {
			layer, _, _ = ctx.ensureAssetLayer(id)
		}
		if layer == nil {
			return
		}
		layer.Visible = true
		layer.Enabled = true
		layer.Mode = engine.AnimLoop
		layer.Playing = true
		layer.TaskDriven = false
		layer.Accumulator = 0
	})
}

func loc08BoatLayerID(ctx *Context) string {
	for _, id := range []string{"Boat", "Boot"} {
		if _, ok := ctx.layer(id); ok {
			return id
		}
	}
	for _, id := range []string{"Boat", "Boot"} {
		if _, _, err := ctx.ensureAssetLayer(id); err == nil {
			return id
		}
	}
	return "Boat"
}

func loc08LoopBoat(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		id := loc08BoatLayerID(ctx)
		layer, ok := ctx.layer(id)
		if !ok || layer == nil {
			return
		}
		layer.Visible = true
		layer.Enabled = true
		layer.Mode = engine.AnimLoop
		layer.Playing = true
		layer.TaskDriven = false
		layer.Accumulator = 0
	})
}

func loc08MakeBoatClickable(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		id := loc08BoatLayerID(ctx)
		layer, ok := ctx.layer(id)
		if !ok || layer == nil || layer.Source == nil || ctx.session.scene == nil {
			return
		}
		w := layer.Source.Width() * layer.Zoom / 100
		h := layer.Source.Height() * layer.Zoom / 100
		area, exists := ctx.session.scene.Areas[id]
		if !exists {
			area = &engine.Area{ID: id, CursorType: 10}
			ctx.session.localizeArea(area, ctx.session.state.Location)
			ctx.session.scene.Areas[id] = area
			ctx.session.scene.AreaOrder = append(ctx.session.scene.AreaOrder, id)
		}
		area.X1 = layer.X
		area.Y1 = layer.Y
		area.X2 = layer.X + w
		area.Y2 = layer.Y + h
		area.Enabled = true
	})
}

func loc08HideBoat(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		for _, id := range []string{"Boat", "Boot"} {
			if layer, ok := ctx.layer(id); ok {
				layer.Visible = false
				layer.Playing = false
				if area, exists := ctx.session.scene.Areas[id]; exists {
					area.Enabled = false
				}
			}
		}
	})
}

func loc08ShowClickableLayer(ctx *Context, id string) engine.Task {
	return engine.Sequence(ctx.ShowLayer(id), ctx.MakeLayerClickable(id))
}

// loc08SabSpeechTask reproduces the original SabStand speech binding for
// Location 8: the unknown lady talks only inside the authored 4..7 range.
// Keeping this scene-local avoids stepping into the adjacent stand/turn pose.
type loc08SabSpeechTask struct {
	ctx         *Context
	line        string
	layer       *engine.Layer
	voice       engine.Task
	started     bool
	accumulator float64
	prevFrame   int
	prevPlaying bool
	prevDriven  bool
	prevVisible bool
	prevEnabled bool
}

func (t *loc08SabSpeechTask) Update(dt float64) bool {
	if !t.started {
		t.started = true
		t.voice = t.ctx.PlayVoiceover(t.line, "")
		if layer, ok := t.ctx.layer("SabStand"); ok {
			t.layer = layer
			t.prevFrame = layer.Frame
			t.prevPlaying = layer.Playing
			t.prevDriven = layer.TaskDriven
			t.prevVisible = layer.Visible
			t.prevEnabled = layer.Enabled
			layer.Visible = true
			layer.Enabled = true
			layer.Playing = false
			layer.TaskDriven = true
			layer.Frame = 4
			layer.Accumulator = 0
		}
	}

	if t.layer != nil {
		fps := t.layer.FPS
		if fps <= 0 {
			fps = 20
		}
		t.accumulator += dt
		frameDuration := 1.0 / float64(fps)
		for t.accumulator >= frameDuration {
			t.accumulator -= frameDuration
			t.layer.Frame++
			if t.layer.Frame > 7 || t.layer.Frame < 4 {
				t.layer.Frame = 4
			}
		}
	}

	if t.voice != nil && t.voice.Update(dt) {
		if t.layer != nil {
			t.layer.Frame = t.prevFrame
			t.layer.Accumulator = 0
			t.layer.Playing = t.prevPlaying
			t.layer.TaskDriven = t.prevDriven
			t.layer.Visible = t.prevVisible
			t.layer.Enabled = t.prevEnabled
		}
		return true
	}
	return false
}

func loc08SabSpeech(ctx *Context, line string) engine.Task {
	return &loc08SabSpeechTask{ctx: ctx, line: line}
}

func loc08PoseSab100(ctx *Context) engine.Task {
	return engine.Sequence(loc08SetLayerTransform(ctx, "SabStand", 346, 149, 167, 48), ctx.ShowLayer("SabStand"), ctx.FreezeLayer("SabStand", 3), loc08MakeSabClickable(ctx))
}

func loc08Sab100(ctx *Context, line string) engine.Task {
	return engine.Sequence(ctx.ShowLayer("SabStand"), loc08SabSpeech(ctx, line))
}

func loc08BoatInteraction(ctx *Context, actor string) engine.Task {
	if actor == "" {
		return nil
	}
	state := ctx.session.state.OriginalState
	if getOriginalFlag(state, loc08StateForestGate) != 0 {
		return ctx.Say(actor, "100_ROD_01", "")
	}
	if getOriginalFlag(state, loc08StateBoatFlag) == 0 {
		return loc08Sab100(ctx, "100_SAB_01")
	}
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0x174, 0xe6, 7),
		ctx.HideActor(actor),
		loc08HideBoat(ctx),
		ctx.HideLayer("BootFahne"),
		ctx.ShowLayer("BoatAway"),
		ctx.PlayLayerFrames("BoatAway", 0, 0x18),
		ctx.HideLayer("SabStand"),
		loc08SetLayerTransform(ctx, "SabWalk", 346, 149, 167, 48),
		ctx.ShowLayer("SabWalk"),
		engine.Parallel(
			loc08WalkTween(ctx, "SabWalk", 344, 146, 119, 57, 10, 0x0b, 0x17),
			ctx.PlayLayerFrames("BoatAway", 0x19, 0x23),
		),
		ctx.HideLayer("SabWalk"),
		ctx.PlayLayerFrames("BoatAway", 0x24, 0x50),
		ctx.PlaySFXVolume("Sfx_Paddling.wav", 50),
		ctx.PlayLayerFrames("BoatAway", 0x51, 0x68),
		ctx.PlaySFXVolume("Sfx_Paddling.wav", 50),
		ctx.PlayLayerFrames("BoatAway", 0x69, 0x80),
		ctx.PlaySFXVolume("Sfx_Paddling.wav", 50),
		ctx.PlayLayerFrames("BoatAway", 0x81, -1),
		loc08TweenLayer(ctx, "BoatAway", -100, 0, 0, 0, 10, true, false, false, false),
		ctx.HideLayer("BoatAway"),
		loc08SetOriginal(ctx, loc08StateHarbour, 4),
		ctx.ChangeLocation(23, "103"),
	)
}

func loc08Prepare99(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		for _, id := range []string{"RodReinCam", "Busch", "Klappe", "RodClimbIn", "RodClimbOut", "SabJump", "RodSchreck", "SabPoint", "SabLever", "SabClimbIn", "SabClimbOut", "SabWalk", "SabStand", "099_PirateShip"} {
			_, _, _ = ctx.ensureAssetLayer(id)
			if layer, ok := ctx.layer(id); ok {
				layer.Playing = false
				layer.TaskDriven = false
				layer.Accumulator = 0
				if id == "RodReinCam" {
					layer.Presentation = true
					layer.ColorKeyed = false
				}
				if id != "099_PirateShip" {
					layer.Visible = false
				}
			}
		}
	})
}

func loc08Barrel(ctx *Context, actor string) engine.Task {
	if actor == "" {
		return nil
	}
	state := ctx.session.state.OriginalState
	if getOriginalFlag(state, loc08StateBarrelFirst) == 2 {
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x1e6, 0xfe, 4), ctx.Say(actor, "099_ROD_08", ""))
	}
	if getOriginalFlag(state, loc08StateBarrelFirst) != 1 {
		return nil
	}
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0x1e6, 0xfe, 4),
		loc08SetOriginal(ctx, loc08StateBarrelFirst, 2),
		loc08SetLayerPosition(ctx, "SabJump", 0x1ac, 0xb4),
		ctx.ShowLayer("SabJump"),
		engine.Parallel(
			loc08TweenLayer(ctx, "SabJump", 428, 167, 92, 0, 0x0f, true, true, true, false),
			ctx.PlayLayerFrames("SabJump", 0, 0x12),
		),
		ctx.PlaySFX("Sfx_Knock_Wet.wav"),
		ctx.HideActor(actor),
		ctx.ShowLayer("RodSchreck"),
		ctx.PlayLayerFrames("RodSchreck", 0, 7),
		ctx.PlayLayerFrames("SabJump", 0x12, 0x15),
		ctx.PlaySFX("Sfx_Woosh.wav"),
		ctx.PlayLayerFrames("SabJump", 0x15, 0x29),
		ctx.PlayLayerFrames("RodSchreck", 7, -1),
		ctx.PlayLayerFrames("SabJump", 0x29, -1),
		ctx.HideLayer("RodSchreck"),
		ctx.HideLayer("SabJump"),
		loc08SetOriginal(ctx, loc08StateHarbour, 1),
		loc08PoseSab(ctx, 1),
		ctx.SetActorDirection(actor, 3),
		ctx.ShowActor(actor),
		loc08Sab(ctx, "099_SAB_01"),
	)
}

func loc08SabConversation(ctx *Context, actor string) engine.Task {
	if actor == "" {
		return nil
	}
	state := ctx.session.state.OriginalState
	stage := getOriginalFlag(state, loc08StateHarbour)
	if stage == 1 {
		talk := getOriginalFlag(state, loc08StateSabTalk)
		switch talk {
		case 1:
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x1e6, 0xfe, 3), loc08SetOriginal(ctx, loc08StateSabTalk, 2), ctx.Say(actor, "099_ROD_01", ""), loc08Sab(ctx, "099_SAB_02"))
		case 2:
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x1e6, 0xfe, 3), loc08SetOriginal(ctx, loc08StateSabTalk, 3), ctx.Say(actor, "099_ROD_02", ""), loc08Sab(ctx, "099_SAB_03"))
		case 3:
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x1e6, 0xfe, 3), loc08SetOriginal(ctx, loc08StateSabTalk, 4), ctx.Say(actor, "099_ROD_03", ""), ctx.WalkToFacingPerspective(actor, 0x1e6, 0x102, 0), ctx.Say(actor, "099_ROD_04", ""), loc08Sab(ctx, "099_SAB_04"))
		case 4:
			return engine.Sequence(
				ctx.WalkToFacingPerspective(actor, 0x1e6, 0xfe, 3),
				loc08SetOriginal(ctx, loc08StateSabTalk, 5),
				ctx.Say(actor, "099_ROD_05", ""),
				loc08Sab(ctx, "099_SAB_05"),
				ctx.HideLayer("SabStand"),
				loc08SetLayerPosition(ctx, "SabPoint", 0x1b0, 0xb0),
				ctx.ShowLayer("SabPoint"), ctx.PlayLayer("SabPoint"), ctx.HideLayer("SabPoint"),
				ctx.ShowLayer("SabStand"),
			)
		case 5:
			return loc08SabLeverSequence(ctx, actor)
		}
	}
	if stage == 2 {
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x15f, 0xe9, 6), ctx.Say(actor, "099_ROD_07", ""), loc08Sab(ctx, "099_SAB_07"))
	}
	if stage == 3 {
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x12f, 0xe8, 5), ctx.Say(actor, "099_ROD_07", ""), loc08Sab(ctx, "099_SAB_07"))
	}
	return nil
}

func loc08SabLeverSequence(ctx *Context, actor string) engine.Task {
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0x1e6, 0xfe, 3),
		ctx.Say(actor, "099_ROD_06", ""),
		loc08Sab(ctx, "099_SAB_06"),
		ctx.HideLayer("SabStand"),
		loc08SetLayerTransform(ctx, "SabWalk", 433, 176, 92, 49),
		ctx.ShowLayer("SabWalk"),
		engine.Parallel(
			loc08TweenLayer(ctx, "SabWalk", 368, 0, 0, 0, 0x14, true, false, false, false),
			ctx.PlayLayerFrames("SabWalk", 0x23, 0x2f),
		),
		engine.Parallel(
			loc08TweenLayer(ctx, "SabWalk", 368, 171, 143, 40, 0x14, true, true, true, true),
			ctx.PlayLayerFrames("SabWalk", 0x3b, 0x47),
		),
		ctx.HideLayer("SabWalk"),
		loc08SetOriginal(ctx, loc08StateHarbour, 2),
		ctx.ShowLayer("SabLever"),
		ctx.PlayLayerFrames("SabLever", 0, 5),
		ctx.PlaySFX("Sfx_Work_FixBelt.wav"),
		ctx.PlayLayerFrames("SabLever", 5, 0xe),
		ctx.PlaySFX("Sfx_Door_CreakShort.wav"),
		ctx.ShowLayer("Klappe"), ctx.FreezeLayer("Klappe", 9),
		ctx.PlayLayerFrames("SabLever", 0xe, -1),
		ctx.HideLayer("SabLever"),
		loc08PoseSab(ctx, 2),
		ctx.EnableArea("099_Gang"),
	)
}

func loc08Gang(ctx *Context, actor string) engine.Task {
	stage := getOriginalFlag(ctx.session.state.OriginalState, loc08StateHarbour)
	tasks := []engine.Task{
		ctx.WalkToFacingPerspective(actor, 0x167, 0xdb, 0),
		ctx.HideActor(actor),
		ctx.ShowLayer("RodClimbIn"), ctx.PlayLayer("RodClimbIn"), ctx.HideLayer("RodClimbIn"),
	}
	if stage == 2 {
		tasks = append(tasks,
			ctx.HideLayer("SabStand"),
			loc08SetLayerTransform(ctx, "SabWalk", 368, 171, 143, 40),
			ctx.ShowLayer("SabWalk"),
			engine.Parallel(
				loc08TweenLayer(ctx, "SabWalk", 330, 171, 143, 40, 0x14, true, true, true, true),
				ctx.PlayLayerFrames("SabWalk", 0x23, 0x2f),
			),
			engine.Parallel(
				loc08TweenLayer(ctx, "SabWalk", 347, 172, 205, 33, 0x14, true, true, true, true),
				ctx.PlayLayerFrames("SabWalk", 0x53, 0x5f),
			),
			ctx.HideLayer("SabWalk"),
			ctx.ShowLayer("SabClimbIn"), ctx.PlayLayer("SabClimbIn"), ctx.HideLayer("SabClimbIn"),
		)
	} else if stage == 3 {
		tasks = append(tasks,
			ctx.HideLayer("SabStand"),
			loc08SetLayerTransform(ctx, "SabWalk", 306, 173, 205, 30),
			ctx.ShowLayer("SabWalk"),
			engine.Parallel(
				loc08TweenLayer(ctx, "SabWalk", 347, 172, 205, 33, 0x14, true, true, true, true),
				ctx.PlayLayerFrames("SabWalk", 0x53, 0x5f),
			),
			ctx.HideLayer("SabWalk"),
			ctx.ShowLayer("SabClimbIn"), ctx.PlayLayer("SabClimbIn"), ctx.HideLayer("SabClimbIn"),
		)
	}
	tasks = append(tasks, ctx.ChangeLocation(8, "100"))
	return engine.Sequence(tasks...)
}

func loc08Sab(ctx *Context, line string) engine.Task {
	return engine.Sequence(ctx.ShowLayer("SabStand"), loc08SabSpeech(ctx, line))
}

func loc08PoseSab(ctx *Context, stage int) engine.Task {
	x, y, z, zoom := 433, 176, 92, 49
	if stage == 2 {
		x, y, z, zoom = 368, 171, 143, 40
	} else if stage >= 3 {
		x, y, z, zoom = 306, 173, 205, 30
	}
	return engine.Sequence(loc08SetLayerTransform(ctx, "SabStand", x, y, z, zoom), ctx.ShowLayer("SabStand"), ctx.FreezeLayer("SabStand", 3), loc08MakeSabClickable(ctx))
}

func loc08MakeSabClickable(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		layer, ok := ctx.layer("SabStand")
		if !ok || layer.Source == nil || ctx.session.scene == nil {
			return
		}
		w := layer.Source.Width() * layer.Zoom / 100
		h := layer.Source.Height() * layer.Zoom / 100
		area, exists := ctx.session.scene.Areas["SabStand"]
		if !exists {
			area = &engine.Area{ID: "SabStand", CursorType: 10}
			ctx.session.localizeArea(area, ctx.session.state.Location)
			ctx.session.scene.Areas["SabStand"] = area
			ctx.session.scene.AreaOrder = append(ctx.session.scene.AreaOrder, "SabStand")
		}
		area.X1 = layer.X
		area.Y1 = layer.Y
		area.X2 = layer.X + w
		area.Y2 = layer.Y + h
		area.Enabled = true
	})
}

func loc08PlayAction(ctx *Context, id string) engine.Task {
	return engine.Sequence(ctx.ShowLayer(id), ctx.PlayLayer(id), ctx.HideLayer(id))
}

func loc08SetLayerPosition(ctx *Context, id string, x, y int) engine.Task {
	return engine.Immediate(func() {
		if layer, ok := ctx.layer(id); ok {
			layer.X = x
			layer.Y = y
		}
	})
}

func loc08SetLayerTransform(ctx *Context, id string, x, y, z, zoom int) engine.Task {
	return engine.Immediate(func() {
		if layer, ok := ctx.layer(id); ok {
			layer.X = x
			layer.Y = y
			layer.Z = z
			layer.Zoom = zoom
		}
	})
}

type loc08LayerTween struct {
	ctx                       *Context
	id                        string
	targetX, targetY, targetZ int
	targetZoom, steps         int
	changeX, changeY, changeZ bool
	changeZoom                bool
	started                   bool
	elapsed                   float64
	startX, startY, startZ    int
	startZoom                 int
}

func loc08TweenLayer(ctx *Context, id string, x, y, z, zoom, steps int, changeX, changeY, changeZ, changeZoom bool) engine.Task {
	return &loc08LayerTween{ctx: ctx, id: id, targetX: x, targetY: y, targetZ: z, targetZoom: zoom, steps: steps, changeX: changeX, changeY: changeY, changeZ: changeZ, changeZoom: changeZoom}
}

func (t *loc08LayerTween) Update(dt float64) bool {
	layer, ok := t.ctx.layer(t.id)
	if !ok {
		return true
	}
	if !t.started {
		t.started = true
		t.startX, t.startY, t.startZ, t.startZoom = layer.X, layer.Y, layer.Z, layer.Zoom
	}
	if t.steps <= 0 {
		if t.changeX {
			layer.X = t.targetX
		}
		if t.changeY {
			layer.Y = t.targetY
		}
		if t.changeZ {
			layer.Z = t.targetZ
		}
		if t.changeZoom {
			layer.Zoom = t.targetZoom
		}
		return true
	}
	t.elapsed += dt
	p := t.elapsed / (float64(t.steps) / 20.0)
	if p > 1 {
		p = 1
	}
	if t.changeX {
		layer.X = t.startX + int(float64(t.targetX-t.startX)*p)
	}
	if t.changeY {
		layer.Y = t.startY + int(float64(t.targetY-t.startY)*p)
	}
	if t.changeZ {
		layer.Z = t.startZ + int(float64(t.targetZ-t.startZ)*p)
	}
	if t.changeZoom {
		layer.Zoom = t.startZoom + int(float64(t.targetZoom-t.startZoom)*p)
	}
	return p >= 1
}

type loc08WalkTweenTask struct {
	ctx                                   *Context
	id                                    string
	targetX, targetY, targetZ, targetZoom int
	steps                                 int
	startFrame, endFrame                  int
	started                               bool
	startX, startY, startZ, startZoom     int
	elapsed                               float64
}

func (t *loc08WalkTweenTask) Update(dt float64) bool {
	layer, ok := t.ctx.layer(t.id)
	if !ok {
		return true
	}
	if !t.started {
		t.started = true
		t.startX, t.startY, t.startZ, t.startZoom = layer.X, layer.Y, layer.Z, layer.Zoom
		layer.Visible = true
		layer.Enabled = true
		layer.Playing = false
		layer.TaskDriven = true
		layer.Frame = t.startFrame
		layer.Accumulator = 0
	}

	// The original mode 0x0D keeps the authored walk range advancing through
	// the normal sprite animation machinery while the transform tween runs.
	// Use AdvanceScripted rather than assigning frames ourselves so AVI/frame
	// timing and frame-entry events behave exactly like other scripted layers.
	layer.Playing = true
	layer.Mode = engine.AnimOnce
	layer.AdvanceScripted(dt)
	if layer.Frame > t.endFrame || layer.Frame < t.startFrame {
		layer.Frame = t.startFrame
		layer.Accumulator = 0
	}

	if t.steps <= 0 {
		layer.X, layer.Y, layer.Z, layer.Zoom = t.targetX, t.targetY, t.targetZ, t.targetZoom
		return true
	}
	t.elapsed += dt
	p := t.elapsed / (float64(t.steps) / 20.0)
	if p > 1 {
		p = 1
	}
	layer.X = t.startX + int(float64(t.targetX-t.startX)*p)
	layer.Y = t.startY + int(float64(t.targetY-t.startY)*p)
	layer.Z = t.startZ + int(float64(t.targetZ-t.startZ)*p)
	layer.Zoom = t.startZoom + int(float64(t.targetZoom-t.startZoom)*p)
	if p >= 1 {
		layer.Playing = false
		layer.TaskDriven = false
		layer.Accumulator = 0
		return true
	}
	return false
}

func loc08WalkTween(ctx *Context, id string, x, y, z, zoom, steps, startFrame, endFrame int) engine.Task {
	return &loc08WalkTweenTask{ctx: ctx, id: id, targetX: x, targetY: y, targetZ: z, targetZoom: zoom, steps: steps, startFrame: startFrame, endFrame: endFrame}
}

func loc08SetOriginal(ctx *Context, offset, value int) engine.Task {
	return engine.Immediate(func() { putOriginalFlag(ctx.session.state.OriginalState, offset, value) })
}

type loc08Ambient struct {
	ctx   *Context
	wait  float64
	alive bool
}

func (t *loc08Ambient) Update(dt float64) bool {
	if !t.alive {
		t.alive = true
		t.wait = 1 + rand.Float64()*2
	}
	scene := loc08SceneID(t.ctx.session.state.Scene)
	if scene != "S99" && scene != "S100" {
		return true
	}
	t.wait -= dt
	if t.wait > 0 {
		return false
	}
	switch rand.IntN(16) + 1 {
	case 7, 8, 9:
		_ = t.ctx.PlaySFX("Sfx_Seagull.wav").Update(0)
	case 10:
		_ = t.ctx.PlaySFX("Sfx_Thunder.wav").Update(0)
	case 11:
		_ = t.ctx.PlaySFX("Sfx_Thunder2.wav").Update(0)
	}
	t.wait = 1 + rand.Float64()*2
	return false
}
