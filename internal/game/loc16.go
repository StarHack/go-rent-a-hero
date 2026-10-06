package game

import (
	"fmt"
	"log"
	"math/rand/v2"

	"github.com/wok/rent-a-hero/internal/engine"
)

const (
	loc16StateNight          = 0x3e80
	loc16StateLandingTalk    = 0x3f40
	loc16StatePirateBurn     = 0x3fa8
	loc16StateEntryTalk      = 0x3fa4
	loc16StateDragonStage    = 0x3fac
	loc16StateNightStage     = 0x3fb0
	loc16StateFlightStage    = 0x3fb4
	loc16StateCanExit59      = 0x3fb8
	loc16StateDragonPresent  = 0x3fbc
	loc16StateHut            = 0x3fc0
	loc16StateHand           = 0x3fc4
	loc16StatePiratesGone    = 0x3fc8
	loc16StateDragonOverride = 0x409c
)

type LOC16Controller struct{}

func (LOC16Controller) Enter(ctx *Context, scene, from string) engine.Task {
	if scene != "58" && scene != "S58" {
		return nil
	}
	if getOriginalFlag(ctx.session.state.OriginalState, loc16StatePirateBurn) != 0 {
		if _, _, err := ctx.ensureAssetLayer("S58_PirateBurn"); err != nil {
			log.Printf("game: load S58_PirateBurn: %v", err)
		}
	}
	actor := loc15ActorID(ctx)
	tasks := []engine.Task{loc16InitializeScene(ctx, actor, from), ctx.PlayMusic("Loc16_DragonsLair.wav")}
	if getOriginalFlag(ctx.session.state.OriginalState, loc16StateNight) != 0 {
		tasks = append(tasks, ctx.SetBackgroundByStem("S58_Back_nd"), ctx.ShowLayer("S58_DragonTalk_nd"))
	} else {
		tasks = append(tasks, ctx.ShowLayer("S58_DragonTalk"))
	}
	if getOriginalFlag(ctx.session.state.OriginalState, loc16StateHand) != 0 {
		tasks = append(tasks, ctx.ShowLayer("S58_Hand"), ctx.MakeLayerClickable("S58_Hand"))
	} else {
		tasks = append(tasks, ctx.HideLayer("S58_Hand"), ctx.DisableArea("S58_Hand"))
	}
	if getOriginalFlag(ctx.session.state.OriginalState, loc16StateHut) != 0 {
		tasks = append(tasks, ctx.ShowLayer("S58_Hut"), ctx.MakeLayerClickable("S58_Hut"))
	} else {
		tasks = append(tasks, ctx.HideLayer("S58_Hut"), ctx.DisableArea("S58_Hut"))
	}
	if actor != "" {
		switch loc15NumericScene(from) {
		case 59:
			tasks = append(tasks, ctx.PlaceActorPerspective(actor, 500, 0xda))
			if getOriginalFlag(ctx.session.state.OriginalState, loc16StatePirateBurn) == 0 {
				tasks = append(tasks, ctx.WalkToFacingPerspective(actor, 0x1e0, 0x13b, 1))
			} else {
				tasks = append(tasks,
					ctx.WalkToFacingPerspective(actor, 0x15d, 0x138, 5),
					loc16DragonVoice(ctx, "058_DRA_03", 0, 8),
					ctx.PlaySFX("Sfx_DangerStringsHigh.wav"),
					ctx.Say(actor, "058_ROD_03", ""),
					ctx.HideActor(actor),
					ctx.ShowLayer("S58_PirateBurn"),
					ctx.PlaySFX("Sfx_DangerStrings.wav"),
					ctx.PlayLayerFrames("S58_PirateBurn", 0, 0x13),
					ctx.PlaySFX("Sfx_Knock_Wet.wav"),
					ctx.PlayLayerFrames("S58_PirateBurn", 0x14, 0x1f),
					ctx.PlaySFX("Sfx_Knock_Wet.wav"),
					ctx.PlayLayerFrames("S58_PirateBurn", 0x20, 0x28),
					ctx.PlaySFX("Sfx_Nose_BeingBroken.wav"),
					ctx.PlayLayerFrames("S58_PirateBurn", 0x29, 0x31),
					ctx.PlaySFX("Sfx_Knock_Wet.wav"),
					ctx.PlayLayerFrames("S58_PirateBurn", 0x32, 0x57),
					ctx.PlaySFX("Sfx_DragonFire.wav"),
					ctx.PlayLayerFrames("S58_PirateBurn", 0x58, -1),
					ctx.HideLayer("S58_PirateBurn"),
					ctx.ShowActor(actor),
					loc15SetOriginal(ctx, loc16StateHand, 1),
					ctx.ShowLayer("S58_Hand"), ctx.MakeLayerClickable("S58_Hand"),
					loc15SetOriginal(ctx, loc16StateHut, 1),
					ctx.ShowLayer("S58_Hut"), ctx.MakeLayerClickable("S58_Hut"),
					ctx.PlaceActorPerspective(actor, 0x15d, 0x138),
					ctx.WalkToFacingPerspective(actor, 0x178, 0x13d, 4),
					loc16Conversation(ctx, "058_ROD_04", "058_DRA_04"),
					loc15SetOriginal(ctx, loc16StatePirateBurn, 0),
				)
			}
		case 93:
			tasks = append(tasks,
				ctx.ShowLayer("S58_DragonLanding"),
				ctx.PlayLayerFrames("S58_DragonLanding", 0, 0x21),
				ctx.PlaySFX("Sfx_Clothes_Scratched.wav"),
				ctx.PlayLayerFrames("S58_DragonLanding", 0x22, -1),
				ctx.HideLayer("S58_DragonLanding"),
				ctx.PlaceActorPerspective(actor, 0x1ab, 0x11e),
				ctx.WalkToFacingPerspective(actor, 0x11d, 0x140, 6),
			)
			if !ctx.HasItem(5) && getOriginalFlag(ctx.session.state.OriginalState, 0x3e90) != 0 && getOriginalFlag(ctx.session.state.OriginalState, loc16StateLandingTalk) == 0 {
				tasks = append(tasks, loc16Conversation(ctx, "058_ROD_05", "058_DRA_05"), loc16Conversation(ctx, "058_ROD_06", "058_DRA_06"), loc15SetOriginal(ctx, loc16StateLandingTalk, 1))
			}
		default:
			tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x10, 300))
			if getOriginalFlag(ctx.session.state.OriginalState, loc16StateEntryTalk) == 0 {
				tasks = append(tasks, ctx.WalkToFacingPerspective(actor, 0xc0, 0x125, 6))
			} else {
				tasks = append(tasks, ctx.WalkToFacingPerspective(actor, 0xc0, 0x125, 6), loc16DragonVoice(ctx, "058_DRA_01", 0, 8), loc16Conversation(ctx, "058_ROD_01", "058_DRA_02"), loc15SetOriginal(ctx, loc16StateEntryTalk, 0))
			}
		}
	}
	tasks = append(tasks, loc16ConfigureDragonAccess(ctx), ctx.EnableArea("S58_Dragon"), loc16InstallDragonIdle(ctx), ctx.RunAmbient(&loc16Ambient{ctx: ctx}))
	return engine.Sequence(tasks...)
}

