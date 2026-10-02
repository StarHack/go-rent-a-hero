package game

import (
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/wok/rent-a-hero/internal/engine"
)

const (
	loc21StateDrillLocation = 0x4034
	loc21StateGloomUsed     = 0x4038
	loc21StateSwitch        = 0x403c
	loc21StateSwitchTalk    = 0x4040
	loc21StateRubble76      = 0x4044
	loc21StateTinder        = 0x4048
	loc21StateRubble79      = 0x404c
	loc21StateStone         = 0x4054
	loc21StateTryStone      = 0x4058
	loc21StateWallPrimed    = 0x405c
)

type LOC21Controller struct{}

func loc21SceneID(scene string) string {
	s := strings.ToUpper(strings.TrimSpace(scene))
	s = strings.TrimPrefix(s, "S")
	if n, err := strconv.Atoi(s); err == nil && n >= 71 && n <= 80 {
		return "S" + strconv.Itoa(n)
	}
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(scene)), "S") {
		return strings.ToUpper(strings.TrimSpace(scene))
	}
	return s
}

func loc21Original(ctx *Context, off int) int {
	return getOriginalFlag(ctx.session.state.OriginalState, off)
}

func loc21SetOriginal(ctx *Context, off, value int) engine.Task {
	return engine.Immediate(func() { putOriginalFlag(ctx.session.state.OriginalState, off, value) })
}

func loc21SetLayerNow(ctx *Context, id string, visible bool, frame int, loop bool) {
	layer, ok := ctx.session.scene.Layers[id]
	if !ok || layer == nil {
		return
	}
	layer.Visible = visible
	layer.Enabled = true
	layer.Frame = frame
	layer.Accumulator = 0
	layer.TaskDriven = false
	if visible && loop {
		layer.Mode = engine.AnimLoop
		layer.Playing = true
	} else {
		layer.Playing = false
	}
	if area, ok := ctx.session.scene.Areas[id]; ok && !visible {
		area.Enabled = false
	}
}

func loc21SetLayerTransform(ctx *Context, id string, x, y, z, zoom int) engine.Task {
	return engine.Immediate(func() {
		if layer, ok := ctx.layer(id); ok {
			layer.X = x
			layer.Y = y
			layer.Z = z
			layer.Zoom = zoom
			layer.Visible = true
			layer.Enabled = true
		}
	})
}

func loc21DormantScene(ctx *Context, scene string) {
	var ids []string
	switch scene {
	case "S71":
		ids = []string{"S71_RodTakesShovel", "S71_Shovel"}
	case "S72":
		ids = []string{"S72_RodFeelsWall"}
	case "S73":
		ids = []string{"S73_RodUsesGloom", "S73_Drill", "S73_Hebel", "Gen_RodGreif", "Gen_SmallSteam"}
	case "S74":
		ids = []string{"S74_Hebel", "S74_RodUsesWeiche", "S74_RodLooksWeiche"}
	case "S76":
		ids = []string{"Gen_DrillLeaves", "Gen_DrillDrills", "Gen_Schutt", "Gen_Schutt2", "Gen_RodGreif", "Gen_SmallSteam", "Gen_RodPickUp", "S76_Zunder"}
	case "S79":
		ids = []string{"Gen_DrillLeaves", "Gen_DrillDrills", "Gen_Schutt", "Gen_Schutt2", "Gen_RodGreif", "Gen_SmallSteam"}
	case "S80":
		ids = []string{"S80_RodHacks", "S80_GloomOnFloor", "S80_RodTakesPickAxe", "S80_RodTakesStone", "S80_RodTriesTake", "S80_PickAxe"}
	}
	for _, id := range ids {
		loc21SetLayerNow(ctx, id, false, 0, false)
	}
}

func loc21PrioritizeAreas(ctx *Context, ids ...string) engine.Task {
	return engine.Immediate(func() {
		if ctx.session.scene == nil || len(ids) == 0 {
			return
		}
		wanted := make(map[string]bool, len(ids))
		ordered := make([]string, 0, len(ctx.session.scene.AreaOrder))
		for _, id := range ids {
			if _, ok := ctx.session.scene.Areas[id]; ok && !wanted[id] {
				wanted[id] = true
				ordered = append(ordered, id)
			}
		}
		for _, id := range ctx.session.scene.AreaOrder {
			if !wanted[id] {
				ordered = append(ordered, id)
			}
		}
		ctx.session.scene.AreaOrder = ordered
	})
}

func loc21ShowStatic(ctx *Context, id string, frame int) engine.Task {
	return engine.Immediate(func() { loc21SetLayerNow(ctx, id, true, frame, false) })
}

func loc21ShowLoop(ctx *Context, id string) engine.Task {
	return engine.Immediate(func() { loc21SetLayerNow(ctx, id, true, 0, true) })
}

func loc21PlayHide(ctx *Context, id string) engine.Task {
	return engine.Sequence(ctx.ShowLayer(id), ctx.PlayLayer(id), ctx.HideLayer(id), ctx.FreezeLayer(id, 0))
}

func loc21RangeHide(ctx *Context, id string, from, to int) engine.Task {
	return engine.Sequence(ctx.ShowLayer(id), ctx.PlayLayerFrames(id, from, to), ctx.HideLayer(id), ctx.FreezeLayer(id, 0))
}

func loc21Examine(ctx *Context, actor string, x, y float64, facing int, line string) engine.Task {
	return engine.Sequence(ctx.WalkToFacingPerspective(actor, x, y, facing), ctx.Say(actor, line, "["+line+"]"))
}

