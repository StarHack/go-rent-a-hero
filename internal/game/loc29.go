package game

import (
	"strconv"
	"strings"

	"github.com/wok/rent-a-hero/internal/engine"
)

const (
	loc29Scene33      = "S33"
	loc29StateJug     = 0x40f4
	loc29StatePirates = 0x40f8
)

type LOC29Controller struct{}

func loc29SceneID(scene string) string {
	s := strings.ToUpper(strings.TrimSpace(scene))
	s = strings.TrimPrefix(s, "S")
	if n, err := strconv.Atoi(s); err == nil && n == 33 {
		return loc29Scene33
	}
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(scene)), "S") {
		return strings.ToUpper(strings.TrimSpace(scene))
	}
	return s
}

func loc29Original(ctx *Context, off int) int {
	return getOriginalFlag(ctx.session.state.OriginalState, off)
}

func loc29SetOriginal(ctx *Context, off, value int) engine.Task {
	return engine.Immediate(func() { putOriginalFlag(ctx.session.state.OriginalState, off, value) })
}

func loc29FrameStart(frame int) int {
	if frame > 0 {
		return frame - 1
	}
	return frame
}

func loc29LayerSpeech(ctx *Context, layerID, line string, start, end int) engine.Task {
	if _, ok := ctx.layer(layerID); !ok {
		return ctx.PlayVoiceover(line, "["+line+"]")
	}
	return ctx.PlaySpeechBoundToLayer(layerID, line, "["+line+"]", loc29FrameStart(start), end)
}

func loc29PlayRange(ctx *Context, id string, from, to int) engine.Task {
	return ctx.PlayLayerFrames(id, loc29FrameStart(from), to)
}

func loc29ShowStatic(ctx *Context, id string, frame int) engine.Task {
	return engine.Sequence(ctx.FreezeLayer(id, loc29FrameStart(frame)), ctx.ShowLayer(id))
}

func loc29SetJugDormant(ctx *Context) {
	if layer, ok := ctx.layer("S33_Krug1"); ok && layer != nil {
		layer.Visible = false
		layer.Enabled = false
		layer.Playing = false
		layer.TaskDriven = false
		layer.Frame = 0
		layer.Accumulator = 0
	}
	if area, ok := ctx.session.scene.Areas["S33_Krug1"]; ok && area != nil {
		area.Enabled = false
	}
	order := ctx.session.scene.AreaOrder[:0]
	for _, id := range ctx.session.scene.AreaOrder {
		if id != "S33_Krug1" {
			order = append(order, id)
		}
	}
	ctx.session.scene.AreaOrder = order
}

func loc29PrioritizeJug(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		order := make([]string, 0, len(ctx.session.scene.AreaOrder))
		for _, id := range ctx.session.scene.AreaOrder {
			if id != "S33_Krug1" {
				order = append(order, id)
			}
		}
		insert := len(order)
		for i, id := range order {
			if id == "S33_Pirate1" || id == "S33_Pirate2" || id == "S33_Pirate3" {
				insert = i
				break
			}
		}
		order = append(order, "")
		copy(order[insert+1:], order[insert:])
		order[insert] = "S33_Krug1"
		ctx.session.scene.AreaOrder = order
	})
}

func loc29RevealJug(ctx *Context) engine.Task {
	return engine.Sequence(
		engine.Immediate(func() {
			layer, ok := ctx.layer("S33_Krug1")
			if !ok || layer == nil {
				return
			}
			layer.Playing = false
			layer.TaskDriven = false
			layer.Frame = 0
			layer.Accumulator = 0
			layer.Enabled = true
			layer.Visible = true
		}),
		ctx.MakeLayerClickable("S33_Krug1"),
		ctx.EnableArea("S33_Krug1"),
		loc29PrioritizeJug(ctx),
	)
}

