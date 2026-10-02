package game

import (
	"log"
	"math/rand/v2"
	"strings"

	"github.com/wok/rent-a-hero/internal/engine"
)

const (
	loc19StateTo70       = 0x3ff0
	loc19StateJasVisible = 0x3ff4
	loc19StateStoryTalk  = 0x3ffc
	loc19StateJasGate    = 0x4004
	loc19StateJasTalk    = 0x4008
	loc19StateRamTalk    = 0x400c
	loc19StateDwarfTalk  = 0x4010
	loc19StateJasActor   = 0x4014
	loc19StateRamMoved   = 0x4018
	loc19StateRamPose    = 0x401c
	loc19StateTo71       = 0x4028
)

type LOC19Controller struct{}

func loc19SceneID(scene string) string {
	s := strings.ToUpper(strings.TrimSpace(scene))
	s = strings.TrimPrefix(s, "S")
	switch s {
	case "68":
		return "S68"
	case "69":
		return "S69"
	case "101":
		return "S101"
	}
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(scene)), "S") {
		return strings.ToUpper(strings.TrimSpace(scene))
	}
	return s
}

func loc19Original(ctx *Context, off int) int {
	return getOriginalFlag(ctx.session.state.OriginalState, off)
}

func loc19SetOriginal(ctx *Context, off, value int) engine.Task {
	return engine.Immediate(func() { putOriginalFlag(ctx.session.state.OriginalState, off, value) })
}

func loc19FrameStart(frame int) int {
	if frame > 0 {
		return frame - 1
	}
	return frame
}

func loc19Range(ctx *Context, id string, from, to int) engine.Task {
	return ctx.PlayLayerFrames(id, loc19FrameStart(from), to)
}

func loc19LayerSpeech(ctx *Context, id, line string, from, to int) engine.Task {
	if _, ok := ctx.layer(id); !ok {
		return ctx.PlayVoiceover(line, "["+line+"]")
	}
	return ctx.PlaySpeechBoundToLayer(id, line, "["+line+"]", loc19FrameStart(from), to)
}

func loc19Dormant69(ctx *Context) engine.Task {
	ids := []string{
		"S69_Tuer", "S69_Wache",
		"S69_JasLauf", "S69_JasWalkKette", "S69_RamBoese", "S69_RamWalk", "S69_RamRun",
		"S69_RodStarrt", "S69_RodZeig", "S69_ThaTalk", "S69_ThaWalk", "S69_Umarm",
		"S69_Zwerg1", "S69_RamBack", "S69_RamTalkCloseup", "S69_JasBack", "S69_JasTalkCloseup",
		"S69_JasTalk", "S69_RamBackTalk", "S69_RamTalkAnim", "S69_WacheStauntBack", "S69_WacheTalk",
	}
	tasks := make([]engine.Task, 0, len(ids)*2)
	for _, id := range ids {
		if _, ok := ctx.layer(id); ok {
			tasks = append(tasks, ctx.HideLayer(id), ctx.FreezeLayer(id, 0))
		}
	}
	return engine.Sequence(tasks...)
}

func loc19ShowStatic(ctx *Context, id string, frame int) engine.Task {
	return engine.Sequence(ctx.FreezeLayer(id, loc19FrameStart(frame)), ctx.ShowLayer(id))
}

func loc19SetLayerState(ctx *Context, id string, x, y, z, zoom, frame int) engine.Task {
	return engine.Immediate(func() {
		if layer, ok := ctx.layer(id); ok {
			layer.X = x
			layer.Y = y
			layer.Z = z
			layer.Zoom = zoom
			wantedFrame := loc19FrameStart(frame)
			if layer.Source == nil || (wantedFrame >= 0 && wantedFrame < layer.Source.Frames()) {
				layer.Frame = wantedFrame
			}
			layer.Visible = true
			layer.Enabled = true
			layer.Playing = false
			layer.TaskDriven = false
		}
	})
}

type loc19LayerMoveTask struct {
	ctx                       *Context
	id                        string
	targetX, targetY, targetZ int
	targetZoom, steps         int
	elapsed                   float64
	startX, startY, startZ    int
	startZoom                 int
	animate                   bool
	rangeLoop                 bool
	rangeFrom, rangeTo        int
	started                   bool
}

func loc19MoveLayer(ctx *Context, id string, x, y, z, zoom, steps int, animate bool) engine.Task {
	return &loc19LayerMoveTask{ctx: ctx, id: id, targetX: x, targetY: y, targetZ: z, targetZoom: zoom, steps: steps, animate: animate}
}

