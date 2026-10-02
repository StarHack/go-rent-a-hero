package game

import (
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/wok/rent-a-hero/internal/engine"
)

const (
	loc28Scene31 = "S31"
	loc28Scene32 = "S32"
	loc28Scene35 = "S35"

	loc28StateVisited      = 0x3e8c
	loc28StateCaptainTalk  = 0x40c4
	loc28StateReturnLine   = 0x40c8
	loc28StateScene32Line  = 0x40cc
	loc28StateGuardReturn  = 0x40d0
	loc28StateWagon        = 0x40d4
	loc28StateStreetTalk   = 0x40d8
	loc28StateGuardTalk    = 0x40dc
	loc28StateFredericSeen = 0x40e0
	loc28StateDisguise     = 0x40e4
	loc28StateBushTalk     = 0x40e8
	loc28StateInsideCity   = 0x40f0
	loc28ItemWagonPickup   = 0x19
)

type LOC28Controller struct{}

func loc28SceneID(scene string) string {
	s := strings.ToUpper(strings.TrimSpace(scene))
	s = strings.TrimPrefix(s, "S")
	if n, err := strconv.Atoi(s); err == nil {
		switch n {
		case 30, 31, 36:
			return loc28Scene31
		case 32:
			return loc28Scene32
		case 35:
			return loc28Scene35
		}
	}
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(scene)), "S") {
		return strings.ToUpper(strings.TrimSpace(scene))
	}
	return s
}

func loc28Original(ctx *Context, off int) int {
	return getOriginalFlag(ctx.session.state.OriginalState, off)
}

func loc28SetOriginal(ctx *Context, off, value int) engine.Task {
	return engine.Immediate(func() { putOriginalFlag(ctx.session.state.OriginalState, off, value) })
}

func loc28Actor(ctx *Context) string {
	if actor := ctx.session.PlayerActor(); actor != nil {
		return actor.ID
	}
	return "RodrigoSmall"
}

func loc28FrameStart(frame int) int {
	if frame > 0 {
		return frame - 1
	}
	return frame
}

func loc28LayerSpeech(ctx *Context, layerID, line string, start, end int) engine.Task {
	if _, ok := ctx.layer(layerID); !ok {
		return ctx.PlayVoiceover(line, "["+line+"]")
	}
	return ctx.PlaySpeechBoundToLayer(layerID, line, "["+line+"]", loc28FrameStart(start), end)
}

func loc28Rod(ctx *Context, line string) engine.Task {
	return ctx.Say(loc28Actor(ctx), line, "["+line+"]")
}

func loc28ShowStatic(ctx *Context, id string, frame int) engine.Task {
	return engine.Sequence(ctx.FreezeLayer(id, loc28FrameStart(frame)), ctx.ShowLayer(id))
}

func loc28PlayHide(ctx *Context, id string) engine.Task {
	return engine.Sequence(ctx.ShowLayer(id), ctx.PlayLayer(id), ctx.HideLayer(id))
}

func loc28PlayRange(ctx *Context, id string, from, to int) engine.Task {
	return ctx.PlayLayerFrames(id, loc28FrameStart(from), to)
}

func loc28PlayRangeHide(ctx *Context, id string, from, to int) engine.Task {
	return engine.Sequence(loc28PlayRange(ctx, id, from, to), ctx.HideLayer(id))
}

func loc28Hide31ScriptLayers(ctx *Context) engine.Task {
	ids := []string{
		"S31_EnterCity", "S31_FreArrives", "S31_FreEntersCity", "S31_FreParole", "S31_FreTalk",
		"S31_JasClimbsOut", "S31_PirGlider", "S31_RodBuschRunterRauf", "S31_RodBushTalk",
		"S31_RodClimbsOut", "S31_RodEntersBush", "S31_RodGliderAnim", "S31_RodLeavesBush",
		"S31_RodPirBushRunterRauf", "S31_RodPirBushTalk", "S31_RodPirEntersBush",
		"S31_RodPirLeavesBush", "S31_RodPirParole",
	}
	tasks := make([]engine.Task, 0, len(ids)*2)
	for _, id := range ids {
		if _, ok := ctx.layer(id); ok {
			tasks = append(tasks, ctx.HideLayer(id), ctx.FreezeLayer(id, 0))
		}
	}
	return engine.Sequence(tasks...)
}

