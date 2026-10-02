package game

import (
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/wok/rent-a-hero/internal/audio"
	"github.com/wok/rent-a-hero/internal/engine"
)

const loc25StateSequence = 0x4090

type LOC25Controller struct{}

func loc25SceneID(scene string) int {
	s := strings.ToUpper(strings.TrimSpace(scene))
	s = strings.TrimPrefix(s, "S")
	n, _ := strconv.Atoi(s)
	if n == 104 {
		return 105
	}
	return n
}

func loc25Original(ctx *Context, off int) int {
	return getOriginalFlag(ctx.session.state.OriginalState, off)
}

func loc25SetOriginal(ctx *Context, off, value int) engine.Task {
	return engine.Immediate(func() { putOriginalFlag(ctx.session.state.OriginalState, off, value) })
}

func loc25EnsureLayers(ctx *Context) {
	ids := []string{
		"S105_Rod", "S105_DraHead", "S105_Jas", "S105_MadHead", "S105_Ran", "S105_SabHead",
		"S105_RodMorph", "S105_WizAttack", "S105_SabAttack", "S105_DraAttack", "S105_MadAttack",
		"S105_LouAttack", "S105_SanAttack", "S105_CynAttack", "S105_WizWalk", "S105_WizExplodes",
		"S105_JasHead", "S105_LouHead", "S105_RanHead", "S105_CynHead", "S105_SanHead",
	}
	for _, id := range ids {
		_, _, _ = ctx.ensureAssetLayer(id)
		if layer, ok := ctx.layer(id); ok {
			layer.Visible = false
			layer.Playing = false
			layer.TaskDriven = false
			layer.Accumulator = 0
		}
	}
}

func loc25BindFrameSFX(ctx *Context) {
	events := map[string]map[int]string{
		"S105_DraAttack":   {1: "Sfx_DragonBreath.wav", 0x10: "Sfx_DragonFire.wav"},
		"S105_WizExplodes": {5: "Sfx_Explosion_Misc2.wav"},
		"S105_WizAttack":   {6: "Sfx_Lazer3.wav"},
		"S105_SabAttack":   {0xb: "Sfx_Woosh.wav", 0xd: "Sfx_Knock_Wet.wav"},
		"S105_MadAttack":   {9: "Sfx_Woosh.wav", 10: "Sfx_Knock_Wet.wav"},
		"S105_LouAttack":   {6: "Sfx_WaterSplash.wav"},
		"S105_SanAttack":   {0xf: "Sfx_SignalHorn_Modern.wav"},
		"S105_CynAttack":   {8: "Sfx_Woosh.wav", 10: "Sfx_Knock_Wet.wav"},
	}
	for id, frames := range events {
		layer, ok := ctx.layer(id)
		if !ok || layer == nil {
			continue
		}
		for frame, name := range frames {
			if len(layer.FrameEvents[frame]) != 0 {
				continue
			}
			name := name
			layer.AddFrameEvent(frame, func() { _ = ctx.PlaySFX(name).Update(0) })
		}
	}
}

func loc25SetLayerXY(ctx *Context, id string, x, y int) engine.Task {
	return engine.Immediate(func() {
		if layer, ok := ctx.layer(id); ok {
			layer.X = x
			layer.Y = y
		}
	})
}

func loc25SetLayerZoom(ctx *Context, id string, zoom int) engine.Task {
	return engine.Immediate(func() {
		if layer, ok := ctx.layer(id); ok {
			layer.Zoom = zoom
		}
	})
}

func loc25SetLayerXYZoom(ctx *Context, id string, x, y, zoom int) engine.Task {
	return engine.Immediate(func() {
		if layer, ok := ctx.layer(id); ok {
			layer.X = x
			layer.Y = y
			layer.Zoom = zoom
		}
	})
}

type loc25AttackDrawState struct {
	z     int
	index int
	valid bool
}

