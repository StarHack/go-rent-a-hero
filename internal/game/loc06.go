package game

import (
	"strconv"
	"strings"

	"github.com/wok/rent-a-hero/internal/engine"
)

const (
	loc06StateRanamaSeen = 0x3e8c
	loc06StateVisitStage = 0x3ee8
	loc06StateRingStage  = 0x3eec
	loc06StateTalkStage  = 0x3ef0
	loc06Scene26         = "026"
	loc06Scene27         = "027"
	loc06ItemEar         = 0x0e
)

type LOC06Controller struct{}

func loc06Original(ctx *Context, off int) int {
	return getOriginalFlag(ctx.session.state.OriginalState, off)
}

func loc06SetOriginal(ctx *Context, off, value int) engine.Task {
	return engine.Immediate(func() {
		putOriginalFlag(ctx.session.state.OriginalState, off, value)
	})
}

func loc06SceneID(scene string) string {
	s := strings.ToUpper(strings.TrimSpace(scene))
	s = strings.TrimPrefix(s, "S")
	if n, err := strconv.Atoi(s); err == nil {
		return strconv.Itoa(n)
	}
	return s
}

func loc06NumericScene(scene string) int {
	s := loc06SceneID(scene)
	n, _ := strconv.Atoi(s)
	return n
}

func loc06ActorID(ctx *Context) string {
	if actor := ctx.session.PlayerActor(); actor != nil {
		return actor.ID
	}
	return "RodrigoSmall"
}

func loc06LayerSpeech(ctx *Context, id, line string) engine.Task {
	layer, ok := ctx.layer(id)
	if !ok || layer.Source == nil {
		return ctx.PlayVoiceover(line, "["+line+"]")
	}
	return ctx.PlaySpeechBoundToLayer(id, line, "["+line+"]", -1, -1)
}

func loc06RodSpeech(ctx *Context, line string) engine.Task {
	if loc06SceneID(ctx.session.state.Scene) == "27" {
		return loc06LayerSpeech(ctx, "S27_RodTalk", line)
	}
	return ctx.Say(loc06ActorID(ctx), line, "["+line+"]")
}

func loc06RanSpeech(ctx *Context, line string) engine.Task {
	if loc06SceneID(ctx.session.state.Scene) == "27" {
		return loc06LayerSpeech(ctx, "S27_RanTalk", line)
	}
	if loc06Original(ctx, loc06StateVisitStage) == 1 {
		return loc06LayerSpeech(ctx, "S26_RanTalk", line)
	}
	return loc06LayerSpeech(ctx, "S26_RanBesen", line)
}

func (LOC06Controller) LoadConditionMask(ctx *Context, scene string) int {
	scene = loc06SceneID(scene)
	prev := loc06NumericScene(ctx.session.state.PreviousScene)
	switch prev {
	case 1003:
		putOriginalFlag(ctx.session.state.OriginalState, loc06StateVisitStage, 2)
	case 1001:
		putOriginalFlag(ctx.session.state.OriginalState, loc06StateVisitStage, 0)
	case 1002:
		putOriginalFlag(ctx.session.state.OriginalState, loc06StateVisitStage, 1)
	case 2001:
		putOriginalFlag(ctx.session.state.OriginalState, loc06StateVisitStage, 2)
		putOriginalFlag(ctx.session.state.OriginalState, loc06StateRanamaSeen, 1)
	case 2003:
		putOriginalFlag(ctx.session.state.OriginalState, loc06StateVisitStage, 2)
		putOriginalFlag(ctx.session.state.OriginalState, loc06StateRingStage, 3)
		putOriginalFlag(ctx.session.state.OriginalState, loc06StateRanamaSeen, 1)
	}
	if scene == "27" {
		putOriginalFlag(ctx.session.state.OriginalState, loc06StateVisitStage, 2)
		if loc06Original(ctx, loc06StateRanamaSeen) != 0 {
			return 1
		}
		return 0
	}
	stage := loc06Original(ctx, loc06StateVisitStage) + 1
	putOriginalFlag(ctx.session.state.OriginalState, loc06StateVisitStage, stage)
	if stage == 1 {
		return 1
	}
	return 2
}

func (LOC06Controller) Enter(ctx *Context, scene string, from string) engine.Task {
	scene = loc06SceneID(scene)
	if scene == "27" {
		if loc06Original(ctx, loc06StateRanamaSeen) != 0 {
			_, _, _ = ctx.ensureAssetLayer("S27_OhrGreif")
			if layer, ok := ctx.layer("S27_OhrGreif"); ok {
				layer.Presentation = true
				layer.ColorKeyed = false
				layer.Visible = false
				layer.Playing = false
				layer.TaskDriven = false
				layer.Frame = 0
				layer.Accumulator = 0
			}
		}
		return loc06Enter27(ctx)
	}
	return loc06Enter26(ctx, loc06SceneID(from))
}

