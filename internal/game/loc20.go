package game

import (
	"github.com/wok/rent-a-hero/internal/engine"
	"math/rand/v2"
)

const (
	loc20StateStoneTaken = 0x3fec
	loc20StateIntro      = 0x4020
	loc20StateStone      = 0x4024
	loc20StateTo71       = 0x4028
	loc20StateCanLeave   = 0x402c
	loc20StateTalk       = 0x4030
)

type LOC20Controller struct{}

func loc20Original(ctx *Context, off int) int {
	return getOriginalFlag(ctx.session.state.OriginalState, off)
}

func loc20SetOriginal(ctx *Context, off, value int) engine.Task {
	return engine.Immediate(func() { putOriginalFlag(ctx.session.state.OriginalState, off, value) })
}

func loc20LayerSpeech(ctx *Context, id, line string, from, to int) engine.Task {
	start := from
	if start > 0 {
		start--
	}
	return ctx.PlaySpeechBoundToLayer(id, line, "["+line+"]", start, to)
}

func loc20Dormant70(ctx *Context) engine.Task {
	ids := []string{"S70_RodCatchStone", "S70_Tharain", "S70_StoneFlies", "S70_ThaTakesStone"}
	tasks := make([]engine.Task, 0, len(ids)*2)
	for _, id := range ids {
		if _, ok := ctx.layer(id); ok {
			tasks = append(tasks, ctx.HideLayer(id), ctx.FreezeLayer(id, 0))
		}
	}
	return engine.Sequence(tasks...)
}

func loc20ShowLoop(ctx *Context, id string) engine.Task {
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

func loc20PlayHide(ctx *Context, id string) engine.Task {
	return engine.Sequence(
		ctx.ShowLayer(id),
		ctx.PlayLayer(id),
		ctx.HideLayer(id),
	)
}

func loc20PlayRangeHide(ctx *Context, id string, from, to int) engine.Task {
	start := from
	if start > 0 {
		start--
	}
	return engine.Sequence(
		ctx.PlayLayerFrames(id, start, to),
		ctx.HideLayer(id),
	)
}

func (LOC20Controller) LoadConditionMask(ctx *Context, scene string) int {
	if normalizeLocationSceneID(20, scene) == "S70" && loc20Original(ctx, loc20StateStoneTaken) == 0 {
		return 1
	}
	return 0
}
func (LOC20Controller) Exit(*Context, string, string) engine.Task { return nil }
func (LOC20Controller) UseItem(*Context, int, string) engine.Task { return nil }
func (LOC20Controller) SelectItem(*Context, int) engine.Task      { return nil }

func (LOC20Controller) Enter(ctx *Context, scene, from string) engine.Task {
	if normalizeLocationSceneID(20, scene) != "S70" {
		return nil
	}
	actor := loc10ActorID(ctx)
	tasks := []engine.Task{
		ctx.PlayMusic("Loc20_Library.wav"),
		ctx.ShowActor(actor),
		loc20Dormant70(ctx),
		ctx.EnableArea("S70_To69"),
		ctx.ShowLayer("S70_Pult"),
		ctx.ShowLayer("S70_Book"),
		ctx.ShowLayer("S70_Blue"),
		ctx.ShowLayer("S70_Cyan"),
		ctx.ShowLayer("S70_Green"),
		ctx.ShowLayer("S70_Red"),
		ctx.ShowLayer("S70_Violet"),
		ctx.ShowLayer("S70_Yellow"),
	}
	if loc20Original(ctx, loc20StateIntro) == 0 {
		colors := []string{"S70_Blue", "S70_Cyan", "S70_Green", "S70_Red", "S70_Violet", "S70_Yellow"}
		tasks = append(tasks, ctx.HideLayer(colors[rand.IntN(len(colors))]))
	}
	if loc20Original(ctx, loc20StateStoneTaken) != 0 {
		tasks = append(tasks, ctx.EnableArea("S70_Boden"))
	} else {
		tasks = append(tasks,
			ctx.DisableArea("S70_Boden"),
			loc20ShowLoop(ctx, "S70_Tharain"),
			ctx.MakeLayerClickable("S70_Tharain"),
			ctx.MakeLayerClickable("S70_Book"),
			ctx.MakeLayerClickable("S70_Pult"),
		)
		if loc20Original(ctx, loc20StateTalk) < 6 {
			tasks = append(tasks, ctx.MakeLayerClickable("S70_Stone"))
		} else {
			tasks = append(tasks, ctx.HideLayer("S70_Stone"), ctx.FreezeLayer("S70_Stone", 0))
		}
	}
	if ctx.session.state.PreviousLocation != 20 || loc19SceneID(from) == "S69" {
		tasks = append(tasks, ctx.WalkToFacingPerspective(actor, 0xfd, 0x159, 5))
	}
	if loc20Original(ctx, loc20StateIntro) != 0 {
		tasks = append(tasks,
			loc20LayerSpeech(ctx, "S70_Tharain", "070_THA_01", 0xf, 0x2d),
			ctx.Say(actor, "070_ROD_01", "[070_ROD_01]"),
			loc20LayerSpeech(ctx, "S70_Tharain", "070_THA_02", 0xf, 0x2d),
			loc20SetOriginal(ctx, loc20StateIntro, 0),
		)
	}
	return engine.Sequence(tasks...)
}

func (LOC20Controller) Click(ctx *Context, area string) engine.Task {
	if normalizeLocationSceneID(20, ctx.session.state.Scene) != "S70" {
		return nil
	}
	actor := loc10ActorID(ctx)
	switch area {
	case "S70_To69":
		if loc20Original(ctx, loc20StateCanLeave) == 0 {
			return engine.Sequence(
				ctx.WalkToFacingPerspective(actor, 0x7c, 0x150, 3),
				loc20LayerSpeech(ctx, "S70_Tharain", "070_THA_13", 0xf, 0x2d),
				ctx.WalkToFacingPerspective(actor, 0xfd, 0x159, 5),
				ctx.SetActorOrientation(actor, 0),
				ctx.Say(actor, "070_ROD_13", "[070_ROD_13]"),
				ctx.SetActorOrientation(actor, 4),
			)
		}
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0x1c, 0x138, 2),
			ctx.ChangeLocation(19, "69"),
		)
	case "S70_Boden":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x1b3, 0x132, 1), ctx.Say(actor, "070_ROD_14", "[070_ROD_14]"))
	case "S70_Tharain", "S70_Book", "S70_Pult", "S70_Stone":
		return loc20Talk(ctx, actor)
	}
	return nil
}

