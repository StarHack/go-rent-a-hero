package game

import (
	"math/rand/v2"
	"os"
	"strconv"
	"strings"

	"github.com/wok/rent-a-hero/internal/audio"
	"github.com/wok/rent-a-hero/internal/engine"
	"github.com/wok/rent-a-hero/internal/formats/acs"
)

const (
	loc27StateAltConversation = 0x4094
	loc27StateIslandUnlocked  = 0x4098
	loc27StateSecretReady     = 0x40a8
	loc27StateFloorDiscover   = 0x40ac
	loc27StateFloorStage      = 0x40b0
	loc27StateFirstLabVisit   = 0x40b4
	loc27StateEyeUnlocked     = 0x40b8
	loc27StateHelmetDropped   = 0x40bc
	loc27StateMegStage        = 0x40c0
	loc27StateDragonFollowup  = 0x3e90
)

type LOC27Controller struct{}

func loc27SceneID(scene string) int {
	s := strings.ToUpper(strings.TrimSpace(scene))
	s = strings.TrimPrefix(s, "S")
	n, _ := strconv.Atoi(s)
	return n
}

func loc27Actor(ctx *Context) string {
	if actor := ctx.session.PlayerActor(); actor != nil {
		return actor.ID
	}
	return "RodrigoSmall"
}

func loc27SetOriginal(ctx *Context, offset, value int) engine.Task {
	return engine.Immediate(func() {
		putOriginalFlag(ctx.session.state.OriginalState, offset, value)
	})
}

func loc27Rod(ctx *Context, line string) engine.Task {
	return ctx.Say(loc27Actor(ctx), line, "["+line+"]")
}

func loc27Voice(ctx *Context, line string) engine.Task {
	return ctx.PlayVoiceover(line, "["+line+"]")
}

func loc27LoopSFX(ctx *Context, name string) engine.Task {
	sound, err := engine.LoadSound(ctx.session.idx, name)
	if err != nil {
		return engine.Immediate(func() {})
	}
	return engine.Immediate(func() {
		ctx.session.audioEngine.PlayLooping(sound, audio.CategorySFX)
	})
}

func loc27AddFrameSFX(ctx *Context, layerID string, frame int, name string) {
	layer, ok := ctx.layer(layerID)
	if !ok {
		return
	}
	if len(layer.FrameEvents[frame]) != 0 {
		return
	}
	layer.AddFrameEvent(frame, func() { _ = ctx.PlaySFX(name).Update(0) })
}

func loc27EnsureLayer(ctx *Context, id string) {
	_, _, _ = ctx.ensureAssetLayer(id)
}

func loc27PreparePresentation(ctx *Context, id string) {
	loc27EnsureLayer(ctx, id)
	if layer, ok := ctx.layer(id); ok {
		layer.Presentation = true
		layer.ColorKeyed = false
	}
}

func loc27Prepare95(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		for _, id := range []string{"S95_EnterHut", "S95_EnterSecretRoom"} {
			loc27PreparePresentation(ctx, id)
			if layer, ok := ctx.layer(id); ok {
				layer.Visible = false
				layer.Playing = false
				layer.TaskDriven = false
				layer.Frame = 0
				layer.Accumulator = 0
			}
		}
		loc27AddFrameSFX(ctx, "S95_EnterHut", 5, "Gen_StepLeft.wav")
		loc27AddFrameSFX(ctx, "S95_EnterHut", 0xf, "Gen_StepRight.wav")
		loc27AddFrameSFX(ctx, "S95_EnterSecretRoom", 4, "Sfx_MobileUp.wav")
	})
}