func (LOC06Controller) Exit(ctx *Context, scene string, to string) engine.Task {
	return nil
}

func loc06Enter26(ctx *Context, from string) engine.Task {
	actor := loc06ActorID(ctx)
	stage := loc06Original(ctx, loc06StateVisitStage)
	tasks := []engine.Task{ctx.PlayMusic("Loc06_Ranama.wav"), engine.Immediate(func() { ctx.session.state.Flags["loc06_s26_bottle_count"] = 0 }), ctx.EnableArea("S26_Exit"), ctx.EnableArea("S26_Bottle"), ctx.EnableArea("S26_Telescope"), ctx.EnableArea("S26_Background")}
	if stage == 1 {
		tasks = append(tasks, ctx.HideLayer("S26_RanFaellt"), ctx.ShowLayer("S26_RanTalk"), ctx.MakeLayerClickable("S26_RanTalk"), ctx.DisableArea("S26_Ranama"))
	} else {
		tasks = append(tasks, ctx.ShowLayer("S26_RanBesen"), ctx.FreezeLayer("S26_RanBesen", 0x33), ctx.EnableArea("S26_Ranama"))
	}
	if from == "27" {
		tasks = append(tasks, ctx.PlaceActor(actor, 0x13f, 0x160), ctx.ShowActor(actor))
		if ctx.HasItem(loc06ItemEar) {
			tasks = append(tasks, loc06Leave26AfterEar(ctx))
		}
		return engine.Sequence(tasks...)
	}
	tasks = append(tasks, ctx.PlaceActor(actor, 0xc, 0x140), ctx.ShowActor(actor))
	if stage == 1 {
		tasks = append(tasks,
			ctx.RunAmbient(&loc06BounceTask{ctx: ctx, layerID: "S26_RanTalk", target: 111, steps: 13}),
			ctx.WalkToFacingPerspective(actor, 0x95, 0x156, 6),
			loc06RodSpeech(ctx, "026_ROD_01"),
			loc06RanSpeech(ctx, "026_RAN_01"),
			ctx.WalkToFacingPerspective(actor, 0x13f, 0x160, 6),
			loc06RodSpeech(ctx, "026_ROD_02"),
			loc06RanSpeech(ctx, "026_RAN_02"),
			loc06RodSpeech(ctx, "026_ROD_03"),
		)
		return engine.Sequence(tasks...)
	}
	if stage == 2 {
		tasks = append(tasks,
			ctx.RunAmbient(ctx.WalkToFacingPerspective(actor, 0x13f, 0x160, 6)),
			ctx.PlayLayerFrames("S26_RanBesen", 0, 6),
			ctx.PlaySFX("Sfx_Boing2.wav"),
			ctx.PlayLayerFrames("S26_RanBesen", 7, 0x11),
			ctx.PlaySFX("Sfx_Boing2.wav"),
			ctx.PlayLayerFrames("S26_RanBesen", 0x12, 0x2c),
			ctx.PlaySFX("Sfx_Tock.wav"),
			ctx.PlayLayerFrames("S26_RanBesen", 0x2d, 0x31),
			ctx.PlaySFX("Sfx_Tock.wav"),
			ctx.PlayLayerFrames("S26_RanBesen", 0x32, 0x34),
			ctx.PlaySpeechBoundToLayer("S26_RanBesen", "026_RAN_03", "[026_RAN_03]", 0x2e, 0x34),
			loc06RodSpeech(ctx, "026_ROD_04"),
		)
		return engine.Sequence(tasks...)
	}
	if from != "27" {
		tasks = append(tasks, ctx.WalkToFacingPerspective(actor, 0x13f, 0x160, 6))
	}
	return engine.Sequence(tasks...)
}

func loc06Enter27(ctx *Context) engine.Task {
	tasks := []engine.Task{
		ctx.PlayMusic("Loc06_Ranama.wav"),
		engine.Immediate(func() { ctx.session.state.Flags["loc06_s27_transitioning"] = 0 }),
		ctx.EnableArea("S27_Exit"),
		ctx.ShowLayer("S27_RanTalkRing"),
		ctx.MakeLayerClickable("S27_RanTalkRing"),
		ctx.ShowLayer("S27_RodTalk"),
		ctx.ShowLayer("S27_RanTalk"),
		ctx.MakeLayerClickable("S27_RanTalk"),
	}
	if loc06Original(ctx, loc06StateRanamaSeen) != 0 {
		tasks = append(tasks,
			ctx.HideLayer("S27_RanSchautRing"),
			ctx.MakeLayerClickable("S27_RanSchautRing"),
			ctx.DisableArea("S27_RanSchautRing"),
			ctx.HideLayer("S27_RanSchaut"),
			ctx.MakeLayerClickable("S27_RanSchaut"),
			ctx.DisableArea("S27_RanSchaut"),
		)
	}
	return engine.Sequence(tasks...)
}

