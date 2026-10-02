package game

import (
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/wok/rent-a-hero/internal/engine"
)

const (
	loc30Scene34        = "S34"
	loc30StateWindow    = 0x40fc
	loc30StateCanEscape = 0x4100
	loc30StateTalkA     = 0x4104
	loc30StateTalkB     = 0x4108
)

type LOC30Controller struct{}

func loc30SceneID(scene string) string {
	s := strings.ToUpper(strings.TrimSpace(scene))
	s = strings.TrimPrefix(s, "S")
	if n, err := strconv.Atoi(s); err == nil && n == 34 {
		return loc30Scene34
	}
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(scene)), "S") {
		return strings.ToUpper(strings.TrimSpace(scene))
	}
	return s
}

func loc30Actor(ctx *Context) string {
	if actor := ctx.session.PlayerActor(); actor != nil {
		return actor.ID
	}
	return "RodrigoSmall"
}

func loc30Original(ctx *Context, off int) int {
	return getOriginalFlag(ctx.session.state.OriginalState, off)
}

func loc30SetOriginal(ctx *Context, off, value int) engine.Task {
	return engine.Immediate(func() { putOriginalFlag(ctx.session.state.OriginalState, off, value) })
}

func loc30FrameStart(frame int) int {
	if frame > 0 {
		return frame - 1
	}
	return frame
}

func loc30PlayRange(ctx *Context, id string, from, to int) engine.Task {
	return ctx.PlayLayerFrames(id, loc30FrameStart(from), to)
}

func loc30ShowStatic(ctx *Context, id string, frame int) engine.Task {
	return engine.Sequence(ctx.FreezeLayer(id, loc30FrameStart(frame)), ctx.ShowLayer(id))
}

func loc30PlayHide(ctx *Context, id string) engine.Task {
	return engine.Sequence(ctx.ShowLayer(id), ctx.PlayLayer(id), ctx.HideLayer(id))
}

func loc30Jas(ctx *Context, line string) engine.Task {
	if _, ok := ctx.layer("S34_JasStehTalk"); !ok {
		return ctx.PlayVoiceover(line, "["+line+"]")
	}
	return ctx.PlaySpeechBoundToLayer("S34_JasStehTalk", line, "["+line+"]", -1, -1)
}

func loc30JasRange(ctx *Context, line string, start, end int) engine.Task {
	if _, ok := ctx.layer("S34_JasStehTalk"); !ok {
		return ctx.PlayVoiceover(line, "["+line+"]")
	}
	return ctx.PlaySpeechBoundToLayer("S34_JasStehTalk", line, "["+line+"]", loc30FrameStart(start), end)
}

func loc30Rod(ctx *Context, line string) engine.Task {
	return ctx.Say(loc30Actor(ctx), line, "["+line+"]")
}

func loc30ClearInventory(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		ctx.session.state.Inventory = nil
		ctx.session.ClearSelectedItem()
	})
}

func (LOC30Controller) LoadConditionMask(*Context, string) int { return 0 }

func (LOC30Controller) Enter(ctx *Context, scene string, from string) engine.Task {
	if loc30SceneID(scene) != loc30Scene34 {
		return nil
	}

	fromID := loc30SceneID(from)
	if fromID != "S35" {
		_, _, _ = ctx.ensureAssetLayer("S34_RodReinCam")
	}

	tasks := []engine.Task{
		ctx.PlayMusic("Loc30_PrisionInEndavin.wav"),
		ctx.HideLayer("S34_JasAufsteh"),
		ctx.FreezeLayer("S34_JasAufsteh", 0),
		ctx.HideLayer("S34_JasStehTalk"),
		ctx.FreezeLayer("S34_JasStehTalk", 0),
		ctx.HideLayer("S34_RodKnock"),
		ctx.FreezeLayer("S34_RodKnock", 0),
		ctx.HideLayer("S34_GridOpen"),
		ctx.FreezeLayer("S34_GridOpen", 0),
		ctx.HideLayer("S34_RodOpensGrid"),
		ctx.FreezeLayer("S34_RodOpensGrid", 0),
		ctx.HideLayer("S34_RodInSchacht"),
		ctx.FreezeLayer("S34_RodInSchacht", 0),
		ctx.HideLayer("S34_RodSchautStroh"),
		ctx.FreezeLayer("S34_RodSchautStroh", 0),
		loc30ShowStatic(ctx, "S34_Sun", 0),
		loc30ShowStatic(ctx, "S34_Straw", 0),
		ctx.EnableArea("S34_Door"),
		ctx.EnableArea("S34_Ring1"),
		ctx.EnableArea("S34_Ring2"),
		ctx.EnableArea("S34_Plate"),
		ctx.EnableArea("S34_Window"),
		ctx.MakeLayerClickable("S34_Straw"),
		ctx.MakeLayerClickable("S34_JasStehTalk"),
		ctx.DisableArea("S34_JasStehTalk"),
	}

	if fromID == "S35" {
		tasks = append(tasks,
			ctx.ShowActor(loc30Actor(ctx)),
			ctx.PlaceActor(loc30Actor(ctx), 0xb6, 0x122),
			loc30ShowStatic(ctx, "S34_JasStehTalk", 0),
			ctx.EnableArea("S34_JasStehTalk"),
		)
	} else {
		tasks = append(tasks,
			ctx.HideActor(loc30Actor(ctx)),
			ctx.ShowLayer("S34_RodReinCam"),
			ctx.PlayLayerFrames("S34_RodReinCam", 0, 0xe),
			ctx.PlaySFX("Sfx_Tock.wav"),
			ctx.PlayLayerFrames("S34_RodReinCam", 0xf, 0x18),
			ctx.PlaySFX("Sfx_Knock_Wet.wav"),
			ctx.PlayLayerFrames("S34_RodReinCam", 0x19, -1),
			ctx.HideLayer("S34_RodReinCam"),
			loc30ClearInventory(ctx),
			ctx.ShowActor(loc30Actor(ctx)),
			ctx.PlaceActor(loc30Actor(ctx), 0x118, 0x122),
			ctx.RunAmbient(loc30RiseJas(ctx)),
			ctx.WalkToFacing(loc30Actor(ctx), 0x112, 0x132, 0),
			loc30Rod(ctx, "034_ROD_01"),
		)
	}

	return engine.Sequence(tasks...)
}