func loc27Prepare96(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		for _, id := range []string{
			"S96_RodEntersRoom", "S96_RodAndEye", "S96_RodDropsHelm", "S96_Helm",
			"S96_RodWinkt", "S96_Eye", "S96_EyeDown", "S96_Steam", "S96_Bubble",
			"S96_Microphone", "S96_Pump", "S96_HelmAuf",
		} {
			loc27EnsureLayer(ctx, id)
		}
		loc27PreparePresentation(ctx, "S96_RodEntersRoom")
		loc27PreparePresentation(ctx, "S96_HelmAuf")

		for _, id := range []string{
			"S96_RodEntersRoom", "S96_RodAndEye", "S96_RodDropsHelm", "S96_Helm",
			"S96_RodWinkt", "S96_Eye", "S96_EyeDown", "S96_Steam", "S96_Bubble", "S96_HelmAuf",
		} {
			if layer, ok := ctx.layer(id); ok {
				layer.Visible = false
				layer.Playing = false
				layer.TaskDriven = false
				layer.Frame = 0
				layer.Accumulator = 0
			}
		}
		for _, id := range []string{"S96_Microphone", "S96_Pump"} {
			if layer, ok := ctx.layer(id); ok {
				layer.Visible = true
				layer.Enabled = true
				layer.Playing = false
				layer.TaskDriven = false
				layer.Frame = 0
				layer.Accumulator = 0
			}
		}

		loc27AddFrameSFX(ctx, "S96_RodEntersRoom", 5, "Gen_StepLeft.wav")
		loc27AddFrameSFX(ctx, "S96_RodEntersRoom", 0xc, "Sfx_BigStone.wav")
		loc27AddFrameSFX(ctx, "S96_RodEntersRoom", 0xd, "Sfx_DragonWings2.wav")
		loc27AddFrameSFX(ctx, "S96_RodEntersRoom", 0xf, "Gen_StepRight.wav")
		loc27AddFrameSFX(ctx, "S96_RodEntersRoom", 0x19, "Sfx_BigStone.wav")
		loc27AddFrameSFX(ctx, "S96_RodEntersRoom", 0x22, "Sfx_DragonWings2.wav")
		loc27AddFrameSFX(ctx, "S96_Bubble", 0x15, "Sfx_Bubble.wav")
		loc27AddFrameSFX(ctx, "S96_Pump", 3, "Sfx_DragonWings2.wav")
		loc27AddFrameSFX(ctx, "S96_Pump", 0x16, "Sfx_DragonWings2.wav")
		loc27AddFrameSFX(ctx, "S96_Steam", 1, "Sfx_DragonWings.wav")
		loc27AddFrameSFX(ctx, "S96_RodAndEye", 1, "Sfx_ServoMotor.wav")
		loc27AddFrameSFX(ctx, "S96_RodAndEye", 10, "Sfx_ServoMotor.wav")
		loc27AddFrameSFX(ctx, "S96_RodAndEye", 0xf, "Sfx_ServoMotor.wav")
		loc27AddFrameSFX(ctx, "S96_RodAndEye", 0x14, "Sfx_ServoMotor.wav")
		loc27AddFrameSFX(ctx, "S96_RodWinkt", 6, "Sfx_ServoMotor.wav")
		loc27AddFrameSFX(ctx, "S96_HelmAuf", 0xe, "Sfx_DragonWings2.wav")
		loc27AddFrameSFX(ctx, "S96_HelmAuf", 0x21, "Sfx_DragonWings2.wav")
		loc27AddFrameSFX(ctx, "S96_RodDropsHelm", 0x25, "Sfx_Door_Opened.wav")
	})
}

func (LOC27Controller) Enter(ctx *Context, scene, from string) engine.Task {
	switch loc27SceneID(scene) {
	case 95:
		return loc27Enter95(ctx, from)
	case 96:
		return loc27Enter96(ctx, from)
	default:
		return nil
	}
}

func loc27Enter95(ctx *Context, from string) engine.Task {
	actor := loc27Actor(ctx)
	tasks := []engine.Task{
		ctx.PlayMusic("Loc26_IslandOfMegophias.wav"),
		loc27LoopSFX(ctx, "Sfx_Ocean.wav"),
		loc27Prepare95(ctx),
		ctx.EnableArea("S95_WindowW"),
		ctx.EnableArea("S95_WindowN"),
		ctx.EnableArea("S95_WindowE"),
		ctx.EnableArea("S95_Chair"),
		ctx.EnableArea("S95_ExitTo93"),
	}
	if getOriginalFlag(ctx.session.state.OriginalState, loc27StateFloorDiscover) == 1 {
		tasks = append(tasks, ctx.DisableArea("S95_Floor"))
	} else {
		tasks = append(tasks, ctx.EnableArea("S95_Floor"))
	}
	if loc27SceneID(from) == 96 {
		tasks = append(tasks,
			ctx.ShowActor(actor),
			ctx.PlaceActorPerspective(actor, 0x138, 0x125),
		)
	} else {
		tasks = append(tasks,
			ctx.HideActor(actor),
			ctx.ShowLayer("S95_EnterHut"),
			ctx.PlayLayer("S95_EnterHut"),
			ctx.HideLayer("S95_EnterHut"),
			ctx.ShowActor(actor),
			ctx.PlaceActorPerspective(actor, 0x14c, 0x165),
			ctx.WalkToPerspective(actor, 0x17b, 0x141),
		)
	}
	tasks = append(tasks, ctx.RunAmbient(&loc27Ambient{ctx: ctx, wait: float64(5 + rand.IntN(6))}))
	return engine.Sequence(tasks...)
}