func loc25PlaceAttackAboveMage(ctx *Context, id string, state *loc25AttackDrawState) engine.Task {
	return engine.Immediate(func() {
		attack, ok := ctx.layer(id)
		if !ok || attack == nil || ctx.session.scene == nil {
			return
		}
		order := ctx.session.scene.LayerOrder
		for i, name := range order {
			if name == id {
				state.z = attack.Z
				state.index = i
				state.valid = true
				break
			}
		}
		if mage, ok := ctx.layer("S105_WizWalk"); ok && mage != nil {
			attack.Z = mage.Z
		}
		for i, name := range order {
			if name != id {
				continue
			}
			copy(order[i:], order[i+1:])
			order[len(order)-1] = id
			break
		}
	})
}

func loc25RestoreAttackDrawState(ctx *Context, id string, state *loc25AttackDrawState) engine.Task {
	return engine.Immediate(func() {
		if !state.valid || ctx.session.scene == nil {
			return
		}
		if attack, ok := ctx.layer(id); ok && attack != nil {
			attack.Z = state.z
		}
		order := ctx.session.scene.LayerOrder
		current := -1
		for i, name := range order {
			if name == id {
				current = i
				break
			}
		}
		if current < 0 {
			return
		}
		copy(order[current:], order[current+1:])
		order = order[:len(order)-1]
		index := state.index
		if index < 0 {
			index = 0
		}
		if index > len(order) {
			index = len(order)
		}
		order = append(order, "")
		copy(order[index+1:], order[index:])
		order[index] = id
		ctx.session.scene.LayerOrder = order
	})
}

func loc25LoopLayer(ctx *Context, id string) engine.Task {
	return engine.Immediate(func() {
		if layer, ok := ctx.layer(id); ok {
			layer.Visible = true
			layer.Enabled = true
			layer.Mode = engine.AnimLoop
			layer.Playing = true
			layer.TaskDriven = false
			layer.Accumulator = 0
		}
	})
}

func loc25LayerStartSFX(ctx *Context, id string) engine.Task {
	if id == "S105_RodMorph" {
		return ctx.PlaySFX("Sfx_Explosion_Bass.wav")
	}
	return engine.Immediate(func() {})
}

func loc25PlayLayerOnceAutoHide(ctx *Context, id string) engine.Task {
	return engine.Sequence(
		loc25LayerStartSFX(ctx, id),
		ctx.PlayLayerFrames(id, 0, -1),
		ctx.HideLayer(id),
	)
}

func loc25PlayLayerFramesAutoHide(ctx *Context, id string, from, to int) engine.Task {
	return engine.Sequence(
		ctx.PlayLayerFrames(id, from, to),
		ctx.HideLayer(id),
	)
}

func loc25StartLayerOnceAutoHide(ctx *Context, id string) engine.Task {
	return engine.Sequence(
		loc25LayerStartSFX(ctx, id),
		engine.Immediate(func() {
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
			layer.TaskDriven = true
			ctx.session.ambientTasks = append(ctx.session.ambientTasks, engine.Sequence(
				&layerRangeTask{layer: layer, target: -1, freeze: true},
				ctx.HideLayer(id),
			))
		}),
	)
}

type loc25MoveLayerTask struct {
	ctx                  *Context
	id                   string
	x, y, z, zoom, steps int
	started              bool
	elapsed              float64
	sx, sy, sz, szoom    int
}

func loc25MoveLayer(ctx *Context, id string, x, y, z, zoom, steps int) engine.Task {
	return &loc25MoveLayerTask{ctx: ctx, id: id, x: x, y: y, z: z, zoom: zoom, steps: steps}
}