func loc30RiseJas(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.ShowLayer("S34_JasAufsteh"),
		ctx.PlayLayer("S34_JasAufsteh"),
		ctx.HideLayer("S34_JasAufsteh"),
		loc30Jas(ctx, "034_JAS_01"),
		loc30ShowStatic(ctx, "S34_JasStehTalk", 0),
		ctx.EnableArea("S34_JasStehTalk"),
	)
}

func (LOC30Controller) Exit(*Context, string, string) engine.Task { return nil }

func (LOC30Controller) Click(ctx *Context, area string) engine.Task {
	if loc30SceneID(ctx.session.state.Scene) != loc30Scene34 {
		return nil
	}
	switch area {
	case "S34_Door":
		return engine.Sequence(
			ctx.WalkToFacing(loc30Actor(ctx), 0x128, 0x119, 5),
			ctx.HideActor(loc30Actor(ctx)),
			ctx.RunAmbient(loc30JasRange(ctx, "034_JAS_12", 0, 4)),
			ctx.ShowLayer("S34_RodKnock"),
			ctx.PlayLayerFrames("S34_RodKnock", 0, 0xb),
			ctx.PlaySFX("Sfx_Tock.wav"),
			ctx.PlayLayerFrames("S34_RodKnock", 0xc, 0x13),
			ctx.PlaySFX("Sfx_Tock.wav"),
			ctx.PlayLayerFrames("S34_RodKnock", 0x14, -1),
			ctx.HideLayer("S34_RodKnock"),
			ctx.ShowActor(loc30Actor(ctx)),
			ctx.PlaceActor(loc30Actor(ctx), 0x128, 0x122),
			ctx.FreezeLayer("S34_JasStehTalk", 0),
		)
	case "S34_Ring1":
		return engine.Sequence(
			ctx.WalkToFacing(loc30Actor(ctx), 0x1ba, 0x148, 4),
			loc30Jas(ctx, "034_JAS_16"),
		)
	case "S34_Ring2":
		return engine.Sequence(
			ctx.WalkToFacing(loc30Actor(ctx), 0x1c8, 0x14b, 7),
			loc30Jas(ctx, "034_JAS_17"),
		)
	case "S34_Plate":
		return engine.Sequence(
			ctx.WalkToFacing(loc30Actor(ctx), 0x1c6, 0x153, 7),
			loc30Jas(ctx, "034_JAS_15"),
		)
	case "S34_Window":
		return engine.Sequence(
			ctx.WalkToFacing(loc30Actor(ctx), 0xb6, 0x122, 4),
			loc30SetOriginal(ctx, loc30StateWindow, 1),
			ctx.ChangeLocation(28, "35"),
		)
	case "S34_JasStehTalk":
		return loc30TalkJas(ctx)
	case "S34_Straw":
		return loc30UseStraw(ctx)
	}
	return nil
}

