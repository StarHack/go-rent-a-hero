package game

import (
	"encoding/binary"
	"math/rand/v2"

	"github.com/wok/rent-a-hero/internal/engine"
)

const (
	loc07StateFirstDescent = 0x3ef4
	loc07StateSack         = 0x3ef8
	loc07StateRope         = 0x3efc
	loc07StateRopeFixed    = 0x3f00
	loc07StateGateClosed   = 0x3f04
	loc07StatePiratePeek   = 0x3f08
	loc07StateRopeBroken   = 0x3f30
	loc07StatePiratesGone  = 0x3fc8
)

const flagLoc07MusicStarted = "loc07.music_started"

type LOC07Controller struct{}

func (LOC07Controller) Enter(ctx *Context, scene string, from string) engine.Task {
	actor := loc07ActorID(ctx)
	tasks := make([]engine.Task, 0, 13)
	tasks = append(tasks, engine.Immediate(func() {
		if a, ok := ctx.session.scene.Characters[actor]; ok {
			a.StepLeft = "Gen_StepLeft.wav"
			a.StepRight = "Gen_StepRight.wav"
		}
	}))
	if ctx.session.state.PreviousLocation != 7 || ctx.GetFlag(flagLoc07MusicStarted) == 0 {
		tasks = append(tasks, ctx.PlayMusic("Loc07_Sewage.wav"), ctx.SetFlag(flagLoc07MusicStarted, 1))
	}
	tasks = append(tasks, loc07PrepareScene(ctx, scene))

	var enter engine.Task
	switch scene {
	case "47":
		enter = loc07Enter47(ctx, actor, from)
	case "48":
		enter = loc07EnterSimple(ctx, actor, 0x40, 0xce, 0x4e, 0xf8, 0)
	case "49":
		enter = loc07Enter49(ctx, actor, from)
	case "50":
		enter = loc07Enter50(ctx, actor, from)
	case "51":
		enter = loc07Enter51(ctx, actor, from)
	case "52":
		enter = loc07Enter52(ctx, actor, from)
	case "53":
		enter = loc07Enter53(ctx, actor, from)
	case "54":
		enter = loc07Enter54(ctx, actor, from)
	case "55":
		enter = loc07EnterSimple(ctx, actor, 0x90, 0xbc, 0xa7, 0xcb, 0)
	case "56":
		enter = loc07EnterSimple(ctx, actor, 0x40, 0xce, 0x4e, 0xf8, 7)
	default:
		return engine.Sequence(tasks...)
	}
	if enter != nil {
		tasks = append(tasks, enter)
	}
	tasks = append(tasks, ctx.RunAmbient(newLoc07BubbleAmbient(ctx)))
	return engine.Sequence(tasks...)
}

func (LOC07Controller) Exit(ctx *Context, scene string, to string) engine.Task { return nil }

