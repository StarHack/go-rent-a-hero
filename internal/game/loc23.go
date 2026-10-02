package game

import (
	"strings"

	"github.com/wok/rent-a-hero/internal/engine"
)

const (
	loc23StateCrate      = 0x407c
	loc23StateDoorOpen   = 0x4080
	loc23StateDoorLocked = 0x4084
)

type LOC23Controller struct{}

func loc23SceneID(scene string) string {
	s := strings.ToUpper(strings.TrimSpace(scene))
	s = strings.TrimPrefix(s, "S")
	s = strings.TrimLeft(s, "0")
	if s == "" {
		s = "0"
	}
	if s == "102" || s == "103" {
		return "S102"
	}
	return strings.ToUpper(strings.TrimSpace(scene))
}

func loc23Original(ctx *Context, off int) int {
	return getOriginalFlag(ctx.session.state.OriginalState, off)
}

func loc23SetOriginal(ctx *Context, off, value int) engine.Task {
	return engine.Immediate(func() { putOriginalFlag(ctx.session.state.OriginalState, off, value) })
}

func loc23EnsureLayers(ctx *Context) {
	ids := []string{
		"S102_Escape", "S102_EnterShip", "S102_KisteLinks", "S102_KisteNachLinks",
		"S102_KisteNachRechts", "S102_KisteRechts", "S102_TuerZu", "S102_TuerOffen",
		"S102_RodRuettel", "S102_SabTuer", "SabWalk",
	}
	for _, id := range ids {
		_, _, _ = ctx.ensureAssetLayer(id)
		if layer, ok := ctx.layer(id); ok {
			if id == "S102_EnterShip" {
				layer.Presentation = true
				layer.ColorKeyed = false
			}
			layer.Visible = false
			layer.Playing = false
			layer.TaskDriven = false
			layer.Accumulator = 0
		}
	}
}

func loc23BindFrameSFX(ctx *Context) {
	events := map[string]map[int][]string{
		"S102_SabTuer": {
			0x0d: {"Sfx_Door_Slammed.wav"},
		},
		"SabWalk": {
			3: {"Gen_StepLeft.wav"}, 8: {"Gen_StepRight.wav"}, 0x0e: {"Gen_StepLeft.wav"}, 0x13: {"Gen_StepRight.wav"},
			0x19: {"Gen_StepLeft.wav"}, 0x1e: {"Gen_StepRight.wav"}, 0x26: {"Gen_StepLeft.wav"}, 0x2b: {"Gen_StepRight.wav"},
			0x31: {"Gen_StepLeft.wav"}, 0x36: {"Gen_StepRight.wav"}, 0x3c: {"Gen_StepLeft.wav"}, 0x42: {"Gen_StepRight.wav"},
			0x49: {"Gen_StepLeft.wav"}, 0x4f: {"Gen_StepRight.wav"}, 0x55: {"Gen_StepLeft.wav"}, 0x5b: {"Gen_StepRight.wav"},
		},
		"S102_KisteNachLinks": {
			0x0c: {"Sfx_Box_Pushed.wav"}, 0x29: {"Sfx_Box_Pushed_Short.wav"},
		},
		"S102_KisteNachRechts": {
			0x0c: {"Sfx_Box_Pushed.wav"},
		},
		"S102_Escape": {
			0x15: {"Sfx_WaterSplash.wav"},
			0x34: {"Sfx_Wood_Creaking.wav"},
			0x45: {"Sfx_Wood_Creaking.wav"},
			0x48: {"Sfx_WaterSplash.wav"},
			0x51: {"Sfx_Wood_Creaking.wav", "Sfx_WaterSplash.wav"},
			0x5d: {"Sfx_Wood_Creaking.wav", "Sfx_WaterSplash.wav"},
			0x66: {"Sfx_Wood_Creaking.wav", "Sfx_WaterSplash.wav"},
			0x71: {"Sfx_Wood_Creaking.wav", "Sfx_WaterSplash.wav"},
			0x85: {"Sfx_Explosion_Misc2.wav"},
			0x87: {"Sfx_Explosion_Bass.wav"},
		},
		"S102_RodRuettel": {
			0x1b: {"Sfx_Wood_Creaking.wav"}, 7: {"Sfx_Wood_Creaking.wav"}, 0x30: {"Sfx_Metal_DoubleHit.wav"},
		},
	}
	for id, frames := range events {
		layer, ok := ctx.layer(id)
		if !ok || layer == nil {
			continue
		}
		for frame, sounds := range frames {
			if len(layer.FrameEvents[frame]) != 0 {
				continue
			}
			for _, sound := range sounds {
				sound := sound
				layer.AddFrameEvent(frame, func() { _ = ctx.PlaySFX(sound).Update(0) })
			}
		}
	}
}