func (LOC28Controller) LoadConditionMask(*Context, string) int { return 0 }

func (LOC28Controller) Enter(ctx *Context, scene string, from string) engine.Task {
	switch loc28SceneID(scene) {
	case loc28Scene31:
		return loc28Enter31(ctx, loc28SceneID(from))
	case loc28Scene32:
		return loc28Enter32(ctx, loc28SceneID(from))
	case loc28Scene35:
		return loc28Enter35(ctx)
	}
	return nil
}

func (LOC28Controller) Exit(*Context, string, string) engine.Task { return nil }

func loc28Enter31(ctx *Context, from string) engine.Task {
	actor := loc28Actor(ctx)
	_, _, _ = ctx.ensureAssetLayer("S31_EnterCity")
	tasks := []engine.Task{
		ctx.PlayMusic("Loc28_Endavin.wav"),
		loc28SetOriginal(ctx, loc28StateVisited, 1),
		ctx.HideActor(actor),
		loc28Hide31ScriptLayers(ctx),
		loc28ShowStatic(ctx, "S31_Deckel", 0),
		loc28ShowStatic(ctx, "S31_Wache", 0),
		ctx.MakeLayerClickable("S31_Wache"),
		ctx.EnableArea("S31_To30_113"),
		ctx.EnableArea("S31_To32"),
		ctx.EnableArea("S31_Wall"),
	}

	switch from {
	case loc28Scene32:
		tasks = append(tasks,
			ctx.ShowActor(actor),
			ctx.PlaceActor(actor, 0x104, 0xf4),
			ctx.WalkToFacingPerspective(actor, 200, 0xe9, 2),
			loc28LayerSpeech(ctx, "S31_Wache", "031_PW1_13", 0x1e, 0x2b),
			ctx.WalkToFacingPerspective(actor, 0x5a, 0x106, 6),
			loc28SetOriginal(ctx, loc28StateInsideCity, 1),
		)
	case "34", "S34":
		tasks = append(tasks, loc28Enter31From34(ctx))
	default:
		tasks = append(tasks, loc28Enter31Bush(ctx))
	}

	tasks = append(tasks, ctx.RunAmbient(&loc28BirdTask{ctx: ctx, wait: float64(5 + rand.IntN(11))}))
	return engine.Sequence(tasks...)
}

func loc28Enter31Bush(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.HideActor(loc28Actor(ctx)),
		loc28PlayHide(ctx, "S31_RodEntersBush"),
		loc28ShowStatic(ctx, "S31_RodBushTalk", 0),
		loc28Deferred(func() engine.Task {
			if loc28Original(ctx, loc28StateReturnLine) != 0 {
				return engine.Sequence(
					loc28LayerSpeech(ctx, "S31_RodBushTalk", "031_ROD_03", 0, 8),
					loc28SetOriginal(ctx, loc28StateReturnLine, 0),
				)
			}
			if ctx.HasItem(1) && loc28Original(ctx, loc28StateFredericSeen) == 0 && rand.IntN(3) == 2 {
				return loc28FredericArrival(ctx)
			}
			return nil
		}),
		loc28Deferred(func() engine.Task {
			if ctx.HasItem(1) && ctx.HasItem(4) && ctx.HasItem(3) && ctx.HasItem(0xe) && loc28Original(ctx, loc28StateFredericSeen) == 0 {
				return loc28FredericArrival(ctx)
			}
			return nil
		}),
	)
}

func loc28FredericArrival(ctx *Context) engine.Task {
	return engine.Sequence(
		loc28PlayHide(ctx, "S31_FreArrives"),
		loc28ShowStatic(ctx, "S31_FreTalk", 0),
		loc28LayerSpeech(ctx, "S31_Wache", "031_PW1_01", 0x1e, 0x2b),
		loc28LayerSpeech(ctx, "S31_FreTalk", "031_FRE_01", 0, 8),
		loc28LayerSpeech(ctx, "S31_Wache", "031_PW1_02b", 0x1e, 0x2b),
		ctx.HideLayer("S31_FreTalk"),
		loc28PlayHide(ctx, "S31_FreParole"),
		loc28ShowStatic(ctx, "S31_FreEntersCity", 0),
		loc28LayerSpeech(ctx, "S31_Wache", "031_PW1_02", 0x1e, 0x2b),
		loc28PlayHide(ctx, "S31_FreEntersCity"),
		loc28LayerSpeech(ctx, "S31_RodBushTalk", "031_ROD_02", 0, 8),
		loc28SetOriginal(ctx, loc28StateFredericSeen, 1),
	)
}