func loc27Enter96(ctx *Context, from string) engine.Task {
	actor := loc27Actor(ctx)
	if getOriginalFlag(ctx.session.state.OriginalState, loc27StateAltConversation) != 0 && getOriginalFlag(ctx.session.state.OriginalState, loc27StateMegStage) > 3 {
		putOriginalFlag(ctx.session.state.OriginalState, loc27StateMegStage, 1)
	}
	tasks := []engine.Task{
		ctx.PlayMusic("Loc26_IslandOfMegophias.wav"),
		loc27LoopSFX(ctx, "Sfx_EndTimeFeeling.wav"),
		loc27Prepare96(ctx),
		ctx.EnableArea("S96_ExitTo95"),
		ctx.EnableArea("S96_Anlagen"),
	}
	if getOriginalFlag(ctx.session.state.OriginalState, loc27StateEyeUnlocked) != 0 {
		tasks = append(tasks, ctx.EnableArea("S96_Steckkontakt"))
	} else {
		tasks = append(tasks, ctx.DisableArea("S96_Steckkontakt"))
	}
	if !ctx.HasItem(5) && getOriginalFlag(ctx.session.state.OriginalState, loc27StateHelmetDropped) == 0 {
		tasks = append(tasks, ctx.EnableArea("S96_Megophias"))
	} else {
		tasks = append(tasks, ctx.DisableArea("S96_Megophias"))
	}
	if getOriginalFlag(ctx.session.state.OriginalState, loc27StateHelmetDropped) != 0 {
		tasks = append(tasks,
			ctx.ShowLayer("S96_Helm"),
			ctx.MakeLayerClickable("S96_Helm"),
		)
	}

	if loc27SceneID(from) == 95 {
		first := getOriginalFlag(ctx.session.state.OriginalState, loc27StateFirstLabVisit) != 0
		tasks = append(tasks,
			ctx.HideActor(actor),
			ctx.ShowLayer("S96_RodEntersRoom"),
			ctx.PlayLayer("S96_RodEntersRoom"),
		)
		if first {
			tasks = append(tasks, loc27Voice(ctx, "096_ROD_01"))
		}
		tasks = append(tasks,
			ctx.HideLayer("S96_RodEntersRoom"),
			ctx.ShowLayer("S96_RodAndEye"),
			ctx.PlayLayer("S96_RodAndEye"),
			ctx.HideLayer("S96_RodAndEye"),
			ctx.ShowActor(actor),
			ctx.PlaceActorPerspective(actor, 0xeb, 0x165),
			ctx.WalkToFacingPerspective(actor, 0xfa, 0x15e, 5),
			ctx.ShowLayer("S96_Eye"),
			ctx.MakeLayerClickable("S96_Eye"),
		)
		if first {
			tasks = append(tasks,
				loc27SetOriginal(ctx, loc27StateFirstLabVisit, 0),
				loc27Meg(ctx, "096_MEG_01"),
			)
		}
	} else {
		tasks = append(tasks,
			ctx.ShowActor(actor),
			ctx.ShowLayer("S96_Eye"),
			ctx.MakeLayerClickable("S96_Eye"),
		)
	}
	tasks = append(tasks, ctx.RunAmbient(&loc27Ambient{ctx: ctx, wait: float64(5 + rand.IntN(6))}))
	return engine.Sequence(tasks...)
}

func (LOC27Controller) Exit(ctx *Context, scene, to string) engine.Task {
	if loc27SceneID(scene) != 95 && loc27SceneID(scene) != 96 {
		return nil
	}
	return engine.Immediate(func() {
		ctx.session.audioEngine.StopCategory(audio.CategorySFX)
	})
}

func (LOC27Controller) Click(ctx *Context, area string) engine.Task {
	switch loc27SceneID(ctx.session.state.Scene) {
	case 95:
		return loc27Click95(ctx, area)
	case 96:
		return loc27Click96(ctx, area)
	default:
		return nil
	}
}