func loc20Talk(ctx *Context, actor string) engine.Task {
	base := []engine.Task{ctx.WalkToFacingPerspective(actor, 0xfd, 0x159, 5)}
	switch loc20Original(ctx, loc20StateTalk) {
	case 1:
		return engine.Sequence(append(base,
			ctx.Say(actor, "070_ROD_03", "[070_ROD_03]"),
			loc20LayerSpeech(ctx, "S70_Tharain", "070_THA_03", 0xf, 0x2d),
			loc20SetOriginal(ctx, loc20StateTalk, 2),
		)...)
	case 2:
		return engine.Sequence(append(base,
			ctx.Say(actor, "070_ROD_04", "[070_ROD_04]"),
			loc20LayerSpeech(ctx, "S70_Tharain", "070_THA_04", 0xf, 0x2d),
			loc20SetOriginal(ctx, loc20StateTalk, 3),
		)...)
	case 3:
		return engine.Sequence(append(base,
			ctx.Say(actor, "070_ROD_05", "[070_ROD_05]"),
			loc20LayerSpeech(ctx, "S70_Tharain", "070_THA_05", 0xf, 0x2d),
			ctx.Say(actor, "070_ROD_06", "[070_ROD_06]"),
			loc20SetOriginal(ctx, loc20StateTalk, 4),
		)...)
	case 4:
		return engine.Sequence(append(base,
			ctx.Say(actor, "070_ROD_07", "[070_ROD_07]"),
			loc20LayerSpeech(ctx, "S70_Tharain", "070_THA_07", 0xf, 0x2d),
			loc20SetOriginal(ctx, loc20StateTalk, 5),
		)...)
	case 5:
		return engine.Sequence(append(base,
			ctx.Say(actor, "070_ROD_08", "[070_ROD_08]"),
			loc20LayerSpeech(ctx, "S70_Tharain", "070_THA_08", 0xf, 0x2d),
			ctx.HideLayer("S70_Stone"),
			ctx.RunAmbient(loc20PlayHide(ctx, "S70_ThaTakesStone")),
			ctx.PlayLayerFrames("S70_Tharain", 0, 0xe),
			ctx.ShowLayer("S70_StoneFlies"),
			loc20SetOriginal(ctx, loc20StateTalk, 6),
		)...)
	case 6:
		return engine.Sequence(append(base,
			ctx.Say(actor, "070_ROD_09", "[070_ROD_09]"),
			loc20LayerSpeech(ctx, "S70_Tharain", "070_THA_09", 0xf, 0x2d),
			ctx.RunAmbient(loc20LayerSpeech(ctx, "S70_Tharain", "070_THA_10", 0xf, 0x2d)),
			ctx.RunAmbient(ctx.WalkToFacingPerspective(actor, 0x126, 0x138, 5)),
			ctx.PlayLayerFrames("S70_StoneFlies", 0, 0xc),
			ctx.HideActor(actor),
			ctx.RunAmbient(loc20PlayRangeHide(ctx, "S70_StoneFlies", 0xd, -1)),
			loc20PlayHide(ctx, "S70_RodCatchStone"),
			ctx.ShowActor(actor),
			ctx.WalkToFacingPerspective(actor, 0xfd, 0x159, 5),
			ctx.AddItem(0x15),
			loc20SetOriginal(ctx, loc20StateTalk, 7),
		)...)
	case 7:
		return engine.Sequence(append(base,
			ctx.Say(actor, "070_ROD_11", "[070_ROD_11]"),
			loc20LayerSpeech(ctx, "S70_Tharain", "070_THA_11", 0xf, 0x2d),
			loc20SetOriginal(ctx, loc20StateTalk, 8),
		)...)
	case 8:
		return engine.Sequence(append(base,
			ctx.Say(actor, "070_ROD_12", "[070_ROD_12]"),
			loc20LayerSpeech(ctx, "S70_Tharain", "070_THA_12", 0xf, 0x2d),
			loc20SetOriginal(ctx, loc20StateStone, 1),
			loc20SetOriginal(ctx, loc20StateTo71, 1),
			loc20SetOriginal(ctx, loc20StateCanLeave, 1),
		)...)
	}
	return nil
}