func loc30TalkJas(ctx *Context) engine.Task {
	walk := ctx.WalkToFacing(loc30Actor(ctx), 0x165, 300, 6)
	if loc30Original(ctx, loc30StateWindow) == 0 {
		switch loc30Original(ctx, loc30StateTalkA) {
		case 1:
			return engine.Sequence(walk, loc30SetOriginal(ctx, loc30StateTalkA, 2), loc30Rod(ctx, "034_ROD_02"), loc30Jas(ctx, "034_JAS_02"))
		case 2:
			return engine.Sequence(walk, loc30SetOriginal(ctx, loc30StateTalkA, 3), loc30Rod(ctx, "034_ROD_03"), loc30Jas(ctx, "034_JAS_03"))
		case 3:
			n := rand.IntN(3) + 1
			switch n {
			case 1:
				return engine.Sequence(walk, loc30Rod(ctx, "034_ROD_04"), loc30Jas(ctx, "034_JAS_04"))
			case 2:
				return engine.Sequence(walk, loc30Rod(ctx, "034_ROD_05"), loc30Jas(ctx, "034_JAS_05"))
			default:
				line := "034_JAS_06"
				if rand.IntN(2) == 0 {
					line = "034_JAS_01"
				}
				return engine.Sequence(walk, loc30Rod(ctx, "034_ROD_06"), loc30Jas(ctx, line))
			}
		}
		return walk
	}

	switch loc30Original(ctx, loc30StateTalkB) {
	case 1:
		return engine.Sequence(walk, loc30SetOriginal(ctx, loc30StateTalkB, 2), loc30Rod(ctx, "034_ROD_07"), loc30Jas(ctx, "034_JAS_07"))
	case 2:
		return engine.Sequence(walk, loc30SetOriginal(ctx, loc30StateTalkB, 3), loc30Rod(ctx, "034_ROD_08"), loc30Jas(ctx, "034_JAS_08"))
	case 3:
		return engine.Sequence(
			walk,
			loc30SetOriginal(ctx, loc30StateTalkB, 4),
			loc30Rod(ctx, "034_ROD_09"),
			loc30JasRange(ctx, "034_JAS_09", 0, 4),
			loc30Jas(ctx, "034_JAS_10"),
		)
	case 4:
		return engine.Sequence(
			walk,
			loc30SetOriginal(ctx, loc30StateCanEscape, 1),
			loc30Rod(ctx, "034_ROD_11"),
			loc30Jas(ctx, "034_JAS_11"),
		)
	}
	return walk
}

func loc30UseStraw(ctx *Context) engine.Task {
	if loc30Original(ctx, loc30StateCanEscape) == 0 {
		return engine.Sequence(
			ctx.WalkToFacing(loc30Actor(ctx), 0xe5, 0x13b, 7),
			loc30JasRange(ctx, "034_JAS_13", 0, 4),
			ctx.HideActor(loc30Actor(ctx)),
			ctx.ShowLayer("S34_RodSchautStroh"),
			ctx.PlayLayer("S34_RodSchautStroh"),
			ctx.HideLayer("S34_RodSchautStroh"),
			ctx.ShowActor(loc30Actor(ctx)),
			ctx.FreezeLayer("S34_JasStehTalk", 0),
		)
	}

	return engine.Sequence(
		ctx.WalkToFacing(loc30Actor(ctx), 0xe5, 0x13b, 7),
		loc30JasRange(ctx, "034_JAS_14", 0, 4),
		ctx.HideActor(loc30Actor(ctx)),
		ctx.HideLayer("S34_Straw"),
		ctx.DisableArea("S34_Straw"),
		ctx.ShowLayer("S34_RodOpensGrid"),
		ctx.PlayLayerFrames("S34_RodOpensGrid", 0, 0x18),
		ctx.PlaySFX("Sfx_Box_Pushed_Short.wav"),
		ctx.PlayLayerFrames("S34_RodOpensGrid", 0x19, 0x29),
		ctx.AddItem(0xf),
		ctx.PlayLayerFrames("S34_RodOpensGrid", loc30FrameStart(0x2a), 0x42),
		ctx.PlaySFX("Sfx_Winding.wav"),
		ctx.PlayLayerFrames("S34_RodOpensGrid", 0x43, -1),
		ctx.HideLayer("S34_RodOpensGrid"),
		ctx.FreezeLayer("S34_JasStehTalk", 0),
		ctx.RunAmbient(ctx.PlayVoiceover("034_ROD_12", "[034_ROD_12]")),
		ctx.ShowLayer("S34_RodInSchacht"),
		ctx.PlayLayerFrames("S34_RodInSchacht", 0, 0x16),
		ctx.PlaySFX("Sfx_Clothes_Scratched.wav"),
		ctx.PlayLayerFrames("S34_RodInSchacht", 0x17, -1),
		ctx.HideLayer("S34_RodInSchacht"),
		ctx.ChangeLocation(28, "36"),
	)
}

func (LOC30Controller) UseItem(*Context, int, string) engine.Task { return nil }
func (LOC30Controller) SelectItem(*Context, int) engine.Task      { return nil }