func loc19MoveLayerRange(ctx *Context, id string, x, y, z, zoom, steps, from, to int) engine.Task {
	return &loc19LayerMoveTask{ctx: ctx, id: id, targetX: x, targetY: y, targetZ: z, targetZoom: zoom, steps: steps, animate: true, rangeLoop: true, rangeFrom: loc19FrameStart(from), rangeTo: to}
}

func (t *loc19LayerMoveTask) Update(dt float64) bool {
	layer, ok := t.ctx.layer(t.id)
	if !ok {
		return true
	}
	if !t.started {
		t.started = true
		t.startX, t.startY, t.startZ, t.startZoom = layer.X, layer.Y, layer.Z, layer.Zoom
		if t.animate {
			layer.Visible = true
			layer.Enabled = true
			layer.Mode = engine.AnimLoop
			layer.Accumulator = 0
			layer.Playing = true
			if t.rangeLoop {
				if layer.Source == nil || (t.rangeFrom >= 0 && t.rangeFrom < layer.Source.Frames()) {
					layer.Frame = t.rangeFrom
				} else {
					layer.Playing = false
				}
			}
			layer.TaskDriven = true
		}
	}
	if t.steps <= 0 {
		layer.X, layer.Y, layer.Z, layer.Zoom = t.targetX, t.targetY, t.targetZ, t.targetZoom
		if t.animate {
			layer.Playing = false
			layer.TaskDriven = false
		}
		return true
	}
	if t.animate {
		if t.rangeLoop {
			fps := layer.FPS
			if fps <= 0 {
				fps = 10
			}
			layer.Accumulator += dt
			frameDuration := 1.0 / float64(fps)
			for layer.Accumulator >= frameDuration && layer.Playing {
				layer.Accumulator -= frameDuration
				next := layer.Frame + 1
				if next > t.rangeTo {
					next = t.rangeFrom
				}
				if layer.Source != nil && (next < 0 || next >= layer.Source.Frames()) {
					layer.Playing = false
					break
				}
				layer.Frame = next
			}
		} else {
			layer.AdvanceScripted(dt)
		}
	}
	t.elapsed += dt
	duration := float64(t.steps) / 20.0
	p := t.elapsed / duration
	if p > 1 {
		p = 1
	}
	layer.X = t.startX + int(float64(t.targetX-t.startX)*p)
	layer.Y = t.startY + int(float64(t.targetY-t.startY)*p)
	layer.Z = t.startZ + int(float64(t.targetZ-t.startZ)*p)
	layer.Zoom = t.startZoom + int(float64(t.targetZoom-t.startZoom)*p)
	if t.elapsed < duration {
		return false
	}
	if t.animate {
		layer.Playing = false
		layer.TaskDriven = false
	}
	return true
}

func (LOC19Controller) LoadConditionMask(*Context, string) int    { return 0 }
func (LOC19Controller) Exit(*Context, string, string) engine.Task { return nil }
func (LOC19Controller) UseItem(*Context, int, string) engine.Task { return nil }
func (LOC19Controller) SelectItem(*Context, int) engine.Task      { return nil }

func (LOC19Controller) Enter(ctx *Context, scene, from string) engine.Task {
	switch loc19SceneID(scene) {
	case "S68":
		return loc19Enter68(ctx, loc19SceneID(from))
	case "S69":
		return loc19Enter69(ctx, loc19SceneID(from))
	case "S101":
		return loc19Enter101(ctx, loc19SceneID(from))
	}
	return nil
}

func loc19Enter68(ctx *Context, from string) engine.Task {
	actor := loc10ActorID(ctx)
	jas := loc10JasminActorID(ctx)
	tasks := []engine.Task{
		ctx.PlayMusic("Loc19_Esragoth.wav"),
		ctx.HideActor(jas),
		ctx.ShowActor(actor),
		ctx.EnableArea("S68_To67"),
		ctx.EnableArea("S68_To69"),
	}
	if from == "S38" {
		tasks = append(tasks, ctx.PlaceActorPerspective(actor, 4, 0x14b))
		if loc10Original(ctx, loc10StateJas) != 0 {
			tasks = append(tasks,
				ctx.ShowActor(jas),
				ctx.PlaceActorPerspective(jas, 4, 0x138),
				ctx.RunAmbient(ctx.WalkToFacingPerspective(jas, 0x153, 0x13e, 5)),
			)
		}
		tasks = append(tasks, ctx.WalkToFacingPerspective(actor, 0x10a, 0x145, 6))
	} else if from == "S69" {
		tasks = append(tasks,
			ctx.PlaceActorPerspective(actor, 0x22a, 0x111),
			ctx.WalkToFacingPerspective(actor, 0x10a, 0x145, 2),
		)
	}
	tasks = append(tasks, ctx.RunAmbient(&loc19Ambient{ctx: ctx}))
	return engine.Sequence(tasks...)
}

