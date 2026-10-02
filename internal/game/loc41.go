package game

import (
	"strings"

	"github.com/wok/rent-a-hero/internal/audio"
	"github.com/wok/rent-a-hero/internal/engine"
)

type LOC41Controller struct{}

func loc41EnsureLayers(ctx *Context) {
	for _, id := range []string{"S112_Credits", "S112_EndeSequenz", "S112_MeerFadeOut", "S112_MeerLoop"} {
		_, _, _ = ctx.ensureAssetLayer(id)
		if layer, ok := ctx.layer(id); ok {
			layer.Visible = false
			layer.Playing = false
			layer.TaskDriven = false
			layer.Accumulator = 0
		}
	}
}

func loc41BindEndEvents(ctx *Context) {
	layer, ok := ctx.layer("S112_EndeSequenz")
	if !ok || layer == nil {
		return
	}
	for frame, name := range map[int]string{
		0x1f: "Sfx_Thunder2.wav",
		0x25: "Sfx_Thunder.wav",
		0x67: "Sfx_Explosion_Bass2.wav",
		0x78: "Sfx_Explosion_Bass2.wav",
		0x8e: "Sfx_Explosion_Bass2.wav",
	} {
		frame, name := frame, name
		if len(layer.FrameEvents[frame]) == 0 {
			layer.AddFrameEvent(frame, func() { _ = ctx.PlaySFX(name).Update(0) })
		}
	}
	for frame, name := range map[int]string{
		0x5a: "Sfx_Bird.wav",
		0x69: "Sfx_Bird2.wav",
		0x7b: "Sfx_Bird3.wav",
		0x91: "Sfx_Bird2.wav",
		0x9e: "Sfx_Chirp.wav",
		0xab: "Sfx_Bird3.wav",
		0xb4: "Sfx_Bird.wav",
	} {
		frame, name := frame, name
		layer.AddFrameEvent(frame, func() { _ = ctx.PlaySFXVolume(name, 50).Update(0) })
	}
}

func loc41PlayMusicFrom(ctx *Context, location int, name string) engine.Task {
	if location == ctx.session.state.Location {
		return ctx.PlayMusic(name)
	}
	return engine.Immediate(func() {
		idx, _, found, err := resolveBuiltinLocation(location)
		if err != nil || !found || idx == nil {
			return
		}
		sound, err := engine.LoadSound(idx, name)
		if err != nil {
			return
		}
		s := ctx.session
		if s.musicHandle != nil {
			s.musicHandle.Stop()
		}
		s.musicHandle = s.audioEngine.PlayLooping(sound, audio.CategoryMusic)
		s.musicName = name
		s.musicLocation = location
	})
}

func loc41LoopLayer(ctx *Context, id string) engine.Task {
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

func loc41FinishLoop(ctx *Context, id string) engine.Task {
	layer, ok := ctx.layer(id)
	if !ok || layer == nil {
		return engine.Immediate(func() {})
	}
	return engine.Sequence(
		engine.Immediate(func() {
			layer.Visible = true
			layer.Enabled = true
			layer.Mode = engine.AnimOnce
			layer.Playing = true
			layer.TaskDriven = true
			layer.Accumulator = 0
		}),
		&layerRangeTask{layer: layer, target: -1, freeze: true},
		ctx.HideLayer(id),
	)
}

func loc41PlayEndSequence(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.PlayLayerFrames("S112_EndeSequenz", 0, 0x81),
		loc41PlayMusicFrom(ctx, 41, "Loc41_InALand.wav"),
		ctx.PlayLayerFrames("S112_EndeSequenz", 0x82, 0x9f),
		ctx.RunAmbient(ctx.PlayVoiceover("112_SPR_01", "[112_SPR_01]")),
		ctx.PlayLayerFrames("S112_EndeSequenz", 0xa0, -1),
		ctx.HideLayer("S112_EndeSequenz"),
	)
}

func loc41PlayCredits(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.PlayLayerFrames("S112_Credits", 0, 0x671),
		loc41PlayMusicFrom(ctx, 35, "Loc35_RodrigoOnGlider.wav"),
		ctx.PlayLayerFrames("S112_Credits", 0x672, 0x76b),
		loc41PlayMusicFrom(ctx, 5, "Loc05_LouisOffice.wav"),
		ctx.PlayLayerFrames("S112_Credits", 0x76c, 0x833),
		loc41PlayMusicFrom(ctx, 6, "Loc06_Ranama.wav"),
		ctx.PlayLayerFrames("S112_Credits", 0x834, 0x8fb),
		loc41PlayMusicFrom(ctx, 26, "Loc26_IslandOfMegophias.wav"),
		ctx.PlayLayerFrames("S112_Credits", 0x8fc, 0x9c3),
		loc41PlayMusicFrom(ctx, 14, "Loc14_Cynthia.wav"),
		ctx.PlayLayerFrames("S112_Credits", 0x9c4, 0xb3f),
		ctx.RunAmbient(&loc41MusicFadeTask{ctx: ctx, duration: 5}),
		ctx.PlayLayerFrames("S112_Credits", 0xb40, -1),
		ctx.HideLayer("S112_Credits"),
	)
}

type loc41MusicFadeTask struct {
	ctx      *Context
	duration float64
	elapsed  float64
}

func (t *loc41MusicFadeTask) Update(dt float64) bool {
	if t.ctx.session.state.Location != 41 || !strings.EqualFold(t.ctx.session.state.Scene, "112") {
		return true
	}
	if t.duration <= 0 {
		if t.ctx.session.musicHandle != nil {
			t.ctx.session.musicHandle.SetVolume(0)
		}
		return true
	}
	t.elapsed += dt
	p := t.elapsed / t.duration
	if p > 1 {
		p = 1
	}
	if t.ctx.session.musicHandle != nil {
		t.ctx.session.musicHandle.SetVolume(1 - p)
	}
	return p >= 1
}

func (LOC41Controller) Enter(ctx *Context, scene string, from string) engine.Task {
	if !strings.EqualFold(strings.TrimPrefix(scene, "S"), "112") {
		return nil
	}
	loc41EnsureLayers(ctx)
	loc41BindEndEvents(ctx)
	return engine.Sequence(
		loc41PlayMusicFrom(ctx, 41, "Loc41_Outro.wav"),
		loc41PlayEndSequence(ctx),
		loc41LoopLayer(ctx, "S112_MeerLoop"),
		loc41PlayCredits(ctx),
		loc41FinishLoop(ctx, "S112_MeerLoop"),
		ctx.PlayLayerFrames("S112_MeerFadeOut", 0, -1),
		ctx.HideLayer("S112_MeerFadeOut"),
	)
}

func (LOC41Controller) Exit(*Context, string, string) engine.Task { return nil }
func (LOC41Controller) Click(*Context, string) engine.Task        { return nil }
func (LOC41Controller) UseItem(*Context, int, string) engine.Task { return nil }
func (LOC41Controller) SelectItem(*Context, int) engine.Task      { return nil }
func (LOC41Controller) LoadConditionMask(*Context, string) int    { return 0 }
