package game

import (
	"math/rand/v2"

	"github.com/wok/rent-a-hero/internal/engine"
)

const (
	loc05Scene = "10"

	FlagLoc05PirateAttackSeen = "loc05.pirate_attack_seen"

	flagLoc05DefaultsInitialized = "loc05.defaults_initialized"
	flagLoc05GreetingPending     = "loc05.greeting_pending"
	flagLoc05Conversation        = "loc05.conversation"
	flagLoc05Flowers             = "loc05.flowers"
	flagLoc05LouiDepartureDone   = "loc05.loui_departure_done"
	flagLoc05QuestA              = "loc05.quest_a"
	flagLoc05QuestB              = "loc05.quest_b"

	loc05StateQuestA            = 0x3e84
	loc05StateQuestB            = 0x3ea0
	loc05StatePirateAttackSeen  = 0x3ec8
	loc05StateGreetingPending   = 0x3ed8
	loc05StateConversation      = 0x3edc
	loc05StateFlowers           = 0x3ee0
	loc05StateLouiDepartureDone = 0x3ee4
)

type LOC05Controller struct{}

func (LOC05Controller) Enter(ctx *Context, scene string, from string) engine.Task {
	if scene != loc05Scene && scene != "S10" {
		return nil
	}
	actor := loc05ActorID(ctx)
	if actor != "" {
		if a, _, ok := ctx.session.actorByNameOrAsset(actor); ok {
			a.Visible = false
			a.X, a.Y = 0x137, 0xDA
			engine.UpdateActorPerspective(ctx.session.scene, a)
		}
	}
	tasks := []engine.Task{
		ctx.PlayMusic("Loc05_LouisOffice.wav"),
		loc05InitializeLayers(ctx),
		loc05BindLouiWalkSFX(ctx),
		ctx.RunAmbient(ctx.PlayLayerFrames("S10_LouiTalk", 1, 0x0F)),
	}
	if from == "" {
		from = ctx.session.state.PreviousScene
	}
	tasks = append(tasks,
		ctx.PlayLayerFrames("S10_DoorOpen", 0, 4),
		ctx.PlaySFX("Sfx_Door_CreakShort.wav"),
		ctx.PlayLayerFrames("S10_DoorOpen", 5, -1),
	)
	if actor != "" {
		tasks = append(tasks,
			loc05PlaceActorPerspective(ctx, actor, 0x137, 0xDA),
			ctx.ShowActor(actor),
		)
		if loc05State(ctx, loc05StateGreetingPending) == 0 {
			tasks = append(tasks, ctx.WalkToFacingPerspective(actor, 0x1A3, 0xFF, 1))
		} else {
			tasks = append(tasks,
				loc05SetState(ctx, loc05StateGreetingPending, 0),
				ctx.WalkToFacingPerspective(actor, 0x11B, 0x150, 1),
				loc05LouiSpeak(ctx, "010_LOU_01"),
				loc05Exchange(ctx, "010_ROD_01", "010_LOU_02"),
				loc05Exchange(ctx, "010_ROD_02", "010_LOU_03"),
			)
		}
	}
	tasks = append(tasks, ctx.RunAmbient(newLoc05LouiAmbient(ctx)))
	return engine.Sequence(tasks...)
}

func (LOC05Controller) Exit(ctx *Context, scene string, to string) engine.Task { return nil }

func (LOC05Controller) Click(ctx *Context, area string) engine.Task {
	actor := loc05ActorID(ctx)
	if actor == "" {
		return nil
	}
	switch area {
	case "S10_Exit":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x137, 0xDA, 2), ctx.ChangeLocation(1, "9"))
	case "S10_Desk":
		return loc05SpeakAt(ctx, actor, 0xFB, 0x141, 2, "010_ROD_04")
	case "S10_Flowers1":
		return loc05Flowers(ctx, actor, 0x156, 0x111, 3)
	case "S10_Flowers2":
		return loc05Flowers(ctx, actor, 0x1D8, 0x148, 0)
	case "S10_Kitchen":
		return loc05SpeakAt(ctx, actor, 0x1C4, 0xF0, 4, "010_ROD_07")
	case "S10_Loui":
		return loc05TalkToLoui(ctx, actor)
	case "S10_Truhe":
		return loc05SpeakAt(ctx, actor, 0x1C9, 0x11F, 6, "010_ROD_08")
	case "S10_Krug":
		return loc05SpeakAt(ctx, actor, 0x172, 0xEA, 3, "010_ROD_12")
	case "S10_Ofen":
		return loc05SpeakAt(ctx, actor, 0x1B0, 0xF0, 4, "010_ROD_11")
	case "S10_KastenLinks":
		return loc05SpeakAt(ctx, actor, 0x17E, 0xEF, 4, "010_ROD_09")
	case "S10_KastenRechts":
		return loc05SpeakAt(ctx, actor, 0x1CF, 0xF1, 5, "010_ROD_10")
	}
	return nil
}

