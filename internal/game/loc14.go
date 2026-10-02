package game

import (
	"strings"

	"github.com/wok/rent-a-hero/internal/audio"
	"github.com/wok/rent-a-hero/internal/engine"
)

const (
	loc14StateFireEnabled       = 0x3f7c
	loc14StateFireConversation  = 0x3f80
	loc14StateRomanceReady      = 0x3f84
	loc14StateConversationStage = 0x3f88
	loc14StateCynthiaStage      = 0x3f8c
)

type LOC14Controller struct{}

func (LOC14Controller) LoadConditionMask(ctx *Context, scene string) int {
	if loc14SceneID(scene) != "S90" {
		return 0
	}
	if loc09SceneID(ctx.session.state.Scene) == "S89" {
		return 1
	}
	return 0
}

func (LOC14Controller) Enter(ctx *Context, scene, from string) engine.Task {
	if loc14SceneID(scene) != "S90" {
		return nil
	}
	actor := loc14ActorID(ctx)
	tasks := []engine.Task{
		loc14PrepareScene(ctx, from),
		loc14BindBettFrameEvents(ctx),
		loc14StartMusic(ctx, "Loc14_Cynthia.wav", 80, 2000),
		ctx.RunAmbient(newLoc14FireTask(ctx)),
		ctx.MakeLayerClickable("S90_CynTalk"),
	}
	if getOriginalFlag(ctx.session.state.OriginalState, loc14StateFireEnabled) == 0 {
		tasks = append(tasks, ctx.DisableArea("S90_KaminArea"))
	} else {
		tasks = append(tasks, ctx.EnableArea("S90_KaminArea"))
	}
	if loc14FromS89(from) {
		tasks = append(tasks, loc14WakeupSequence(ctx, actor))
	} else {
		tasks = append(tasks,
			ctx.ShowLayer("S90_CynTalk"),
		)
	}
	return engine.Sequence(tasks...)
}

func (LOC14Controller) Exit(*Context, string, string) engine.Task { return nil }

func (LOC14Controller) Click(ctx *Context, area string) engine.Task {
	if loc14SceneID(ctx.session.state.Scene) != "S90" {
		return nil
	}
	actor := loc14ActorID(ctx)
	switch strings.ToLower(area) {
	case "s90_krankenbett":
		return engine.Sequence(ctx.WalkToFacing(actor, 0x1d8, 0x142, 5), loc14Rod(ctx, "090_ROD_11"))
	case "s90_doppelbett":
		return loc14Doppelbett(ctx, actor)
	case "s90_blumen":
		return engine.Sequence(ctx.WalkToFacing(actor, 0x180, 0x10f, 5), loc14Rod(ctx, "090_ROD_17"), ctx.SetActorDirection(actor, 7), loc14Cyn(ctx, "090_CYN_17"), loc14SetOriginal(ctx, loc14StateFireConversation, 1))
	case "s90_exit":
		return engine.Sequence(ctx.WalkToFacing(actor, 0x15b, 299, 5), loc14Rod(ctx, "090_ROD_21"), loc14Cyn(ctx, "090_CYN_21"), ctx.WalkToFacing(actor, 0, 0x15e, 2), ctx.ChangeLocation(12, "91"))
	case "s90_trog":
		return engine.Sequence(ctx.WalkToFacing(actor, 0x1fc, 0x155, 7), loc14Rod(ctx, "090_ROD_22"))
	case "s90_regale":
		return engine.Sequence(ctx.WalkToFacing(actor, 0x96, 0x143, 3), loc14Rod(ctx, "090_ROD_18"))
	case "s90_kaminarea":
		return loc14Kamin(ctx, actor)
	case "s90_cyntalk":
		return loc14CynthiaConversation(ctx, actor)
	}
	return nil
}

func (LOC14Controller) UseItem(*Context, int, string) engine.Task { return nil }
func (LOC14Controller) SelectItem(*Context, int) engine.Task      { return nil }

func loc14SceneID(scene string) string {
	switch strings.ToUpper(strings.TrimSpace(scene)) {
	case "90", "090", "S90", "S090":
		return "S90"
	default:
		return scene
	}
}

func loc14FromS89(from string) bool {
	return loc09SceneID(from) == "S89"
}