func loc21DrillLeavesForward(ctx *Context, id string) engine.Task {
	return engine.Sequence(
		ctx.PlayLayerFrames(id, 0, 1), ctx.PlaySFXVolume("Sfx_Winding.wav", 90),
		ctx.PlayLayerFrames(id, 2, 0xd), ctx.PlaySFX("Sfx_Winding.wav"),
		ctx.PlayLayerFrames(id, 0xe, 0x19), ctx.PlaySFXVolume("Sfx_Winding.wav", 80),
		ctx.PlayLayerFrames(id, 0x1a, 0x25), ctx.PlaySFXVolume("Sfx_Winding.wav", 70),
		ctx.PlayLayerFrames(id, 0x26, 0x31), ctx.PlaySFXVolume("Sfx_Winding.wav", 60),
		ctx.PlayLayerFrames(id, 0x32, 0x3d), ctx.PlaySFXVolume("Sfx_Winding.wav", 50),
		ctx.PlayLayerFrames(id, 0x3e, 0x49), ctx.PlaySFXVolume("Sfx_Winding.wav", 40),
		ctx.PlayLayerFrames(id, 0x4a, 0x55), ctx.PlaySFXVolume("Sfx_Winding.wav", 30),
		ctx.PlayLayerFrames(id, 0x56, -1),
		ctx.HideLayer(id),
	)
}

func loc21PlayGrab(ctx *Context, x, y, z, zoom int, frame int) engine.Task {
	return engine.Sequence(
		loc21SetLayerTransform(ctx, "Gen_RodGreif", x, y, z, zoom),
		ctx.PlayLayerFrames("Gen_RodGreif", 0, frame),
		ctx.PlaySFX("Sfx_Door_Opened.wav"),
		ctx.PlayLayerFrames("Gen_RodGreif", frame+1, -1),
		ctx.HideLayer("Gen_RodGreif"),
	)
}

func loc21PlayGrabAuthored(ctx *Context, frame int) engine.Task {
	return engine.Sequence(
		ctx.ShowLayer("Gen_RodGreif"),
		ctx.PlayLayerFrames("Gen_RodGreif", 0, frame),
		ctx.PlaySFX("Sfx_Door_Opened.wav"),
		ctx.PlayLayerFrames("Gen_RodGreif", frame+1, -1),
		ctx.HideLayer("Gen_RodGreif"),
	)
}

func loc21BindDrillSFX(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		layer, ok := ctx.layer("Gen_DrillDrills")
		if !ok || layer == nil {
			return
		}
		// Original Location 21 constructor binds Sfx_Drill to frame 1 of
		// Gen_DrillDrills with FUN_00442bb0. Register it once per scene
		// layer so every loop through frame 1 retriggers the SFX.
		if len(layer.FrameEvents[1]) == 0 {
			layer.AddFrameEvent(1, func() {
				_ = ctx.PlaySFX("Sfx_Drill.wav").Update(0)
			})
		}
	})
}

func loc21BindRubbleSFX(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		bind := func(layerID string, frame, count int) {
			layer, ok := ctx.layer(layerID)
			if !ok || layer == nil {
				return
			}
			for len(layer.FrameEvents[frame]) < count {
				layer.AddFrameEvent(frame, func() {
					_ = ctx.PlaySFX("Sfx_Explosion_Bass2.wav").Update(0)
				})
			}
		}

		// Original Location 21 constructor attaches Sfx_Explosion_Bass2
		// directly to the two rubble animations. Gen_Schutt deliberately
		// has two registrations at frame 0x1e, so preserve both callbacks.
		bind("Gen_Schutt", 0x0a, 1)
		bind("Gen_Schutt", 0x14, 1)
		bind("Gen_Schutt", 0x1e, 2)
		bind("Gen_Schutt2", 0x14, 1)
		bind("Gen_Schutt2", 0x21, 1)
	})
}

func loc21StartDrill(ctx *Context) engine.Task {
	return loc21ShowLoop(ctx, "Gen_DrillDrills")
}

func loc21StopDrill(ctx *Context) engine.Task {
	return ctx.FreezeLayer("Gen_DrillDrills", 0)
}

func (LOC21Controller) LoadConditionMask(*Context, string) int    { return 0 }
func (LOC21Controller) Exit(*Context, string, string) engine.Task { return nil }
func (LOC21Controller) SelectItem(ctx *Context, item int) engine.Task {
	actor := loc10ActorID(ctx)
	switch loc21SceneID(ctx.session.state.Scene) {
	case "S73":
		// Original Location 21 entry registers inventory item 0x17 directly
		// to callback 0x23 while S73 is active.
		if item == 0x17 {
			return loc21UseGloom(ctx, actor)
		}
	case "S74":
		// Original S74 registers inventory item 0x0b directly to callback
		// 0x2e; selecting the shovel invokes the switch interaction.
		if item == 0xb {
			return loc21UseSwitch(ctx, actor)
		}
	case "S80":
		// Original S80 registers item 9 -> callback 0x68 and item 0x0b ->
		// callback 0x69. These are direct inventory-selection callbacks, not
		// "arm item, then click a hotspot" handlers.
		if item == 9 {
			return loc21HackStone(ctx, actor)
		}
		if item == 0xb {
			return loc21TakeStone(ctx, actor)
		}
	}
	return nil
}