func (LOC05Controller) UseItem(ctx *Context, item int, area string) engine.Task { return nil }

func (LOC05Controller) SelectItem(ctx *Context, item int) engine.Task { return nil }

func (LOC05Controller) LoadConditionMask(ctx *Context, scene string) int {
	if (scene == loc05Scene || scene == "S10") && loc05State(ctx, loc05StatePirateAttackSeen) != 0 {
		return 1
	}
	return 0
}

func loc05State(ctx *Context, offset int) int {
	name := ""
	switch offset {
	case loc05StateQuestA:
		name = flagLoc05QuestA
	case loc05StateQuestB:
		name = flagLoc05QuestB
	case loc05StatePirateAttackSeen:
		name = FlagLoc05PirateAttackSeen
	case loc05StateGreetingPending:
		name = flagLoc05GreetingPending
	case loc05StateConversation:
		name = flagLoc05Conversation
	case loc05StateFlowers:
		name = flagLoc05Flowers
	case loc05StateLouiDepartureDone:
		name = flagLoc05LouiDepartureDone
	}
	if name != "" {
		if value, ok := ctx.session.state.Flags[name]; ok {
			return value
		}
	}
	return getOriginalFlag(ctx.session.state.OriginalState, offset)
}

func loc05SetState(ctx *Context, offset, value int) engine.Task {
	return engine.Immediate(func() {
		putOriginalFlag(ctx.session.state.OriginalState, offset, value)
		switch offset {
		case loc05StateQuestA:
			ctx.session.state.Flags[flagLoc05QuestA] = value
		case loc05StateQuestB:
			ctx.session.state.Flags[flagLoc05QuestB] = value
		case loc05StatePirateAttackSeen:
			ctx.session.state.Flags[FlagLoc05PirateAttackSeen] = value
		case loc05StateGreetingPending:
			ctx.session.state.Flags[flagLoc05GreetingPending] = value
		case loc05StateConversation:
			ctx.session.state.Flags[flagLoc05Conversation] = value
		case loc05StateFlowers:
			ctx.session.state.Flags[flagLoc05Flowers] = value
		case loc05StateLouiDepartureDone:
			ctx.session.state.Flags[flagLoc05LouiDepartureDone] = value
		}
	})
}

func loc05BindLouiWalkSFX(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		if loc05State(ctx, loc05StatePirateAttackSeen) == 0 {
			return
		}
		layer, ok := ctx.layer("S10_LouiWalk")
		if !ok {
			return
		}
		events := []struct {
			frame int
			sound string
		}{
			{0x2a, "Gen_StepLeft.wav"},
			{0x2f, "Gen_StepRight.wav"},
			{0x36, "Gen_StepLeft.wav"},
			{0x3a, "Gen_StepRight.wav"},
			{0x5d, "Sfx_WaterSplash.wav"},
			{0x70, "Sfx_WaterSplash.wav"},
			{0x81, "Sfx_Tock.wav"},
			{0x8c, "Gen_StepLeft.wav"},
			{0x92, "Gen_StepRight.wav"},
			{0x97, "Gen_StepLeft.wav"},
		}
		for _, event := range events {
			if len(layer.FrameEvents[event.frame]) != 0 {
				continue
			}
			sound := event.sound
			layer.AddFrameEvent(event.frame, func() { _ = ctx.PlaySFX(sound).Update(0) })
		}
	})
}

func loc05InitializeLayers(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		for _, name := range ctx.session.scene.LayerOrder {
			if l := ctx.session.scene.Layers[name]; l != nil {
				l.Visible = false
				l.Playing = false
				l.Accumulator = 0
				l.TaskDriven = false
			}
		}
	})
}