func loc27Click95(ctx *Context, area string) engine.Task {
	actor := loc27Actor(ctx)
	switch strings.ToLower(area) {
	case "s95_windoww":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xb0, 0x153, 3), loc27Rod(ctx, "095_ROD_01"))
	case "s95_windown":
		if getOriginalFlag(ctx.session.state.OriginalState, loc27StateFloorDiscover) == 1 {
			return engine.Sequence(ctx.EnableArea("S95_Floor"), ctx.WalkToFacingPerspective(actor, 0x138, 0x125, 5), loc27Rod(ctx, "095_ROD_01"), loc27SetOriginal(ctx, loc27StateFloorDiscover, 2))
		}
		if getOriginalFlag(ctx.session.state.OriginalState, loc27StateSecretReady) == 0 {
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x138, 0x125, 5), loc27Rod(ctx, "095_ROD_02"))
		}
		tasks := []engine.Task{ctx.WalkToFacingPerspective(actor, 0x138, 0x125, 5)}
		if getOriginalFlag(ctx.session.state.OriginalState, loc27StateFirstLabVisit) != 0 {
			tasks = append(tasks, loc27Rod(ctx, "095_ROD_02"))
		}
		tasks = append(tasks,
			ctx.HideActor(actor),
			ctx.ShowLayer("S95_EnterSecretRoom"),
			ctx.PlayLayer("S95_EnterSecretRoom"),
			ctx.HideLayer("S95_EnterSecretRoom"),
			ctx.ChangeScene("96"),
		)
		return engine.Sequence(tasks...)
	case "s95_windowe":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x1c7, 0x159, 7), loc27Rod(ctx, "095_ROD_01"))
	case "s95_chair":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xf2, 0x12a, 3), loc27Rod(ctx, "095_ROD_03"))
	case "s95_floor":
		if getOriginalFlag(ctx.session.state.OriginalState, loc27StateFloorStage) == 1 {
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xed, 299, 7), loc27Rod(ctx, "095_ROD_04"), loc27SetOriginal(ctx, loc27StateFloorStage, 2))
		}
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xed, 299, 7), loc27Rod(ctx, "095_ROD_05"), loc27SetOriginal(ctx, loc27StateSecretReady, 1))
	case "s95_exitto93":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x136, 0x162, 0), ctx.ChangeLocation(26, "93"))
	}
	return nil
}

func loc27Click96(ctx *Context, area string) engine.Task {
	actor := loc27Actor(ctx)
	switch strings.ToLower(area) {
	case "s96_exitto95":
		if getOriginalFlag(ctx.session.state.OriginalState, loc27StateHelmetDropped) == 0 {
			if getOriginalFlag(ctx.session.state.OriginalState, loc27StateIslandUnlocked) == 0 {
				return engine.Sequence(
					ctx.WalkToPerspective(actor, 3, 0x157),
					engine.Wait(1),
					ctx.SetActorOrientation(actor, 4),
					loc27Meg(ctx, "096_MEG_26"),
					ctx.WalkToFacingPerspective(actor, 0xfa, 0x155, 5),
				)
			}
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 3, 0x157, 2), ctx.ChangeScene("95"))
		}
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x2f, 0x14d, 2), ctx.SetActorOrientation(actor, 0), loc27Rod(ctx, "096_ROD_26"))
	case "s96_megophias":
		return loc27MegophiasConversation(ctx)
	case "s96_steckkontakt":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xda, 0x121, 3), loc27Rod(ctx, "096_ROD_25"), loc27Meg(ctx, "096_MEG_25"))
	case "s96_anlagen":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xe2, 0x118, 5), loc27Meg(ctx, "096_MEG_23"))
	case "s96_eye":
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0xeb, 0x165, 5),
			ctx.HideActor(actor),
			ctx.HideLayer("S96_Eye"),
			ctx.ShowLayer("S96_RodWinkt"),
			engine.Parallel(
				loc27Voice(ctx, "096_ROD_24"),
				ctx.PlayLayer("S96_RodWinkt"),
			),
			ctx.HideLayer("S96_RodWinkt"),
			ctx.ShowActor(actor),
			ctx.WalkToFacingPerspective(actor, 0xfa, 0x15e, 5),
			ctx.ShowLayer("S96_Eye"),
			ctx.MakeLayerClickable("S96_Eye"),
			loc27Meg(ctx, "096_MEG_24"),
		)
	case "s96_helm":
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0xa0, 0x133, 5),
			ctx.HideActor(actor),
			ctx.ShowLayer("S96_HelmAuf"),
			ctx.PlayLayer("S96_HelmAuf"),
			ctx.HideLayer("S96_HelmAuf"),
			ctx.ChangeLocation(25, "104"),
		)
	}
	return nil
}