func loc14ActorID(ctx *Context) string {
	if actor := ctx.session.PlayerActor(); actor != nil {
		return actor.ID
	}
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

func loc14PrepareScene(ctx *Context, from string) engine.Task {
	return engine.Immediate(func() {
		if loc14FromS89(from) {
			if layer, _, err := ctx.ensureAssetLayer("S90_RodWachtAuf"); err == nil {
				loc14DormantActionLayer(layer)
			}
			for _, id := range []string{"S90_Rodaufsteh", "S90_Cynsitz", "S90_Rodaufsetz", "S90_RodTalk"} {
				if layer, ok := ctx.layer(id); ok {
					loc14DormantActionLayer(layer)
				}
			}
		}

		if layer, ok := ctx.layer("S90_CynTalk"); ok {
			loc14DormantActionLayer(layer)
		}

		if layer, ok := ctx.layer("S90_Kamin"); ok {
			layer.Visible = true
			layer.Enabled = true
			layer.Mode = engine.AnimLoop
			layer.Frame = 0
			layer.Accumulator = 0
			layer.Playing = true
			layer.TaskDriven = false
		}

		if getOriginalFlag(ctx.session.state.OriginalState, loc14StateCynthiaStage) < 3 {
			if layer, _, err := ctx.ensureAssetLayer("S90_Bett"); err == nil {
				loc14DormantActionLayer(layer)
			}
		}
	})
}

func loc14DormantActionLayer(layer *engine.Layer) {
	if layer == nil {
		return
	}
	layer.Visible = false
	layer.Enabled = true
	layer.Playing = false
	layer.TaskDriven = false
	layer.Mode = engine.AnimOnce
	layer.Frame = 0
	layer.Accumulator = 0
}

func loc14BindBettFrameEvents(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		layer, ok := ctx.layer("S90_Bett")
		if !ok || layer == nil {
			return
		}
		if len(layer.FrameEvents[0x1a]) == 0 {
			layer.AddFrameEvent(0x1a, func() { _ = ctx.PlaySFX("Sfx_ServoMotor_ShortClang.wav").Update(0) })
		}
		if len(layer.FrameEvents[0x2b]) == 0 {
			layer.AddFrameEvent(0x2b, func() { _ = ctx.PlaySFX("Sfx_Woosh.wav").Update(0) })
		}
		if len(layer.FrameEvents[0x38]) == 0 {
			layer.AddFrameEvent(0x38, func() { _ = ctx.PlaySFX("Sfx_Work_Hack.wav").Update(0) })
		}
	})
}

func loc14StartLayerAsyncOnce(ctx *Context, id string) engine.Task {
	return engine.Immediate(func() {
		layer, ok := ctx.layer(id)
		if !ok || layer == nil {
			return
		}
		layer.Visible = true
		layer.Enabled = true
		layer.Mode = engine.AnimOnce
		layer.Frame = 0
		layer.Accumulator = 0
		layer.Playing = true
		layer.TaskDriven = false
	})
}

func loc14PlayPresentation(ctx *Context, id string) engine.Task {
	return engine.Sequence(
		ctx.ShowLayer(id),
		ctx.PlayLayer(id),
		ctx.HideLayer(id),
	)
}

func loc14WakeupSequence(ctx *Context, actor string) engine.Task {
	return engine.Sequence(
		ctx.ShowLayer("S90_Cynsitz"),
		ctx.ShowLayer("S90_Rodaufsetz"),
		loc14PlayPresentation(ctx, "S90_RodWachtAuf"),
		ctx.PlayLayer("S90_Rodaufsetz"),
		ctx.ShowLayer("S90_RodTalk"),
		loc14LayerSpeech(ctx, "S90_RodTalk", "090_ROD_01"),
		ctx.Wait(0.5),
		ctx.HideLayer("S90_RodTalk"),
		ctx.HideLayer("S90_Rodaufsetz"),
		loc14StartLayerAsyncOnce(ctx, "S90_Cynsitz"),
		ctx.ShowLayer("S90_Rodaufsteh"),
		ctx.PlayLayerFrames("S90_Rodaufsteh", 3, -1),
		ctx.HideLayer("S90_Rodaufsteh"),
		ctx.PlaceActor(actor, 0x1cb, 0x127),
		ctx.WalkToFacing(actor, 0x197, 0x145, 5),
		ctx.HideLayer("S90_Cynsitz"),
		ctx.ShowLayer("S90_CynTalk"),
		loc14Cyn(ctx, "090_CYN_01"),
	)
}