func loc28Enter31From34(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.Wait(1),
		ctx.PlayVoiceover("031_ROD_17", "[031_ROD_17]"),
		ctx.PlayVoiceover("031_JAS_17", "[031_JAS_17]"),
		ctx.PlayVoiceover("031_ROD_18", "[031_ROD_18]"),
		ctx.PlayVoiceover("031_JAS_18", "[031_JAS_18]"),
		ctx.PlayVoiceover("031_ROD_19", "[031_ROD_19]"),
		ctx.PlaySFX("Sfx_Winding.wav"),
		ctx.PlayLayer("S31_Deckel"),
		engine.Immediate(func() {
			if layer, ok := ctx.layer("S31_Deckel"); ok {
				layer.X = 500
				layer.Y = 232
				layer.Z = 159
				layer.Zoom = 100
			}
		}),
		ctx.ShowLayer("S31_RodClimbsOut"),
		ctx.RunAmbient(engine.Sequence(
			ctx.PlayLayerFrames("S31_RodClimbsOut", 0, 0x23),
			ctx.PlaySFX("Gen_StepLeft.wav"),
			ctx.PlayLayerFrames("S31_RodClimbsOut", 0x24, 0x27),
			ctx.PlaySFX("Gen_StepRight.wav"),
			ctx.PlayLayerFrames("S31_RodClimbsOut", 0x28, 0x2f),
			ctx.PlaySFX("Gen_StepLeft.wav"),
			ctx.PlayLayerFrames("S31_RodClimbsOut", 0x30, -1),
		)),
		ctx.Wait(1.3),
		ctx.ShowLayer("S31_JasClimbsOut"),
		ctx.PlayLayerFrames("S31_JasClimbsOut", 0, 0x1f),
		ctx.PlaySFX("Gen_StepLeft.wav"),
		ctx.PlayLayerFrames("S31_JasClimbsOut", 0x20, 0x24),
		ctx.PlaySFX("Gen_StepRight.wav"),
		ctx.PlayLayerFrames("S31_JasClimbsOut", 0x25, 0x30),
		ctx.PlaySFX("Gen_StepLeft.wav"),
		ctx.PlayVoiceover("031_PW2_20", "[031_PW2_20]"),
		loc28PlayRange(ctx, "S31_Wache", 0x5b, 100),
		loc28LayerSpeech(ctx, "S31_Wache", "031_PW1_20", -1, -1),
		loc28PlayRange(ctx, "S31_Wache", 0x69, 0x75),
		ctx.ShowLayer("S31_RodGliderAnim"),
		ctx.PlayLayerFrames("S31_RodGliderAnim", 0, 1),
		ctx.PlaySFXVolume("Sfx_Glider_PassingBy.wav", 0x3c),
		ctx.PlayLayerFrames("S31_RodGliderAnim", 2, -1),
		ctx.RunAmbient(loc28PlayRange(ctx, "S31_Wache", 0x76, -1)),
		ctx.ShowLayer("S31_PirGlider"),
		ctx.PlaySFX("Sfx_Glider_PassingBy3.wav"),
		ctx.PlayLayer("S31_PirGlider"),
		ctx.ChangeLocation(10, "66"),
	)
}

type loc28DeferredTask struct {
	build   func() engine.Task
	inner   engine.Task
	started bool
}

func loc28Deferred(build func() engine.Task) engine.Task {
	return &loc28DeferredTask{build: build}
}

func (t *loc28DeferredTask) Update(dt float64) bool {
	if !t.started {
		t.started = true
		t.inner = t.build()
		if t.inner == nil {
			return true
		}
	}
	return t.inner.Update(dt)
}