func loc16InitializeScene(ctx *Context, actor, from string) engine.Task {
	return engine.Immediate(func() {
		night := getOriginalFlag(ctx.session.state.OriginalState, loc16StateNight) != 0
		if !night && getOriginalFlag(ctx.session.state.OriginalState, loc16StatePirateBurn) != 0 {
			loc16PreparePresentation(ctx, "S58_PirateBurn")
		}
		if night {
			if getOriginalFlag(ctx.session.state.OriginalState, loc16StateDragonPresent) != 0 {
				loc16PreparePresentation(ctx, "S58_DragonTakeOff")
			}
			if loc15NumericScene(from) == 93 {
				loc16PreparePresentation(ctx, "S58_DragonLanding")
			}
		}
		for _, id := range []string{"S58_Hut", "S58_Hand", "S58_DragonTalk", "S58_DragonTalk_nd", "S58_RodAngriff", "S58_RodLooksSky", "S58_RodPickUp", "S58_RodSitDown", "S58_PirateBurn", "S58_DragonTakeOff", "S58_DragonLanding"} {
			if layer, ok := ctx.session.scene.Layers[id]; ok {
				layer.Visible = false
				layer.Playing = false
				layer.TaskDriven = false
				layer.Frame = 0
				layer.Accumulator = 0
			}
		}
		if night && actor != "" {
			if a, ok := ctx.session.scene.Characters[actor]; ok {
				a.Color.R = -52
				a.Color.G = -11
				a.Color.B = 43
			}
		}
	})
}