func (t *loc25MoveLayerTask) Update(dt float64) bool {
	layer, ok := t.ctx.layer(t.id)
	if !ok || layer == nil {
		return true
	}
	if !t.started {
		t.started = true
		t.sx, t.sy, t.sz, t.szoom = layer.X, layer.Y, layer.Z, layer.Zoom
		layer.Visible = true
		layer.Enabled = true
	}
	if t.steps <= 0 {
		layer.X, layer.Y, layer.Z, layer.Zoom = t.x, t.y, t.z, t.zoom
		return true
	}
	t.elapsed += dt * 10
	p := t.elapsed / float64(t.steps)
	if p > 1 {
		p = 1
	}
	layer.X = t.sx + int(float64(t.x-t.sx)*p)
	layer.Y = t.sy + int(float64(t.y-t.sy)*p)
	layer.Z = t.sz + int(float64(t.z-t.sz)*p)
	layer.Zoom = t.szoom + int(float64(t.zoom-t.szoom)*p)
	return p >= 1
}

func loc25StartEndTime(ctx *Context) engine.Task {
	sound, err := engine.LoadSound(ctx.session.idx, "Sfx_EndTimeFeeling.wav")
	if err != nil {
		return engine.Immediate(func() {})
	}
	return engine.Immediate(func() {
		ctx.session.audioEngine.PlayLoopingWithVolume(sound, audio.CategorySFX, loc09DirectSoundGain(20))
	})
}

func loc25RevealHeads(ctx *Context) engine.Task {
	heads := []struct {
		id string
		z  int
	}{
		{"S105_DraHead", 0xb},
		{"S105_MadHead", 0xc},
		{"S105_SabHead", 0xd},
		{"S105_JasHead", 0xe},
		{"S105_LouHead", 0xf},
		{"S105_RanHead", 0x10},
		{"S105_CynHead", 0x11},
		{"S105_SanHead", 0x12},
	}
	tasks := make([]engine.Task, 0, len(heads)*4+1)
	for _, head := range heads {
		head := head
		tasks = append(tasks,
			ctx.ShowLayer(head.id),
			engine.Wait(0.1),
			ctx.SetLayerZ(head.id, head.z),
			ctx.MakeLayerClickable(head.id),
		)
	}
	tasks = append(tasks, loc25SetOriginal(ctx, loc25StateSequence, 1))
	return engine.Sequence(tasks...)
}

func loc25InitialArrival(ctx *Context) engine.Task {
	tasks := []engine.Task{
		ctx.HideLayer("S105_WizWalk"),
		ctx.PlayLayerFrames("S105_WizAttack", 0, 5),
		loc25StartLayerOnceAutoHide(ctx, "S105_RodMorph"),
		loc25PlayLayerFramesAutoHide(ctx, "S105_WizAttack", 6, -1),
		loc25LoopLayer(ctx, "S105_WizWalk"),
		ctx.MakeLayerClickable("S105_WizWalk"),
		ctx.PlayVoiceover("105_MEG_01", "[105_MEG_01]"),
	}
	if loc25Original(ctx, loc25StateSequence) == 0 {
		tasks = append(tasks, ctx.PlaySFX("Sfx_MobileUp.wav"), loc25RevealHeads(ctx))
	}
	return engine.Sequence(tasks...)
}

func loc25ResetWizard(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.PlaySFX("Sfx_DangerStrings.wav"),
		loc25SetLayerXYZoom(ctx, "S105_WizWalk", 0x172, -0xa7, 0x50),
		loc25LoopLayer(ctx, "S105_WizWalk"),
		loc25MoveLayer(ctx, "S105_WizWalk", 0x165, 0x2a, 0, 100, 0x14),
		ctx.MakeLayerClickable("S105_WizWalk"),
	)
}

func loc25Finale(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.PlaySFX("Sfx_Gong.wav"),
		engine.Wait(3),
		ctx.PlaySFX("Sfx_MobileDown.wav"),
		ctx.HideLayer("S105_DraHead"), engine.Wait(0.1),
		ctx.HideLayer("S105_MadHead"), engine.Wait(0.1),
		ctx.HideLayer("S105_SabHead"), engine.Wait(0.1),
		ctx.HideLayer("S105_JasHead"), engine.Wait(0.1),
		ctx.HideLayer("S105_LouHead"), engine.Wait(0.1),
		ctx.HideLayer("S105_RanHead"), engine.Wait(0.1),
		ctx.HideLayer("S105_CynHead"), engine.Wait(0.1),
		ctx.HideLayer("S105_SanHead"),
		engine.Wait(1),
		ctx.HideLayer("S105_Rod"),
		loc25PlayLayerOnceAutoHide(ctx, "S105_RodMorph"),
		engine.Wait(2),
		ctx.ChangeLocation(41, "112"),
	)
}