func loc27MegophiasConversation(ctx *Context) engine.Task {
	actor := loc27Actor(ctx)
	stage := getOriginalFlag(ctx.session.state.OriginalState, loc27StateMegStage)
	if getOriginalFlag(ctx.session.state.OriginalState, loc27StateAltConversation) != 0 {
		switch stage {
		case 1:
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xfa, 0x155, 5), loc27Rod(ctx, "096_ROD_20"), loc27Meg(ctx, "096_MEG_20"), loc27SetOriginal(ctx, loc27StateMegStage, 2))
		case 2:
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xfa, 0x155, 5), loc27Rod(ctx, "096_ROD_21"), loc27Meg(ctx, "096_MEG_21"), loc27SetOriginal(ctx, loc27StateMegStage, 3))
		case 3:
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xfa, 0x155, 5), loc27Rod(ctx, "096_ROD_22"), loc27Meg(ctx, "096_MEG_22"), loc27Voice(ctx, "096_ROD_23"), loc27SetOriginal(ctx, loc27StateDragonFollowup, 1))
		}
		return nil
	}

	if stage < 1 || stage > 0x10 {
		stage = 1
	}
	if stage >= 1 && stage <= 8 {
		n := stage + 1
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xfa, 0x155, 5), loc27Rod(ctx, "096_ROD_"+twoDigit(n)), loc27Meg(ctx, "096_MEG_"+twoDigit(n)), loc27SetOriginal(ctx, loc27StateMegStage, stage+1))
	}
	if stage == 9 {
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xfa, 0x155, 5), loc27Rod(ctx, "096_ROD_10"), loc27Meg(ctx, "096_MEG_10"), loc27Rod(ctx, "096_ROD_11"), loc27Meg(ctx, "096_MEG_11"), loc27SetOriginal(ctx, loc27StateMegStage, 10))
	}
	if stage >= 10 && stage <= 14 {
		n := stage + 2
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xfa, 0x155, 5), loc27Rod(ctx, "096_ROD_"+twoDigit(n)), loc27Meg(ctx, "096_MEG_"+twoDigit(n)), loc27SetOriginal(ctx, loc27StateMegStage, stage+1))
	}
	if stage == 15 {
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0xfa, 0x155, 5),
			loc27Rod(ctx, "096_ROD_17"),
			loc27Meg(ctx, "096_MEG_17"),
			ctx.HideActor(actor),
			ctx.HideLayer("S96_Eye"),
			ctx.ShowLayer("S96_RodAndEye"),
			ctx.PlayLayer("S96_RodAndEye"),
			ctx.HideLayer("S96_RodAndEye"),
			ctx.ShowActor(actor),
			ctx.PlaceActorPerspective(actor, 0xee, 0x15e),
			ctx.ShowLayer("S96_Eye"),
			ctx.MakeLayerClickable("S96_Eye"),
			loc27SetOriginal(ctx, loc27StateEyeUnlocked, 1),
			ctx.EnableArea("S96_Steckkontakt"),
			ctx.SetActorOrientation(actor, 3),
			loc27SetOriginal(ctx, loc27StateMegStage, 0x10),
		)
	}
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0xfa, 0x155, 5),
		loc27Rod(ctx, "096_ROD_18"),
		loc27Meg(ctx, "096_MEG_18"),
		loc27Rod(ctx, "096_ROD_19"),
		loc27Meg(ctx, "096_MEG_19"),
		loc27SetOriginal(ctx, loc27StateIslandUnlocked, 1),
	)
}