func loc19Enter101(ctx *Context, from string) engine.Task {
	actor := loc10ActorID(ctx)
	jas := loc10JasminActorID(ctx)
	tasks := []engine.Task{
		ctx.SetBackgroundByStem("S68_Back_nd"),
		ctx.PlayMusic("Loc10_ForestInDarkness.wav"),
		ctx.HideActor(jas),
		ctx.ShowActor(actor),
		ctx.EnableArea("S68_To67"),
		ctx.DisableArea("S68_To69"),
		ctx.DisableArea("S101_To69"),
		engine.Immediate(func() {
			if bg := ctx.session.scene.Background; bg != nil {
				if nav := bg.NavGrid(); nav != nil {
					nav.SetDynamicRect(0x168, 200, 0x276, 0x159, true)
				}
			}
		}),
	}
	if from == "S38" {
		tasks = append(tasks,
			ctx.PlaceActorPerspective(actor, 7, 0x124),
			ctx.WalkToFacingPerspective(actor, 0x101, 0x138, 6),
			ctx.Say(actor, "101_ROD_01", "[101_ROD_01]"),
		)
	}
	tasks = append(tasks, ctx.RunAmbient(&loc19Ambient{ctx: ctx}))
	return engine.Sequence(tasks...)
}

func loc19Enter69(ctx *Context, from string) engine.Task {
	actor := loc10ActorID(ctx)
	jas := loc10JasminActorID(ctx)
	tasks := []engine.Task{
		ctx.PlayMusic("Loc19_Esragoth.wav"),
		ctx.ShowActor(actor),
		ctx.HideActor(jas),
		loc19Dormant69(ctx),
		loc19ShowStatic(ctx, "S69_Tuer", 0),
		loc19ShowStatic(ctx, "S69_Wache", 0),
		ctx.EnableArea("S69_To68"),
		ctx.DisableArea("S69_To70"),
		ctx.DisableArea("S69_To71"),
	}
	switch loc19Original(ctx, loc19StateRamPose) {
	case 1:
		tasks = append(tasks,
			loc19SetLayerState(ctx, "S69_JasWalkKette", 179, 163, 43, 106, 12),
			loc19SetLayerState(ctx, "S69_RamTalkAnim", 225, 205, 65, 90, 0),
			ctx.MakeLayerClickable("S69_JasWalkKette"),
		)
	case 2:
		if loc19Original(ctx, loc19StateRamTalk) < 3 {
			tasks = append(tasks, loc19SetLayerState(ctx, "S69_JasTalk", 410, 188, 47, 82, 0))
		}
		tasks = append(tasks,
			loc19SetLayerState(ctx, "S69_RamBackTalk", 425, 214, 42, 82, 0),
			ctx.MakeLayerClickable("S69_RamBackTalk"),
		)
	}
	if loc19Original(ctx, loc19StateJasVisible) != 0 {
		tasks = append(tasks,
			loc19SetLayerState(ctx, "S69_Zwerg1", -13, 210, 67, 100, 12),
			ctx.MakeLayerClickable("S69_Zwerg1"),
		)
	}
	if loc19Original(ctx, loc19StateRamTalk) != 1 {
		tasks = append(tasks, ctx.MakeLayerClickable("S69_Wache"))
	}
	if loc19Original(ctx, loc19StateTo70) != 0 {
		tasks = append(tasks, ctx.EnableArea("S69_To70"))
	} else {
		tasks = append(tasks, ctx.DisableArea("S69_To70"))
	}
	if loc19Original(ctx, loc19StateTo71) != 0 {
		tasks = append(tasks, ctx.EnableArea("S69_To71"))
	} else {
		tasks = append(tasks, ctx.DisableArea("S69_To71"))
	}
	if from == "S68" {
		tasks = append(tasks, loc19Enter69From68(ctx, actor))
	} else if from == "S70" {
		tasks = append(tasks,
			ctx.PlaceActorPerspective(actor, 0x13c, 0x115),
			ctx.PlaySFX("Sfx_Door_Creaking_Medium.wav"),
			ctx.PlayLayer("S69_Tuer"),
			ctx.WalkToFacingPerspective(actor, 0x14c, 0x138, 0),
			ctx.PlaySFX("Sfx_Door_Creaking_Medium1.wav"),
			loc19Range(ctx, "S69_Tuer", 10, 0),
			ctx.ShowLayer("S69_Wache"),
		)
	} else if from == "S71" {
		tasks = append(tasks,
			ctx.PlaceActorPerspective(actor, 0x27, 0x12a),
			ctx.WalkToFacingPerspective(actor, 0x4e, 0x137, 7),
		)
	}
	tasks = append(tasks, ctx.RunAmbient(&loc19Ambient{ctx: ctx}))
	return engine.Sequence(tasks...)
}