func (LOC07Controller) Click(ctx *Context, area string) engine.Task {
	actor := loc07ActorID(ctx)
	switch area {
	case "047_DoorTo50":
		return loc07WalkChange(ctx, actor, 4, 0xea, 2, 7, "50")
	case "047_DoorTo51":
		return loc07WalkChange(ctx, actor, 0x1de, 0xb7, 4, 7, "51")
	case "047_DoorTo49":
		return loc07WalkChange(ctx, actor, 0x232, 0xbb, 4, 7, "49")
	case "047_DeadEnd":
		return loc07WalkSay(ctx, actor, 0x13f, 0xd8, 4, "000_ROD_01")
	case "047_Ladder":
		return loc07ClimbOut(ctx, actor)
	case "048_DoorTo50":
		return loc07WalkChange(ctx, actor, 0x40, 0xce, 4, 7, "50")
	case "048_DeadEnd":
		return loc07WalkSay(ctx, actor, 499, 0xcf, 5, "000_ROD_01")
	case "049_DoorTo52":
		return loc07WalkChange(ctx, actor, 0x3e, 0xcc, 4, 7, "52")
	case "049_DoorTo47":
		return loc07WalkChange(ctx, actor, 0x202, 0xbd, 4, 7, "47")
	case "050_DoorTo54":
		return loc07WalkChange(ctx, actor, 0x20, 0xde, 4, 7, "54")
	case "050_DoorTo48":
		return loc07WalkChange(ctx, actor, 0x110, 0xea, 4, 7, "48")
	case "050_DoorTo47":
		return loc07WalkChange(ctx, actor, 0x230, 0xdd, 4, 7, "47")
	case "050_DeadEnd":
		return loc07WalkSay(ctx, actor, 0x1d2, 0xe0, 4, "000_ROD_01")
	case "051_DoorTo54":
		return loc07WalkChange(ctx, actor, 0x1f, 0xd4, 4, 7, "54")
	case "051_DoorTo47":
		return loc07WalkChange(ctx, actor, 0xf4, 0xd1, 4, 7, "47")
	case "051_DoorTo52":
		return loc07WalkChange(ctx, actor, 0x1b2, 0xd3, 4, 7, "52")
	case "051_DoorTo55":
		return loc07WalkChange(ctx, actor, 0x1fc, 0xf7, 4, 7, "55")
	case "052_DoorTo56":
		return loc07WalkChange(ctx, actor, 0x0b, 0xd8, 4, 7, "56")
	case "052_DoorTo51":
		return loc07WalkChange(ctx, actor, 0xe9, 0xcb, 4, 7, "51")
	case "052_DoorTo49":
		return loc07WalkChange(ctx, actor, 0x1a5, 0xcf, 4, 7, "49")
	case "052_DoorTo53":
		return loc07DoorTo53(ctx, actor)
	case "52_Kurbel":
		return loc07Crank(ctx, actor)
	case "52_Sack":
		return loc07TakeSack(ctx, actor)
	case "053_DoorTo52":
		return loc07Door53To52(ctx, actor)
	case "053_DoorTo57":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x219, 0xb8, 4), ctx.ChangeLocation(15, "57"))
	case "054_DoorTo50":
		return loc07WalkChange(ctx, actor, 0x6c, 0xb8, 4, 7, "50")
	case "054_DoorTo51":
		return loc07WalkChange(ctx, actor, 0x210, 0xb4, 4, 7, "51")
	case "055_DoorTo51":
		return loc07WalkChange(ctx, actor, 0x90, 0xbc, 4, 7, "51")
	case "056_DoorTo52":
		return loc07WalkChange(ctx, actor, 0x40, 0xce, 4, 7, "52")
	}
	return nil
}

func (LOC07Controller) UseItem(ctx *Context, item int, area string) engine.Task { return nil }

func (LOC07Controller) SelectItem(ctx *Context, item int) engine.Task {
	if item == 10 && ctx.session.state.Scene == "52" && loc07Original(ctx, loc07StateRope) == 0 {
		return loc07FixRope(ctx, loc07ActorID(ctx))
	}
	return nil
}

func (LOC07Controller) LoadConditionMask(ctx *Context, scene string) int {
	if scene != "52" {
		return 0
	}
	mask := 0
	if loc07Original(ctx, loc07StateSack) != 0 {
		mask |= 1
	}
	if loc07Original(ctx, loc07StateGateClosed) != 0 {
		mask |= 2
	}
	return mask
}

func loc07PrepareScene(ctx *Context, scene string) engine.Task {
	return engine.Immediate(func() {
		loops := map[string]string{
			"47": "47_BackAnim", "48": "48_BackAnim", "49": "48_BackAnim", "50": "50_BackAnim",
			"51": "51_BackAnim", "52": "52_BackAnim", "53": "53_BackAnim", "54": "54_BackAnim",
			"55": "55_BackAnim", "56": "56_BackAnim",
		}
		if id := loops[scene]; id != "" {
			if l, ok := ctx.layer(id); ok {
				l.Visible = true
				l.Enabled = true
				l.Mode = engine.AnimLoop
				l.Playing = true
				l.TaskDriven = false
			}
		}
		for _, id := range []string{
			"47_RodRaufStart", "47_RodRauf", "47_RodRunter", "47_RodRunterStop",
			"51_PiratGuck", "52_SeilReisst", "52_RodRuettel", "52_RodKurbelStart",
			"52_RodKurbel", "52_RodKurbelStop", "52_RodRunterbeug", "52_RodFixRope",
		} {
			if l, ok := ctx.layer(id); ok {
				l.Visible = false
				l.Playing = false
				l.TaskDriven = false
			}
		}
		for _, id := range []string{"51_Light", "54_Light", "52_Behind"} {
			if l, ok := ctx.layer(id); ok {
				l.Visible = true
				l.Enabled = true
			}
		}
	})
}