func loc28Enter32(ctx *Context, from string) engine.Task {
	actor := loc28Actor(ctx)
	tasks := []engine.Task{
		ctx.PlayMusic("Loc28_Endavin.wav"),
		ctx.HideLayer("S32_RodKramtAnim"),
		ctx.FreezeLayer("S32_RodKramtAnim", 0),
		ctx.ShowActor(actor),
		loc28ShowStatic(ctx, "S32_Wache", 0),
		ctx.MakeLayerClickable("S32_Wache"),
		ctx.EnableArea("S32_To31"),
		ctx.EnableArea("S32_Wagen"),
		ctx.EnableArea("S32_FensterLinks"),
		ctx.EnableArea("S32_FensterRechts"),
		ctx.EnableArea("S32_Kneipe"),
		ctx.EnableArea("S32_Treppen"),
		ctx.EnableArea("S32_SchuttLinks"),
		ctx.EnableArea("S32_SchuttRechts"),
		ctx.EnableArea("S32_FassLinks"),
		ctx.EnableArea("S32_FassRechts"),
		ctx.EnableArea("S32_Kerker"),
	}
	if from == loc28Scene31 {
		tasks = append(tasks, ctx.PlaceActor(actor, 0x69, 0x167))
	} else if from == "33" || from == "S33" {
		tasks = append(tasks, ctx.PlaceActor(actor, 0xf7, 0x112))
	}
	if loc28Original(ctx, loc28StateScene32Line) != 0 {
		tasks = append(tasks,
			loc28SetOriginal(ctx, loc28StateScene32Line, 0),
			loc28LayerSpeech(ctx, "S32_Wache", "032_ROD_01", 0, 8),
		)
	}
	return engine.Sequence(tasks...)
}

func loc28Enter35(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.PlayMusic("Loc30_PrisionInEndavin.wav"),
		loc28ShowStatic(ctx, "S35_Fremder", 0),
		loc28ShowStatic(ctx, "S35_Captain", 0),
		loc28ShowStatic(ctx, "S35_Window", 0),
		ctx.EnableArea("S35_To34"),
		ctx.EnableArea("S35_To34_2"),
		ctx.RunAmbient(&loc28S35DialogueTask{ctx: ctx, timer: 1, event: 0x21}),
	)
}

func (LOC28Controller) Click(ctx *Context, area string) engine.Task {
	switch loc28SceneID(ctx.session.state.Scene) {
	case loc28Scene31:
		return loc28Click31(ctx, area)
	case loc28Scene32:
		return loc28Click32(ctx, area)
	case loc28Scene35:
		return loc28Click35(ctx, area)
	}
	return nil
}

func loc28Click31(ctx *Context, area string) engine.Task {
	actor := loc28Actor(ctx)
	switch area {
	case "S31_To30_113":
		if loc28Original(ctx, loc28StateInsideCity) == 0 {
			tasks := []engine.Task{}
			if loc28Original(ctx, loc28StateDisguise) != 0 {
				tasks = append(tasks,
					loc28LayerSpeech(ctx, "S31_RodPirBushTalk", "031_ROD_04", 0, 8),
					ctx.HideLayer("S31_RodPirBushTalk"),
					loc28PlayRangeHide(ctx, "S31_RodPirBushRunterRauf", 0, 10),
					loc28PlayRangeHide(ctx, "S31_RodBuschRunterRauf", 10, 0),
					loc28ShowStatic(ctx, "S31_RodBushTalk", 0),
					loc28SetOriginal(ctx, loc28StateDisguise, 0),
				)
			}
			tasks = append(tasks,
				ctx.HideLayer("S31_RodBushTalk"),
				loc28PlayHide(ctx, "S31_RodLeavesBush"),
				ctx.ChangeLocation(31, "S113"),
			)
			return engine.Sequence(tasks...)
		}
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 4, 0xa0, 2),
			loc28SetOriginal(ctx, loc28StateInsideCity, 0),
			ctx.HideActor(actor),
			ctx.Wait(2),
			loc28PlayHide(ctx, "S31_RodPirEntersBush"),
			loc28ShowStatic(ctx, "S31_RodPirBushTalk", 0),
		)
	case "S31_To32":
		if loc28Original(ctx, loc28StateInsideCity) != 0 {
			return engine.Sequence(
				ctx.WalkToFacingPerspective(actor, 0xf4, 0xf5, 5),
				ctx.HideActor(actor),
				loc28PlayHide(ctx, "S31_EnterCity"),
				ctx.ChangeScene(loc28Scene32),
			)
		}
		return loc28S31BushInteraction(ctx)
	case "S31_Wall":
		return loc28LayerSpeech(ctx, "S31_RodBushTalk", "031_ROD_10", 0, 8)
	case "S31_Wache":
		if loc28Original(ctx, loc28StateInsideCity) != 0 {
			n := rand.IntN(3) + 1
			return engine.Sequence(
				ctx.WalkToFacingPerspective(actor, 0xc6, 0x88, 6),
				loc28Rod(ctx, "031_ROD_1"+strconv.Itoa(3+n)),
				loc28LayerSpeech(ctx, "S31_Wache", "031_PW1_1"+strconv.Itoa(3+n), 0, 8),
			)
		}
		return loc28S31BushInteraction(ctx)
	}
	return nil
}