func (LOC06Controller) Click(ctx *Context, area string) engine.Task {
	scene := loc06SceneID(ctx.session.state.Scene)
	if scene == "27" {
		return loc06Click27(ctx, area)
	}
	return loc06Click26(ctx, area)
}

func loc06Click26(ctx *Context, area string) engine.Task {
	actor := loc06ActorID(ctx)
	switch area {
	case "S26_Exit":
		return loc06Exit26(ctx)
	case "S26_Bottle":
		line := "026_ROD_07"
		key := "loc06_s26_bottle_count"
		if ctx.session.state.Flags[key] == 0 {
			line = "026_ROD_06"
		}
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0x62, 0x163, 1),
			ctx.Say(actor, line, "["+line+"]"),
			engine.Immediate(func() { ctx.session.state.Flags[key]++ }),
		)
	case "S26_Telescope":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x13f, 0x160, 5), ctx.Say(actor, "026_ROD_08", "[026_ROD_08]"))
	case "S26_Background":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x103, 0x158, 4), ctx.Say(actor, "026_ROD_09", "[026_ROD_09]"))
	case "S26_Ranama", "S26_RanBesen", "S26_RanTalk":
		if loc06Original(ctx, loc06StateVisitStage) == 1 {
			return ctx.Say(actor, "026_ROD_05", "[026_ROD_05]")
		}
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x13f, 0x160, 6), ctx.ChangeLocation(6, loc06Scene27))
	}
	return nil
}

func loc06Exit26(ctx *Context) engine.Task {
	actor := loc06ActorID(ctx)
	if loc06Original(ctx, loc06StateVisitStage) != 1 {
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xc, 0x140, 2), ctx.ChangeLocation(1, "7"))
	}
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0xc, 0x140, 2),
		ctx.HideLayer("S26_RanTalk"),
		ctx.ShowLayer("S26_RanFaellt"),
		ctx.PlayLayerFrames("S26_RanFaellt", 0, 0x17),
		ctx.PlaySFX("Sfx_Knock_Wet.wav"),
		ctx.PlaySFX("Sfx_Gong.wav"),
		ctx.PlayLayerFrames("S26_RanFaellt", 0x18, -1),
		ctx.Wait(1),
		ctx.ChangeLocation(1, "7"),
	)
}

func loc06Leave26AfterEar(ctx *Context) engine.Task {
	actor := loc06ActorID(ctx)
	return engine.Sequence(
		ctx.PlaySpeechBoundToLayer("S26_RanBesen", "027_RAN_09", "[027_RAN_09]", 0x2e, 0x34),
		ctx.SetLayerZ("S26_RanBesen", 20),
		ctx.WalkToFacingPerspective(actor, 0xc, 0x140, 2),
		ctx.SetLayerZ("S26_RanBesen", 10),
		ctx.HideActor(actor),
		ctx.PlaySpeechBoundToLayer("S26_RanBesen", "027_RAN_10", "[027_RAN_10]", 0x2e, 0x34),
		ctx.ChangeLocation(1, "7"),
	)
}

func loc06Click27(ctx *Context, area string) engine.Task {
	switch area {
	case "S27_Exit":
		return ctx.ChangeLocation(6, loc06Scene26)
	case "S27_RanSchautRing":
		return loc06TakeEar(ctx)
	case "S27_RanTalkRing":
		return loc06RingConversation(ctx)
	case "S27_RanTalk", "S27_RanSchaut":
		return loc06RanamaConversation(ctx)
	}
	return nil
}

func loc06TakeEar(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.DisableArea("S27_RanSchautRing"),
		ctx.DisableArea("S27_RanSchaut"),
		ctx.ShowLayer("S27_OhrGreif"),
		ctx.PlayLayerFrames("S27_OhrGreif", 0, 5),
		ctx.PlaySFX("Sfx_Ratsch.wav"),
		ctx.PlayLayerFrames("S27_OhrGreif", 6, -1),
		ctx.AddItem(loc06ItemEar),
		ctx.ChangeLocation(6, loc06Scene26),
	)
}

func loc06RanamaConversation(ctx *Context) engine.Task {
	if ctx.session.state.Flags["loc06_s27_transitioning"] != 0 {
		return nil
	}
	if loc06Original(ctx, loc06StateTalkStage) == 1 {
		return engine.Sequence(
			loc06RodSpeech(ctx, "027_ROD_01"),
			loc06RanSpeech(ctx, "027_RAN_01"),
			loc06RodSpeech(ctx, "027_ROD_02"),
			loc06RanSpeech(ctx, "027_RAN_02"),
			loc06SetOriginal(ctx, loc06StateTalkStage, 2),
		)
	}
	return engine.Sequence(
		loc06RodSpeech(ctx, "027_ROD_03"),
		loc06RanSpeech(ctx, "027_RAN_03"),
		loc06RodSpeech(ctx, "027_ROD_04"),
		loc06RanSpeech(ctx, "027_RAN_04"),
		loc06RodSpeech(ctx, "027_ROD_05"),
		loc06SetOriginal(ctx, loc06StateTalkStage, 2),
	)
}