func (LOC21Controller) Enter(ctx *Context, scene, from string) engine.Task {
	scene = loc21SceneID(scene)
	from = loc21SceneID(from)
	loc21DormantScene(ctx, scene)
	actor := loc10ActorID(ctx)
	tasks := []engine.Task{ctx.PlayMusic("Loc21_DwarvesMine.wav"), ctx.ShowActor(actor)}

	switch scene {
	case "S71":
		tasks = append(tasks, ctx.EnableArea("S71_To74"), ctx.EnableArea("S71_Wall"), ctx.EnableArea("S71_To69"))
		if !ctx.HasItem(0xb) {
			tasks = append(tasks, loc21ShowStatic(ctx, "S71_Shovel", 0), ctx.MakeLayerClickable("S71_Shovel"), loc21PrioritizeAreas(ctx, "S71_Shovel"))
		}
		if from == "S74" {
			tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x55, 0xe4), ctx.WalkToFacingPerspective(actor, 0x61, 0x113, 0))
		} else {
			tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x276, 0x136), ctx.WalkToFacingPerspective(actor, 0x21d, 0x136, 2))
		}
	case "S72":
		tasks = append(tasks, ctx.EnableArea("S72_To74"), ctx.EnableArea("S72_Wall"), ctx.EnableArea("S72_To73"))
		if from == "S73" {
			tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x1d9, 0xe0), ctx.WalkToFacingPerspective(actor, 0x1b4, 0x110, 1))
		} else if from == "S74" {
			tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x82, 0xdc), ctx.WalkToFacingPerspective(actor, 0x91, 0x10b, 0))
		}
	case "S73":
		tasks = append(tasks,
			ctx.EnableArea("S73_To72"), ctx.EnableArea("S73_To75"), ctx.EnableArea("S73_Wall"),
			ctx.DisableArea("S73_Drill"), ctx.DisableArea("S73_Hebel"),
		)
		if loc21Original(ctx, loc21StateDrillLocation) == 1 {
			tasks = append(tasks, loc21ShowStatic(ctx, "S73_Drill", 0), ctx.MakeLayerClickable("S73_Drill"), loc21ShowStatic(ctx, "S73_Hebel", 0), ctx.MakeLayerClickable("S73_Hebel"), loc21PrioritizeAreas(ctx, "S73_Hebel", "S73_Drill"))
		}
		if from == "S72" {
			tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0xbd, 0xdc), ctx.WalkToFacingPerspective(actor, 0xba, 0x103, 0))
		} else if from == "S75" {
			tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x215, 0xe4), ctx.WalkToFacingPerspective(actor, 0x1de, 0x120, 1))
		}
	case "S74":
		tasks = append(tasks, ctx.EnableArea("S74_To71"), ctx.EnableArea("S74_To72"), ctx.EnableArea("S74_To75"), ctx.EnableArea("S74_To77"), ctx.EnableArea("S74_Wall"))
		leverFrame := 0
		if loc21Original(ctx, loc21StateSwitch) != 0 {
			leverFrame = 10
		}
		tasks = append(tasks, loc21ShowStatic(ctx, "S74_Hebel", leverFrame), ctx.MakeLayerClickable("S74_Hebel"), loc21PrioritizeAreas(ctx, "S74_Hebel"))
		switch from {
		case "S71":
			tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x26d, 0xd8), ctx.WalkToFacingPerspective(actor, 0x250, 0xed, 1))
		case "S72":
			tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0xd3, 0xb4), ctx.WalkToFacingPerspective(actor, 0xea, 0xcd, 0))
		case "S75":
			tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x1d2, 0xb0), ctx.WalkToFacingPerspective(actor, 0x1c1, 0xd0, 1))
		case "S77":
			tasks = append(tasks, ctx.PlaceActorPerspective(actor, 10, 0xe4), ctx.WalkToFacingPerspective(actor, 0x46, 0xf7, 7))
		}
	case "S75":
		tasks = append(tasks, ctx.EnableArea("S75_To74"), ctx.EnableArea("S75_To73"), ctx.EnableArea("S75_To76"), ctx.EnableArea("S75_Wall"))
		switch from {
		case "S73":
			tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x15c, 200), ctx.WalkToFacingPerspective(actor, 0x14f, 0x105, 0))
		case "S74":
			tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x49, 0xcc), ctx.WalkToFacingPerspective(actor, 0x59, 0x10b, 0))
		case "S76":
			tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x27b, 0x107), ctx.WalkToFacingPerspective(actor, 0x224, 0x10f, 2))
		}
	case "S76":
		tasks = append(tasks, loc21BindDrillSFX(ctx), loc21BindRubbleSFX(ctx))
		tasks = append(tasks,
			ctx.EnableArea("S76_To75"), ctx.EnableArea("S76_Wall"),
			ctx.DisableArea("S76_DriveLever"), ctx.DisableArea("S76_DrillLever"),
		)
		switch loc21Original(ctx, loc21StateRubble76) {
		case 1:
			tasks = append(tasks, loc21ShowStatic(ctx, "Gen_Schutt", 0), loc21ShowStatic(ctx, "Gen_Schutt2", 0))
		case 2:
			tasks = append(tasks, loc21ShowStatic(ctx, "Gen_Schutt", 0x14), loc21ShowStatic(ctx, "Gen_Schutt2", 0x15))
		case 3:
			tasks = append(tasks, loc21ShowStatic(ctx, "Gen_Schutt", 0x28), loc21ShowStatic(ctx, "Gen_Schutt2", 0x2b))
		}
		if loc21Original(ctx, loc21StateDrillLocation) == 2 {
			tasks = append(tasks, loc21ShowStatic(ctx, "Gen_DrillLeaves", 0), ctx.EnableArea("S76_DriveLever"), ctx.EnableArea("S76_DrillLever"), loc21PrioritizeAreas(ctx, "S76_DrillLever", "S76_DriveLever"))
		}
		if loc21Original(ctx, loc21StateTinder) != 0 {
			tasks = append(tasks, loc21ShowStatic(ctx, "S76_Zunder", 0), ctx.MakeLayerClickable("S76_Zunder"), loc21PrioritizeAreas(ctx, "S76_Zunder"))
		}
		tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0xd5, 0xd0), ctx.WalkToFacingPerspective(actor, 200, 0x104, 0))
	case "S77":
		tasks = append(tasks, ctx.EnableArea("S77_To74"), ctx.EnableArea("S77_To78"), ctx.EnableArea("S77_To80"), ctx.EnableArea("S77_Wall"))
		switch from {
		case "S74":
			tasks = append(tasks, ctx.PlaceActorPerspective(actor, 10, 0xe7), ctx.WalkToFacingPerspective(actor, 0x73, 0x118, 7))
		case "S78":
			tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x13e, 0xd0), ctx.WalkToFacingPerspective(actor, 0x145, 0xfb, 0))
		case "S80":
			tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x276, 0xd8), ctx.WalkToFacingPerspective(actor, 0x243, 0x10f, 1))
		}
	case "S78":
		tasks = append(tasks, ctx.EnableArea("S78_Wall"), ctx.EnableArea("S78_To77"), ctx.EnableArea("S78_To79"))
		if from == "S77" {
			tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x2f, 0xdd), ctx.WalkToFacingPerspective(actor, 0x6e, 0x107, 7))
		} else if from == "S79" {
			tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x270, 0x101), ctx.WalkToFacingPerspective(actor, 0x211, 0x115, 2))
		}
	case "S79":
		// The original controller only binds S79_To81 after the second
		// drilling/rubble pass has advanced 0x404c to 3. SZN areas are
		// enabled by default in this engine, so suppress it synchronously
		// before the first frame can expose the hidden-chamber hotspot.
		if ctx.session.scene != nil {
			if a, ok := ctx.session.scene.Areas["S79_To81"]; ok {
				a.Enabled = false
			}
		}
		tasks = append(tasks, loc21BindDrillSFX(ctx), loc21BindRubbleSFX(ctx))
		tasks = append(tasks,
			ctx.EnableArea("S79_To78"), ctx.EnableArea("S79_Wall"),
			ctx.DisableArea("S79_To81"),
			ctx.DisableArea("S79_DriveLever"), ctx.DisableArea("S79_DrillLever"),
		)
		switch loc21Original(ctx, loc21StateRubble79) {
		case 1:
			tasks = append(tasks, loc21ShowStatic(ctx, "Gen_Schutt", 0), loc21ShowStatic(ctx, "Gen_Schutt2", 0))
		case 2:
			tasks = append(tasks, loc21ShowStatic(ctx, "Gen_Schutt", 0x14), loc21ShowStatic(ctx, "Gen_Schutt2", 0x15))
		case 3:
			tasks = append(tasks, loc21ShowStatic(ctx, "Gen_Schutt", 0x28), loc21ShowStatic(ctx, "Gen_Schutt2", 0x2b), ctx.EnableArea("S79_To81"))
		}
		if loc21Original(ctx, loc21StateDrillLocation) == 3 {
			tasks = append(tasks, loc21ShowStatic(ctx, "Gen_DrillLeaves", 0), ctx.EnableArea("S79_DriveLever"), ctx.EnableArea("S79_DrillLever"), loc21PrioritizeAreas(ctx, "S79_DrillLever", "S79_DriveLever"))
		}
		if from == "S78" {
			tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0xcf, 0xd0), ctx.WalkToFacingPerspective(actor, 0xc4, 0x10b, 0))
		} else if from == "S81" {
			tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x231, 0xd8), ctx.WalkToFacingPerspective(actor, 0x1f7, 0x119, 1))
		}
	case "S80":
		tasks = append(tasks, ctx.SetFlag("loc21_s80_wall_seen", 0), ctx.EnableArea("S80_To77"), ctx.EnableArea("S80_Wall"))
		if !ctx.HasItem(9) {
			tasks = append(tasks, loc21ShowStatic(ctx, "S80_PickAxe", 0), ctx.MakeLayerClickable("S80_PickAxe"), loc21PrioritizeAreas(ctx, "S80_PickAxe"))
		}
		if loc21Original(ctx, loc21StateStone) == 2 {
			tasks = append(tasks, loc21ShowStatic(ctx, "S80_GloomOnFloor", 0), ctx.MakeLayerClickable("S80_GloomOnFloor"), loc21PrioritizeAreas(ctx, "S80_GloomOnFloor"))
		}
		tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x99, 0xdc), ctx.WalkToFacingPerspective(actor, 0xa6, 0x110, 0))
	}

	tasks = append(tasks, ctx.RunAmbient(&loc21DripAmbient{ctx: ctx}))
	return engine.Sequence(tasks...)
}