func loc19Enter69From68(ctx *Context, actor string) engine.Task {
	if loc10Original(ctx, loc10StateJas) == 0 {
		return engine.Sequence(
			ctx.PlaceActorPerspective(actor, 0x277, 0xf4),
			ctx.WalkToFacingPerspective(actor, 0x22e, 0x104, 2),
		)
	}
	jas := loc10JasminActorID(ctx)
	return engine.Sequence(
		loc19SetOriginal(ctx, loc10StateJas, 0),
		loc19SetOriginal(ctx, loc19StateJasActor, 1),
		ctx.ShowActor(jas),
		ctx.PlaceActorPerspective(actor, 0x277, 0xf4),
		ctx.PlaceActorPerspective(jas, 0x277, 0xf4),
		ctx.RunAmbient(ctx.WalkToFacingPerspective(jas, 0x218, 0x108, 5)),
		ctx.WalkToFacingPerspective(actor, 0x22e, 0x104, 2),
		ctx.PlayVoiceover("069_ZW2_01", "[069_ZW2_01]"),
		loc19Range(ctx, "S69_Wache", 0, 0x12),
		ctx.PlayVoiceover("069_JAS_01", "[069_JAS_01]"),
		loc19LayerSpeech(ctx, "S69_Wache", "069_ZW2_02", 0x18, 0x1e),
		ctx.PlayVoiceover("069_JAS_02", "[069_JAS_02]"),
		ctx.Say(actor, "069_ROD_02", "[069_ROD_02]"),
		loc19Range(ctx, "S69_Wache", 0x12, 0),
		loc19SetLayerState(ctx, "S69_Zwerg1", -13, 210, 67, 100, 0),
		loc19MoveLayer(ctx, "S69_Zwerg1", 36, 210, 67, 100, 12, true),
		ctx.FreezeLayer("S69_Zwerg1", loc19FrameStart(0xc)),
		loc19LayerSpeech(ctx, "S69_Zwerg1", "069_ZW1_02", 0xc, 0x10),
		ctx.FreezeLayer("S69_Zwerg1", loc19FrameStart(0xc)),
		ctx.MakeLayerClickable("S69_Zwerg1"),
		loc19SetOriginal(ctx, loc19StateJasVisible, 1),
	)
}

func (LOC19Controller) Click(ctx *Context, area string) engine.Task {
	switch loc19SceneID(ctx.session.state.Scene) {
	case "S68":
		return loc19Click68(ctx, area)
	case "S69":
		return loc19Click69(ctx, area)
	case "S101":
		return loc19Click101(ctx, area)
	}
	return nil
}

func loc19Click68(ctx *Context, area string) engine.Task {
	actor := loc10ActorID(ctx)
	switch area {
	case "S68_To67":
		if loc10Original(ctx, loc10StateJas) == 0 {
			return engine.Sequence(
				ctx.WalkToFacingPerspective(actor, 4, 0x14b, 2),
				ctx.ChangeLocation(10, "38"),
			)
		}
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 4, 0x14b, 2),
			ctx.PlayVoiceover("068_JAS_01", "[068_JAS_01]"),
			ctx.WalkToFacingPerspective(actor, 0x10a, 0x145, 6),
		)
	case "S68_To69":
		jas := loc10JasminActorID(ctx)
		tasks := []engine.Task{ctx.WalkToFacingPerspective(actor, 0x1b3, 299, 6)}
		if loc10Original(ctx, loc10StateJas) != 0 {
			tasks = append(tasks, ctx.RunAmbient(ctx.WalkToPerspective(jas, 0x22a, 0x10c)))
		}
		tasks = append(tasks,
			ctx.WalkToFacingPerspective(actor, 0x22a, 0x111, 6),
			ctx.ChangeScene("S69"),
		)
		return engine.Sequence(tasks...)
	}
	return nil
}

func loc19Click101(ctx *Context, area string) engine.Task {
	actor := loc10ActorID(ctx)
	switch area {
	case "S68_To67":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 7, 0x124, 2), ctx.ChangeLocation(10, "38"))
	case "S101_To69":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x101, 0x138, 6), ctx.Say(actor, "101_ROD_02", "[101_ROD_02]"))
	}
	return nil
}