func loc16PreparePresentation(ctx *Context, id string) {
	layer, _, err := ctx.ensureAssetLayer(id)
	if err != nil {
		log.Printf("game: load %s: %v", id, err)
		return
	}
	layer.Presentation = true
	layer.ColorKeyed = false
}

func loc16ConfigureDragonAccess(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		blocked := getOriginalFlag(ctx.session.state.OriginalState, loc16StateCanExit59) == 0 || getOriginalFlag(ctx.session.state.OriginalState, loc16StatePiratesGone) != 0
		if bg := ctx.session.scene.Background; bg != nil {
			if nav := bg.NavGrid(); nav != nil {
				nav.SetDynamicRect(0, 0, 0x27f, 0x106, blocked)
			}
		}
	})
}

func (LOC16Controller) Exit(*Context, string, string) engine.Task { return nil }

func (LOC16Controller) Click(ctx *Context, area string) engine.Task {
	actor := loc15ActorID(ctx)
	switch area {
	case "S58_To57":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x10, 300, 2), ctx.ChangeLocation(15, "57"))
	case "S58_To59":
		if getOriginalFlag(ctx.session.state.OriginalState, loc16StatePiratesGone) != 0 {
			return engine.Sequence(loc16DragonVoice(ctx, "058_DRA_34", 0, 8), ctx.PlayVoiceover("058_ROD_35", ""))
		}
		if getOriginalFlag(ctx.session.state.OriginalState, loc16StateCanExit59) == 0 {
			return engine.Sequence(loc16DragonReturn(ctx, false), ctx.WalkToFacingPerspective(actor, 0x201, 0x11b, 2), loc16DragonPosedVoice(ctx, "058_DRA_33", 3, 0x49, 0x4d), ctx.WalkToFacingPerspective(actor, 0x1ca, 0x138, 3), loc16DragonReturn(ctx, false), loc16ScheduleDragonIdle(ctx))
		}
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 500, 0xda, 4), ctx.ChangeLocation(17, "59"))
	case "S58_Dragon":
		return loc16DragonClick(ctx, actor)
	case "S58_Hut":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x115, 0x15e, 4), ctx.HideActor(actor), loc16PlaceLayer(ctx, "S58_RodPickUp", 0xf8, 0xf6), ctx.ShowLayer("S58_RodPickUp"), ctx.PlayLayerFrames("S58_RodPickUp", 0, 9), ctx.HideLayer("S58_Hut"), ctx.DisableArea("S58_Hut"), ctx.PlayLayerFrames("S58_RodPickUp", 10, -1), ctx.HideLayer("S58_RodPickUp"), loc15SetOriginal(ctx, loc16StateHut, 0), ctx.AddItem(4), ctx.ShowActor(actor))
	case "S58_Hand":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x137, 0x15b, 4), ctx.HideActor(actor), loc16PlaceLayer(ctx, "S58_RodPickUp", 0x11a, 0xf3), ctx.ShowLayer("S58_RodPickUp"), ctx.PlayLayerFrames("S58_RodPickUp", 0, 9), ctx.HideLayer("S58_Hand"), ctx.DisableArea("S58_Hand"), ctx.PlayLayerFrames("S58_RodPickUp", 10, -1), ctx.HideLayer("S58_RodPickUp"), loc15SetOriginal(ctx, loc16StateHand, 0), ctx.AddItem(3), ctx.ShowActor(actor))
	}
	return nil
}