func loc05PlaceActorPerspective(ctx *Context, actorID string, x, y float64) engine.Task {
	return engine.Immediate(func() {
		actor, _, ok := ctx.session.actorByNameOrAsset(actorID)
		if !ok {
			return
		}
		actor.X, actor.Y = x, y
		engine.UpdateActorPerspective(ctx.session.scene, actor)
	})
}

func loc05ActorID(ctx *Context) string {
	for _, name := range []string{"RodrigoLarge", "RodrigoSmall"} {
		if _, id, ok := ctx.session.actorByNameOrAsset(name); ok {
			return id
		}
	}
	if ctx.session.scene != nil && len(ctx.session.scene.CharacterOrder) != 0 {
		return ctx.session.scene.CharacterOrder[0]
	}
	return ""
}

func loc05SpeakAt(ctx *Context, actor string, x, y float64, dir int, line string) engine.Task {
	return engine.Sequence(ctx.WalkToFacingPerspective(actor, x, y, dir), ctx.Say(actor, line, "["+line+"]"))
}

func loc05Flowers(ctx *Context, actor string, x, y float64, dir int) engine.Task {
	switch loc05State(ctx, loc05StateFlowers) {
	case 1:
		return engine.Sequence(loc05SpeakAt(ctx, actor, x, y, dir, "010_ROD_05"), loc05SetState(ctx, loc05StateFlowers, 2))
	case 2:
		return loc05SpeakAt(ctx, actor, x, y, dir, "010_ROD_06")
	}
	return nil
}

func loc05LouiSpeak(ctx *Context, line string) engine.Task {
	return engine.Sequence(
		ctx.ShowLayer("S10_LouiTalk"),
		ctx.PlaySpeechBoundToLayer("S10_LouiTalk", line, "["+line+"]", 0xF, 0x17),
		ctx.FreezeLayer("S10_LouiTalk", 0x0F),
	)
}

func loc05Exchange(ctx *Context, rod, loui string) engine.Task {
	actor := loc05ActorID(ctx)
	if actor == "" {
		return loc05LouiSpeak(ctx, loui)
	}
	return engine.Sequence(ctx.Say(actor, rod, "["+rod+"]"), loc05LouiSpeak(ctx, loui))
}

func loc05WalkToLoui(ctx *Context, actor string) engine.Task {
	return ctx.WalkToFacingPerspective(actor, 0x11B, 0x150, 1)
}

func loc05TalkToLoui(ctx *Context, actor string) engine.Task {
	state := loc05State(ctx, loc05StateConversation)
	prefix := []engine.Task{loc05WalkToLoui(ctx, actor)}
	var body engine.Task
	switch state {
	case 1:
		body = engine.Sequence(loc05SetState(ctx, loc05StateConversation, 2), loc05Exchange(ctx, "028_ROD_01", "028_LOU_01"))
	case 2:
		body = engine.Sequence(loc05SetState(ctx, loc05StateConversation, 3), loc05Exchange(ctx, "028_ROD_02", "028_LOU_02"))
	case 3:
		body = engine.Sequence(
			loc05SetState(ctx, loc05StateConversation, 4),
			loc05Exchange(ctx, "028_ROD_03", "028_LOU_03"),
			loc05SetState(ctx, loc05StateQuestA, 1),
			loc05SetState(ctx, loc05StateQuestB, 1),
		)
	case 4:
		body = engine.Sequence(loc05SetState(ctx, loc05StateConversation, 5), loc05Exchange(ctx, "028_ROD_04", "028_LOU_04"))
	case 5:
		if loc05State(ctx, loc05StatePirateAttackSeen) != 0 && loc05State(ctx, loc05StateLouiDepartureDone) == 0 {
			body = engine.Sequence(loc05SetState(ctx, loc05StateConversation, 6), loc05Exchange(ctx, "028_ROD_08", "028_LOU_08"))
		} else {
			choice := rand.IntN(3)
			body = loc05Exchange(ctx, []string{"028_ROD_05", "028_ROD_06", "028_ROD_07"}[choice], []string{"028_LOU_05", "028_LOU_06", "028_LOU_07"}[choice])
		}
	case 6:
		body = engine.Sequence(loc05SetState(ctx, loc05StateConversation, 7), loc05Exchange(ctx, "028_ROD_09", "028_LOU_09"))
	case 7:
		body = engine.Sequence(loc05SetState(ctx, loc05StateConversation, 8), loc05Exchange(ctx, "028_ROD_10", "028_LOU_10"))
	case 8:
		if loc05State(ctx, loc05StateQuestA) == 2 {
			body = engine.Sequence(loc05Exchange(ctx, "028_ROD_11", "028_LOU_11"), loc05SetState(ctx, loc05StateConversation, 9))
		} else {
			body = engine.Sequence(loc05SetState(ctx, loc05StateConversation, 9), loc05LouiDeparture(ctx, actor))
		}
	case 9:
		body = loc05LouiDeparture(ctx, actor)
	default:
		body = engine.Sequence(loc05SetState(ctx, loc05StateConversation, 1), loc05Exchange(ctx, "028_ROD_01", "028_LOU_01"), loc05SetState(ctx, loc05StateConversation, 2))
	}
	prefix = append(prefix, body)
	return engine.Sequence(prefix...)
}

