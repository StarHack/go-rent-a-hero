package game

import (
	"math/rand/v2"
	"strings"

	"github.com/wok/rent-a-hero/internal/engine"
)

const (
	loc22StateIntro = 0x4060
	loc22Slot0      = 0x4064

	// FUN_00425cf0 initializes the controller-local first-removal hint flag
	// (+0x128) to one every time Location 22 is constructed. Keep the Go
	// equivalent transient to an S1081 visit rather than storing it in the
	// original-state block.
	loc22RemoveHintFlag = "Loc22S1081RemoveHint"
)

var loc22S81StoneLayers = []string{
	"S81_Stone12Uhr",
	"S81_Stone02Uhr",
	"S81_Stone04Uhr",
	"S81_Stone06Uhr",
	"S81_Stone08Uhr",
	"S81_Stone10Uhr",
}

var loc22S1081RodLayers = []string{
	"S1081_Rod12Uhr",
	"S1081_Rod02Uhr",
	"S1081_Rod04Uhr",
	"S1081_Rod06Uhr",
	"S1081_Rod08Uhr",
	"S1081_Rod10Uhr",
}

var loc22S1081StoneLayers = []string{
	"S1081_Stone12Uhr",
	"S1081_Stone02Uhr",
	"S1081_Stone04Uhr",
	"S1081_Stone06Uhr",
	"S1081_Stone08Uhr",
	"S1081_Stone10Uhr",
}

// FUN_00425cf0 stores the six authored S1081_BigStone ranges at controller
// offsets +0xf8..+0x124. FUN_00426840 indexes those values by inventory ID.
var loc22BigStoneRanges = map[int][2]int{
	0x11: {0x1e, 0x28},
	0x12: {0x0a, 0x14},
	0x13: {0x14, 0x1e},
	0x14: {0x28, 0x32},
	0x15: {0x00, 0x0a},
	0x16: {0x32, 0x3c},
}

var loc22Solution = [6]int{0x15, 0x16, 0x11, 0x12, 0x14, 0x13}

type LOC22Controller struct{}

func loc22SceneID(scene string) string {
	s := strings.ToUpper(strings.TrimSpace(scene))
	s = strings.TrimPrefix(s, "S")
	switch s {
	case "81":
		return "S81"
	case "1081":
		return "S1081"
	}
	return strings.ToUpper(strings.TrimSpace(scene))
}

func loc22Original(ctx *Context, off int) int {
	return getOriginalFlag(ctx.session.state.OriginalState, off)
}

func loc22SetOriginal(ctx *Context, off, value int) engine.Task {
	return engine.Immediate(func() { putOriginalFlag(ctx.session.state.OriginalState, off, value) })
}

func loc22AnyStonePlaced(ctx *Context) bool {
	for i := 0; i < 6; i++ {
		if loc22Original(ctx, loc22Slot0+i*4) != 0 {
			return true
		}
	}
	return false
}

func loc22EnsureLayer(ctx *Context, id string) {
	_, _, _ = ctx.ensureAssetLayer(id)
}

func loc22SetLayerNow(ctx *Context, id string, visible bool, frame int) {
	layer, ok := ctx.layer(id)
	if !ok || layer == nil {
		return
	}
	layer.Visible = visible
	layer.Enabled = true
	layer.Playing = false
	layer.TaskDriven = false
	layer.Accumulator = 0
	if frame >= 0 && layer.Source != nil && frame < layer.Source.Frames() {
		layer.Frame = frame
	}
}

func loc22EnsureClickableNow(ctx *Context, id string, enabled bool) {
	if ctx.session.scene == nil {
		return
	}
	if area, ok := ctx.session.scene.Areas[id]; ok && area != nil {
		area.Enabled = enabled
		return
	}
	layer, ok := ctx.layer(id)
	if !ok || layer == nil || layer.Source == nil {
		return
	}
	w := layer.Source.Width() * layer.Zoom / 100
	h := layer.Source.Height() * layer.Zoom / 100
	area := &engine.Area{
		ID:         id,
		X1:         layer.X,
		Y1:         layer.Y,
		X2:         layer.X + w,
		Y2:         layer.Y + h,
		CursorType: 10,
		Enabled:    enabled,
	}
	ctx.session.localizeArea(area, ctx.session.state.Location)
	ctx.session.scene.Areas[id] = area
	ctx.session.scene.AreaOrder = append(ctx.session.scene.AreaOrder, id)
}