func loc16PlaceLayer(ctx *Context, id string, x, y int) engine.Task {
	return engine.Immediate(func() {
		if layer, ok := ctx.layer(id); ok {
			layer.X = x
			layer.Y = y
		}
	})
}

func (LOC16Controller) UseItem(*Context, int, string) engine.Task { return nil }
func (LOC16Controller) SelectItem(*Context, int) engine.Task      { return nil }
func (LOC16Controller) LoadConditionMask(ctx *Context, scene string) int {
	if scene != "58" && scene != "S58" {
		return 0
	}
	if getOriginalFlag(ctx.session.state.OriginalState, loc16StateNight) != 0 {
		return 2
	}
	return 1
}

func loc16DragonClick(ctx *Context, actor string) engine.Task {
	state := ctx.session.state.OriginalState
	move := engine.Task(engine.Immediate(func() {}))
	if getOriginalFlag(state, loc16StateNight) == 0 || getOriginalFlag(state, loc16StateDragonPresent) == 0 || getOriginalFlag(state, loc16StateDragonOverride) != 0 {
		move = ctx.WalkToFacingPerspective(actor, 0x11d, 0x140, 6)
	}
	if getOriginalFlag(state, loc16StatePiratesGone) == 0 {
		stage := getOriginalFlag(state, loc16StateDragonStage)
		if stage == 1 {
			return engine.Sequence(move, ctx.PlayVoiceover("058_ROD_07", ""), ctx.HideActor(actor), ctx.ShowLayer("S58_RodAngriff"), ctx.PlayLayerFrames("S58_RodAngriff", 0, 4), ctx.PlaySFX("Sfx_Woosh.wav"), ctx.PlayLayerFrames("S58_RodAngriff", 5, 10), loc16DragonVoice(ctx, "058_DRA_07", 0, 8), ctx.PlayLayerFrames("S58_RodAngriff", 11, -1), ctx.HideLayer("S58_RodAngriff"), ctx.ShowActor(actor), loc15SetOriginal(ctx, loc16StateDragonStage, 2))
		}
		if stage == 9 {
			return engine.Sequence(move, loc16DragonReturn(ctx, false), ctx.PlayVoiceover("058_ROD_15", ""), loc16DragonPosedVoice(ctx, "058_DRA_15", 4, 0x6e, 0x76), loc16DragonReturn(ctx, false), loc15SetOriginal(ctx, loc16StateCanExit59, 1), loc16ConfigureDragonAccess(ctx), ctx.WalkToFacingPerspective(actor, 500, 0xda, 4), loc15SetOriginal(ctx, loc16StateDragonStage, 10), ctx.ChangeLocation(17, "59"))
		}
		rod := fmt.Sprintf("058_ROD_%02d", stage+6)
		dra := fmt.Sprintf("058_DRA_%02d", stage+6)
		tasks := []engine.Task{move, loc16Conversation(ctx, rod, dra)}
		if stage < 11 {
			tasks = append(tasks, loc15SetOriginal(ctx, loc16StateDragonStage, stage+1))
		}
		return engine.Sequence(tasks...)
	}
	if getOriginalFlag(state, loc16StateDragonPresent) != 0 {
		stage := getOriginalFlag(state, loc16StateFlightStage)
		switch stage {
		case 1, 2, 3:
			return engine.Sequence(move, loc16Conversation(ctx, fmt.Sprintf("058_ROD_%02d", stage+24), fmt.Sprintf("058_DRA_%02d", stage+24)), loc15SetOriginal(ctx, loc16StateFlightStage, stage+1))
		case 4:
			return engine.Sequence(move, loc16Conversation(ctx, "058_ROD_28", "058_DRA_28"), ctx.PlayVoiceover("058_ROD_29", ""), loc15SetOriginal(ctx, loc16StateFlightStage, 5))
		case 5:
			return engine.Sequence(move, ctx.WalkToFacingPerspective(actor, 0x1ab, 0x11e, 4), loc16DragonVoice(ctx, "058_DRA_30", 0, 8), loc16Conversation(ctx, "058_ROD_30", "058_DRA_31"), loc15SetOriginal(ctx, loc16StateFlightStage, 6), loc16DragonTakeoff(ctx, actor))
		case 6:
			return engine.Sequence(move, &loc16FlightStage6Task{ctx: ctx, actor: actor})
		}
	}
	if getOriginalFlag(state, loc16StateNight) == 0 {
		return engine.Sequence(move, loc16Conversation(ctx, "058_ROD_18", "058_DRA_18"))
	}
	stage := getOriginalFlag(state, loc16StateNightStage)
	var talk engine.Task
	switch stage {
	case 1:
		talk = engine.Sequence(loc16DragonReturn(ctx, false), ctx.PlayVoiceover("058_ROD_19", ""), loc16DragonPosedVoice(ctx, "058_DRA_19", 5, 0x8b, 0x94), loc16DragonReturn(ctx, false), loc16ScheduleDragonIdle(ctx))
	case 4:
		talk = engine.Sequence(ctx.SetActorDirection(actor, 5), ctx.HideActor(actor), ctx.ShowLayer("S58_RodLooksSky"), ctx.PlayLayer("S58_RodLooksSky"), ctx.HideLayer("S58_RodLooksSky"), ctx.ShowActor(actor), ctx.PlayVoiceover("058_ROD_22", ""), ctx.WalkToFacingPerspective(actor, 0x160, 0x155, 5), ctx.HideActor(actor), ctx.ShowLayer("S58_RodSitDown"), ctx.PlayLayerFrames("S58_RodSitDown", 0, 10), loc16DragonVoice(ctx, "058_DRA_22", 0, 8), ctx.PlayLayerFrames("S58_RodSitDown", 11, -1), ctx.HideLayer("S58_RodSitDown"), ctx.ShowActor(actor))
	case 6:
		talk = engine.Sequence(loc16Conversation(ctx, "058_ROD_24", "058_DRA_24"), loc16SetFlag1To2(ctx, 0x3f74))
	default:
		talk = loc16Conversation(ctx, fmt.Sprintf("058_ROD_%02d", stage+18), fmt.Sprintf("058_DRA_%02d", stage+18))
	}
	tasks := []engine.Task{move, talk}
	if stage < 6 {
		tasks = append(tasks, loc15SetOriginal(ctx, loc16StateNightStage, stage+1))
	}
	return engine.Sequence(tasks...)
}