func (LOC21Controller) Click(ctx *Context, area string) engine.Task {
	actor := loc10ActorID(ctx)
	switch loc21SceneID(ctx.session.state.Scene) {
	case "S71":
		switch area {
		case "S71_To69":
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x276, 0x136, 6), ctx.ChangeLocation(19, "69"))
		case "S71_To74":
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x55, 0xe4, 4), ctx.ChangeScene("S74"))
		case "S71_Wall":
			return loc21Examine(ctx, actor, 0x14a, 0x12a, 4, "071_ROD_02")
		case "S71_Shovel":
			if ctx.HasItem(0xb) {
				return nil
			}
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xfb, 0x117, 5), ctx.HideActor(actor), ctx.HideLayer("S71_Shovel"), ctx.DisableArea("S71_Shovel"), ctx.AddItem(0xb), loc21PlayHide(ctx, "S71_RodTakesShovel"), ctx.ShowActor(actor), ctx.Say(actor, "071_ROD_01", "[071_ROD_01]"))
		}
	case "S72":
		switch area {
		case "S72_To73":
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x1d9, 0xe0, 4), ctx.ChangeScene("S73"))
		case "S72_To74":
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x82, 0xdc, 4), ctx.ChangeScene("S74"))
		case "S72_Wall":
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x11e, 0x123, 5), ctx.HideActor(actor), loc21PlayHide(ctx, "S72_RodFeelsWall"), ctx.ShowActor(actor), ctx.Say(actor, "072_ROD_01", "[072_ROD_01]"))
		}
	case "S73":
		switch area {
		case "S73_To72":
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xbd, 0xdc, 4), ctx.ChangeScene("S72"))
		case "S73_To75":
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x215, 0xe4, 4), ctx.ChangeScene("S75"))
		case "S73_Wall":
			return loc21Examine(ctx, actor, 0x143, 0x124, 4, "073_ROD_01")
		case "S73_Drill":
			line := "073_ROD_03"
			if loc21Original(ctx, loc21StateGloomUsed) != 0 {
				line = "073_ROD_02"
			}
			return loc21Examine(ctx, actor, 0x202, 0x147, 3, line)
		case "S73_Hebel":
			return loc21UseDrillLever73(ctx, actor)
		}
	case "S74":
		switch area {
		case "S74_To71":
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x26d, 0xd8, 5), ctx.ChangeScene("S71"))
		case "S74_To72":
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xd3, 0xb4, 4), ctx.ChangeScene("S72"))
		case "S74_To75":
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x1d2, 0xb0, 5), ctx.ChangeScene("S75"))
		case "S74_To77":
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 10, 0xe4, 2), ctx.ChangeScene("S77"))
		case "S74_Wall":
			return loc21Examine(ctx, actor, 0x159, 0xdc, 4, "074_ROD_02")
		case "S74_Hebel":
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xf0, 0xe7, 2), ctx.HideActor(actor), loc21PlayHide(ctx, "S74_RodLooksWeiche"), ctx.ShowActor(actor), ctx.Say(actor, "074_ROD_01", "[074_ROD_01]"))
		}
	case "S75":
		switch area {
		case "S75_To73":
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x15c, 200, 4), ctx.ChangeScene("S73"))
		case "S75_To74":
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x49, 0xcc, 4), ctx.ChangeScene("S74"))
		case "S75_To76":
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x27b, 0x107, 6), ctx.ChangeScene("S76"))
		case "S75_Wall":
			return loc21Examine(ctx, actor, 0xcf, 0x10c, 4, "075_ROD_01")
		}
	case "S76":
		switch area {
		case "S76_To75":
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xd5, 0xd0, 4), ctx.ChangeScene("S75"))
		case "S76_Wall":
			return loc21Examine(ctx, actor, 0x157, 0x134, 4, "076_ROD_01")
		case "S76_DriveLever":
			return loc21DriveLever76(ctx, actor)
		case "S76_DrillLever":
			return loc21DrillLever76(ctx, actor)
		case "S76_Zunder":
			return loc21TakeTinder(ctx, actor)
		}
	case "S77":
		switch area {
		case "S77_To74":
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 10, 0xe7, 2), ctx.ChangeScene("S74"))
		case "S77_To78":
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x13e, 0xd0, 4), ctx.ChangeScene("S78"))
		case "S77_To80":
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x276, 0xd8, 4), ctx.ChangeScene("S80"))
		case "S77_Wall":
			return loc21Examine(ctx, actor, 0x1c5, 0x10c, 4, "077_ROD_01")
		}
	case "S78":
		switch area {
		case "S78_To77":
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x2f, 0xdd, 2), ctx.ChangeScene("S77"))
		case "S78_To79":
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x270, 0x101, 6), ctx.ChangeScene("S79"))
		case "S78_Wall":
			return loc21Examine(ctx, actor, 0x142, 0x114, 4, "078_ROD_01")
		}
	case "S79":
		switch area {
		case "S79_To78":
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xcf, 0xd0, 4), ctx.ChangeScene("S78"))
		case "S79_To81":
			if loc21Original(ctx, loc21StateRubble79) != 3 {
				return nil
			}
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x231, 0xd8, 4), ctx.ChangeLocation(22, "81"))
		case "S79_Wall":
			return loc21Examine(ctx, actor, 0x14b, 0x124, 4, "079_ROD_01")
		case "S79_DriveLever":
			return loc21DriveLever79(ctx, actor)
		case "S79_DrillLever":
			return loc21DrillLever79(ctx, actor)
		}
	case "S80":
		switch area {
		case "S80_To77":
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x99, 0xdc, 4), ctx.ChangeScene("S77"))
		case "S80_Wall":
			return loc21Wall80(ctx, actor)
		case "S80_PickAxe":
			return loc21TakePickAxe(ctx, actor)
		case "S80_GloomOnFloor":
			return loc21TryStone(ctx, actor)
		}
	}
	return nil
}

