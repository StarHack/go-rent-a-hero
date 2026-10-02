package game

import (
	"math"
	"sort"

	"github.com/wok/rent-a-hero/internal/audio"
	"github.com/wok/rent-a-hero/internal/engine"
)

const (
	loc09StateNight       = 0x3e80
	loc09StateStory       = 0x3f04
	loc09StateTree        = 0x3f48
	loc09StateForestStage = 0x3f54
)

type LOC09Controller struct{}

func (LOC09Controller) Enter(ctx *Context, scene, from string) engine.Task {
	if loc09SceneID(scene) != "S89" {
		return nil
	}
	return engine.Sequence(
		engine.Immediate(func() {
			_, _, _ = ctx.ensureAssetLayer("Attack")
			if layer, ok := ctx.layer("Attack"); ok {
				layer.Presentation = true
				layer.ColorKeyed = false
				layer.Visible = false
				layer.Playing = false
				layer.TaskDriven = false
				layer.Frame = 0
				layer.Accumulator = 0
			}
		}),
		ctx.PlayMusic("Loc09_SmashvilleAttack.wav"),
		engine.Immediate(func() {
			putOriginalFlag(ctx.session.state.OriginalState, loc09StateNight, 1)
			putOriginalFlag(ctx.session.state.OriginalState, loc09StateStory, 1)
			putOriginalFlag(ctx.session.state.OriginalState, loc09StateTree, 0)
			putOriginalFlag(ctx.session.state.OriginalState, loc09StateForestStage, 4)
		}),
		&loc09AttackTask{ctx: ctx},
		engine.Immediate(func() {
			putOriginalFlag(ctx.session.state.OriginalState, loc09StateNight, 1)
		}),
		ctx.ChangeLocation(14, "90"),
	)
}

func (LOC09Controller) Exit(*Context, string, string) engine.Task { return nil }
func (LOC09Controller) Click(*Context, string) engine.Task        { return nil }
func (LOC09Controller) UseItem(*Context, int, string) engine.Task { return nil }
func (LOC09Controller) SelectItem(*Context, int) engine.Task      { return nil }
func (LOC09Controller) LoadConditionMask(*Context, string) int    { return 0 }

func loc09SceneID(scene string) string {
	switch scene {
	case "89", "089", "S89", "S089":
		return "S89"
	default:
		return scene
	}
}

type loc09CueAction int

const (
	loc09CueOneShot loc09CueAction = iota
	loc09CueOceanVolume
	loc09CueEndTimeStart
	loc09CueEndTimeVolume
)

type loc09FrameCue struct {
	frame          int
	action         loc09CueAction
	sfx            string
	volume         int
	explicitVolume bool
	fadeMS         int
	voice          string
}