func loc19Click69(ctx *Context, area string) engine.Task {
	actor := loc10ActorID(ctx)
	switch area {
	case "S69_To71":
		if loc10Original(ctx, loc10StateTree) != 0 {
			return engine.Sequence(
				ctx.WalkToFacingPerspective(actor, 0x11, 0x114, 2),
				ctx.ChangeLocation(21, "71"),
			)
		}
		if loc19Original(ctx, loc19StateJasGate) == 1 {
			return loc19NecklaceReturnCutscene(ctx, actor)
		}
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x75, 0x13c, 2), ctx.Say(actor, "069_ROD_24", "[069_ROD_24]"))
	case "S69_To70":
		if loc19Original(ctx, loc19StateTo70) == 0 {
			return nil
		}
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0x16f, 0x137, 3),
			ctx.PlaySFX("Sfx_Door_Creaking_Medium.wav"),
			ctx.PlayLayer("S69_Tuer"),
			ctx.WalkToFacingPerspective(actor, 0x13c, 0x115, 4),
			ctx.ChangeLocation(20, "70"),
		)
	case "S69_To68":
		return loc19Leave69To68(ctx, actor)
	case "S69_RamBackTalk":
		return loc19RamTalk(ctx, actor)
	case "S69_JasWalkKette":
		return loc19JasTalk(ctx, actor)
	case "S69_Zwerg1":
		return loc19DwarfTalk(ctx, actor)
	case "S69_Wache":
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0x1d3, 0x124, 5),
			ctx.Say(actor, "069_ROD_14", "[069_ROD_14]"),
			loc19LayerSpeech(ctx, "S69_Wache", "069_ZW2_14", 0, 2),
		)
	}
	return nil
}

func loc19Leave69To68(ctx *Context, actor string) engine.Task {
	if loc19Original(ctx, loc19StateRamMoved) == 0 {
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0x23f, 0x106, 6),
			ctx.PlayVoiceover("069_JAS_03", "[069_JAS_03]"),
			ctx.WalkToFacingPerspective(actor, 0x212, 0x11d, 2),
		)
	}
	if loc19Original(ctx, loc19StateJasActor) != 0 {
		return nil
	}
	if loc19Original(ctx, loc19StateTo70) != 0 {
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x277, 0xf4, 6), ctx.ChangeScene("S68"))
	}
	return engine.Sequence(
		ctx.Say(actor, "069_ROD_04", "[069_ROD_04]"),
		ctx.PlayVoiceover("069_THA_04", "[069_THA_04]"),
		ctx.WalkToFacingPerspective(actor, 0x16f, 0x137, 3),
		ctx.ShowLayer("S69_ThaWalk"),
		ctx.PlaySFX("Sfx_Door_Creaking_Medium.wav"),
		ctx.PlayLayer("S69_Tuer"),
		loc19LayerSpeech(ctx, "S69_Wache", "069_ZW2_04", 0, 4),
		ctx.Say(actor, "069_ROD_05", "[069_ROD_05]"),
		loc19LayerSpeech(ctx, "S69_ThaTalk", "069_THA_05", 0, 8),
		loc19SetOriginal(ctx, loc19StateTo70, 1),
		loc19SetOriginal(ctx, loc19StateJasVisible, 0),
		loc19SetOriginal(ctx, loc19StateRamPose, -1),
		ctx.EnableArea("S69_To70"),
		ctx.ChangeLocation(20, "70"),
	)
}

func loc19RamTalk(ctx *Context, actor string) engine.Task {
	switch loc19Original(ctx, loc19StateRamTalk) {
	case 1:
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0x15a, 0x13d, 7),
			ctx.Say(actor, "069_ROD_09", "[069_ROD_09]"),
			loc19LayerSpeech(ctx, "S69_RamBackTalk", "069_RAM_09", 10, 0xc),
			ctx.Say(actor, "069_ROD_10", "[069_ROD_10]"),
			loc19Range(ctx, "S69_RamBackTalk", 10, 0),
			loc19SetOriginal(ctx, loc19StateRamTalk, 2),
			ctx.MakeLayerClickable("S69_Wache"),
		)
	case 2:
		jas := loc10JasminActorID(ctx)
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0x15a, 0x13d, 7),
			ctx.Say(actor, "069_ROD_11", "[069_ROD_11]"),
			loc19LayerSpeech(ctx, "S69_RamBackTalk", "069_RAM_11", 10, 0xc),
			loc19LayerSpeech(ctx, "S69_JasTalk", "069_JAS_11", 0, 8),
			ctx.HideLayer("S69_JasTalk"),
			ctx.ShowActor(jas),
			ctx.PlaceActorPerspective(jas, 0x1a8, 0x137),
			ctx.WalkToFacingPerspective(jas, 4, 0x15c, 1),
			ctx.HideActor(jas),
			loc19SetOriginal(ctx, loc19StateRamMoved, 1),
			loc19SetOriginal(ctx, loc19StateJasActor, 0),
			loc19SetOriginal(ctx, loc19StateRamTalk, 3),
		)
	case 3:
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0x172, 0x13d, 7),
			ctx.Say(actor, "069_ROD_12", "[069_ROD_12]"),
			loc19LayerSpeech(ctx, "S69_RamBackTalk", "069_RAM_12", 0, 4),
			loc19SetOriginal(ctx, loc19StateRamTalk, 4),
		)
	case 4:
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0x172, 0x13d, 7),
			ctx.Say(actor, "069_ROD_13", "[069_ROD_13]"),
			loc19LayerSpeech(ctx, "S69_RamBackTalk", "069_RAM_13", 0, 4),
		)
	}
	return nil
}