func loc07Enter47(ctx *Context, actor, from string) engine.Task {
	if from == "46" {
		return loc07DescendLadder(ctx, actor)
	}
	switch from {
	case "49":
		return engine.Sequence(ctx.PlaceActorPerspective(actor, 0x232, 0xbb), ctx.WalkToFacingPerspective(actor, 0x21e, 0xc3, 1))
	case "50":
		return engine.Sequence(ctx.PlaceActorPerspective(actor, 4, 0xea), ctx.WalkToFacingPerspective(actor, 0x38, 0xf2, 7))
	case "51":
		return engine.Sequence(ctx.PlaceActorPerspective(actor, 0x1de, 0xb7), ctx.WalkToFacingPerspective(actor, 0x1de, 0xb9, 0))
	}
	return nil
}

func loc07DescendLadder(ctx *Context, actor string) engine.Task {
	state := loc07Original(ctx, loc07StateFirstDescent)
	tasks := []engine.Task{
		ctx.HideActor(actor),
		loc07SetLayerPosition(ctx, "47_RodRunter", 0x98, -0x46),
		loc07LoopLayer(ctx, "47_RodRunter"),
		loc07MoveLayerOffset(ctx, "47_RodRunter", 0, 80, 30),
		ctx.HideLayer("47_RodRunter"),
		ctx.ShowLayer("47_RodRunterStop"),
		loc07DeferredPlayLayer(ctx, "47_RodRunterStop"),
		ctx.HideLayer("47_RodRunterStop"),
		ctx.PlaceActorPerspective(actor, 0xaf, 0xdb),
		ctx.ShowActor(actor),
		ctx.WalkToFacingPerspective(actor, 0xaf, 0xe1, 0),
	}
	if state == 0 {
		tasks = append(tasks, ctx.Say(actor, "047_ROD_01", "[047_ROD_01]"), engine.Immediate(func() { loc07SetOriginal(ctx, loc07StateFirstDescent, 1) }))
	} else if state == 1 {
		tasks = append(tasks, ctx.Say(actor, "047_ROD_02", "[047_ROD_02]"))
	}
	return engine.Sequence(tasks...)
}

func loc07ClimbOut(ctx *Context, actor string) engine.Task {
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0xaf, 0xdb, 4),
		ctx.HideActor(actor),
		ctx.ShowLayer("47_RodRaufStart"),
		loc07DeferredPlayLayer(ctx, "47_RodRaufStart"),
		ctx.HideLayer("47_RodRaufStart"),
		loc07LoopLayer(ctx, "47_RodRauf"),
		loc07MoveLayerOffset(ctx, "47_RodRauf", 0, -80, 50),
		ctx.ChangeLocation(1, "46"),
	)
}

func loc07EnterSimple(ctx *Context, actor string, px, py, wx, wy float64, dir int) engine.Task {
	return engine.Sequence(ctx.PlaceActorPerspective(actor, px, py), ctx.WalkToFacingPerspective(actor, wx, wy, dir))
}

func loc07Enter49(ctx *Context, actor, from string) engine.Task {
	if from == "47" {
		return engine.Sequence(ctx.PlaceActorPerspective(actor, 0x202, 0xbd), ctx.WalkToFacingPerspective(actor, 0x1f8, 0xcb, 1))
	}
	if from == "52" {
		return engine.Sequence(ctx.PlaceActorPerspective(actor, 0x3e, 0xcc), ctx.WalkToFacingPerspective(actor, 0x50, 0xfa, 0))
	}
	return nil
}

func loc07Enter50(ctx *Context, actor, from string) engine.Task {
	switch from {
	case "47":
		return engine.Sequence(ctx.PlaceActorPerspective(actor, 0x230, 0xdd), ctx.WalkToFacingPerspective(actor, 0x216, 0xf1, 1))
	case "48":
		return engine.Sequence(ctx.PlaceActorPerspective(actor, 0x10f, 0xe4), ctx.WalkToFacingPerspective(actor, 0x10f, 0xea, 0))
	case "54":
		return engine.Sequence(ctx.PlaceActorPerspective(actor, 0x20, 0xde), ctx.WalkToFacingPerspective(actor, 0x35, 0xef, 7))
	}
	return nil
}