func (LOC21Controller) UseItem(ctx *Context, item int, area string) engine.Task {
	actor := loc10ActorID(ctx)
	switch loc21SceneID(ctx.session.state.Scene) {
	case "S73":
		if item == 0x17 {
			return loc21UseGloom(ctx, actor)
		}
	case "S74":
		if item == 0xb {
			return loc21UseSwitch(ctx, actor)
		}
	case "S80":
		// The original S80 controller registers inventory item 9 directly
		// to callback 0x68 and item 0x0b directly to callback 0x69. The
		// callbacks gate on the story state themselves; they are not tied
		// to a particular hotspot id.
		if item == 9 {
			return loc21HackStone(ctx, actor)
		}
		if item == 0xb {
			return loc21TakeStone(ctx, actor)
		}
	}
	return nil
}

func loc21UseGloom(ctx *Context, actor string) engine.Task {
	if loc21Original(ctx, loc21StateGloomUsed) != 0 {
		return nil
	}
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0x1b9, 0x155, 3), ctx.HideActor(actor), ctx.RemoveItem(0x17),
		ctx.ShowLayer("S73_RodUsesGloom"), ctx.PlayLayerFrames("S73_RodUsesGloom", 0, 0x29), ctx.PlaySFXVolume("Sfx_WaterSplash.wav", 60), ctx.PlayLayerFrames("S73_RodUsesGloom", 0x2a, -1), ctx.HideLayer("S73_RodUsesGloom"),
		engine.Parallel(
			engine.Sequence(ctx.ShowLayer("Gen_SmallSteam"), ctx.PlayLayerFrames("Gen_SmallSteam", 0, 2), ctx.PlaySFX("Sfx_DragonWings.wav"), ctx.PlayLayerFrames("Gen_SmallSteam", 3, -1), ctx.HideLayer("Gen_SmallSteam")),
			engine.Sequence(loc21SetOriginal(ctx, loc21StateGloomUsed, 1), ctx.ShowActor(actor), ctx.Say(actor, "073_ROD_02", "[073_ROD_02]")),
		),
	)
}

