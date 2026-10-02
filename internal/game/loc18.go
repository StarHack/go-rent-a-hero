package game

import (
	"math/rand/v2"

	"github.com/wok/rent-a-hero/internal/engine"
)

type LOC18Controller struct{}

func (LOC18Controller) Enter(ctx *Context, scene, from string) engine.Task {
	if loc15NumericScene(scene) != 61 {
		return nil
	}
	actor := loc15ActorID(ctx)
	tasks := []engine.Task{engine.Immediate(func() {
		for _, id := range []string{"S61_ClimbIn", "S61_ClimbOut", "S61_OpenJaws", "S61_Streu"} {
			_, _, _ = ctx.ensureAssetLayer(id)
		}
		for _, id := range []string{"S61_ClimbIn", "S61_ClimbOut", "S61_OpenJaws", "S61_Streu", "S61_Smoke", "S61_SmokeClone"} {
			if l, ok := ctx.layer(id); ok {
				l.Visible = false
				l.Playing = false
				l.TaskDriven = false
				l.Frame = 0
			}
		}
	}), ctx.ShowLayer("S61_PilzLinks"), ctx.ShowLayer("S61_PilzRechts"), ctx.EnableArea("S61_To60"), ctx.EnableArea("S61_Brain"), ctx.EnableArea("S61_To60_2")}
	if actor != "" {
		tasks = append(tasks, ctx.HideActor(actor), ctx.ShowLayer("S61_ClimbIn"), ctx.PlayLayer("S61_ClimbIn"), ctx.HideLayer("S61_ClimbIn"), ctx.PlaceActorPerspective(actor, 0x8b, 0xad), ctx.ShowActor(actor), ctx.WalkToFacingPerspective(actor, 0x91, 0xbe, 0))
		if getOriginalFlag(ctx.session.state.OriginalState, loc17StateFirstBrain) != 0 {
			tasks = append(tasks, loc15SetOriginal(ctx, loc17StateFirstBrain, 0), ctx.Say(actor, "061_ROD_01", ""))
		}
	}
	if getOriginalFlag(ctx.session.state.OriginalState, loc17StateBrainUsed) != 0 {
		tasks = append(tasks, ctx.RunAmbient(&loc18Smoke{ctx: ctx}))
	}
	tasks = append(tasks, ctx.RunAmbient(&loc18Ambient{ctx: ctx}))
	return engine.Sequence(tasks...)
}

func (LOC18Controller) Exit(*Context, string, string) engine.Task { return nil }

func (LOC18Controller) Click(ctx *Context, area string) engine.Task {
	actor := loc15ActorID(ctx)
	switch area {
	case "S61_To60":
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x8c, 0xc0, 3), ctx.HideActor(actor), ctx.ShowLayer("S61_ClimbOut"), ctx.PlayLayer("S61_ClimbOut"), ctx.HideLayer("S61_ClimbOut"), ctx.ChangeLocation(17, "60"))
	case "S61_Brain":
		stage := getOriginalFlag(ctx.session.state.OriginalState, loc17StateBrainStage)
		line := "061_ROD_04"
		next := 2
		if stage == 2 {
			line, next = "061_ROD_05", 3
		} else if stage == 3 {
			line, next = "061_ROD_06", 1
		}
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x100, 0x123, 5), ctx.Say(actor, line, ""), loc15SetOriginal(ctx, loc17StateBrainStage, next))
	case "S61_To60_2":
		if getOriginalFlag(ctx.session.state.OriginalState, loc17StateBrainUsed) == 0 {
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x235, 0xf7, 5), ctx.Say(actor, "061_ROD_02", ""))
		}
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x235, 0xf7, 5), ctx.Say(actor, "061_ROD_03", ""), ctx.HideActor(actor), ctx.ShowLayer("S61_OpenJaws"), ctx.PlayLayer("S61_OpenJaws"), ctx.HideLayer("S61_OpenJaws"), ctx.ShowActor(actor), loc15SetOriginal(ctx, loc17StateJawsOpened, 1), ctx.ChangeLocation(17, "60"))
	}
	return nil
}

func loc18UsePoisonLeaves(ctx *Context) engine.Task {
	if getOriginalFlag(ctx.session.state.OriginalState, loc17StateBrainUsed) != 0 {
		return nil
	}
	actor := loc15ActorID(ctx)
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0x100, 0x123, 5),
		ctx.Say(actor, "061_ROD_08", ""),
		ctx.HideActor(actor),
		ctx.ShowLayer("S61_Streu"),
		ctx.PlayLayerSplit("S61_Streu", 0x1d, "Sfx_Leafs.wav"),
		ctx.HideLayer("S61_Streu"),
		ctx.ShowActor(actor),
		ctx.PlaySFX("Sfx_Acid.wav"),
		ctx.RunAmbient(&loc18Smoke{ctx: ctx, armed: true, wait: 1}),
		ctx.Wait(3),
		ctx.Say(actor, "061_ROD_09", ""),
		loc15SetOriginal(ctx, loc17StateBrainUsed, 1),
		ctx.RemoveItem(6),
	)
}

func (LOC18Controller) UseItem(ctx *Context, item int, area string) engine.Task {
	if item == 6 && area == "S61_Brain" {
		return loc18UsePoisonLeaves(ctx)
	}
	return nil
}

func (LOC18Controller) SelectItem(ctx *Context, item int) engine.Task {
	if item == 6 {
		return loc18UsePoisonLeaves(ctx)
	}
	return nil
}
func (LOC18Controller) LoadConditionMask(*Context, string) int { return 0 }

type loc18Ambient struct {
	ctx  *Context
	wait float64
}

func (t *loc18Ambient) Update(dt float64) bool {
	if t.ctx.session.state.Location != 18 || loc15NumericScene(t.ctx.session.state.Scene) != 61 {
		return true
	}
	if t.wait <= 0 {
		t.wait = float64(3 + rand.IntN(4))
	}
	t.wait -= dt
	if t.wait > 0 {
		return false
	}
	sounds := []string{"Sfx_Fly1.wav", "Sfx_Fly2.wav"}
	_ = t.ctx.PlaySFX(sounds[rand.IntN(2)]).Update(0)
	t.wait = float64(3 + rand.IntN(4))
	return false
}

type loc18Smoke struct {
	ctx   *Context
	wait  float64
	which bool
	armed bool
	inner engine.Task
}

func (t *loc18Smoke) Update(dt float64) bool {
	if t.ctx.session.state.Location != 18 || loc15NumericScene(t.ctx.session.state.Scene) != 61 {
		return true
	}
	if !t.armed && getOriginalFlag(t.ctx.session.state.OriginalState, loc17StateBrainUsed) == 0 {
		return true
	}
	if t.inner != nil {
		if !t.inner.Update(dt) {
			return false
		}
		t.inner = nil
		t.wait = 4
		return false
	}
	if t.wait <= 0 {
		t.wait = 1
	}
	t.wait -= dt
	if t.wait > 0 {
		return false
	}
	id := "S61_Smoke"
	if t.which {
		id = "S61_SmokeClone"
	}
	t.which = !t.which
	t.inner = engine.Sequence(t.ctx.ShowLayer(id), t.ctx.PlayLayer(id), t.ctx.HideLayer(id))
	return false
}