func loc14Doppelbett(ctx *Context, actor string) engine.Task {
	state := ctx.session.state.OriginalState
	stage := getOriginalFlag(state, loc14StateCynthiaStage)
	conversation := getOriginalFlag(state, loc14StateConversationStage)
	romanceReady := getOriginalFlag(state, loc14StateRomanceReady)
	prefix := []engine.Task{ctx.WalkToFacing(actor, 0x16a, 0x115, 2)}
	if stage == 1 {
		return engine.Sequence(append(prefix, loc14Rod(ctx, "090_ROD_12"), ctx.SetActorDirection(actor, 6), loc14Cyn(ctx, "090_CYN_12"), loc14SetOriginal(ctx, loc14StateCynthiaStage, 2))...)
	}
	if stage != 2 {
		return engine.Sequence(append(prefix, ctx.SetActorDirection(actor, 0), loc14Rod(ctx, "090_ROD_16"))...)
	}
	if romanceReady != 0 && conversation == 2 {
		return engine.Sequence(append(prefix,
			ctx.SetActorDirection(actor, 6),
			loc14Rod(ctx, "090_ROD_14"),
			loc14Cyn(ctx, "090_CYN_14"),
			ctx.SetActorDirection(actor, 2),
			loc14StartMusic(ctx, "Loc14_LoveIsBreakingTheNight.wav", 100, 1000),
			loc14PlayPresentation(ctx, "S90_Bett"),
			loc14StartMusic(ctx, "Loc14_Cynthia.wav", 80, 2000),
			ctx.SetActorDirection(actor, 6),
			loc14Rod(ctx, "090_ROD_15"),
			loc14Cyn(ctx, "090_CYN_15"),
			loc14SetOriginal(ctx, loc14StateCynthiaStage, 3),
		)...)
	}
	return engine.Sequence(append(prefix, loc14Rod(ctx, "090_ROD_13"), ctx.SetActorDirection(actor, 6), loc14Cyn(ctx, "090_CYN_13"))...)
}

func loc14Kamin(ctx *Context, actor string) engine.Task {
	state := ctx.session.state.OriginalState
	tasks := []engine.Task{ctx.WalkToFacing(actor, 0x16a, 0x127, 4), ctx.SetActorDirection(actor, 5)}
	if getOriginalFlag(state, loc14StateFireConversation) == 0 {
		tasks = append(tasks, loc14Rod(ctx, "090_ROD_19"), loc14Cyn(ctx, "090_CYN_19"))
		return engine.Sequence(tasks...)
	}
	tasks = append(tasks, loc14Rod(ctx, "090_ROD_20"), loc14Cyn(ctx, "090_CYN_20"), loc14SetOriginal(ctx, loc14StateRomanceReady, 1))
	if getOriginalFlag(state, loc14StateConversationStage) > 2 {
		tasks = append(tasks, loc14SetOriginal(ctx, loc14StateConversationStage, 1))
	}
	return engine.Sequence(tasks...)
}

func loc14CynthiaConversation(ctx *Context, actor string) engine.Task {
	state := ctx.session.state.OriginalState
	stage := getOriginalFlag(state, loc14StateConversationStage)
	tasks := []engine.Task{ctx.WalkToFacing(actor, 0x17e, 0x116, 6)}
	if getOriginalFlag(state, loc14StateRomanceReady) != 0 {
		if stage != 1 {
			tasks = append(tasks, loc14Rod(ctx, "090_ROD_10"), loc14Cyn(ctx, "090_CYN_10"))
			return engine.Sequence(tasks...)
		}
		tasks = append(tasks, loc14Rod(ctx, "090_ROD_09"), loc14Cyn(ctx, "090_CYN_09"), loc14SetOriginal(ctx, loc14StateConversationStage, 2))
		return engine.Sequence(tasks...)
	}
	switch stage {
	case 1:
		tasks = append(tasks, loc14SetOriginal(ctx, loc14StateConversationStage, 2), loc14Rod(ctx, "090_ROD_02"), loc14Cyn(ctx, "090_CYN_02"))
	case 2:
		tasks = append(tasks, loc14SetOriginal(ctx, loc14StateConversationStage, 3), loc14Rod(ctx, "090_ROD_03"), loc14Cyn(ctx, "090_CYN_03"))
	case 3:
		tasks = append(tasks, loc14SetOriginal(ctx, loc14StateConversationStage, 4), loc14Rod(ctx, "090_ROD_04"), loc14Cyn(ctx, "090_CYN_04"), ctx.SetActorDirection(actor, 0), ctx.Wait(0.7), ctx.SetActorDirection(actor, 6))
	case 4:
		tasks = append(tasks,
			loc14SetOriginal(ctx, loc14StateConversationStage, 5),
			loc14Rod(ctx, "090_ROD_05"),
			loc14Cyn(ctx, "090_CYN_05"),
			loc14Rod(ctx, "090_ROD_06"),
			ctx.WalkToFacing(actor, 0x17e, 0x163, 0),
			loc14Rod(ctx, "090_ROD_07"),
			ctx.SetActorDirection(actor, 5),
			loc14SetOriginal(ctx, loc14StateFireEnabled, 1),
			ctx.EnableArea("S90_KaminArea"),
		)
	case 5:
		tasks = append(tasks, loc14Rod(ctx, "090_ROD_08"), loc14Cyn(ctx, "090_CYN_08"))
	}
	return engine.Sequence(tasks...)
}