func loc16DragonTakeoff(ctx *Context, actor string) engine.Task {
	return engine.Sequence(loc16DragonReturn(ctx, true), ctx.WalkToFacingPerspective(actor, 0x1ab, 0x11e, 4), ctx.HideActor(actor), ctx.ShowLayer("S58_DragonTakeOff"), ctx.PlayLayerFrames("S58_DragonTakeOff", 0, 7), ctx.PlaySFX("Sfx_Clothes_Scratched.wav"), ctx.PlayLayerFrames("S58_DragonTakeOff", 8, 0x43), ctx.PlaySFX("Sfx_DragonWings2.wav"), ctx.PlayLayerFrames("S58_DragonTakeOff", 0x44, 0x57), ctx.PlaySFX("Sfx_DragonWings2.wav"), ctx.PlayLayerFrames("S58_DragonTakeOff", 0x58, -1), ctx.HideLayer("S58_DragonTakeOff"), ctx.ChangeLocation(26, "93"))
}

func loc16DragonLayer(ctx *Context) string {
	if getOriginalFlag(ctx.session.state.OriginalState, loc16StateNight) != 0 {
		return "S58_DragonTalk_nd"
	}
	return "S58_DragonTalk"
}

func loc16DragonVoice(ctx *Context, name string, start, end int) engine.Task {
	return engine.Sequence(loc16DragonReturn(ctx, true), loc16DragonRawVoice(ctx, name, start, end), loc16ScheduleDragonIdle(ctx))
}