func loc21UseDrillLever73(ctx *Context, actor string) engine.Task {
	tasks := []engine.Task{ctx.WalkToFacingPerspective(actor, 0x1cf, 0x158, 4), ctx.HideActor(actor), loc21PlayGrabAuthored(ctx, 11), ctx.ShowActor(actor)}
	if loc21Original(ctx, loc21StateGloomUsed) == 0 {
		return engine.Sequence(append(tasks, ctx.Say(actor, "073_ROD_05", "[073_ROD_05]"))...)
	}
	tasks = append(tasks, ctx.HideLayer("S73_Hebel"), ctx.DisableArea("S73_Hebel"), ctx.DisableArea("S73_Drill"))
	if loc21Original(ctx, loc21StateSwitch) == 0 {
		tasks = append(tasks, loc21SetOriginal(ctx, loc21StateDrillLocation, 2), loc21DrillLeavesForward(ctx, "S73_Drill"), ctx.Wait(3))
	} else {
		tasks = append(tasks, ctx.ShowLayer("S73_Drill"), ctx.PlayLayer("S73_Drill"), ctx.PlayLayerFrames("S73_Drill", 99, 0), loc21ShowStatic(ctx, "S73_Hebel", 0), ctx.MakeLayerClickable("S73_Hebel"), ctx.MakeLayerClickable("S73_Drill"), loc21PrioritizeAreas(ctx, "S73_Hebel", "S73_Drill"), ctx.Say(actor, "076_ROD_04", "[076_ROD_04]"))
	}
	return engine.Sequence(tasks...)
}

func loc21UseSwitch(ctx *Context, actor string) engine.Task {
	tasks := []engine.Task{ctx.WalkToFacingPerspective(actor, 0xc2, 0xec, 5), ctx.HideActor(actor), ctx.ShowLayer("S74_RodUsesWeiche")}
	if loc21Original(ctx, loc21StateSwitch) == 0 {
		tasks = append(tasks,
			ctx.PlayLayerFrames("S74_RodUsesWeiche", 0, 0x26),
			engine.Parallel(
				ctx.PlayLayerFrames("S74_Hebel", 0, 10),
				engine.Sequence(ctx.PlaySFX("Sfx_Winding.wav"), ctx.PlayLayerFrames("S74_RodUsesWeiche", 0x27, -1)),
			),
			loc21SetOriginal(ctx, loc21StateSwitch, 1),
		)
	} else {
		tasks = append(tasks,
			ctx.PlayLayerFrames("S74_RodUsesWeiche", 0x4b, 0x32),
			engine.Parallel(
				ctx.PlayLayerFrames("S74_Hebel", 10, 0),
				engine.Sequence(ctx.PlaySFX("Sfx_Winding.wav"), ctx.PlayLayerFrames("S74_RodUsesWeiche", 0x31, 0)),
			),
			loc21SetOriginal(ctx, loc21StateSwitch, 0),
		)
	}
	tasks = append(tasks, ctx.HideLayer("S74_RodUsesWeiche"), ctx.ShowActor(actor))
	if loc21Original(ctx, loc21StateSwitchTalk) == 1 {
		tasks = append(tasks, ctx.Say(actor, "074_ROD_03", "[074_ROD_03]"), loc21SetOriginal(ctx, loc21StateSwitchTalk, 2))
	} else {
		tasks = append(tasks, ctx.Say(actor, "074_ROD_04", "[074_ROD_04]"))
	}
	return engine.Sequence(tasks...)
}