var loc09AttackCues = func() []loc09FrameCue {
	cues := []loc09FrameCue{
		{0x3b, loc09CueOneShot, "Sfx_Seagull.wav", 80, true, 0, ""},
		{0x7b, loc09CueOneShot, "Sfx_Seagull.wav", 100, true, 0, ""},
		{0xa8, loc09CueOneShot, "Sfx_ServoMotor_ShortClang2.wav", 100, false, 0, ""},
		{0xaf, loc09CueOneShot, "Sfx_CannonFire.wav", 100, false, 0, ""},
		{0xb0, loc09CueOneShot, "Sfx_BombApproaching.wav", 0, true, 0, ""},
		{0xbb, loc09CueOneShot, "Sfx_CannonFire2.wav", 100, false, 0, ""},
		{0xbd, loc09CueOneShot, "Sfx_CannonFire2.wav", 100, false, 0, ""},
		{0xbf, loc09CueOneShot, "Sfx_CannonFire2.wav", 100, false, 0, ""},
		{0xc1, loc09CueOneShot, "Sfx_CannonFire2.wav", 100, false, 0, ""},
		{0xc2, loc09CueOneShot, "Sfx_BombApproaching2.wav", 0, true, 0, ""},
		{199, loc09CueOneShot, "Sfx_Explosion_Bass.wav", 0, true, 0, ""},
		{0xce, loc09CueOneShot, "Sfx_Explosion_Misc2.wav", 0, true, 0, ""},
		{0xd3, loc09CueOneShot, "Sfx_BombApproaching.wav", 0, true, 0, ""},
		{0xd5, loc09CueOneShot, "Sfx_Explosion_Bass.wav", 0, true, 0, ""},
		{0xd9, loc09CueOneShot, "Sfx_Explosion_Bass2.wav", 0, true, 0, ""},
		{0xdd, loc09CueOneShot, "Sfx_Explosion_Misc2.wav", 0, true, 0, ""},
		{0xee, loc09CueOceanVolume, "", 1, true, 0, ""},
		{0xef, loc09CueOneShot, "Sfx_BigStone.wav", 100, false, 0, ""},
		{0xf5, loc09CueOneShot, "Sfx_BigStone.wav", 100, false, 0, ""},
		{0xfa, loc09CueOneShot, "Sfx_BigStone.wav", 100, false, 0, ""},
		{0xfc, loc09CueOneShot, "Sfx_Thunder.wav", 100, false, 0, ""},
		{0xfc, loc09CueOneShot, "Sfx_Thunder.wav", 100, false, 0, ""},
		{0xff, loc09CueOneShot, "Sfx_BigStone.wav", 100, false, 0, ""},
		{0x104, loc09CueOneShot, "Sfx_BigStone.wav", 100, false, 0, ""},
		{0x105, loc09CueOneShot, "Sfx_Thunder2.wav", 100, false, 0, "084_ROD_01"},
		{0x105, loc09CueOneShot, "Sfx_Thunder2.wav", 100, false, 0, ""},
		{0x105, loc09CueOneShot, "Sfx_Thunder2.wav", 100, false, 0, ""},
		{0x109, loc09CueOneShot, "Sfx_BigStone.wav", 100, false, 0, ""},
		{0x10e, loc09CueOneShot, "Sfx_BigStone.wav", 100, false, 0, ""},
		{0x113, loc09CueOneShot, "Sfx_BigStone.wav", 100, false, 0, ""},
		{0x118, loc09CueOneShot, "Sfx_BigStone.wav", 100, false, 0, ""},
		{0x11d, loc09CueOneShot, "Sfx_BigStone.wav", 100, false, 0, ""},
		{0x122, loc09CueOneShot, "Sfx_BigStone.wav", 100, false, 0, ""},
		{0x123, loc09CueOneShot, "", 100, false, 0, "084_ROD_02"},
		{0x127, loc09CueOneShot, "Sfx_BigStone.wav", 100, false, 0, ""},
		{300, loc09CueOneShot, "Sfx_BigStone.wav", 100, false, 0, ""},
		{0x131, loc09CueOneShot, "Sfx_BigStone.wav", 100, false, 0, ""},
		{0x136, loc09CueOneShot, "Sfx_BigStone.wav", 100, false, 0, ""},
		{0x13b, loc09CueOneShot, "Sfx_BigStone.wav", 100, false, 0, ""},
		{0x140, loc09CueOneShot, "Sfx_BigStone.wav", 100, false, 0, ""},
		{0x145, loc09CueOneShot, "Sfx_BigStone.wav", 100, false, 0, ""},
		{0x14a, loc09CueOneShot, "Sfx_BigStone.wav", 100, false, 0, ""},
		{0x14b, loc09CueOneShot, "", 100, false, 0, "084_ROD_03"},
		{0x14f, loc09CueOneShot, "Sfx_BigStone.wav", 100, false, 0, ""},
		{0x151, loc09CueOceanVolume, "", 70, true, 0, ""},
		{0x163, loc09CueEndTimeStart, "", 60, true, 2000, ""},
		{0x177, loc09CueOneShot, "Sfx_Thunder.wav", 100, false, 0, ""},
		{0x17a, loc09CueOneShot, "Sfx_Thunder2.wav", 100, false, 0, ""},
		{0x18e, loc09CueOneShot, "Sfx_Thunder.wav", 100, false, 0, ""},
		{400, loc09CueOneShot, "Sfx_WaterSplash.wav", 100, false, 0, ""},
		{0x194, loc09CueOneShot, "Sfx_WaterSplash.wav", 100, false, 0, ""},
		{0x197, loc09CueOneShot, "Sfx_Seagull.wav", 80, true, 0, ""},
		{0x19d, loc09CueOneShot, "Sfx_WaterSplash.wav", 100, false, 0, ""},
		{0x1a5, loc09CueOneShot, "Sfx_Seagull.wav", 60, true, 0, ""},
		{0x1d9, loc09CueOceanVolume, "", 1, true, 0, ""},
		{0x23a, loc09CueOneShot, "Sfx_Thunder.wav", 100, false, 0, ""},
		{0x23f, loc09CueOneShot, "Sfx_Thunder2.wav", 100, false, 0, ""},
		{0x25f, loc09CueOneShot, "Sfx_Explosion_Misc2.wav", 100, false, 0, ""},
		{0x26a, loc09CueEndTimeVolume, "", 1, true, 1000, ""},
	}
	sort.SliceStable(cues, func(i, j int) bool { return cues[i].frame < cues[j].frame })
	return cues
}()

type loc09VolumeEnvelope struct {
	currentVolume int
	targetVolume  int
	step          int
	accumulator   float64
	handle        audio.Handle
}

func (v *loc09VolumeEnvelope) setImmediate(volume int) {
	v.currentVolume = volume
	v.targetVolume = volume
	v.step = 0
	v.accumulator = 0
	v.apply()
}