func loc28S31BushInteraction(ctx *Context) engine.Task {
	if loc28Original(ctx, loc28StateInsideCity) != 0 {
		return nil
	}
	if ctx.HasItem(1) && ctx.HasItem(3) && ctx.HasItem(4) && ctx.HasItem(0xe) {
		return loc28ApproachGuard(ctx)
	}
	if ctx.HasItem(1) || ctx.HasItem(3) || ctx.HasItem(4) || ctx.HasItem(0xe) {
		return loc28LayerSpeech(ctx, "S31_RodBushTalk", "031_ROD_09", 0, 8)
	}
	stage := loc28Original(ctx, loc28StateBushTalk)
	switch stage {
	case 1:
		return engine.Sequence(
			loc28LayerSpeech(ctx, "S31_RodBushTalk", "031_ROD_05", 0, 8),
			loc28SetOriginal(ctx, loc28StateBushTalk, 2),
		)
	case 2:
		return engine.Sequence(
			loc28SetOriginal(ctx, loc28StateBushTalk, 3),
			loc28LayerSpeech(ctx, "S31_RodBushTalk", "031_ROD_06", 0, 8),
			ctx.PlayVoiceover("031_PW2_06", "[031_PW2_06]"),
			ctx.Wait(2),
			loc28PlayRange(ctx, "S31_Wache", 0x32, 0x3c),
			loc28LayerSpeech(ctx, "S31_Wache", "031_PW1_06", 0x3c, 0x46),
			ctx.PlayVoiceover("031_PW2_07", "[031_PW2_07]"),
			ctx.RunAmbient(loc28PlayRange(ctx, "S31_Wache", 0x50, 0x5a)),
			loc28LayerSpeech(ctx, "S31_RodBushTalk", "031_ROD_07", 0, 8),
		)
	case 3:
		return loc28LayerSpeech(ctx, "S31_RodBushTalk", "031_ROD_08", 0, 8)
	default:
		return nil
	}
}

func loc28ApproachGuard(ctx *Context) engine.Task {
	actor := loc28Actor(ctx)
	tasks := []engine.Task{}
	if loc28Original(ctx, loc28StateDisguise) == 0 {
		tasks = append(tasks,
			loc28SetOriginal(ctx, loc28StateDisguise, 1),
			loc28LayerSpeech(ctx, "S31_RodBushTalk", "031_ROD_22", 0, 8),
			ctx.HideLayer("S31_RodBushTalk"),
			loc28PlayHide(ctx, "S31_RodBuschRunterRauf"),
			loc28PlayRangeHide(ctx, "S31_RodPirBushRunterRauf", 10, 0),
			loc28ShowStatic(ctx, "S31_RodPirBushTalk", 0),
		)
	} else {
		tasks = append(tasks, ctx.HideLayer("S31_RodPirBushTalk"))
	}
	tasks = append(tasks,
		loc28PlayHide(ctx, "S31_RodPirLeavesBush"),
		ctx.Wait(2),
		loc28SetOriginal(ctx, loc28StateInsideCity, 1),
		ctx.ShowActor(actor),
		ctx.PlaceActor(actor, 4, 0x108),
		ctx.WalkToFacingPerspective(actor, 0x5a, 0x106, 6),
		loc28Deferred(func() engine.Task {
			if loc28Original(ctx, loc28StateGuardReturn) == 0 {
				return nil
			}
			return engine.Sequence(
				loc28SetOriginal(ctx, loc28StateGuardReturn, 0),
				loc28LayerSpeech(ctx, "S31_Wache", "031_PW1_11", 0, 8),
				loc28Rod(ctx, "031_ROD_11"),
				loc28LayerSpeech(ctx, "S31_Wache", "031_PW1_12b", 0, 8),
				ctx.HideActor(actor),
				loc28PlayHide(ctx, "S31_RodPirParole"),
				ctx.ShowActor(actor),
				loc28LayerSpeech(ctx, "S31_Wache", "031_PW1_12", 0, 8),
			)
		}),
	)
	return engine.Sequence(tasks...)
}