func loc25WrongAttack(ctx *Context, attack string) engine.Task {
	return engine.Sequence(
		ctx.HideLayer("S105_WizWalk"),
		ctx.PlayLayerFrames("S105_WizAttack", 0, 5),
		loc25StartLayerOnceAutoHide(ctx, "S105_RodMorph"),
		ctx.HideLayer(attack),
		ctx.ShowLayer("S105_Rod"),
		loc25PlayLayerFramesAutoHide(ctx, "S105_WizAttack", 6, -1),
		loc25LoopLayer(ctx, "S105_WizWalk"),
		ctx.MakeLayerClickable("S105_WizWalk"),
		loc25SetOriginal(ctx, loc25StateSequence, 1),
	)
}

func loc25Attack(ctx *Context, expected int, attack string, dragon bool) engine.Task {
	state := loc25Original(ctx, loc25StateSequence)
	if state != expected {
		return engine.Sequence(
			ctx.ShowLayer(attack),
			func() engine.Task {
				if dragon {
					return engine.Sequence(
						loc25SetLayerXYZoom(ctx, attack, 0xb9, 0xa9, 0x19),
						ctx.RunAmbient(loc25MoveLayer(ctx, attack, 1, 1, 0, 100, 10)),
					)
				}
				return engine.Immediate(func() {})
			}(),
			ctx.HideLayer("S105_Rod"),
			loc25PlayLayerOnceAutoHide(ctx, "S105_RodMorph"),
			func() engine.Task {
				if dragon {
					return loc25SetLayerXY(ctx, attack, 0, 0)
				}
				return engine.Immediate(func() {})
			}(),
			loc25WrongAttack(ctx, attack),
		)
	}

	drawState := &loc25AttackDrawState{}
	tasks := []engine.Task{ctx.ShowLayer(attack), loc25PlaceAttackAboveMage(ctx, attack, drawState)}
	if dragon {
		tasks = append(tasks,
			loc25SetLayerXYZoom(ctx, attack, 0xb9, 0xa9, 0x19),
			ctx.RunAmbient(loc25MoveLayer(ctx, attack, 1, 1, 0, 100, 10)),
		)
	}
	tasks = append(tasks,
		ctx.HideLayer("S105_Rod"),
		loc25PlayLayerOnceAutoHide(ctx, "S105_RodMorph"),
	)
	if dragon {
		tasks = append(tasks, loc25SetLayerXY(ctx, attack, 0, 0))
	}
	tasks = append(tasks,
		loc25SetLayerXY(ctx, "S105_WizWalk", 0x165, 0x2a),
		loc25SetLayerZoom(ctx, "S105_WizExplodes", 100),
	)
	if !dragon {
		tasks = append(tasks, loc25MoveLayer(ctx, "S105_WizWalk", 0xf7, 0x17, 0, 150, 10))
	}
	tasks = append(tasks,
		ctx.PlayLayerFrames(attack, 0, expected-1),
		ctx.PlayLayerFrames(attack, expected, -1),
		ctx.HideLayer("S105_WizWalk"),
	)
	if dragon {
		tasks = append(tasks, loc25SetLayerXYZoom(ctx, "S105_WizExplodes", 0x10f, -0x15, 100))
	} else {
		tasks = append(tasks, loc25SetLayerXYZoom(ctx, "S105_WizExplodes", 0x76, -0x49, 0x96))
	}
	tasks = append(tasks,
		loc25PlayLayerOnceAutoHide(ctx, "S105_WizExplodes"),
		ctx.HideLayer(attack),
		loc25RestoreAttackDrawState(ctx, attack, drawState),
		ctx.ShowLayer("S105_Rod"),
		loc25StartLayerOnceAutoHide(ctx, "S105_RodMorph"),
	)
	if state < 6 {
		tasks = append(tasks, loc25ResetWizard(ctx))
	} else {
		tasks = append(tasks, loc25Finale(ctx))
	}
	tasks = append(tasks, loc25SetOriginal(ctx, loc25StateSequence, state+1))
	return engine.Sequence(tasks...)
}