func loc16DragonRawVoice(ctx *Context, name string, start, end int) engine.Task {
	return ctx.PlaySpeechBoundToLayer(loc16DragonLayer(ctx), name, "", start, end)
}

func loc16Conversation(ctx *Context, rod, dragon string) engine.Task {
	actor := loc15ActorID(ctx)
	return engine.Sequence(loc16DragonReturn(ctx, false), ctx.Say(actor, rod, ""), loc16DragonReturn(ctx, true), loc16DragonRawVoice(ctx, dragon, 0, 8), loc16ScheduleDragonIdle(ctx))
}

func loc16DragonPosedVoice(ctx *Context, name string, pose, start, end int) engine.Task {
	return engine.Sequence(loc16DragonReturn(ctx, true), loc16DragonSetPose(ctx, pose), loc16DragonRawVoice(ctx, name, start, end))
}

func loc16DragonReturn(ctx *Context, blocking bool) engine.Task {
	task := engine.Sequence(loc16PauseDragonIdle(ctx), &loc16DragonReturnTask{ctx: ctx})
	if blocking {
		return task
	}
	return ctx.RunAmbient(task)
}

func loc16DragonSetPose(ctx *Context, pose int) engine.Task {
	from, to := 0, 0
	switch pose {
	case 1:
		from, to = 0x55, 0x5a
	case 2:
		from, to = 0x81, 0x84
	case 3:
		from, to = 0x30, 0x49
	case 4:
		from, to = 0x56, 0x6e
	case 5:
		from, to = 0x81, 0x8b
	default:
		return engine.Immediate(func() {})
	}
	return ctx.PlayLayerFrames(loc16DragonLayer(ctx), from, to)
}

type loc16DragonReturnTask struct {
	ctx     *Context
	inner   engine.Task
	started bool
}

func (t *loc16DragonReturnTask) Update(dt float64) bool {
	if !t.started {
		layer, ok := t.ctx.layer(loc16DragonLayer(t.ctx))
		if !ok {
			return true
		}
		if layer.TaskDriven {
			return false
		}
		t.started = true
		from, to := loc16DragonReturnRange(layer.Frame)
		if from < 0 {
			return true
		}
		t.inner = t.ctx.PlayLayerFrames(layer.ID, from, to)
	}
	if t.inner == nil {
		return true
	}
	return t.inner.Update(dt)
}

func loc16DragonReturnRange(frame int) (int, int) {
	switch {
	case frame >= 0x55 && frame <= 0x5a:
		return 0x5a, 0x55
	case frame >= 0x81 && frame <= 0x84:
		return 0x84, 0x81
	case frame >= 0x30 && frame <= 0x4d:
		return 0x4d, 0x56
	case frame >= 0x56 && frame <= 0x76:
		return 0x76, 0x81
	case frame >= 0x8b:
		return 0x8b, 0x81
	default:
		return -1, -1
	}
}

type loc16FlightStage6Task struct {
	ctx     *Context
	actor   string
	inner   engine.Task
	started bool
}

func (t *loc16FlightStage6Task) Update(dt float64) bool {
	if !t.started {
		t.started = true
		if loc16ActorNear(t.ctx, t.actor, 0x1ab, 0x11e, 10) {
			t.inner = loc16DragonTakeoff(t.ctx, t.actor)
		} else {
			t.inner = engine.Sequence(loc16Conversation(t.ctx, "058_ROD_32", "058_DRA_32"), t.ctx.WalkToFacingPerspective(t.actor, 0x1ab, 0x11e, 4))
		}
	}
	if t.inner == nil {
		return true
	}
	return t.inner.Update(dt)
}