func loc07Enter51(ctx *Context, actor, from string) engine.Task {
	switch from {
	case "47":
		return engine.Sequence(ctx.PlaceActorPerspective(actor, 0xf4, 0xd1), ctx.WalkToFacingPerspective(actor, 0xf4, 0xea, 0))
	case "52":
		return engine.Sequence(ctx.PlaceActorPerspective(actor, 0x1b2, 0xd3), ctx.WalkToFacingPerspective(actor, 0x1b2, 0xe9, 0))
	case "54":
		if loc07Original(ctx, loc07StatePiratePeek) == 0 && loc07Original(ctx, loc07StatePiratesGone) == 0 {
			return loc07PiratePeek(ctx, actor)
		}
		return engine.Sequence(ctx.PlaceActorPerspective(actor, 0x1f, 0xd4), ctx.WalkToFacingPerspective(actor, 0x3a, 0xe4, 7))
	case "55":
		return engine.Sequence(ctx.PlaceActorPerspective(actor, 0x1fc, 0xf7), ctx.WalkToFacingPerspective(actor, 0x1e0, 0xf7, 2))
	}
	return nil
}

func loc07PiratePeek(ctx *Context, actor string) engine.Task {
	baseX, baseY := -12, 0xa5
	return engine.Sequence(
		ctx.PlaceActorPerspective(actor, 0x1f, 0xd4),
		ctx.WalkToFacingPerspective(actor, 0x5d, 0xfd, 7),
		loc07SetLayerPosition(ctx, "51_PiratGuck", baseX, baseY),
		ctx.PlaySFX("Sfx_DangerStringsLow.wav"),
		ctx.ShowLayer("51_PiratGuck"),
		loc07MoveLayerFromBase(ctx, "51_PiratGuck", &baseX, &baseY, 0, 5, 10),
		loc07DeferredPlayLayer(ctx, "51_PiratGuck"),
		loc07MoveLayerFromBase(ctx, "51_PiratGuck", &baseX, &baseY, 0, -12, 10),
		ctx.HideLayer("51_PiratGuck"),
		engine.Immediate(func() { loc07SetOriginal(ctx, loc07StatePiratePeek, 1) }),
	)
}

func loc07Enter52(ctx *Context, actor, from string) engine.Task {
	tasks := []engine.Task{loc07Setup52(ctx)}
	switch from {
	case "49":
		tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x1a5, 0xcf), ctx.WalkToFacingPerspective(actor, 0x1a5, 0xe8, 0))
	case "51":
		tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0xe9, 0xcb), ctx.WalkToFacingPerspective(actor, 0xe9, 0xe8, 0))
	case "53":
		tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x24e, 0xe9), ctx.WalkToFacingPerspective(actor, 0x1f9, 0xff, 1))
	case "56":
		tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x0b, 0xed), ctx.WalkToFacingPerspective(actor, 0x28, 0xed, 7))
	}
	return engine.Sequence(tasks...)
}

func loc07Setup52(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		if l, ok := ctx.layer("52_Kurbel"); ok {
			l.Visible = true
			l.Enabled = true
		}
		if l, ok := ctx.layer("52_Gitter"); ok {
			l.Visible = true
			l.Enabled = true
			if loc07Original(ctx, loc07StateGateClosed) == 0 {
				l.X, l.Y = 0x1fb, -40
			}
		}
		if l, ok := ctx.layer("52_Seil"); ok {
			l.Visible = loc07Original(ctx, loc07StateRope) != 0
			l.Enabled = true
		}
		if l, ok := ctx.layer("52_Sack"); ok {
			l.Visible = loc07Original(ctx, loc07StateSack) != 0
			l.Enabled = l.Visible
		}
		if loc07Original(ctx, loc07StateGateClosed) != 0 {
			ctx.MakeLayerClickable("52_Kurbel").Update(0)
		}
		if loc07Original(ctx, loc07StateSack) != 0 {
			ctx.MakeLayerClickable("52_Sack").Update(0)
		}
	})
}