func loc22PrioritizeAreas(ctx *Context, ids ...string) engine.Task {
	return engine.Immediate(func() {
		if ctx.session.scene == nil || len(ids) == 0 {
			return
		}
		wanted := make(map[string]bool, len(ids))
		order := make([]string, 0, len(ctx.session.scene.AreaOrder))
		for _, id := range ids {
			if _, ok := ctx.session.scene.Areas[id]; ok && !wanted[id] {
				wanted[id] = true
				order = append(order, id)
			}
		}
		for _, id := range ctx.session.scene.AreaOrder {
			if !wanted[id] {
				order = append(order, id)
			}
		}
		ctx.session.scene.AreaOrder = order
	})
}

func loc22RegisterFrameSFX(ctx *Context) {
	// FUN_00425cf0 binds Sfx_Door_Opened to frame 0x1b on all six
	// Rodrigo placement/removal animations.
	for _, id := range loc22S1081RodLayers {
		layer, ok := ctx.layer(id)
		if !ok || layer == nil {
			continue
		}
		if len(layer.FrameEvents[0x1b]) == 0 {
			layer.AddFrameEvent(0x1b, func() {
				_ = ctx.PlaySFX("Sfx_Door_Opened.wav").Update(0)
			})
		}
	}

	// The big crystal mechanism has six impact cues at the authored frames.
	if layer, ok := ctx.layer("S1081_BigStone"); ok && layer != nil {
		for _, frame := range []int{1, 0x0b, 0x15, 0x1f, 0x29, 0x33} {
			if len(layer.FrameEvents[frame]) != 0 {
				continue
			}
			layer.AddFrameEvent(frame, func() {
				_ = ctx.PlaySFX("Sfx_BigStone.wav").Update(0)
			})
		}
	}
}

func loc22S1081ReconstructNow(ctx *Context, actor string) {
	for _, id := range append([]string{"S1081_BigStone", "S1081_RodStreich"}, append(loc22S1081RodLayers, loc22S1081StoneLayers...)...) {
		loc22EnsureLayer(ctx, id)
	}
	loc22RegisterFrameSFX(ctx)

	// S1081 is an authored close-up puzzle. The original uses RodStreich and
	// the six RodXXUhr layers instead of the freely walking actor.
	if actor != "" {
		if a, ok := ctx.actor(actor); ok {
			a.Visible = false
		}
	}

	loc22SetLayerNow(ctx, "S1081_BigStone", true, 0)
	loc22SetLayerNow(ctx, "S1081_RodStreich", true, 0)
	loc22EnsureClickableNow(ctx, "S1081_BigStone", true)

	for i := 0; i < 6; i++ {
		loc22SetLayerNow(ctx, loc22S1081RodLayers[i], false, 0)
		occupied := loc22Original(ctx, loc22Slot0+i*4) != 0
		loc22SetLayerNow(ctx, loc22S1081StoneLayers[i], occupied, 0)
		loc22EnsureClickableNow(ctx, loc22S1081StoneLayers[i], occupied)
	}
	loc22EnsureClickableNow(ctx, "S1081_To81", true)

	// Controller-local +0x128 starts at one on every original construction.
	if ctx.session.state.Flags == nil {
		ctx.session.state.Flags = map[string]int{}
	}
	ctx.session.state.Flags[loc22RemoveHintFlag] = 1
}

func (LOC22Controller) LoadConditionMask(*Context, string) int    { return 0 }
func (LOC22Controller) Exit(*Context, string, string) engine.Task { return nil }