func loc19CloseupSpeech(ctx *Context, line string, jasmine bool, preWait, postWait float64) engine.Task {
	type layerState struct {
		visible bool
		enabled bool
	}
	layers := make(map[string]layerState)
	actors := make(map[string]bool)
	areas := make(map[string]bool)
	backID := "S69_RamBack"
	talkID := "S69_RamTalkCloseup"
	if jasmine {
		backID = "S69_JasBack"
		talkID = "S69_JasTalkCloseup"
	}
	for _, id := range []string{backID, talkID} {
		if _, ok := ctx.layer(id); ok {
			continue
		}
		if _, _, err := ctx.ensureAssetLayer(id); err != nil {
			log.Printf("game: S69 closeup: load layer %q: %v", id, err)
		}
	}
	tasks := []engine.Task{
		engine.Immediate(func() {
			for id, layer := range ctx.session.scene.Layers {
				layers[id] = layerState{visible: layer.Visible, enabled: layer.Enabled}
				layer.Visible = false
			}
			for id, actor := range ctx.session.scene.Characters {
				actors[id] = actor.Visible
				actor.Visible = false
			}
			for id, area := range ctx.session.scene.Areas {
				areas[id] = area.Enabled
				area.Enabled = false
			}
		}),
		ctx.ShowLayer(backID),
		ctx.ShowLayer(talkID),
	}
	if preWait > 0 {
		tasks = append(tasks, ctx.Wait(preWait))
	}
	tasks = append(tasks, loc19LayerSpeech(ctx, talkID, line, 0, 4))
	if postWait > 0 {
		tasks = append(tasks, ctx.Wait(postWait))
	}
	tasks = append(tasks, engine.Immediate(func() {
		for id, state := range layers {
			if layer, ok := ctx.session.scene.Layers[id]; ok {
				layer.Visible = state.visible
				layer.Enabled = state.enabled
			}
		}
		for id, visible := range actors {
			if actor, ok := ctx.session.scene.Characters[id]; ok {
				actor.Visible = visible
			}
		}
		for id, enabled := range areas {
			if area, ok := ctx.session.scene.Areas[id]; ok {
				area.Enabled = enabled
			}
		}
	}))
	return engine.Sequence(tasks...)
}

func loc19JasCloseupSpeech(ctx *Context, line string) engine.Task {
	return loc19CloseupSpeech(ctx, line, true, 0, 0.5)
}

func loc19JasTalk(ctx *Context, actor string) engine.Task {
	if loc19Original(ctx, loc19StateStoryTalk) == 0 {
		if loc19Original(ctx, loc19StateRamMoved) != 0 {
			return nil
		}
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0x112, 0x14d, 3),
			ctx.Say(actor, "069_ROD_16", "[069_ROD_16]"),
			loc19LayerSpeech(ctx, "S69_JasWalkKette", "069_JAS_16", 0, 8),
		)
	}
	switch loc19Original(ctx, loc19StateJasTalk) {
	case 1:
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0x112, 0x14d, 3),
			ctx.HideActor(actor),
			loc19SetLayerState(ctx, "S69_RodZeig", 230, 171, 1, 103, 0),
			ctx.PlayLayer("S69_RodZeig"),
			ctx.PlayVoiceover("069_ROD_17", "[069_ROD_17]"),
			loc19Range(ctx, "S69_RodZeig", 5, 0),
			ctx.HideLayer("S69_RodZeig"),
			ctx.ShowActor(actor),
			loc19Range(ctx, "S69_RamTalkAnim", 6, 0xd),
			loc19JasCloseupSpeech(ctx, "069_JAS_17"),
			ctx.FreezeLayer("S69_RamTalkAnim", 0),
			ctx.MakeLayerClickable("S69_JasWalkKette"),
			loc19SetOriginal(ctx, loc19StateJasTalk, 2),
		)
	case 2:
		return engine.Sequence(
			ctx.PlayVoiceover("069_ROD_18", "[069_ROD_18]"),
			loc19JasCloseupSpeech(ctx, "069_JAS_18"),
			ctx.MakeLayerClickable("S69_JasWalkKette"),
			loc19SetOriginal(ctx, loc19StateJasTalk, 3),
		)
	case 3:
		return engine.Sequence(
			ctx.PlayVoiceover("069_ROD_19", "[069_ROD_19]"),
			loc19JasCloseupSpeech(ctx, "069_JAS_19"),
			ctx.MakeLayerClickable("S69_JasWalkKette"),
			loc19SetOriginal(ctx, loc19StateJasTalk, 4),
		)
	case 4:
		return loc19NecklaceFinale(ctx, actor)
	}
	return nil
}