func loc23PlayTransient(ctx *Context, layer string) engine.Task {
	return engine.Sequence(
		ctx.ShowLayer(layer),
		ctx.PlayLayer(layer),
		ctx.HideLayer(layer),
	)
}

func loc23RebuildState(ctx *Context) engine.Task {
	crate := loc23Original(ctx, loc23StateCrate)
	doorOpen := loc23Original(ctx, loc23StateDoorOpen) != 0
	tasks := []engine.Task{
		ctx.EnableArea("S102_To100"),
		ctx.DisableArea("S102_To103"),
		ctx.HideLayer("S102_KisteLinks"),
		ctx.HideLayer("S102_KisteRechts"),
		ctx.HideLayer("S102_TuerZu"),
		ctx.HideLayer("S102_TuerOffen"),
	}
	if doorOpen {
		tasks = append(tasks, ctx.ShowLayer("S102_TuerOffen"))
	} else {
		tasks = append(tasks, ctx.ShowLayer("S102_TuerZu"))
	}
	if crate == 1 {
		tasks = append(tasks, ctx.ShowLayer("S102_KisteLinks"), ctx.MakeLayerClickable("S102_KisteLinks"), ctx.DisableArea("S102_TuerZu"))
	} else {
		tasks = append(tasks, ctx.ShowLayer("S102_KisteRechts"), ctx.MakeLayerClickable("S102_KisteRechts"))
		if doorOpen {
			tasks = append(tasks, ctx.EnableArea("S102_To103"))
		} else {
			tasks = append(tasks, ctx.MakeLayerClickable("S102_TuerZu"))
		}
	}
	return engine.Sequence(tasks...)
}

func loc23Boarding(ctx *Context, actor string) engine.Task {
	return engine.Sequence(
		ctx.HideActor(actor),
		loc23PlayTransient(ctx, "S102_EnterShip"),
		ctx.ShowLayer("SabWalk"),
		loc08SetLayerTransform(ctx, "SabWalk", 358, 151, 1, 238),
		ctx.FreezeLayer("SabWalk", 0x34),
		ctx.MakeLayerClickable("SabWalk"),
		ctx.PlaceActorPerspective(actor, 0x83, 0x153),
		ctx.ShowActor(actor),
		ctx.WalkToFacingPerspective(actor, 0x83, 0x14a, 4),
	)
}

func loc23EscapeToShore(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.HideActor(loc10ActorID(ctx)),
		loc23PlayTransient(ctx, "S102_Escape"),
		ctx.ChangeLocation(8, "100"),
	)
}

func (LOC23Controller) LoadConditionMask(*Context, string) int { return 0 }

func (LOC23Controller) Exit(*Context, string, string) engine.Task { return nil }

func (LOC23Controller) Enter(ctx *Context, scene, from string) engine.Task {
	if loc23SceneID(scene) != "S102" {
		return nil
	}
	loc23EnsureLayers(ctx)
	loc23BindFrameSFX(ctx)
	actor := loc10ActorID(ctx)
	tasks := []engine.Task{
		ctx.PlayMusic("Loc23_PirateShip.wav"),
		ctx.PlaySFX("Sfx_Ocean.wav"),
	}
	if loc08SceneID(from) == "S100" {
		tasks = append(tasks, loc23RebuildState(ctx))
		if actor != "" {
			tasks = append(tasks, loc23Boarding(ctx, actor))
		}
		return engine.Sequence(tasks...)
	}
	if strings.TrimSpace(from) == "103" || strings.EqualFold(strings.TrimSpace(from), "S103") {
		tasks = append(tasks, loc23EscapeToShore(ctx))
		return engine.Sequence(tasks...)
	}
	tasks = append(tasks, loc23RebuildState(ctx))
	if actor != "" {
		tasks = append(tasks, ctx.ShowActor(actor))
	}
	return engine.Sequence(tasks...)
}

func loc23MoveCrateRight(ctx *Context, actor string) engine.Task {
	tasks := []engine.Task{
		ctx.WalkToFacingPerspective(actor, 0x165, 0xf4, 7),
		ctx.HideActor(actor),
		ctx.HideLayer("S102_KisteLinks"),
		loc23PlayTransient(ctx, "S102_KisteNachRechts"),
		ctx.ShowLayer("S102_KisteRechts"),
		ctx.ShowActor(actor),
		ctx.PlaceActorPerspective(actor, 0x1a7, 0x10a),
		loc23SetOriginal(ctx, loc23StateCrate, 2),
		ctx.PlayVoiceover("102_ROD_02", ""),
		ctx.EnableArea("S102_To103"),
	}
	if loc23Original(ctx, loc23StateDoorOpen) == 0 {
		tasks = append(tasks, ctx.MakeLayerClickable("S102_TuerZu"))
	}
	return engine.Sequence(tasks...)
}