func loc05LouiDeparture(ctx *Context, actor string) engine.Task {
	return engine.Sequence(
		loc05Exchange(ctx, "028_ROD_12", "028_LOU_12"),
		ctx.Say(actor, "029_ROD_01", "[029_ROD_01]"),
		ctx.HideLayer("S10_LouiTalk"),
		ctx.ShowLayer("S10_LouiWalk"),
		ctx.PlayLayerFrames("S10_LouiWalk", 1, 0x2B),
		ctx.SetActorOrientation(actor, 0),
		ctx.PlayLayerFrames("S10_LouiWalk", 0x2C, 0x3C),
		ctx.SetActorOrientation(actor, 7),
		ctx.PlayLayerFrames("S10_LouiWalk", 0x3D, 0x60),
		ctx.PlaySpeechBoundToLayer("S10_LouiWalk", "029_LOU_01", "[029_LOU_01]", 0x60, 0x7A),
		ctx.Say(actor, "029_ROD_02", "[029_ROD_02]"),
		ctx.PlaySpeechBoundToLayer("S10_LouiWalk", "029_LOU_02", "[029_LOU_02]", 0x60, 0x7A),
		ctx.Say(actor, "029_ROD_03", "[029_ROD_03]"),
		ctx.PlaySpeechBoundToLayer("S10_LouiWalk", "029_LOU_03", "[029_LOU_03]", 0x60, 0x7A),
		ctx.Say(actor, "029_ROD_04", "[029_ROD_04]"),
		ctx.PlaySpeechBoundToLayer("S10_LouiWalk", "029_LOU_04", "[029_LOU_04]", 0x60, 0x7A),
		ctx.PlayLayerFrames("S10_LouiWalk", 0x7C, 0x8C),
		ctx.SetActorOrientation(actor, 0),
		ctx.PlayLayerFrames("S10_LouiWalk", 0x8D, 0x96),
		ctx.SetActorOrientation(actor, 1),
		ctx.PlayLayerFrames("S10_LouiWalk", 0x97, 0xAE),
		ctx.HideLayer("S10_LouiWalk"),
		ctx.ShowLayer("S10_LouiTalk"),
		ctx.FreezeLayer("S10_LouiTalk", 0x0F),
		loc05SetState(ctx, loc05StateLouiDepartureDone, 1),
		loc05SetState(ctx, loc05StateConversation, 5),
	)
}

type loc05LouiAmbient struct {
	ctx       *Context
	remaining float64
	active    engine.Task
}

func newLoc05LouiAmbient(ctx *Context) engine.Task {
	return &loc05LouiAmbient{ctx: ctx, remaining: float64(10+rand.IntN(11)) / 10}
}

func (t *loc05LouiAmbient) Update(dt float64) bool {
	if t.ctx.session.scene == nil || (t.ctx.session.state.Scene != loc05Scene && t.ctx.session.state.Scene != "S10") {
		return true
	}
	if t.active != nil {
		if !t.active.Update(dt) {
			return false
		}
		t.active = nil
		t.remaining = float64(10+rand.IntN(11)) / 10
		return false
	}
	if t.ctx.session.Locked() {
		return false
	}
	t.remaining -= dt
	if t.remaining > 0 {
		return false
	}
	if l, ok := t.ctx.session.scene.Layers["S10_LouiTalk"]; ok && l.Visible {
		ranges := [][2]int{{1, 0xF}, {3, 1}, {4, 1}, {3, 0xF}}
		r := ranges[rand.IntN(len(ranges))]
		t.active = t.ctx.PlayLayerFrames("S10_LouiTalk", r[0], r[1])
	} else {
		t.remaining = 1
	}
	return false
}