func loc21DriveLever76(ctx *Context, actor string) engine.Task {
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0x17f, 0x144, 4), ctx.HideActor(actor), loc21PlayGrab(ctx, 336, 156, 18, 106, 8), ctx.ShowActor(actor),
		ctx.HideLayer("Gen_DrillDrills"), ctx.DisableArea("S76_DriveLever"), ctx.DisableArea("S76_DrillLever"),
		loc21DrillLeavesForward(ctx, "Gen_DrillLeaves"),
		loc21SetOriginal(ctx, loc21StateDrillLocation, map[bool]int{true: 3, false: 1}[loc21Original(ctx, loc21StateSwitch) != 0]),
		ctx.RunAmbient(ctx.WalkToFacingPerspective(actor, 0x17f, 0x15e, 4)),
	)
}

func loc21DrillLever76(ctx *Context, actor string) engine.Task {
	tasks := []engine.Task{}
	if loc21Original(ctx, loc21StateRubble76) == 1 {
		tasks = append(tasks, ctx.Say(actor, "076_ROD_02", "[076_ROD_02]"))
	}
	tasks = append(tasks, ctx.WalkToFacingPerspective(actor, 0x1a1, 0x15a, 4), ctx.HideActor(actor), loc21PlayGrab(ctx, 370, 158, 9, 119, 8), ctx.ShowActor(actor), ctx.HideLayer("Gen_DrillLeaves"), loc21StartDrill(ctx), ctx.RunAmbient(ctx.WalkToFacingPerspective(actor, 0x104, 0x163, 6)), ctx.Wait(3))
	switch loc21Original(ctx, loc21StateRubble76) {
	case 1:
		tasks = append(tasks, ctx.PlayLayerFrames("Gen_Schutt", 0, 0x13), ctx.PlayLayerFrames("Gen_Schutt2", 0, 0x14), ctx.Wait(1), loc21StopDrill(ctx), loc21SetOriginal(ctx, loc21StateRubble76, 2), ctx.Say(actor, "076_ROD_04", "[076_ROD_04]"))
	case 2:
		tasks = append(tasks, ctx.PlayLayerFrames("Gen_Schutt", 0x14, -1), ctx.PlayLayerFrames("Gen_Schutt2", 0x15, 0x26), loc21ShowStatic(ctx, "S76_Zunder", 0), ctx.MakeLayerClickable("S76_Zunder"), loc21PrioritizeAreas(ctx, "S76_Zunder"), ctx.PlayLayerFrames("Gen_Schutt2", 0x27, -1), loc21SetOriginal(ctx, loc21StateTinder, 1), ctx.Wait(1), loc21StopDrill(ctx), loc21SetOriginal(ctx, loc21StateRubble76, 3), ctx.Say(actor, "076_ROD_03", "[076_ROD_03]"))
	case 3:
		tasks = append(tasks, loc21StopDrill(ctx), ctx.Say(actor, "076_ROD_04", "[076_ROD_04]"))
	}
	return engine.Sequence(tasks...)
}

func loc21TakeTinder(ctx *Context, actor string) engine.Task {
	if loc21Original(ctx, loc21StateTinder) == 0 {
		return nil
	}
	return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x139, 0x134, 4), ctx.HideActor(actor), ctx.PlayLayerFrames("Gen_RodPickUp", 0, 7), ctx.HideLayer("S76_Zunder"), ctx.DisableArea("S76_Zunder"), ctx.AddItem(0x1a), loc21SetOriginal(ctx, loc21StateTinder, 0), ctx.PlayLayerFrames("Gen_RodPickUp", 8, -1), ctx.HideLayer("Gen_RodPickUp"), ctx.ShowActor(actor), ctx.Say(actor, "069_ROD_26", "[069_ROD_26]"))
}

func loc21DriveLever79(ctx *Context, actor string) engine.Task {
	tasks := []engine.Task{ctx.WalkToFacingPerspective(actor, 0x17f, 0x144, 4), ctx.HideActor(actor), loc21PlayGrab(ctx, 336, 156, 18, 106, 8), ctx.ShowActor(actor), ctx.HideLayer("Gen_DrillDrills")}
	if loc21Original(ctx, loc21StateSwitch) != 0 {
		tasks = append(tasks, ctx.DisableArea("S79_DriveLever"), ctx.DisableArea("S79_DrillLever"), loc21DrillLeavesForward(ctx, "Gen_DrillLeaves"), loc21SetOriginal(ctx, loc21StateDrillLocation, 2), ctx.RunAmbient(ctx.WalkToFacingPerspective(actor, 0x17f, 0x15e, 4)))
	} else {
		tasks = append(tasks, ctx.RunAmbient(ctx.WalkToFacingPerspective(actor, 0x17f, 0x15e, 4)), ctx.PlayLayerFrames("Gen_DrillLeaves", 0, 100), ctx.Wait(3), ctx.PlayLayerFrames("Gen_DrillLeaves", 100, 0), loc21ShowStatic(ctx, "Gen_DrillLeaves", 0), ctx.Say(actor, "079_ROD_06", "[079_ROD_06]"))
	}
	return engine.Sequence(tasks...)
}