func loc07DoorTo53(ctx *Context, actor string) engine.Task {
	if loc07Original(ctx, loc07StateGateClosed) == 0 {
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x24e, 0xe9, 6), ctx.ChangeScene("53"))
	}
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0x1f7, 0xfe, 6),
		ctx.HideActor(actor),
		ctx.ShowLayer("52_RodRuettel"),
		loc07DeferredPlayLayer(ctx, "52_RodRuettel"),
		ctx.HideLayer("52_RodRuettel"),
		ctx.ShowActor(actor),
		ctx.WalkToFacingPerspective(actor, 0x1f7, 0x103, 0),
		ctx.Say(actor, "052_ROD_05", "[052_ROD_05]"),
	)
}

func loc07Crank(ctx *Context, actor string) engine.Task {
	return &loc07DeferredTask{build: func() engine.Task {
		rope := loc07Original(ctx, loc07StateRope)
		fixed := loc07Original(ctx, loc07StateRopeFixed)
		closed := loc07Original(ctx, loc07StateGateClosed)
		if closed == 0 {
			return nil
		}
		if fixed != 0 {
			return loc07OpenGate(ctx, actor)
		}
		if rope != 0 {
			return loc07BreakRope(ctx, actor)
		}
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0x1c4, 0xf5, 6),
			ctx.HideActor(actor),
			ctx.ShowLayer("52_RodKurbelStart"), loc07DeferredPlayLayer(ctx, "52_RodKurbelStart"), ctx.HideLayer("52_RodKurbelStart"),
			ctx.PlaySFX("Sfx_Winding.wav"),
			ctx.ShowLayer("52_RodKurbel"), loc07DeferredPlayLayer(ctx, "52_RodKurbel"), ctx.HideLayer("52_RodKurbel"),
			ctx.ShowLayer("52_RodKurbelStop"), loc07DeferredPlayLayer(ctx, "52_RodKurbelStop"), ctx.HideLayer("52_RodKurbelStop"),
			ctx.ShowActor(actor),
			ctx.WalkToFacingPerspective(actor, 0x1c4, 0xf6, 0),
			ctx.Say(actor, "052_ROD_02", "[052_ROD_02]"),
		)
	}}
}

func loc07BreakRope(ctx *Context, actor string) engine.Task {
	var gateMotionVersion int
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0x1c4, 0xf5, 6),
		ctx.HideActor(actor),
		ctx.ShowLayer("52_RodKurbelStart"), loc07DeferredPlayLayer(ctx, "52_RodKurbelStart"), ctx.HideLayer("52_RodKurbelStart"),
		&loc07DeferredTask{build: func() engine.Task {
			gateMotionVersion++
			return ctx.RunAmbient(loc07MoveLayerToAbsoluteYVersioned(ctx, "52_Gitter", -40, 70, &gateMotionVersion, gateMotionVersion))
		}},
		ctx.PlaySFX("Sfx_Winding.wav"),
		ctx.ShowLayer("52_RodKurbel"), loc07DeferredPlayLayer(ctx, "52_RodKurbel"), ctx.HideLayer("52_RodKurbel"),
		ctx.HideLayer("52_Seil"),
		&loc07DeferredTask{build: func() engine.Task {
			gateMotionVersion++
			return ctx.RunAmbient(loc07MoveLayerToAbsoluteYVersioned(ctx, "52_Gitter", 75, 5, &gateMotionVersion, gateMotionVersion))
		}},
		engine.Immediate(func() { loc07SetOriginal(ctx, loc07StateRope, 0); loc07SetOriginal(ctx, loc07StateRopeBroken, 1) }),
		ctx.PlaySFX("Sfx_Rubberband.wav"),
		loc07StartLayerOnce(ctx, "52_SeilReisst"),
		ctx.ShowLayer("52_RodKurbelStop"),
		loc07DeferredPlayLayerFrames(ctx, "52_RodKurbelStop", 0, 4),
		ctx.PlaySFX("Sfx_Grid_Drops.wav"),
		loc07DeferredPlayLayerFrames(ctx, "52_RodKurbelStop", 5, -1),
		ctx.HideLayer("52_RodKurbelStop"),
		ctx.ShowActor(actor),
		ctx.WalkToFacingPerspective(actor, 0x1c4, 0xf6, 0),
		ctx.PlaySFX("Sfx_PianoStroke.wav"),
		ctx.Say(actor, "052_ROD_01", "[052_ROD_01]"),
	)
}