func loc19NecklaceReturnCutscene(ctx *Context, actor string) engine.Task {
	return engine.Sequence(
		ctx.RunAmbient(ctx.WalkToFacingPerspective(actor, 0x112, 0x14d, 3)),
		loc19SetLayerState(ctx, "S69_JasWalkKette", -90, 174, 17, 130, 0x54),
		loc19SetLayerState(ctx, "S69_RamWalk", -78, 250, 17, 118, 0),
		engine.Parallel(
			loc19MoveLayerRange(ctx, "S69_JasWalkKette", 179, 163, 43, 106, 0x28, 0x54, 0x5f),
			loc19MoveLayerRange(ctx, "S69_RamWalk", 225, 206, 65, 90, 0x28, 0, 7),
		),
		ctx.FreezeLayer("S69_RamWalk", 0),
		ctx.FreezeLayer("S69_JasWalkKette", loc19FrameStart(0xc)),
		ctx.HideActor(actor),
		ctx.HideLayer("S69_RamWalk"),
		loc19CloseupSpeech(ctx, "069_RAM_23", false, 0, 0),
		loc19CloseupSpeech(ctx, "069_JAS_23", true, 1, 0.5),
		loc19SetLayerState(ctx, "S69_RamTalkAnim", 225, 205, 65, 90, 0),
		ctx.MakeLayerClickable("S69_JasWalkKette"),
		loc19SetLayerState(ctx, "S69_RodStarrt", 243, 172, 24, 102, 0),
		loc19Range(ctx, "S69_RodStarrt", 0, 6),
		ctx.Wait(0.5),
		loc19Range(ctx, "S69_RodStarrt", 7, -1),
		ctx.HideLayer("S69_RodStarrt"),
		loc19CloseupSpeech(ctx, "069_JAS_20", true, 1, 0.5),
		loc19SetOriginal(ctx, loc19StateRamPose, 1),
		ctx.ShowActor(actor),
		loc19SetOriginal(ctx, loc19StateStoryTalk, 1),
		loc19SetOriginal(ctx, loc19StateJasGate, 2),
	)
}

func loc19NecklaceFinale(ctx *Context, actor string) engine.Task {
	return engine.Sequence(
		ctx.PlayVoiceover("069_ROD_21", "[069_ROD_21]"),
		loc19Range(ctx, "S69_RamTalkAnim", 6, 0xd),
		loc19LayerSpeech(ctx, "S69_RamTalkAnim", "069_RAM_21", 0xe, 0x12),
		loc19Range(ctx, "S69_RamTalkAnim", 0xd, 6),
		loc19SetLayerState(ctx, "S69_ThaWalk", 288, 196, 93, 65, 0),
		ctx.PlaySFX("Sfx_Door_Creaking_Medium.wav"),
		ctx.RunAmbient(ctx.PlayLayer("S69_Tuer")),
		ctx.PlayVoiceover("069_THA_21", "[069_THA_21]"),
		ctx.PlayVoiceover("069_JAS_21", "[069_JAS_21]"),
		loc19Range(ctx, "S69_RamTalkAnim", 6, 0xd),
		loc19LayerSpeech(ctx, "S69_RamTalkAnim", "069_RAM_22", 0xe, 0x12),
		loc19Range(ctx, "S69_RamTalkAnim", 0xd, 6),
		ctx.HideLayer("S69_JasWalkKette"),
		ctx.DisableArea("S69_JasWalkKette"),
		loc19SetLayerState(ctx, "S69_JasLauf", 178, 178, 36, 102, 0),
		loc19MoveLayer(ctx, "S69_JasLauf", 593, 174, 157, 70, 0x23, true),
		ctx.HideLayer("S69_JasLauf"),
		ctx.Wait(0.5),
		ctx.HideLayer("S69_RamTalkAnim"),
		ctx.DisableArea("S69_RamBackTalk"),
		loc19SetLayerState(ctx, "S69_RamWalk", 211, 209, 48, 96, 0x1c),
		loc19MoveLayerRange(ctx, "S69_RamWalk", -69, 245, 12, 118, 0x28, 0x18, 0x23),
		ctx.HideLayer("S69_RamWalk"),
		loc19Range(ctx, "S69_ThaWalk", 0xb, 0x19),
		ctx.PlaySFX("Sfx_Door_Creaking_Medium1.wav"),
		loc19Range(ctx, "S69_Tuer", 10, 0),
		ctx.HideLayer("S69_ThaWalk"),
		ctx.SetActorOrientation(actor, 0),
		ctx.Say(actor, "069_ROD_22", "[069_ROD_22]"),
		loc19SetOriginal(ctx, loc10StateTree, 1),
		loc19SetOriginal(ctx, loc19StateJasActor, 0),
		loc19SetOriginal(ctx, loc19StateRamPose, -1),
	)
}