func loc29PlayHide(ctx *Context, id string) engine.Task {
	return engine.Sequence(ctx.ShowLayer(id), ctx.PlayLayer(id), ctx.HideLayer(id))
}

func (LOC29Controller) LoadConditionMask(*Context, string) int { return 0 }

func (LOC29Controller) Enter(ctx *Context, scene string, from string) engine.Task {
	if loc29SceneID(scene) != loc29Scene33 {
		return nil
	}
	_, _, _ = ctx.ensureAssetLayer("S33_Enter")
	_, _, _ = ctx.ensureAssetLayer("S33_RodKipptAndLook")
	if loc29Original(ctx, loc29StateJug) == 0 {
		loc29SetJugDormant(ctx)
	}

	tasks := []engine.Task{
		ctx.PlayMusic("Loc29_PubInEndavin.wav"),
		ctx.HideActor(loc28Actor(ctx)),
		ctx.HideLayer("S33_Enter"),
		ctx.FreezeLayer("S33_Enter", 0),
		ctx.HideLayer("S33_RodKipptAndLook"),
		ctx.FreezeLayer("S33_RodKipptAndLook", 0),
		ctx.HideLayer("S33_RodGeht"),
		ctx.FreezeLayer("S33_RodGeht", 0),
		loc29ShowStatic(ctx, "S33_Pirate1", 0),
		loc29ShowStatic(ctx, "S33_Pirate2", 0),
		loc29ShowStatic(ctx, "S33_Pirate3", 0),
		loc29ShowStatic(ctx, "S33_Rodrigo", 0),
		ctx.MakeLayerClickable("S33_Pirate1"),
		ctx.MakeLayerClickable("S33_Pirate2"),
		ctx.MakeLayerClickable("S33_Pirate3"),
		ctx.EnableArea("S33_To32"),
	}

	if loc29Original(ctx, loc29StateJug) == 0 {
		tasks = append(tasks, loc29PlayHide(ctx, "S33_Enter"))
	} else {
		tasks = append(tasks, loc29RevealJug(ctx))
	}

	if loc29Original(ctx, loc29StateJug) != 0 || loc29Original(ctx, loc29StatePirates) == 2 {
		tasks = append(tasks, ctx.RunAmbient(loc29PlayRange(ctx, "S33_Pirate3", 0, -1)))
	}

	tasks = append(tasks, ctx.RunAmbient(loc29PlayRange(ctx, "S33_Pirate2", 0, 10)))
	return engine.Sequence(tasks...)
}

func (LOC29Controller) Exit(*Context, string, string) engine.Task { return nil }

func (LOC29Controller) Click(ctx *Context, area string) engine.Task {
	if loc29SceneID(ctx.session.state.Scene) != loc29Scene33 {
		return nil
	}
	switch area {
	case "S33_To32":
		return engine.Sequence(
			ctx.HideLayer("S33_Rodrigo"),
			loc29PlayHide(ctx, "S33_RodGeht"),
			ctx.ChangeLocation(28, "S32"),
		)
	case "S33_Pirate1", "S33_Pirate2", "S33_Pirate3":
		return loc29PirateTalk(ctx)
	case "S33_Krug1":
		if loc29Original(ctx, loc29StateJug) != 0 {
			return loc29JugSequence(ctx)
		}
	}
	return nil
}