func loc07OpenGate(ctx *Context, actor string) engine.Task {
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0x1c4, 0xf5, 6),
		ctx.HideActor(actor),
		ctx.ShowLayer("52_RodKurbelStart"), loc07DeferredPlayLayer(ctx, "52_RodKurbelStart"), ctx.HideLayer("52_RodKurbelStart"),
		ctx.RunAmbient(loc07MoveLayerToAbsoluteY(ctx, "52_Gitter", -40, 70)),
		engine.Immediate(func() { loc07SetOriginal(ctx, loc07StateGateClosed, 0) }),
		ctx.DisableArea("52_Kurbel"),
		ctx.PlaySFX("Sfx_Winding.wav"), ctx.ShowLayer("52_RodKurbel"), loc07DeferredPlayLayer(ctx, "52_RodKurbel"),
		ctx.PlaySFX("Sfx_Winding.wav"), loc07DeferredPlayLayer(ctx, "52_RodKurbel"),
		ctx.PlaySFX("Sfx_Winding.wav"), loc07DeferredPlayLayer(ctx, "52_RodKurbel"), ctx.HideLayer("52_RodKurbel"),
		ctx.ShowLayer("52_RodKurbelStop"), loc07DeferredPlayLayer(ctx, "52_RodKurbelStop"), ctx.HideLayer("52_RodKurbelStop"),
		ctx.ShowActor(actor),
		ctx.WalkToFacingPerspective(actor, 0x1c4, 0xf6, 0),
		ctx.Say(actor, "052_ROD_06", "[052_ROD_06]"),
	)
}

func loc07TakeSack(ctx *Context, actor string) engine.Task {
	if loc07Original(ctx, loc07StateSack) == 0 {
		return nil
	}
	var baseX, baseY int
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0xf4, 0xf9, 6),
		ctx.HideActor(actor),
		ctx.ShowLayer("52_RodRunterbeug"),
		loc07DeferredPlayLayerFrames(ctx, "52_RodRunterbeug", 0, 0x10),
		engine.Immediate(func() {
			if l, ok := ctx.layer("52_Sack"); ok {
				baseX, baseY = l.X, l.Y
			}
		}),
		ctx.DisableArea("52_Sack"),
		ctx.RunAmbient(loc07MoveLayerFromBase(ctx, "52_Sack", &baseX, &baseY, 640, 0, 40)),
		engine.Immediate(func() { loc07SetOriginal(ctx, loc07StateSack, 0) }),
		loc07DeferredPlayLayerFrames(ctx, "52_RodRunterbeug", 0x11, -1),
		ctx.HideLayer("52_RodRunterbeug"),
		ctx.ShowActor(actor),
		ctx.WalkToFacingPerspective(actor, 0x13d, 0xf1, 6),
		ctx.Say(actor, "052_ROD_03", "[052_ROD_03]"),
	)
}

func loc07FixRope(ctx *Context, actor string) engine.Task {
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0x1c4, 0xf5, 6),
		ctx.HideActor(actor),
		ctx.ShowLayer("52_RodFixRope"),
		loc07DeferredPlayLayerFrames(ctx, "52_RodFixRope", 0, 0x1e),
		ctx.ShowLayer("52_Seil"),
		ctx.RemoveItem(10),
		ctx.PlaySFX("Sfx_Work_FixBelt.wav"),
		loc07DeferredPlayLayerFrames(ctx, "52_RodFixRope", 0x1f, -1),
		ctx.HideLayer("52_RodFixRope"),
		engine.Immediate(func() {
			loc07SetOriginal(ctx, loc07StateRopeFixed, 1)
			loc07SetOriginal(ctx, loc07StateRope, 1)
			loc07SetOriginal(ctx, loc07StateFirstDescent, 2)
		}),
		ctx.ShowActor(actor),
		ctx.WalkToFacingPerspective(actor, 0x1c4, 0xf6, 0),
		ctx.Say(actor, "052_ROD_04", "[052_ROD_04]"),
	)
}