func (v *loc09VolumeEnvelope) setFade(volume, milliseconds int) {
	v.targetVolume = volume
	v.accumulator = 0
	steps := milliseconds / 100
	if steps <= 0 {
		v.setImmediate(volume)
		return
	}
	v.step = (volume - v.currentVolume) / steps
	if v.step == 0 {
		v.setImmediate(volume)
	}
}

func (v *loc09VolumeEnvelope) update(dt float64) {
	if v.handle == nil || v.step == 0 {
		return
	}
	v.accumulator += dt
	for v.accumulator >= 0.1 && v.step != 0 {
		v.accumulator -= 0.1
		v.currentVolume += v.step
		if (v.step < 0 && v.currentVolume <= v.targetVolume) || (v.step > 0 && v.currentVolume >= v.targetVolume) {
			v.currentVolume = v.targetVolume
			v.step = 0
		}
		v.apply()
	}
}

func (v *loc09VolumeEnvelope) apply() {
	if v.handle == nil {
		return
	}
	v.handle.SetVolume(loc09DirectSoundGain(v.currentVolume))
}

type loc09AttackTask struct {
	ctx          *Context
	play         engine.Task
	started      bool
	cueIndex     int
	oceanSound   *audio.Sound
	endTimeSound *audio.Sound
	ocean        loc09VolumeEnvelope
	endTime      loc09VolumeEnvelope
	lastFrame    int
}

func (t *loc09AttackTask) Update(dt float64) bool {
	if !t.started {
		t.started = true
		layer, ok := t.ctx.layer("Attack")
		if !ok {
			return true
		}
		layer.Visible = true
		t.play = engine.PlayLayerOnce(layer)
		t.lastFrame = -1
		if sound, err := engine.LoadSound(t.ctx.session.idx, "Sfx_Ocean.wav"); err == nil {
			t.oceanSound = sound
			t.ocean = loc09VolumeEnvelope{currentVolume: 70, targetVolume: 70}
			t.ocean.handle = t.ctx.session.audioEngine.PlayLoopingWithVolume(t.oceanSound, audio.CategorySFX, loc09DirectSoundGain(70))
		}
		if sound, err := engine.LoadSound(t.ctx.session.idx, "Sfx_EndTimeFeeling.wav"); err == nil {
			t.endTimeSound = sound
			t.endTime = loc09VolumeEnvelope{currentVolume: 0, targetVolume: 0}
		}
	}

	t.ocean.update(dt)
	t.endTime.update(dt)

	layer, ok := t.ctx.layer("Attack")
	if !ok || t.play == nil {
		t.stopLoops()
		return true
	}

	done := t.play.Update(dt)
	frame := layer.Frame
	if frame < t.lastFrame {
		t.lastFrame = frame
	}
	for t.cueIndex < len(loc09AttackCues) && loc09AttackCues[t.cueIndex].frame <= frame {
		cue := loc09AttackCues[t.cueIndex]
		if cue.frame > t.lastFrame {
			t.fireCue(cue)
		}
		t.cueIndex++
	}
	t.lastFrame = frame

	if done {
		t.stopLoops()
		return true
	}
	return false
}

func (t *loc09AttackTask) fireCue(cue loc09FrameCue) {
	switch cue.action {
	case loc09CueOneShot:
		if cue.sfx != "" {
			sound, err := engine.LoadSound(t.ctx.session.idx, cue.sfx)
			if err == nil {
				if cue.explicitVolume {
					t.ctx.session.audioEngine.PlayWithVolume(sound, audio.CategorySFX, loc09DirectSoundGain(cue.volume))
				} else {
					t.ctx.session.audioEngine.Play(sound, audio.CategorySFX)
				}
			}
		}
	case loc09CueOceanVolume:
		t.ocean.setImmediate(cue.volume)
	case loc09CueEndTimeStart:
		if t.endTimeSound != nil && t.endTime.handle == nil {
			t.endTime.handle = t.ctx.session.audioEngine.PlayLoopingWithVolume(t.endTimeSound, audio.CategorySFX, loc09DirectSoundGain(t.endTime.currentVolume))
		}
		t.endTime.setFade(cue.volume, cue.fadeMS)
	case loc09CueEndTimeVolume:
		t.endTime.setFade(cue.volume, cue.fadeMS)
	}
	if cue.voice != "" {
		t.ctx.session.ambientTasks = append(t.ctx.session.ambientTasks, t.ctx.PlayVoiceover(cue.voice, "["+cue.voice+"]"))
	}
}

func (t *loc09AttackTask) stopLoops() {
	if t.ocean.handle != nil {
		t.ocean.handle.Stop()
		t.ocean.handle = nil
	}
	if t.endTime.handle != nil {
		t.endTime.handle.Stop()
		t.endTime.handle = nil
	}
}

func loc09DirectSoundGain(volume int) float64 {
	if volume < 0 {
		volume = 0
	}
	if volume > 100 {
		volume = 100
	}
	attenuation := 3333*(float64(volume)/100.0) - 3333
	return math.Pow(10, attenuation/2000.0)
}