func loc28ToggleDisguise(ctx *Context) engine.Task {
	if loc28Original(ctx, loc28StateDisguise) == 0 {
		return engine.Sequence(
			loc28SetOriginal(ctx, loc28StateDisguise, 1),
			loc28LayerSpeech(ctx, "S31_RodBushTalk", "031_ROD_22", 0, 8),
			ctx.HideLayer("S31_RodBushTalk"),
			loc28PlayHide(ctx, "S31_RodBuschRunterRauf"),
			loc28PlayRangeHide(ctx, "S31_RodPirBushRunterRauf", 10, 0),
			loc28ShowStatic(ctx, "S31_RodPirBushTalk", 0),
		)
	}
	return engine.Sequence(
		loc28SetOriginal(ctx, loc28StateDisguise, 0),
		loc28LayerSpeech(ctx, "S31_RodPirBushTalk", "031_ROD_04", 0, 8),
		ctx.HideLayer("S31_RodPirBushTalk"),
		loc28PlayHide(ctx, "S31_RodPirBushRunterRauf"),
		loc28PlayRangeHide(ctx, "S31_RodBuschRunterRauf", 10, 0),
		loc28ShowStatic(ctx, "S31_RodBushTalk", 0),
	)
}

func loc28Click32(ctx *Context, area string) engine.Task {
	actor := loc28Actor(ctx)
	switch area {
	case "S32_To31":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x69, 0x167, 2), ctx.ChangeScene(loc28Scene31))
	case "S32_Wagen":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x89, 0x137, 4), loc28Rod(ctx, "032_ROD_13"))
	case "S32_FensterLinks":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xb3, 0x120, 4), loc28Rod(ctx, "032_ROD_14"))
	case "S32_FensterRechts":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x130, 0x118, 4), loc28Rod(ctx, "032_ROD_15"))
	case "S32_Kneipe":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x100, 0x124, 4), ctx.ChangeLocation(29, "S33"))
	case "S32_Treppen":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x169, 0x80, 6), loc28Rod(ctx, "032_ROD_16"))
	case "S32_SchuttLinks":
		line := "032_ROD_02"
		if rand.IntN(2) == 1 {
			line = "032_ROD_03"
		}
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x17e, 0xa9, 4), loc28Rod(ctx, line))
	case "S32_SchuttRechts":
		return loc28WagonPickup(ctx)
	case "S32_FassLinks":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x1e2, 0xec, 4), loc28Rod(ctx, "032_ROD_17"))
	case "S32_FassRechts":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x21b, 0xe0, 4), loc28Rod(ctx, "032_ROD_18"))
	case "S32_Kerker":
		return loc28StreetTalk(ctx)
	case "S32_Wache":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xe2, 0x14f, 1), loc28GuardTalk32(ctx))
	}
	return nil
}

func loc28WagonPickup(ctx *Context) engine.Task {
	actor := loc28Actor(ctx)
	stage := loc28Original(ctx, loc28StateWagon)
	if stage <= 1 {
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0x251, 0xcb, 6),
			loc28Rod(ctx, "032_ROD_04"),
			loc28SetOriginal(ctx, loc28StateWagon, 2),
		)
	}
	if stage == 2 {
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0x251, 0xcb, 6),
			loc28SetOriginal(ctx, loc28StateWagon, 3),
			ctx.HideActor(actor),
			loc28PlayHide(ctx, "S32_RodKramtAnim"),
			ctx.AddItem(loc28ItemWagonPickup),
			ctx.ShowActor(actor),
			loc28Rod(ctx, "032_ROD_05"),
		)
	}
	return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x251, 0xcb, 6), loc28Rod(ctx, "032_ROD_06"))
}