func loc07Enter53(ctx *Context, actor, from string) engine.Task {
	tasks := []engine.Task{engine.Immediate(func() {
		if l, ok := ctx.layer("53_Gitter"); ok {
			l.Visible = true
			l.Enabled = true
			if loc07Original(ctx, loc07StateGateClosed) == 0 {
				l.X, l.Y = 0, -100
			}
		}
	})}
	if from == "52" {
		tasks = append(tasks, ctx.PlaceActorPerspective(actor, 100, 0xbc), ctx.WalkToFacingPerspective(actor, 0x6e, 0xd5, 7))
	} else if from == "57" {
		tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x219, 0xb8), ctx.WalkToFacingPerspective(actor, 0x1e6, 0xd2, 1))
	}
	return engine.Sequence(tasks...)
}

func loc07Door53To52(ctx *Context, actor string) engine.Task {
	if loc07Original(ctx, loc07StateGateClosed) != 0 {
		return loc07WalkSay(ctx, actor, 0x8e, 0xd7, 4, "053_ROD_01")
	}
	return engine.Sequence(ctx.WalkToFacingPerspective(actor, 100, 0xbc, 4), ctx.ChangeScene("52"))
}

func loc07Enter54(ctx *Context, actor, from string) engine.Task {
	if from == "50" {
		return engine.Sequence(ctx.PlaceActorPerspective(actor, 0x6c, 0xb8), ctx.WalkToFacingPerspective(actor, 0x72, 0xdd, 7))
	}
	if from == "51" {
		return engine.Sequence(ctx.PlaceActorPerspective(actor, 0x210, 0xb4), ctx.WalkToFacingPerspective(actor, 0x1f8, 0xcb, 1))
	}
	return nil
}

func loc07WalkChange(ctx *Context, actor string, x, y float64, dir, location int, scene string) engine.Task {
	if location == 7 {
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, x, y, dir), ctx.ChangeScene(scene))
	}
	return engine.Sequence(ctx.WalkToFacingPerspective(actor, x, y, dir), ctx.ChangeLocation(location, scene))
}

func loc07WalkSay(ctx *Context, actor string, x, y float64, dir int, line string) engine.Task {
	return engine.Sequence(ctx.WalkToFacingPerspective(actor, x, y, dir), ctx.Say(actor, line, "["+line+"]"))
}

func loc07ActorID(ctx *Context) string {
	if actor := ctx.session.PlayerActor(); actor != nil {
		return actor.ID
	}
	return "RodrigoSmall"
}

func loc07Original(ctx *Context, offset int) int {
	state := ctx.session.state.OriginalState
	if offset < 0 || offset+4 > len(state) {
		return 0
	}
	return int(int32(binary.LittleEndian.Uint32(state[offset : offset+4])))
}

func loc07SetOriginal(ctx *Context, offset, value int) {
	state := ctx.session.state.OriginalState
	if offset < 0 || offset+4 > len(state) {
		return
	}
	binary.LittleEndian.PutUint32(state[offset:offset+4], uint32(int32(value)))
}

func loc07LoopLayer(ctx *Context, id string) engine.Task {
	return engine.Immediate(func() {
		if l, ok := ctx.layer(id); ok {
			l.Visible = true
			l.Enabled = true
			l.Mode = engine.AnimLoop
			l.Playing = true
			l.TaskDriven = false
		}
	})
}

func loc07StartLayerOnce(ctx *Context, id string) engine.Task {
	return engine.Immediate(func() {
		if l, ok := ctx.layer(id); ok {
			l.Visible = true
			l.Enabled = true
			l.Mode = engine.AnimOnce
			l.Frame = 0
			l.Accumulator = 0
			l.Playing = true
			l.TaskDriven = false
		}
	})
}

func loc07SetLayerPosition(ctx *Context, id string, x, y int) engine.Task {
	return engine.Immediate(func() {
		if l, ok := ctx.layer(id); ok {
			l.X, l.Y = x, y
		}
	})
}

type loc07LayerMoveTask struct {
	ctx                *Context
	id                 string
	baseX, baseY       int
	targetOffsetX      int
	targetOffsetY      int
	targetAbsoluteY    int
	useAbsoluteY       bool
	steps              int
	elapsed            float64
	startX, startY     int
	started            bool
	baseXPtr, baseYPtr *int
	captureBaseOnStart bool
	versionPtr         *int
	version            int
}