func loc16ActorNear(ctx *Context, actorID string, x, y, tolerance float64) bool {
	a, ok := ctx.actor(actorID)
	if !ok {
		return false
	}
	dx := a.X - x
	if dx < 0 {
		dx = -dx
	}
	dy := a.Y - y
	if dy < 0 {
		dy = -dy
	}
	return dx < tolerance && dy < tolerance
}

func loc16SetFlag1To2(ctx *Context, offset int) engine.Task {
	return engine.Immediate(func() {
		if getOriginalFlag(ctx.session.state.OriginalState, offset) == 1 {
			putOriginalFlag(ctx.session.state.OriginalState, offset, 2)
		}
	})
}

var loc16DragonIdles = map[*Session]*loc16DragonIdle{}

func loc16InstallDragonIdle(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		idle := &loc16DragonIdle{ctx: ctx, wait: float64(10+rand.IntN(11)) / 10}
		loc16DragonIdles[ctx.session] = idle
		ctx.session.ambientTasks = append(ctx.session.ambientTasks, idle)
	})
}

func loc16PauseDragonIdle(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		if idle := loc16DragonIdles[ctx.session]; idle != nil {
			idle.paused = true
			idle.wait = 0
		}
	})
}

func loc16ScheduleDragonIdle(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		if idle := loc16DragonIdles[ctx.session]; idle != nil {
			idle.paused = false
			idle.wait = float64(10+rand.IntN(11)) / 10
		}
	})
}

type loc16DragonIdle struct {
	ctx    *Context
	wait   float64
	inner  engine.Task
	paused bool
}

func (t *loc16DragonIdle) Update(dt float64) bool {
	if t.ctx.session.state.Location != 16 || loc15NumericScene(t.ctx.session.state.Scene) != 58 {
		if loc16DragonIdles[t.ctx.session] == t {
			delete(loc16DragonIdles, t.ctx.session)
		}
		return true
	}
	layer, ok := t.ctx.layer(loc16DragonLayer(t.ctx))
	if !ok || !layer.Visible {
		return false
	}
	if t.inner != nil {
		if t.inner.Update(dt) {
			t.inner = nil
			if !t.paused {
				t.wait = float64(5 + rand.IntN(3))
			}
		}
		return false
	}
	if t.paused || layer.TaskDriven {
		return false
	}
	t.wait -= dt
	if t.wait > 0 {
		return false
	}
	from, _ := loc16DragonReturnRange(layer.Frame)
	if from >= 0 {
		t.inner = &loc16DragonReturnTask{ctx: t.ctx}
	} else {
		t.inner = loc16DragonSetPose(t.ctx, 1+rand.IntN(2))
	}
	return false
}

type loc16Ambient struct {
	ctx  *Context
	wait float64
}

func (t *loc16Ambient) Update(dt float64) bool {
	if t.ctx.session.state.Location != 16 || loc15NumericScene(t.ctx.session.state.Scene) != 58 {
		return true
	}
	if t.wait <= 0 {
		t.wait = float64(5 + rand.IntN(11))
	}
	t.wait -= dt
	if t.wait > 0 {
		return false
	}
	if getOriginalFlag(t.ctx.session.state.OriginalState, loc16StateNight) != 0 {
		sounds := []string{"Sfx_Owl.wav", "Sfx_Owl.wav", "Sfx_Thunder.wav", "Sfx_Thunder2.wav"}
		_ = t.ctx.PlaySFX(sounds[rand.IntN(len(sounds))]).Update(0)
	} else {
		sounds := []string{"Sfx_Bird.wav", "Sfx_Bird2.wav", "Sfx_Bird3.wav", "Sfx_Chirp.wav"}
		_ = t.ctx.PlaySFX(sounds[rand.IntN(len(sounds))]).Update(0)
	}
	t.wait = float64(5 + rand.IntN(6))
	return false
}