func loc28StreetTalk(ctx *Context) engine.Task {
	actor := loc28Actor(ctx)
	stage := loc28Original(ctx, loc28StateStreetTalk)
	line := "032_ROD_09"
	if stage <= 1 {
		line = "032_ROD_07"
		stage = 2
	} else if stage == 2 {
		line = "032_ROD_08"
		stage = 3
	}
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0x173, 0xd3, 3),
		loc28SetOriginal(ctx, loc28StateStreetTalk, stage),
		loc28Rod(ctx, line),
	)
}

func loc28GuardTalk32(ctx *Context) engine.Task {
	stage := loc28Original(ctx, loc28StateGuardTalk)
	if stage <= 1 {
		return engine.Sequence(
			loc28Rod(ctx, "032_ROD_10"),
			loc28LayerSpeech(ctx, "S32_Wache", "032_PW2_10", -1, -1),
			loc28SetOriginal(ctx, loc28StateGuardTalk, 2),
		)
	}
	if stage == 2 {
		return engine.Sequence(
			loc28Rod(ctx, "032_ROD_11"),
			loc28LayerSpeech(ctx, "S32_Wache", "032_PW2_11", -1, -1),
			loc28SetOriginal(ctx, loc28StateGuardTalk, 3),
		)
	}
	return engine.Sequence(
		loc28Rod(ctx, "032_ROD_12"),
		loc28LayerSpeech(ctx, "S32_Wache", "032_PW2_12", -1, -1),
	)
}

func loc28Click35(ctx *Context, area string) engine.Task {
	switch area {
	case "S35_To34", "S35_To34_2":
		return ctx.ChangeLocation(30, "34")
	}
	return nil
}

func (LOC28Controller) UseItem(ctx *Context, item int, area string) engine.Task {
	if loc28SceneID(ctx.session.state.Scene) != loc28Scene31 {
		return nil
	}
	if area != "" && area != "S31_To32" && area != "S31_Wache" && area != "S31_Wall" {
		return nil
	}
	return loc28S31Item(ctx, item)
}

func (LOC28Controller) SelectItem(ctx *Context, item int) engine.Task {
	switch loc28SceneID(ctx.session.state.Scene) {
	case loc28Scene31:
		return loc28S31Item(ctx, item)
	case loc28Scene32:
		switch item {
		case 1, 3, 4, 0xe:
			return loc28Rod(ctx, "GEN_ROD_01")
		}
	}
	return nil
}

func loc28S31Item(ctx *Context, item int) engine.Task {
	if item != 1 && item != 3 && item != 4 && item != 0xe {
		return nil
	}
	if loc28Original(ctx, loc28StateInsideCity) != 0 {
		return loc28Rod(ctx, "GEN_ROD_01")
	}
	if ctx.HasItem(1) && ctx.HasItem(3) && ctx.HasItem(4) && ctx.HasItem(0xe) {
		return loc28ToggleDisguise(ctx)
	}
	switch item {
	case 1:
		return loc28LayerSpeech(ctx, "S31_RodBushTalk", "031_ROD_25", 0, 8)
	case 4:
		return loc28LayerSpeech(ctx, "S31_RodBushTalk", "031_ROD_26", 0, 8)
	case 0xe:
		if rand.IntN(2) == 0 {
			return loc28LayerSpeech(ctx, "S31_RodBushTalk", "031_ROD_23", 0, 8)
		}
		return loc28LayerSpeech(ctx, "S31_RodBushTalk", "031_ROD_24", 0, 8)
	case 3:
		return loc28LayerSpeech(ctx, "S31_RodBushTalk", "031_ROD_26b", 0, 8)
	}
	return nil
}

type loc28BirdTask struct {
	ctx  *Context
	wait float64
}