func loc19DwarfTalk(ctx *Context, actor string) engine.Task {
	if loc19Original(ctx, loc19StateJasVisible) == 0 {
		return nil
	}
	if loc19Original(ctx, loc19StateDwarfTalk) != 1 {
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0x92, 0x135, 2),
			ctx.Say(actor, "069_ROD_08", "[069_ROD_08]"),
			loc19LayerSpeech(ctx, "S69_Zwerg1", "069_ZW1_08", 0xc, 0x10),
		)
	}
	jas := loc10JasminActorID(ctx)
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0x92, 0x135, 2),
		ctx.Say(actor, "069_ROD_06", "[069_ROD_06]"),
		loc19LayerSpeech(ctx, "S69_Zwerg1", "069_ZW1_06", 0xc, 0x10),
		ctx.RunAmbient(ctx.WalkToFacingPerspective(jas, 0x1a8, 0x137, 1)),
		loc19SetLayerState(ctx, "S69_RamRun", -97, 264, 1, 118, 0),
		loc19MoveLayer(ctx, "S69_RamRun", 359, 223, 1, 96, 30, true),
		ctx.HideLayer("S69_RamRun"),
		ctx.HideActor(jas),
		loc19SetLayerState(ctx, "S69_Umarm", 396, 160, 47, 82, 0),
		ctx.PlayLayer("S69_Umarm"),
		ctx.HideLayer("S69_Umarm"),
		ctx.RunAmbient(ctx.WalkToFacingPerspective(actor, 0x15a, 0x13d, 6)),
		loc19SetLayerState(ctx, "S69_JasTalk", 410, 188, 47, 82, 0),
		loc19SetLayerState(ctx, "S69_RamBackTalk", 425, 214, 42, 82, 0),
		ctx.MakeLayerClickable("S69_RamBackTalk"),
		ctx.PlayVoiceover("069_RAM_06", "[069_RAM_06]"),
		ctx.Wait(1),
		loc19Range(ctx, "S69_RamBackTalk", 0, 10),
		loc19LayerSpeech(ctx, "S69_RamBackTalk", "069_RAM_07", 10, 0xc),
		loc19Range(ctx, "S69_RamBackTalk", 10, 0),
		loc19SetOriginal(ctx, loc19StateRamPose, 2),
		loc19SetOriginal(ctx, loc19StateDwarfTalk, 2),
	)
}

type loc19Ambient struct {
	ctx  *Context
	wait float64
}

func (t *loc19Ambient) Update(dt float64) bool {
	if t.ctx.session.state.Location != 19 {
		return true
	}
	dark := loc19SceneID(t.ctx.session.state.Scene) == "S101"
	if t.wait <= 0 {
		if dark {
			t.wait = float64(5 + rand.IntN(11))
		} else {
			t.wait = float64(2 + rand.IntN(9))
		}
	}
	t.wait -= dt
	if t.wait > 0 {
		return false
	}
	if dark {
		sounds := []string{"Sfx_Thunder.wav", "Sfx_Thunder2.wav"}
		_ = t.ctx.PlaySFX(sounds[rand.IntN(len(sounds))]).Update(0)
		t.wait = float64(5 + rand.IntN(11))
		return false
	}
	sounds := []string{"Sfx_Bird.wav", "Sfx_Bird2.wav", "Sfx_Bird3.wav", "Sfx_Chirp.wav"}
	_ = t.ctx.PlaySFX(sounds[rand.IntN(len(sounds))]).Update(0)
	t.wait = float64(2 + rand.IntN(9))
	return false
}