func loc14Rod(ctx *Context, line string) engine.Task {
	actor := loc14ActorID(ctx)
	if actor == "" {
		return ctx.PlayVoiceover(line, "["+line+"]")
	}
	return ctx.Say(actor, line, "["+line+"]")
}

func loc14Cyn(ctx *Context, line string) engine.Task {
	return loc14LayerSpeech(ctx, "S90_CynTalk", line)
}

func loc14LayerSpeech(ctx *Context, layer, line string) engine.Task {
	return ctx.PlaySpeechBoundToLayer(layer, line, "["+line+"]", -1, -1)
}

func loc14SetOriginal(ctx *Context, offset, value int) engine.Task {
	return engine.Immediate(func() { putOriginalFlag(ctx.session.state.OriginalState, offset, value) })
}

type loc14MusicStartTask struct {
	ctx          *Context
	name         string
	target       int
	milliseconds int
	started      bool
}

func loc14StartMusic(ctx *Context, name string, target, milliseconds int) engine.Task {
	return &loc14MusicStartTask{ctx: ctx, name: name, target: target, milliseconds: milliseconds}
}

func (t *loc14MusicStartTask) Update(float64) bool {
	if t.started {
		return true
	}
	t.started = true
	sound, err := engine.LoadSound(t.ctx.session.idx, t.name)
	if err != nil {
		return true
	}
	if t.ctx.session.musicHandle != nil {
		t.ctx.session.musicHandle.Stop()
	}
	envelope := loc09VolumeEnvelope{currentVolume: 0, targetVolume: 0}
	envelope.handle = t.ctx.session.audioEngine.PlayLoopingWithVolume(sound, audio.CategoryMusic, loc09DirectSoundGain(0))
	t.ctx.session.musicHandle = envelope.handle
	t.ctx.session.musicName = t.name
	t.ctx.session.musicLocation = t.ctx.session.state.Location
	envelope.setFade(t.target, t.milliseconds)
	t.ctx.session.ambientTasks = append(t.ctx.session.ambientTasks, &loc14HandleFadeTask{envelope: envelope})
	return true
}

type loc14HandleFadeTask struct {
	envelope loc09VolumeEnvelope
}

func (t *loc14HandleFadeTask) Update(dt float64) bool {
	t.envelope.update(dt)
	return t.envelope.step == 0
}

type loc14FireTask struct {
	ctx      *Context
	started  bool
	envelope loc09VolumeEnvelope
}

func newLoc14FireTask(ctx *Context) *loc14FireTask {
	return &loc14FireTask{ctx: ctx}
}

func (t *loc14FireTask) Update(dt float64) bool {
	if !t.started {
		t.started = true
		sound, err := engine.LoadSound(t.ctx.session.idx, "Sfx_Fire.wav")
		if err != nil {
			return false
		}
		t.envelope.currentVolume = 0
		t.envelope.targetVolume = 0
		t.envelope.handle = t.ctx.session.audioEngine.PlayLoopingWithVolume(sound, audio.CategorySFX, loc09DirectSoundGain(0))
		t.envelope.setFade(25, 2000)
	}
	t.envelope.update(dt)
	return false
}