func loc07MoveLayerOffset(ctx *Context, id string, x, y, steps int) engine.Task {
	return &loc07LayerMoveTask{ctx: ctx, id: id, targetOffsetX: x, targetOffsetY: y, steps: steps, captureBaseOnStart: true}
}

func loc07MoveLayerFromBase(ctx *Context, id string, baseX, baseY *int, x, y, steps int) engine.Task {
	return &loc07LayerMoveTask{ctx: ctx, id: id, targetOffsetX: x, targetOffsetY: y, steps: steps, baseXPtr: baseX, baseYPtr: baseY}
}

func loc07MoveLayerFromBaseVersioned(ctx *Context, id string, baseX, baseY *int, x, y, steps int, versionPtr *int, version int) engine.Task {
	return &loc07LayerMoveTask{ctx: ctx, id: id, targetOffsetX: x, targetOffsetY: y, steps: steps, baseXPtr: baseX, baseYPtr: baseY, versionPtr: versionPtr, version: version}
}

func loc07MoveLayerToAbsoluteYVersioned(ctx *Context, id string, y, steps int, versionPtr *int, version int) engine.Task {
	return &loc07LayerMoveTask{ctx: ctx, id: id, targetAbsoluteY: y, useAbsoluteY: true, steps: steps, versionPtr: versionPtr, version: version}
}

func loc07MoveLayerToAbsoluteY(ctx *Context, id string, y, steps int) engine.Task {
	return &loc07LayerMoveTask{ctx: ctx, id: id, targetAbsoluteY: y, useAbsoluteY: true, steps: steps}
}

func (t *loc07LayerMoveTask) Update(dt float64) bool {
	if t.versionPtr != nil && *t.versionPtr != t.version {
		return true
	}
	l, ok := t.ctx.layer(t.id)
	if !ok {
		return true
	}
	if !t.started {
		t.started = true
		t.startX, t.startY = l.X, l.Y
		if t.captureBaseOnStart {
			t.baseX, t.baseY = l.X, l.Y
		} else if t.baseXPtr != nil && t.baseYPtr != nil {
			t.baseX, t.baseY = *t.baseXPtr, *t.baseYPtr
		} else {
			t.baseX, t.baseY = l.X, l.Y
		}
	}
	if t.steps <= 0 {
		if t.useAbsoluteY {
			l.Y = t.targetAbsoluteY
		} else {
			l.X, l.Y = t.baseX+t.targetOffsetX, t.baseY+t.targetOffsetY
		}
		return true
	}
	t.elapsed += dt
	duration := float64(t.steps) * 0.1
	p := t.elapsed / duration
	if p > 1 {
		p = 1
	}
	tx, ty := t.baseX+t.targetOffsetX, t.baseY+t.targetOffsetY
	if t.useAbsoluteY {
		ty = t.targetAbsoluteY
		tx = t.startX
	}
	l.X = t.startX + int(float64(tx-t.startX)*p)
	l.Y = t.startY + int(float64(ty-t.startY)*p)
	return t.elapsed >= duration
}

func loc07DeferredPlayLayer(ctx *Context, id string) engine.Task {
	return &loc07DeferredTask{build: func() engine.Task { return ctx.PlayLayer(id) }}
}

func loc07DeferredPlayLayerFrames(ctx *Context, id string, from, to int) engine.Task {
	return &loc07DeferredTask{build: func() engine.Task { return ctx.PlayLayerFrames(id, from, to) }}
}

type loc07DeferredTask struct {
	build   func() engine.Task
	inner   engine.Task
	started bool
}

func (t *loc07DeferredTask) Update(dt float64) bool {
	if !t.started {
		t.started = true
		if t.build != nil {
			t.inner = t.build()
		}
	}
	if t.inner == nil {
		return true
	}
	return t.inner.Update(dt)
}

type loc07BubbleAmbient struct {
	ctx  *Context
	left float64
}

func newLoc07BubbleAmbient(ctx *Context) engine.Task {
	return &loc07BubbleAmbient{ctx: ctx, left: float64(5 + rand.IntN(3))}
}

func (t *loc07BubbleAmbient) Update(dt float64) bool {
	t.left -= dt
	if t.left > 0 {
		return false
	}
	if task := t.ctx.PlaySFX("Sfx_Bubble.wav"); task != nil {
		task.Update(0)
	}
	t.left = float64(5 + rand.IntN(6))
	return false
}