func loc23MoveCrateLeft(ctx *Context, actor string) engine.Task {
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0x19e, 0xf1, 7),
		ctx.HideActor(actor),
		ctx.HideLayer("S102_KisteRechts"),
		loc23PlayTransient(ctx, "S102_KisteNachLinks"),
		ctx.ShowLayer("S102_KisteLinks"),
		ctx.ShowActor(actor),
		ctx.PlaceActorPerspective(actor, 0x16b, 0xf7),
		ctx.WalkToFacingPerspective(actor, 0x157, 0x10b, 7),
		loc23SetOriginal(ctx, loc23StateCrate, 1),
		ctx.PlayVoiceover("102_ROD_02", ""),
		ctx.DisableArea("S102_To103"),
		ctx.DisableArea("S102_TuerZu"),
	)
}

func loc23OpenCabinDoor(ctx *Context, actor string) engine.Task {
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0x13b, 0x163, 7),
		ctx.PlayVoiceover("102_SAB_03", ""),
		ctx.HideActor(actor),
		ctx.ShowLayer("SabWalk"),
		ctx.PlayLayerFrames("SabWalk", 0x30, 0x3b),
		ctx.FreezeLayer("SabWalk", 0x3c),
		ctx.PlayLayerFrames("SabWalk", 0x3c, 0x47),
		ctx.PlayLayerFrames("SabWalk", 0x48, 0x53),
		ctx.HideLayer("SabWalk"),
		ctx.HideLayer("S102_TuerZu"),
		loc23PlayTransient(ctx, "S102_SabTuer"),
		ctx.ShowLayer("S102_TuerOffen"),
		ctx.ShowLayer("SabWalk"),
		ctx.PlayLayerFrames("SabWalk", 0x18, 0x23),
		ctx.PlayLayerFrames("SabWalk", 0x24, 0x2f),
		ctx.FreezeLayer("SabWalk", 0x0c),
		ctx.ShowActor(actor),
		ctx.PlaceActorPerspective(actor, 0x118, 0xf8),
		ctx.EnableArea("S102_To103"),
		loc23SetOriginal(ctx, loc23StateDoorOpen, 1),
	)
}

func (LOC23Controller) Click(ctx *Context, area string) engine.Task {
	if loc23SceneID(ctx.session.state.Scene) != "S102" {
		return nil
	}
	actor := loc10ActorID(ctx)
	switch area {
	case "S102_To100":
		return ctx.PlayVoiceover("102_SAB_05", "")
	case "S102_To103":
		if loc23Original(ctx, loc23StateCrate) != 2 {
			return nil
		}
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x19e, 0xf6, 5), ctx.ChangeLocation(24, "103"))
	case "S102_KisteLinks":
		if loc23Original(ctx, loc23StateCrate) == 1 {
			return loc23MoveCrateRight(ctx, actor)
		}
	case "S102_KisteRechts":
		if loc23Original(ctx, loc23StateCrate) != 1 {
			return loc23MoveCrateLeft(ctx, actor)
		}
	case "SabWalk":
		if loc23Original(ctx, loc23StateDoorLocked) != 0 {
			return nil
		}
		if loc23Original(ctx, loc23StateDoorOpen) == 0 && loc23Original(ctx, loc23StateCrate) == 2 {
			return loc23OpenCabinDoor(ctx, actor)
		}
		return ctx.PlayVoiceover("102_SAB_04", "")
	case "S102_TuerZu":
		if loc23Original(ctx, loc23StateCrate) == 2 && loc23Original(ctx, loc23StateDoorOpen) == 0 {
			return engine.Sequence(
				ctx.WalkToFacingPerspective(actor, 0x19e, 0xf6, 5),
				ctx.HideActor(actor),
				loc23PlayTransient(ctx, "S102_RodRuettel"),
				ctx.ShowActor(actor),
				ctx.PlayVoiceover("102_ROD_01", ""),
			)
		}
	}
	return nil
}

func (LOC23Controller) UseItem(*Context, int, string) engine.Task { return nil }

func (LOC23Controller) SelectItem(*Context, int) engine.Task { return nil }
