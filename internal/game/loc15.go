package game

import (
	"math/rand/v2"

	"github.com/wok/rent-a-hero/internal/engine"
)

const (
	loc15StateNight        = 0x3e80
	loc15StateGateClosed   = 0x3f04
	loc15StateForestGlider = 0x3f24
	loc15StateTo38         = 0x3f38
	loc15StatePiratePeek   = 0x3f90
	loc15StateVoicePending = 0x3f94
	loc15StateTo113        = 0x3f98
	loc15StateGlider       = 0x3f9c
	loc15StateSack         = 0x3fa0
)

type LOC15Controller struct{}

func (LOC15Controller) Enter(ctx *Context, scene, from string) engine.Task {
	if scene != "57" && scene != "S57" {
		return nil
	}
	actor := loc15ActorID(ctx)
	tasks := []engine.Task{loc15InitializeScene(ctx, actor)}
	if getOriginalFlag(ctx.session.state.OriginalState, loc15StateNight) == 0 {
		tasks = append(tasks, ctx.PlayMusic("Loc10_ForestBeforeDarkness.wav"))
	} else {
		tasks = append(tasks, ctx.SetBackgroundByStem("S57_Back_nd"), ctx.PlayMusic("Loc10_ForestInDarkness.wav"))
	}
	if getOriginalFlag(ctx.session.state.OriginalState, loc15StateSack) != 0 {
		tasks = append(tasks, ctx.ShowLayer("S57_Sack"), ctx.MakeLayerClickable("S57_Sack"))
	} else {
		tasks = append(tasks, ctx.HideLayer("S57_Sack"), ctx.DisableArea("S57_Sack"))
	}
	if getOriginalFlag(ctx.session.state.OriginalState, loc15StateGlider) != 0 {
		tasks = append(tasks, ctx.ShowLayer("S57_Gleiter"), ctx.MakeLayerClickable("S57_Gleiter"), ctx.RunAmbient(&loc15GliderBob{ctx: ctx, target: 207}))
	} else {
		tasks = append(tasks, ctx.HideLayer("S57_Gleiter"), ctx.DisableArea("S57_Gleiter"))
	}
	if actor != "" {
		switch loc15NumericScene(from) {
		case 38:
			tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0xd4, 0xb2), ctx.ShowActor(actor), ctx.RunAmbient(ctx.WalkToFacingPerspective(actor, 0x111, 0x14b, 0)))
		case 53:
			layer := loc15RodRausLayer(ctx)
			tasks = append(tasks, ctx.ShowLayer(layer), ctx.PlayLayerSplit(layer, 3, "Sfx_Clothes_Scratched.wav"), ctx.HideLayer(layer), ctx.PlaceActorPerspective(actor, 0x28, 0x142), ctx.ShowActor(actor), ctx.RunAmbient(ctx.WalkToFacingPerspective(actor, 0x64, 0x142, 6)))
		case 113:
			if getOriginalFlag(ctx.session.state.OriginalState, loc15StateNight) == 0 {
				tasks = append(tasks,
					ctx.HideLayer("S57_Gleiter"),
					ctx.ShowLayer("S57_GleiterHin"),
					ctx.PlaySFX("Sfx_Glider_PassingBy.wav"),
					loc15TweenLayerPosition(ctx, "S57_GleiterHin", 0x249, 0x80, 0xe3, 0xc6, 50),
					loc15TweenLayerPosition(ctx, "S57_GleiterHin", 0xe3, 0xc6, 0x7f, 0xd0, 40),
					ctx.HideLayer("S57_GleiterHin"),
					ctx.ShowLayer("S57_RodAbsteig"),
					ctx.PlayLayer("S57_RodAbsteig"),
					ctx.HideLayer("S57_RodAbsteig"),
					ctx.PlaceActorPerspective(actor, 0xcd, 0x136),
					ctx.ShowActor(actor),
					ctx.ShowLayer("S57_Gleiter"),
				)
			} else {
				tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x23a, 300), ctx.ShowActor(actor), ctx.RunAmbient(ctx.WalkToFacingPerspective(actor, 0x1c5, 0x13f, 1)))
			}
		default:
			tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x23a, 300), ctx.ShowActor(actor), ctx.RunAmbient(ctx.WalkToFacingPerspective(actor, 0x1c5, 0x13f, 1)))
		}
	}
	if getOriginalFlag(ctx.session.state.OriginalState, loc15StatePiratePeek) != 0 {
		tasks = append(tasks,
			loc15SetOriginal(ctx, loc15StatePiratePeek, 0),
			loc15SetOriginal(ctx, loc15StateTo113, 1),
			ctx.ShowLayer("S57_PiratGuck"),
			ctx.PlaySFX("Sfx_DangerStringsLow.wav"),
			loc15TweenLayerX(ctx, "S57_PiratGuck", -10, 10),
			ctx.PlayLayer("S57_PiratGuck"),
			engine.Wait(0.3),
			loc15TweenLayerX(ctx, "S57_PiratGuck", -30, 5),
			ctx.HideLayer("S57_PiratGuck"),
		)
	}
	tasks = append(tasks, ctx.RunAmbient(&loc15Ambient{ctx: ctx}))
	return engine.Sequence(tasks...)
}