func (t *loc28BirdTask) Update(dt float64) bool {
	if t.ctx.session.state.Location != 28 || loc28SceneID(t.ctx.session.state.Scene) != loc28Scene31 {
		return true
	}
	t.wait -= dt
	if t.wait > 0 {
		return false
	}
	names := []string{"Sfx_Bird.wav", "Sfx_Bird2.wav", "Sfx_Bird3.wav", "Sfx_Chirp.wav"}
	_ = t.ctx.PlaySFX(names[rand.IntN(4)]).Update(0)
	t.wait = float64(2 + rand.IntN(9))
	return false
}

type loc28S35DialogueTask struct {
	ctx        *Context
	event      int
	timer      float64
	current    engine.Task
	background engine.Task
	after      int
	finalPhase int
	ownsLock   bool
}

func (t *loc28S35DialogueTask) Update(dt float64) bool {
	if t.ctx.session.state.Location != 28 || loc28SceneID(t.ctx.session.state.Scene) != loc28Scene35 {
		if t.ownsLock {
			t.ctx.session.locked = false
			t.ownsLock = false
		}
		return true
	}
	if t.background != nil {
		if t.background.Update(dt) {
			t.background = nil
		}
	}
	if t.current != nil {
		if !t.current.Update(dt) {
			return false
		}
		t.current = nil
		switch t.after {
		case 1:
			t.after = 0
			t.event = 0x22
		case 2:
			t.after = 0
			t.advanceFinal()
		}
		return false
	}
	if t.timer > 0 {
		t.timer -= dt
		if t.timer > 0 {
			return false
		}
		t.timer = 0
	}
	if t.event == 0 {
		return false
	}
	event := t.event
	t.event = 0
	switch event {
	case 0x21:
		t.startDialogueEvent()
	case 0x22:
		t.timer = 0.2
		t.event = 0x21
	}
	return false
}

func (t *loc28S35DialogueTask) startDialogueEvent() {
	stage := loc28Original(t.ctx, loc28StateCaptainTalk)
	switch stage {
	case 1, 3, 5, 7, 9:
		idx := (stage + 1) / 2
		t.current = loc28LayerSpeech(t.ctx, "S35_Captain", "035_CAP_0"+strconv.Itoa(idx), 0, 8)
		putOriginalFlag(t.ctx.session.state.OriginalState, loc28StateCaptainTalk, stage+1)
		t.after = 1
	case 2, 4, 6, 8, 10:
		idx := stage / 2
		t.current = loc28LayerSpeech(t.ctx, "S35_Fremder", "035_FRE_0"+strconv.Itoa(idx), 0, 8)
		putOriginalFlag(t.ctx.session.state.OriginalState, loc28StateCaptainTalk, stage+1)
		t.after = 1
	case 11:
		t.ctx.session.locked = true
		t.ownsLock = true
		t.finalPhase = 0
		t.current = loc28LayerSpeech(t.ctx, "S35_Captain", "035_CAP_06", 0, 8)
		t.after = 2
	}
}

func (t *loc28S35DialogueTask) advanceFinal() {
	switch t.finalPhase {
	case 0:
		t.current = loc28PlayRange(t.ctx, "S35_Captain", 0x1f, 0x28)
		t.finalPhase = 1
		t.after = 2
	case 1:
		t.background = loc28PlayRange(t.ctx, "S35_Captain", 0x29, 0x30)
		t.current = loc28LayerSpeech(t.ctx, "S35_Fremder", "035_FRE_06", 0, 8)
		t.finalPhase = 2
		t.after = 2
	case 2:
		t.background = nil
		t.current = loc28PlayRange(t.ctx, "S35_Captain", 0x31, 0x39)
		t.finalPhase = 3
		t.after = 2
	case 3:
		t.current = loc28LayerSpeech(t.ctx, "S35_Fremder", "035_FRE_07", 0, 8)
		t.finalPhase = 4
		t.after = 2
	case 4:
		t.current = loc28PlayRange(t.ctx, "S35_Captain", 0x3a, -1)
		t.finalPhase = 5
		t.after = 2
	case 5:
		putOriginalFlag(t.ctx.session.state.OriginalState, loc28StateCaptainTalk, 1)
		t.ctx.session.locked = false
		t.ownsLock = false
		t.timer = 1
		t.event = 0x21
		t.after = 0
	}
}