func (LOC22Controller) Enter(ctx *Context, scene, from string) engine.Task {
	scene = loc22SceneID(scene)
	from = loc22SceneID(from)
	actor := loc10ActorID(ctx)
	tasks := []engine.Task{ctx.PlayMusic("Loc22_CrystalCave.wav")}

	switch scene {
	case "S81":
		for _, id := range []string{"S81_Enter", "S81_Exit", "S81_To1081"} {
			loc22EnsureLayer(ctx, id)
		}
		if actor != "" {
			if a, ok := ctx.actor(actor); ok {
				a.Visible = true
			}
		}

		// The generic SZN loader otherwise enables the area immediately. The
		// original has input locked for S81_Enter and only exposes the cave
		// entrance after the authored entry sequence is complete.
		if area, ok := ctx.session.scene.Areas["S81_To1081"]; ok && area != nil {
			area.Enabled = false
		}
		if area, ok := ctx.session.scene.Areas["S81_To79"]; ok && area != nil {
			area.Enabled = true
		}

		for i, id := range loc22S81StoneLayers {
			loc22EnsureLayer(ctx, id)
			loc22SetLayerNow(ctx, id, loc22Original(ctx, loc22Slot0+i*4) != 0, 0)
		}

		tasks = append(tasks, loc22SetOriginal(ctx, 0x3fec, 1))

		if from == "S1081" {
			return engine.Sequence(append(tasks,
				ctx.PlaceActorPerspective(actor, 0x199, 0x13e),
				ctx.WalkToFacingPerspective(actor, 0x186, 0x14a, 1),
				ctx.EnableArea("S81_To1081"),
			)...)
		}

		if !loc22AnyStonePlaced(ctx) {
			tasks = append(tasks,
				ctx.HideActor(actor),
				ctx.ShowLayer("S81_Enter"),
				ctx.PlayLayer("S81_Enter"),
				ctx.HideLayer("S81_Enter"),
				ctx.ShowActor(actor),
			)
		}

		tasks = append(tasks,
			ctx.PlaceActorPerspective(actor, 0xd1, 0x155),
			ctx.WalkToFacingPerspective(actor, 0xdc, 0x15e, 5),
		)
		if loc22Original(ctx, loc22StateIntro) != 0 {
			tasks = append(tasks,
				ctx.Wait(1),
				ctx.Say(actor, "081_ROD_01", "[081_ROD_01]"),
				loc22SetOriginal(ctx, loc22StateIntro, 0),
			)
		}
		tasks = append(tasks, ctx.EnableArea("S81_To1081"))

	case "S1081":
		loc22S1081ReconstructNow(ctx, actor)
		tasks = append(tasks, loc22PrioritizeAreas(ctx, append(loc22S1081StoneLayers, "S1081_BigStone", "S1081_To81")...))
	}

	return engine.Sequence(tasks...)
}

func (LOC22Controller) Click(ctx *Context, area string) engine.Task {
	actor := loc10ActorID(ctx)
	switch loc22SceneID(ctx.session.state.Scene) {
	case "S81":
		switch area {
		case "S81_To1081":
			tasks := []engine.Task{ctx.WalkToFacingPerspective(actor, 0x199, 0x13e, 5)}
			if !loc22AnyStonePlaced(ctx) {
				tasks = append(tasks,
					ctx.HideActor(actor),
					ctx.ShowLayer("S81_To1081"),
					ctx.PlayLayer("S81_To1081"),
					ctx.HideLayer("S81_To1081"),
					ctx.ShowActor(actor),
				)
			}
			tasks = append(tasks, ctx.ChangeScene("S1081"))
			return engine.Sequence(tasks...)
		case "S81_To79":
			tasks := []engine.Task{ctx.WalkToFacingPerspective(actor, 0xd1, 0x14c, 4)}
			if !loc22AnyStonePlaced(ctx) {
				tasks = append(tasks,
					ctx.HideActor(actor),
					ctx.ShowLayer("S81_Exit"),
					ctx.PlayLayer("S81_Exit"),
					ctx.HideLayer("S81_Exit"),
					ctx.ShowActor(actor),
				)
			}
			tasks = append(tasks, ctx.ChangeLocation(21, "79"))
			return engine.Sequence(tasks...)
		}

	case "S1081":
		switch area {
		case "S1081_To81":
			return ctx.ChangeScene("S81")
		case "S1081_BigStone":
			return loc22InspectBigStone(ctx)
		}
		for i, id := range loc22S1081StoneLayers {
			if area == id {
				return loc22RemoveStone(ctx, i)
			}
		}
	}
	return nil
}

func (LOC22Controller) UseItem(*Context, int, string) engine.Task { return nil }

func (LOC22Controller) SelectItem(ctx *Context, item int) engine.Task {
	if loc22SceneID(ctx.session.state.Scene) != "S1081" {
		return nil
	}

	// FUN_00426270 registers item 0x10 -> callback 0x1e and items
	// 0x11..0x16 -> callbacks 0x11..0x16 directly in the inventory table.
	switch item {
	case 0x10:
		return loc22OpenCrystalSet(ctx)
	case 0x11, 0x12, 0x13, 0x14, 0x15, 0x16:
		return loc22PlaceStone(ctx, item)
	}
	return nil
}

func loc22InspectBigStone(ctx *Context) engine.Task {
	// FUN_00440690(this,1,2): uniform original PRNG choice between two
	// responses. Variant one also runs RodStreich in mode 0xc while the
	// voice line is active.
	if rand.IntN(2) == 0 {
		return engine.Parallel(
			ctx.PlayVoiceover("1081_ROD_01", "[1081_ROD_01]"),
			ctx.PlayLayer("S1081_RodStreich"),
		)
	}
	return ctx.PlayVoiceover("1081_ROD_03", "[1081_ROD_03]")
}