func loc15InitializeScene(ctx *Context, actor string) engine.Task {
	return engine.Immediate(func() {
		night := getOriginalFlag(ctx.session.state.OriginalState, loc15StateNight) != 0
		for _, id := range []string{"S57_RodRein_vd", "S57_RodRaus_vd", "S57_RodSchaut", "S57_GleiterWeg", "S57_RodAbsteig", "S57_Gleiter", "S57_GleiterHin", "S57_PiratGuck", "S57_RodRein_nd", "S57_RodRaus_nd", "Gen_RodPickUp", "S57_Sack"} {
			if layer, ok := ctx.session.scene.Layers[id]; ok {
				layer.Visible = false
				layer.Playing = false
				layer.TaskDriven = false
				layer.Frame = 0
				layer.Accumulator = 0
			}
		}
		if actor != "" {
			if a, ok := ctx.session.scene.Characters[actor]; ok {
				a.Visible = false
				if night {
					a.Color.R = -122
					a.Color.G = -84
					a.Color.B = -24
				} else {
					a.Color.R = -10
					a.Color.G = -34
					a.Color.B = -52
				}
			}
		}
	})
}

func (LOC15Controller) Exit(*Context, string, string) engine.Task { return nil }

func (LOC15Controller) Click(ctx *Context, area string) engine.Task {
	actor := loc15ActorID(ctx)
	switch area {
	case "S57_Kanal":
		layer := loc15RodReinLayer(ctx)
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0x2c, 0x143, 2),
			ctx.HideActor(actor),
			ctx.ShowLayer(layer),
			ctx.PlayLayerSplit(layer, 9, "Sfx_Clothes_Scratched.wav"),
			ctx.HideLayer(layer),
			loc15SetOriginalIf(ctx, func() bool {
				return getOriginalFlag(ctx.session.state.OriginalState, loc15StateGlider) != 0 || getOriginalFlag(ctx.session.state.OriginalState, loc15StateForestGlider) != 0 || getOriginalFlag(ctx.session.state.OriginalState, loc15StateNight) != 0
			}, loc15StateGateClosed, 1),
			loc15SetOriginalIf(ctx, func() bool { return getOriginalFlag(ctx.session.state.OriginalState, loc15StateGateClosed) != 0 }, 0x3efc, 0),
			loc15SetOriginalIf(ctx, func() bool { return getOriginalFlag(ctx.session.state.OriginalState, loc15StateGateClosed) != 0 }, 0x3f00, 0),
			ctx.ChangeLocation(7, "53"),
		)
	case "S57_To38":
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0xd4, 0xb2, 4),
			loc15SetOriginal(ctx, loc15StateTo38, 1),
			ctx.ChangeLocation(10, "38"),
		)
	case "S57_To58":
		tasks := []engine.Task{}
		if getOriginalFlag(ctx.session.state.OriginalState, loc15StateVoicePending) != 0 {
			tasks = append(tasks, loc15SetOriginal(ctx, loc15StateVoicePending, 0), ctx.PlayVoiceover("057_ROD_02", ""))
		}
		tasks = append(tasks, ctx.WalkToFacingPerspective(actor, 0x23a, 300, 6), ctx.ChangeLocation(16, "58"))
		return engine.Sequence(tasks...)
	case "S57_Schlucht":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x6d, 0x132, 6), ctx.HideActor(actor), ctx.ShowLayer("S57_RodSchaut"), ctx.PlayLayer("S57_RodSchaut"), ctx.HideLayer("S57_RodSchaut"), ctx.ShowActor(actor), ctx.PlayVoiceover("057_ROD_01", ""))
	case "S57_Gleiter":
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0xcd, 0x136, 0),
			loc15SetOriginal(ctx, loc15StateGlider, 0),
			ctx.HideLayer("S57_Gleiter"),
			ctx.HideActor(actor),
			ctx.ShowLayer("S57_GleiterWeg"),
			ctx.PlayLayerSplit("S57_GleiterWeg", 48, "Sfx_Glider_PassingBy.wav"),
			ctx.HideLayer("S57_GleiterWeg"),
			ctx.ChangeLocation(31, "113"),
		)
	case "S57_Sack":
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0x14, 0x164, 4),
			ctx.HideActor(actor),
			ctx.ShowLayer("Gen_RodPickUp"),
			ctx.PlayLayerFrames("Gen_RodPickUp", 0, 7),
			ctx.HideLayer("S57_Sack"),
			ctx.DisableArea("S57_Sack"),
			ctx.AddItem(7),
			loc15SetOriginal(ctx, loc15StateSack, 0),
			ctx.PlayLayerFrames("Gen_RodPickUp", 8, -1),
			ctx.HideLayer("Gen_RodPickUp"),
			ctx.ShowActor(actor),
			ctx.PlayVoiceover("057_ROD_03", ""),
		)
	}
	return nil
}