func loc29PirateTalk(ctx *Context) engine.Task {
	switch loc29Original(ctx, loc29StatePirates) {
	case 1:
		return engine.Sequence(
			loc29LayerSpeech(ctx, "S33_Rodrigo", "033_ROD_01", 0, 0xf),
			loc29LayerSpeech(ctx, "S33_Pirate1", "033_PI2_01", 0x13, 0x27),
			ctx.RunAmbient(loc29PlayRange(ctx, "S33_Pirate1", 8, 0x13)),
			loc29LayerSpeech(ctx, "S33_Pirate2", "033_PI3_01", 0x55, 0x68),
			loc29SetOriginal(ctx, loc29StatePirates, 2),
			ctx.RunAmbient(loc29PlayRange(ctx, "S33_Pirate3", 0, -1)),
		)
	case 2:
		return engine.Sequence(
			loc29LayerSpeech(ctx, "S33_Rodrigo", "033_ROD_02", 0, 0xf),
			loc29LayerSpeech(ctx, "S33_Pirate1", "033_PI2_02", 0, 8),
			loc29LayerSpeech(ctx, "S33_Pirate2", "033_PI3_02", 0x55, 0x68),
			loc29LayerSpeech(ctx, "S33_Pirate2", "033_PI3_03", 10, 0x16),
			loc29PlayRange(ctx, "S33_Pirate2", 0x2c, 0x49),
			ctx.PlaySFX("Sfx_Tock.wav"),
			ctx.RunAmbient(loc29PlayRange(ctx, "S33_Pirate1", 8, 0x13)),
			loc29PlayRange(ctx, "S33_Pirate2", 0x4a, 0x54),
			loc29RevealJug(ctx),
			ctx.FreezeLayer("S33_Pirate2", loc29FrameStart(0x2c)),
			loc29SetOriginal(ctx, loc29StateJug, 1),
			loc29SetOriginal(ctx, loc29StatePirates, 3),
		)
	case 3:
		return engine.Sequence(
			loc29LayerSpeech(ctx, "S33_Rodrigo", "033_ROD_04", 0, 0xf),
			loc29LayerSpeech(ctx, "S33_Pirate1", "033_PI2_04", 0, 8),
		)
	}
	return nil
}

func loc29JugSequence(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.HideLayer("S33_Pirate1"),
		ctx.HideLayer("S33_Pirate2"),
		ctx.HideLayer("S33_Pirate3"),
		ctx.HideLayer("S33_Rodrigo"),
		ctx.HideLayer("S33_Krug1"),
		ctx.DisableArea("S33_Krug1"),
		ctx.DisableArea("S33_To32"),
		ctx.ShowLayer("S33_RodKipptAndLook"),
		loc29PlayRange(ctx, "S33_RodKipptAndLook", 0, 9),
		ctx.PlaySFX("Sfx_Drinking.wav"),
		loc29PlayRange(ctx, "S33_RodKipptAndLook", 10, 0x26),
		ctx.PlaySFX("Sfx_PianoStroke.wav"),
		loc29PlayRange(ctx, "S33_RodKipptAndLook", 0x27, 0x41),
		ctx.PlaySFX("Sfx_Knock_Wet.wav"),
		loc29PlayRange(ctx, "S33_RodKipptAndLook", 0x42, 0x4c),
		ctx.PlayVoiceover("033_PI2_05", "[033_PI2_05]"),
		ctx.Wait(4),
		loc29PlayRange(ctx, "S33_RodKipptAndLook", 0x4d, 0x89),
		ctx.PlayVoiceover("033_PI3_05", "[033_PI3_05]"),
		loc29PlayRange(ctx, "S33_RodKipptAndLook", 0x8a, -1),
		ctx.HideLayer("S33_RodKipptAndLook"),
		ctx.ChangeLocation(30, "34"),
	)
}

func (LOC29Controller) UseItem(ctx *Context, item int, area string) engine.Task {
	if loc29SceneID(ctx.session.state.Scene) != loc29Scene33 {
		return nil
	}
	switch item {
	case 1, 3, 4, 0xe:
		return ctx.PlayVoiceover("GEN_ROD_01", "[GEN_ROD_01]")
	}
	return nil
}

func (LOC29Controller) SelectItem(ctx *Context, item int) engine.Task {
	if loc29SceneID(ctx.session.state.Scene) != loc29Scene33 {
		return nil
	}
	switch item {
	case 1, 3, 4, 0xe:
		return ctx.PlayVoiceover("GEN_ROD_01", "[GEN_ROD_01]")
	}
	return nil
}