func loc06RingConversation(ctx *Context) engine.Task {
	if loc06Original(ctx, loc06StateRanamaSeen) == 0 {
		return engine.Sequence(loc06RodSpeech(ctx, "027_ROD_06"), loc06RanSpeech(ctx, "027_RAN_05"))
	}
	stage := loc06Original(ctx, loc06StateRingStage)
	if stage == 1 {
		return engine.Sequence(loc06RodSpeech(ctx, "027_ROD_07"), loc06RanSpeech(ctx, "027_RAN_06"), loc06SetOriginal(ctx, loc06StateRingStage, 2))
	}
	if stage == 2 {
		return engine.Sequence(loc06RodSpeech(ctx, "027_ROD_08"), loc06RanSpeech(ctx, "027_RAN_07"), loc06SetOriginal(ctx, loc06StateRingStage, 3))
	}
	return engine.Sequence(
		engine.Immediate(func() { ctx.session.state.Flags["loc06_s27_transitioning"] = 1 }),
		loc06RodSpeech(ctx, "027_ROD_09"),
		ctx.HideLayer("S27_RanTalk"),
		ctx.DisableArea("S27_RanTalk"),
		ctx.HideLayer("S27_RanTalkRing"),
		ctx.DisableArea("S27_RanTalkRing"),
		ctx.ShowLayer("S27_RanSchaut"),
		ctx.PlayLayerFrames("S27_RanSchaut", 0, 0xb),
		ctx.ShowLayer("S27_RanSchautRing"),
		ctx.MakeLayerClickable("S27_RanSchautRing"),
		ctx.EnableArea("S27_RanSchautRing"),
		ctx.MakeLayerClickable("S27_RanSchaut"),
		ctx.EnableArea("S27_RanSchaut"),
		ctx.DisableArea("S27_Exit"),
		ctx.RunAmbient(engine.Sequence(ctx.Wait(2), loc06FinishRingTransition(ctx))),
	)
}

func loc06FinishRingTransition(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.HideLayer("S27_RanSchautRing"),
		ctx.DisableArea("S27_RanSchautRing"),
		ctx.PlayLayerFrames("S27_RanSchaut", 0xc, 0x16),
		ctx.HideLayer("S27_RanSchaut"),
		ctx.DisableArea("S27_RanSchaut"),
		ctx.ShowLayer("S27_RanTalk"),
		ctx.ShowLayer("S27_RanTalkRing"),
		ctx.EnableArea("S27_Exit"),
		ctx.MakeLayerClickable("S27_RanTalk"),
		ctx.EnableArea("S27_RanTalk"),
		ctx.MakeLayerClickable("S27_RanTalkRing"),
		ctx.EnableArea("S27_RanTalkRing"),
		loc06RanSpeech(ctx, "027_RAN_08"),
		loc06SetOriginal(ctx, loc06StateRingStage, 3),
		engine.Immediate(func() { ctx.session.state.Flags["loc06_s27_transitioning"] = 0 }),
	)
}

func (LOC06Controller) UseItem(ctx *Context, item int, area string) engine.Task {
	return nil
}

func (LOC06Controller) SelectItem(ctx *Context, item int) engine.Task {
	return nil
}

type loc06BounceTask struct {
	ctx         *Context
	layerID     string
	target      float64
	start       float64
	steps       int
	step        int
	started     bool
	accumulator float64
}

func (t *loc06BounceTask) Update(dt float64) bool {
	if loc06SceneID(t.ctx.session.state.Scene) != "26" {
		return true
	}
	layer, ok := t.ctx.layer(t.layerID)
	if !ok || !layer.Visible {
		return true
	}
	if !t.started {
		t.started = true
		t.start = float64(layer.Y)
		if t.steps < 1 {
			t.steps = 1
		}
	}
	t.accumulator += dt
	const tick = 1.0 / 25.0
	for t.accumulator >= tick {
		t.accumulator -= tick
		t.step++
		if t.step > t.steps {
			t.step = t.steps
		}
		p := float64(t.step) / float64(t.steps)
		layer.Y = int(t.start + (t.target-t.start)*p)
		if t.step < t.steps {
			continue
		}
		if t.target == 111 {
			t.target = 124
		} else {
			t.target = 111
		}
		t.start = float64(layer.Y)
		t.step = 0
	}
	return false
}