func (LOC15Controller) UseItem(*Context, int, string) engine.Task { return nil }
func (LOC15Controller) SelectItem(*Context, int) engine.Task      { return nil }
func (LOC15Controller) LoadConditionMask(ctx *Context, scene string) int {
	if scene != "57" && scene != "S57" {
		return 0
	}
	night := getOriginalFlag(ctx.session.state.OriginalState, loc15StateNight) != 0
	from := loc15NumericScene(ctx.session.state.Scene)
	if night {
		putOriginalFlag(ctx.session.state.OriginalState, loc15StateGlider, 0)
	} else if from == 113 {
		putOriginalFlag(ctx.session.state.OriginalState, loc15StateGlider, 1)
	}
	mask := 1
	if night {
		mask = 2
	}
	if getOriginalFlag(ctx.session.state.OriginalState, loc15StateGlider) != 0 {
		mask |= 4
	}
	if from == 53 {
		mask |= 8
	} else {
		putOriginalFlag(ctx.session.state.OriginalState, loc15StatePiratePeek, 0)
	}
	return mask
}

func loc15RodReinLayer(ctx *Context) string {
	if getOriginalFlag(ctx.session.state.OriginalState, loc15StateNight) != 0 {
		return "S57_RodRein_nd"
	}
	return "S57_RodRein_vd"
}

func loc15RodRausLayer(ctx *Context) string {
	if getOriginalFlag(ctx.session.state.OriginalState, loc15StateNight) != 0 {
		return "S57_RodRaus_nd"
	}
	return "S57_RodRaus_vd"
}

func loc15ActorID(ctx *Context) string {
	if ctx.session.scene == nil {
		return ""
	}
	for _, id := range ctx.session.scene.CharacterOrder {
		if _, ok := ctx.session.scene.Characters[id]; ok {
			return id
		}
	}
	return ""
}

func loc15NumericScene(scene string) int {
	n := 0
	for _, r := range scene {
		if r >= '0' && r <= '9' {
			n = n*10 + int(r-'0')
		}
	}
	return n
}