func (LOC25Controller) Enter(ctx *Context, scene string, from string) engine.Task {
	if loc25SceneID(scene) != 105 {
		return nil
	}
	loc25EnsureLayers(ctx)
	loc25BindFrameSFX(ctx)
	return engine.Sequence(
		ctx.PlayMusic("Loc25_Cyberspace.wav"),
		loc25StartEndTime(ctx),
		ctx.RemoveItem(5), ctx.RemoveItem(8), ctx.RemoveItem(9), ctx.RemoveItem(10), ctx.RemoveItem(0xb), ctx.RemoveItem(0x18), ctx.RemoveItem(0x1a),
		ctx.PlaySFX("Sfx_DangerStrings.wav"),
		loc25SetLayerXYZoom(ctx, "S105_WizWalk", 0x172, -0xa7, 0x50),
		loc25LoopLayer(ctx, "S105_WizWalk"),
		loc25MoveLayer(ctx, "S105_WizWalk", 0x165, 0x2a, 0, 100, 0x14),
		ctx.ShowLayer("S105_Rod"),
		loc25PlayLayerOnceAutoHide(ctx, "S105_RodMorph"),
		ctx.MakeLayerClickable("S105_WizWalk"),
		ctx.RunAmbient(&loc25ThunderAmbient{ctx: ctx, wait: float64(3 + rand.IntN(3))}),
	)
}

func (LOC25Controller) Exit(*Context, string, string) engine.Task { return nil }

func (LOC25Controller) Click(ctx *Context, area string) engine.Task {
	if loc25SceneID(ctx.session.state.Scene) != 105 {
		return nil
	}
	switch area {
	case "S105_WizWalk":
		return loc25InitialArrival(ctx)
	case "S105_DraHead":
		return loc25Attack(ctx, 6, "S105_DraAttack", true)
	case "S105_MadHead":
		return loc25Attack(ctx, 1, "S105_MadAttack", false)
	case "S105_SabHead":
		return loc25Attack(ctx, 2, "S105_SabAttack", false)
	case "S105_JasHead":
		return loc25Attack(ctx, 0, "S105_Jas", false)
	case "S105_LouHead":
		return loc25Attack(ctx, 4, "S105_LouAttack", false)
	case "S105_RanHead":
		return loc25Attack(ctx, 0, "S105_Ran", false)
	case "S105_CynHead":
		return loc25Attack(ctx, 3, "S105_CynAttack", false)
	case "S105_SanHead":
		return loc25Attack(ctx, 5, "S105_SanAttack", false)
	}
	return nil
}

func (LOC25Controller) UseItem(*Context, int, string) engine.Task { return nil }
func (LOC25Controller) SelectItem(*Context, int) engine.Task      { return nil }
func (LOC25Controller) LoadConditionMask(*Context, string) int    { return 0 }

type loc25ThunderAmbient struct {
	ctx  *Context
	wait float64
}

func (t *loc25ThunderAmbient) Update(dt float64) bool {
	if t.ctx.session.state.Location != 25 || loc25SceneID(t.ctx.session.state.Scene) != 105 {
		return true
	}
	t.wait -= dt
	if t.wait > 0 {
		return false
	}
	if rand.IntN(2) == 0 {
		_ = t.ctx.PlaySFX("Sfx_Thunder.wav").Update(0)
	} else {
		_ = t.ctx.PlaySFX("Sfx_Thunder2.wav").Update(0)
	}
	t.wait = float64(3 + rand.IntN(3))
	return false
}