func loc22OpenCrystalSet(ctx *Context) engine.Task {
	if !ctx.HasItem(0x10) {
		return engine.Immediate(func() {})
	}
	return engine.Sequence(
		ctx.PlayVoiceover("1081_ROD_04", "[1081_ROD_04]"),
		ctx.PlaySFX("Sfx_MobileDown.wav"),
		ctx.RemoveItem(0x10),
		ctx.AddItem(0x11),
		ctx.Wait(0.1),
		ctx.AddItem(0x12),
		ctx.Wait(0.1),
		ctx.AddItem(0x13),
		ctx.Wait(0.1),
		ctx.AddItem(0x14),
		ctx.Wait(0.1),
		ctx.AddItem(0x16),
	)
}

func loc22CheckSolution(ctx *Context) engine.Task {
	var inner engine.Task
	checked := false
	return engine.TaskFunc(func(dt float64) bool {
		if !checked {
			checked = true
			for i, want := range loc22Solution {
				if loc22Original(ctx, loc22Slot0+i*4) != want {
					return true
				}
			}
			inner = engine.Sequence(
				ctx.PlaySFX("Sfx_DangerStringsHigh.wav"),
				ctx.PlayLayer("S1081_BigStone"),
				ctx.ChangeLocation(9, "89"),
			)
		}
		if inner == nil {
			return true
		}
		return inner.Update(dt)
	})
}

func loc22PlaceStone(ctx *Context, item int) engine.Task {
	// Direct inventory callbacks are consumed even if all six slots are full;
	// returning nil would incorrectly arm the inventory item for world use.
	rng, ok := loc22BigStoneRanges[item]
	if !ok {
		return engine.Immediate(func() {})
	}

	slot := -1
	for i := 0; i < 6; i++ {
		if loc22Original(ctx, loc22Slot0+i*4) == 0 {
			slot = i
			break
		}
	}
	if slot < 0 {
		return engine.Immediate(func() {})
	}

	rod := loc22S1081RodLayers[slot]
	stone := loc22S1081StoneLayers[slot]
	tasks := []engine.Task{
		ctx.HideLayer("S1081_RodStreich"),
		ctx.RemoveItem(item),
		ctx.ShowLayer(rod),
		ctx.PlayLayerFrames(rod, 1, -1),
		ctx.HideLayer(rod),
		ctx.ShowLayer("S1081_RodStreich"),
		ctx.ShowLayer(stone),
		engine.Immediate(func() { loc22EnsureClickableNow(ctx, stone, true) }),
		ctx.PlayLayerFrames("S1081_BigStone", rng[0], rng[1]),
		loc22SetOriginal(ctx, loc22Slot0+slot*4, item),
		loc22PrioritizeAreas(ctx, append(loc22S1081StoneLayers, "S1081_BigStone", "S1081_To81")...),
	}

	// FUN_00426840 writes the selected item into the slot state first, then
	// reads all six persistent values to test the solution. Do that check at
	// task execution time, after loc22SetOriginal above has committed.
	tasks = append(tasks, loc22CheckSolution(ctx))
	return engine.Sequence(tasks...)
}

func loc22RemoveStone(ctx *Context, slot int) engine.Task {
	if slot < 0 || slot >= 6 {
		return nil
	}
	item := loc22Original(ctx, loc22Slot0+slot*4)
	if item == 0 {
		return nil
	}

	rod := loc22S1081RodLayers[slot]
	stone := loc22S1081StoneLayers[slot]
	tasks := []engine.Task{
		ctx.HideLayer("S1081_RodStreich"),
		ctx.HideLayer(stone),
		ctx.DisableArea(stone),
	}
	if ctx.GetFlag(loc22RemoveHintFlag) != 0 {
		tasks = append(tasks,
			ctx.PlayVoiceover("1081_ROD_02", "[1081_ROD_02]"),
			ctx.SetFlag(loc22RemoveHintFlag, 0),
		)
	}
	tasks = append(tasks,
		ctx.ShowLayer(rod),
		ctx.PlayLayerFrames(rod, 0x1f, 1),
		ctx.HideLayer(rod),
		ctx.ShowLayer("S1081_RodStreich"),
		ctx.AddItem(item),
		loc22SetOriginal(ctx, loc22Slot0+slot*4, 0),
	)
	return engine.Sequence(tasks...)
}