func loc15SetOriginal(ctx *Context, offset, value int) engine.Task {
	return engine.Immediate(func() { putOriginalFlag(ctx.session.state.OriginalState, offset, value) })
}

func loc15SetOriginalIf(ctx *Context, cond func() bool, offset, value int) engine.Task {
	return engine.Immediate(func() {
		if cond() {
			putOriginalFlag(ctx.session.state.OriginalState, offset, value)
		}
	})
}

type loc15Ambient struct {
	ctx  *Context
	wait float64
}

func (t *loc15Ambient) Update(dt float64) bool {
	if t.ctx.session.state.Location != 15 || loc15NumericScene(t.ctx.session.state.Scene) != 57 {
		return true
	}
	if t.wait <= 0 {
		t.wait = float64(5 + rand.IntN(11))
	}
	t.wait -= dt
	if t.wait > 0 {
		return false
	}
	if getOriginalFlag(t.ctx.session.state.OriginalState, loc15StateNight) != 0 {
		sounds := []string{"Sfx_Owl.wav", "Sfx_Owl.wav", "Sfx_Thunder.wav", "Sfx_Thunder2.wav"}
		_ = t.ctx.PlaySFX(sounds[rand.IntN(len(sounds))]).Update(0)
	} else {
		sounds := []string{"Sfx_Bird.wav", "Sfx_Bird2.wav", "Sfx_Bird3.wav", "Sfx_Chirp.wav"}
		_ = t.ctx.PlaySFX(sounds[rand.IntN(len(sounds))]).Update(0)
	}
	t.wait = float64(5 + rand.IntN(6))
	return false
}

type loc15GliderBob struct {
	ctx         *Context
	target      int
	step        int
	start       int
	accumulator float64
}

func (t *loc15GliderBob) Update(dt float64) bool {
	if t.ctx.session.state.Location != 15 || loc15NumericScene(t.ctx.session.state.Scene) != 57 || getOriginalFlag(t.ctx.session.state.OriginalState, loc15StateGlider) == 0 {
		return true
	}
	l, ok := t.ctx.layer("S57_Gleiter")
	if !ok {
		return true
	}
	t.accumulator += dt
	for t.accumulator >= 0.1 {
		t.accumulator -= 0.1
		oldY := l.Y
		if t.step == 0 {
			t.start = l.Y
		}
		t.step++
		l.Y = t.start + (t.target-t.start)*t.step/2
		if t.step >= 2 {
			l.Y = t.target
			if t.target == 207 {
				t.target = 209
			} else {
				t.target = 207
			}
			t.step = 0
		}
		if area, ok := t.ctx.session.scene.Areas["S57_Gleiter"]; ok {
			delta := l.Y - oldY
			area.Y1 += delta
			area.Y2 += delta
		}
	}
	return false
}

type loc15LayerTween struct {
	ctx   *Context
	id    string
	fromX int
	fromY int
	toX   int
	toY   int
	steps int
	step  int
	xOnly bool
}

func (t *loc15LayerTween) Update(float64) bool {
	l, ok := t.ctx.layer(t.id)
	if !ok || t.steps <= 0 {
		return true
	}
	if t.step == 0 && t.xOnly {
		t.fromX = l.X
		t.fromY = l.Y
	}
	t.step++
	if t.step > t.steps {
		t.step = t.steps
	}
	l.X = t.fromX + (t.toX-t.fromX)*t.step/t.steps
	if !t.xOnly {
		l.Y = t.fromY + (t.toY-t.fromY)*t.step/t.steps
	}
	return t.step >= t.steps
}

func loc15TweenLayerPosition(ctx *Context, id string, fromX, fromY, toX, toY, steps int) engine.Task {
	return &loc15LayerTween{ctx: ctx, id: id, fromX: fromX, fromY: fromY, toX: toX, toY: toY, steps: steps}
}

func loc15TweenLayerX(ctx *Context, id string, toX, steps int) engine.Task {
	return &loc15LayerTween{ctx: ctx, id: id, toX: toX, steps: steps, xOnly: true}
}