func twoDigit(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

func (LOC27Controller) UseItem(*Context, int, string) engine.Task { return nil }

func (LOC27Controller) SelectItem(ctx *Context, item int) engine.Task {
	if loc27SceneID(ctx.session.state.Scene) != 96 || item != 5 || getOriginalFlag(ctx.session.state.OriginalState, loc27StateHelmetDropped) != 0 {
		return nil
	}
	actor := loc27Actor(ctx)
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0x97, 0x137, 6),
		ctx.RemoveItem(5),
		ctx.HideActor(actor),
		ctx.ShowLayer("S96_RodDropsHelm"),
		ctx.PlayLayer("S96_RodDropsHelm"),
		ctx.HideLayer("S96_RodDropsHelm"),
		ctx.ShowActor(actor),
		ctx.PlaceActorPerspective(actor, 0xa0, 0x133),
		ctx.ShowLayer("S96_Helm"),
		ctx.MakeLayerClickable("S96_Helm"),
		ctx.DisableArea("S96_Megophias"),
		loc27SetOriginal(ctx, loc27StateHelmetDropped, 1),
		loc27Meg(ctx, "096_MEG_28"),
		loc27Rod(ctx, "096_ROD_28"),
		loc27Meg(ctx, "096_MEG_29"),
	)
}

func (LOC27Controller) LoadConditionMask(ctx *Context, scene string) int {
	if loc27SceneID(scene) == 96 && (ctx.HasItem(5) || getOriginalFlag(ctx.session.state.OriginalState, loc27StateHelmetDropped) != 0) {
		return 2
	}
	return 0
}

type loc27Ambient struct {
	ctx   *Context
	wait  float64
	inner engine.Task
}

func (t *loc27Ambient) Update(dt float64) bool {
	scene := loc27SceneID(t.ctx.session.state.Scene)
	if t.ctx.session.state.Location != 27 || (scene != 95 && scene != 96) {
		return true
	}
	if t.inner != nil {
		if !t.inner.Update(dt) {
			return false
		}
		t.inner = nil
		t.wait = float64(5 + rand.IntN(6))
		return false
	}
	t.wait -= dt
	if t.wait > 0 {
		return false
	}
	if scene == 95 {
		t.inner = t.ctx.PlaySFXVolume("Sfx_Seagull.wav", 25+rand.IntN(26))
		return false
	}
	ids := []string{"S96_Bubble", "S96_Pump", "S96_Steam"}
	id := ids[rand.IntN(len(ids))]
	if id == "S96_Pump" {
		// The original plays the pump with animation mode 0x0B. Unlike the
		// transient Bubble/Steam effects, it remains visible as part of the
		// machine after the animation finishes.
		t.inner = engine.Sequence(t.ctx.ShowLayer(id), t.ctx.PlayLayer(id))
	} else {
		t.inner = engine.Sequence(t.ctx.ShowLayer(id), t.ctx.PlayLayer(id), t.ctx.HideLayer(id))
	}
	return false
}

type loc27MegSpeechTask struct {
	ctx     *Context
	line    string
	layer   *engine.Layer
	track   *acs.Track
	voice   engine.Task
	elapsed float64
	started bool
}

func loc27Meg(ctx *Context, line string) engine.Task {
	return &loc27MegSpeechTask{ctx: ctx, line: line}
}

func (t *loc27MegSpeechTask) Update(dt float64) bool {
	if !t.started {
		t.started = true
		layer, ok := t.ctx.layer("S96_Microphone")
		if !ok {
			layer, _, _ = t.ctx.ensureAssetLayer("S96_Microphone")
		}
		t.layer = layer
		if path, err := t.ctx.session.idx.ResolveBaseName(t.line + ".ACS"); err == nil {
			if data, err := os.ReadFile(path); err == nil {
				t.track, _ = acs.Parse(data)
			}
		}
		t.voice = t.ctx.PlayVoiceover(t.line, "["+t.line+"]")
		if t.layer != nil {
			t.layer.Visible = true
			t.layer.Enabled = true
			t.layer.TaskDriven = true
			t.layer.Frame = 0
			t.layer.Accumulator = 0
		}
	}

	t.elapsed += dt
	if t.layer != nil && t.track != nil {
		mouth := t.track.MouthStateAt(t.elapsed)
		frame := 0
		if mouth != 0xffff {
			frame = int(mouth)
			if frame > 0 {
				frame--
			}
		}
		if t.layer.Source == nil || (frame >= 0 && frame < t.layer.Source.Frames()) {
			t.layer.Frame = frame
		}
	}
	if t.voice == nil || t.voice.Update(dt) {
		if t.layer != nil {
			t.layer.Frame = 0
			t.layer.Accumulator = 0
			t.layer.TaskDriven = false
		}
		return true
	}
	return false
}