func loc21DrillLever79(ctx *Context, actor string) engine.Task {
	tasks := []engine.Task{}
	if loc21Original(ctx, loc21StateRubble79) == 1 {
		tasks = append(tasks, ctx.Say(actor, "079_ROD_02", "[079_ROD_02]"))
	}
	tasks = append(tasks, ctx.WalkToFacingPerspective(actor, 0x1a1, 0x15a, 4), ctx.HideActor(actor), loc21PlayGrab(ctx, 370, 158, 9, 119, 8), ctx.ShowActor(actor), ctx.HideLayer("Gen_DrillLeaves"), loc21StartDrill(ctx), ctx.RunAmbient(ctx.WalkToFacingPerspective(actor, 0x104, 0x163, 6)), ctx.Wait(3))
	switch loc21Original(ctx, loc21StateRubble79) {
	case 1:
		tasks = append(tasks, ctx.PlayLayerFrames("Gen_Schutt", 0, 0x13), ctx.PlayLayerFrames("Gen_Schutt2", 0, 0x14), ctx.Wait(1), loc21StopDrill(ctx), ctx.Say(actor, "079_ROD_03", "[079_ROD_03]"), loc21SetOriginal(ctx, loc21StateRubble79, 2))
	case 2:
		tasks = append(tasks, ctx.PlayLayerFrames("Gen_Schutt", 0x14, 0x28), ctx.PlayLayerFrames("Gen_Schutt2", 0x15, 0x2b), ctx.Wait(1), loc21StopDrill(ctx), ctx.Say(actor, "079_ROD_05", "[079_ROD_05]"), loc21SetOriginal(ctx, loc21StateRubble79, 3), ctx.EnableArea("S79_To81"))
	case 3:
		tasks = append(tasks, loc21StopDrill(ctx), ctx.Say(actor, "079_ROD_06", "[079_ROD_06]"))
	}
	return engine.Sequence(tasks...)
}

func loc21Wall80(ctx *Context, actor string) engine.Task {
	if loc21Original(ctx, loc21StateStone) == 1 && ctx.GetFlag("loc21_s80_wall_seen") != 0 {
		return engine.Sequence(loc21Examine(ctx, actor, 0x16f, 0x133, 6, "080_ROD_02"), loc21SetOriginal(ctx, loc21StateWallPrimed, 1))
	}
	return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x118, 300, 6), ctx.Wait(0.5), ctx.SetActorOrientation(actor, 2), ctx.Wait(0.5), ctx.SetFlag("loc21_s80_wall_seen", 1), ctx.Say(actor, "080_ROD_01", "[080_ROD_01]"))
}

func loc21TakePickAxe(ctx *Context, actor string) engine.Task {
	if ctx.HasItem(9) {
		return nil
	}
	return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x15a, 300, 5), ctx.HideActor(actor), ctx.HideLayer("S80_PickAxe"), ctx.DisableArea("S80_PickAxe"), ctx.AddItem(9), loc21PlayHide(ctx, "S80_RodTakesPickAxe"), ctx.ShowActor(actor), ctx.Say(actor, "080_ROD_07", "[080_ROD_07]"))
}

func loc21TryStone(ctx *Context, actor string) engine.Task {
	switch loc21Original(ctx, loc21StateTryStone) {
	case 1:
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x16f, 0x133, 6), ctx.HideActor(actor), ctx.ShowLayer("S80_RodTriesTake"), ctx.PlayLayerFrames("S80_RodTriesTake", 0, 9), ctx.PlaySFX("Sfx_Sizzle.wav"), ctx.PlayLayerFrames("S80_RodTriesTake", 10, -1), ctx.HideLayer("S80_RodTriesTake"), loc21SetOriginal(ctx, loc21StateTryStone, 2), ctx.ShowActor(actor), ctx.Say(actor, "080_ROD_03", "[080_ROD_03]"))
	case 2:
		return loc21Examine(ctx, actor, 0x16f, 0x133, 6, "080_ROD_04")
	}
	return nil
}

func loc21HackStone(ctx *Context, actor string) engine.Task {
	if loc21Original(ctx, loc21StateStone) != 1 || loc21Original(ctx, loc21StateWallPrimed) == 0 {
		return nil
	}
	return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x194, 0x12e, 6), ctx.HideActor(actor), ctx.ShowLayer("S80_RodHacks"), ctx.PlayLayerFrames("S80_RodHacks", 0, 0x24), ctx.PlaySFX("Sfx_Work_Hack2.wav"), ctx.PlayLayerFrames("S80_RodHacks", 0x25, -1), ctx.HideLayer("S80_RodHacks"), loc21ShowStatic(ctx, "S80_GloomOnFloor", 0), ctx.MakeLayerClickable("S80_GloomOnFloor"), loc21PrioritizeAreas(ctx, "S80_GloomOnFloor"), loc21SetOriginal(ctx, loc21StateStone, 2), ctx.ShowActor(actor), ctx.Say(actor, "080_ROD_06", "[080_ROD_06]"))
}

func loc21TakeStone(ctx *Context, actor string) engine.Task {
	if loc21Original(ctx, loc21StateStone) != 2 {
		return nil
	}
	return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x16f, 0x133, 6), ctx.HideActor(actor), ctx.ShowLayer("S80_RodTakesStone"), ctx.PlayLayerFrames("S80_RodTakesStone", 0, 0x2c), ctx.HideLayer("S80_GloomOnFloor"), ctx.DisableArea("S80_GloomOnFloor"), loc21SetOriginal(ctx, loc21StateStone, 3), ctx.PlayLayerFrames("S80_RodTakesStone", 0x2d, -1), ctx.HideLayer("S80_RodTakesStone"), ctx.AddItem(0x17), ctx.ShowActor(actor), ctx.Say(actor, "080_ROD_05", "[080_ROD_05]"))
}

type loc21DripAmbient struct {
	ctx  *Context
	wait float64
}

func (t *loc21DripAmbient) Update(dt float64) bool {
	if t.ctx.session.state.Location != 21 {
		return true
	}
	if t.wait <= 0 {
		t.wait = float64(5 + rand.IntN(11))
	}
	t.wait -= dt
	if t.wait > 0 {
		return false
	}
	_ = t.ctx.PlaySFX("Sfx_Drip.wav").Update(0)
	t.wait = 0
	return false
}
