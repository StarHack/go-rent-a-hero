package game

import (
	"log"
	"math"
	"os"
	"strings"

	"github.com/wok/rent-a-hero/internal/assets"
	"github.com/wok/rent-a-hero/internal/audio"
	"github.com/wok/rent-a-hero/internal/engine"
	"github.com/wok/rent-a-hero/internal/formats/acs"
	"github.com/wok/rent-a-hero/internal/video"
)

// videoLayerID is the fixed layer ID a playing cutscene is temporarily
// injected under. Safe as a constant because only one blocking task (and
// therefore at most one video) runs at a time per agents/GAMEPLAY.md
// "Interaction lock".
const videoLayerID = "__video__"

// Context is the gameplay-operations API exposed to Controller
// implementations, per agents/GAMEPLAY.md "Context API". It deliberately
// exposes only gameplay operations, not renderer/session internals, so
// controllers stay deterministic and testable against a fake Context.
type Context struct {
	session *Session
}

func (c *Context) loadSceneFragment(fragmentID string) {
	def, err := c.session.loadSceneDef(c.session.state.Location, fragmentID)
	if err != nil {
		log.Printf("game: scene fragment %q: %v", fragmentID, err)
		return
	}
	characters := make(map[string]*engine.Actor, len(c.session.scene.Characters))
	for id, actor := range c.session.scene.Characters {
		characters[id] = actor
	}
	characterOrder := append([]string(nil), c.session.scene.CharacterOrder...)
	beforeWarnings := len(c.session.scene.Warnings)
	if err := engine.MergeSceneOverlay(c.session.scene, def, c.session.idx, nil, 0); err != nil {
		log.Printf("game: scene fragment %q: %v", fragmentID, err)
		return
	}
	c.session.localizeSceneAreas(c.session.scene, c.session.state.Location)
	c.session.scene.Characters = characters
	c.session.scene.CharacterOrder = characterOrder
	for _, warning := range c.session.scene.Warnings[beforeWarnings:] {
		log.Printf("game: scene fragment %q: %s", fragmentID, warning)
	}
}

func (c *Context) instantiateSceneFragment(fragmentID string) (*engine.Scene, error) {
	def, err := c.session.loadSceneDef(c.session.state.Location, fragmentID)
	if err != nil {
		return nil, err
	}
	fragment, err := engine.InstantiateScene(def, c.session.idx, nil, 0)
	if err != nil {
		return nil, err
	}
	c.session.localizeSceneAreas(fragment, c.session.state.Location)
	for _, warning := range fragment.Warnings {
		log.Printf("game: scene fragment %q: %s", fragmentID, warning)
	}
	return fragment, nil
}

func (c *Context) withSceneFragment(fragmentID string, build func() engine.Task) engine.Task {
	return &sceneFragmentTask{ctx: c, fragmentID: fragmentID, build: build}
}

type sceneFragmentTask struct {
	ctx        *Context
	fragmentID string
	build      func() engine.Task
	base       *engine.Scene
	inner      engine.Task
	started    bool
	done       bool
}

func (t *sceneFragmentTask) Update(dt float64) bool {
	if t.done {
		return true
	}
	if !t.started {
		t.started = true
		fragment, err := t.ctx.instantiateSceneFragment(t.fragmentID)
		if err != nil {
			log.Printf("game: scene fragment %q: %v", t.fragmentID, err)
			t.done = true
			return true
		}
		t.base = t.ctx.session.scene
		t.ctx.session.scene = fragment
		if t.build != nil {
			t.inner = t.build()
		}
	}
	if t.inner != nil && !t.inner.Update(dt) {
		return false
	}
	t.ctx.session.scene = t.base
	t.done = true
	return true
}

func (c *Context) ensureAssetLayer(id string) (*engine.Layer, string, error) {
	if layer, ok := c.session.scene.Layers[id]; ok && layer != nil && layer.Source != nil {
		return layer, "", nil
	}

	// FUN_0043ffb0 resolves controller-owned action animations through the
	// original AVI source type (the fixed DAT_0047672c suffix maps to source
	// type 3 in FUN_00446940/FUN_00446ac0). Do not substitute A16 here.
	source, real, err := engine.LoadSpriteSourceByStem(c.session.idx, id, ".avi")
	if err != nil {
		return nil, "", err
	}
	fps := 10
	if rated, ok := source.(interface{ FPS() float64 }); ok {
		if v := int(rated.FPS() + 0.5); v > 0 {
			fps = v
		}
	}

	x, y, z, zoom := 0, 0, 1, 100
	zBuffered := false
	dirtyAlpha := false
	if def, err := c.session.loadSceneDef(c.session.state.Location, c.session.state.Scene); err == nil {
		for name, layerDef := range def.Layers {
			if !strings.EqualFold(name, id) {
				continue
			}
			x = layerDef.X
			y = layerDef.Y
			if layerDef.Z != nil {
				z = *layerDef.Z
			}
			if layerDef.ZoomVal != nil {
				zoom = *layerDef.ZoomVal
			}
			if layerDef.FPS != nil {
				fps = *layerDef.FPS
			}
			zBuffered = layerDef.ZBuffered == nil || *layerDef.ZBuffered != 0
			dirtyAlpha = layerDef.DirtyAlpha != nil && *layerDef.DirtyAlpha != 0
			break
		}
	}

	_, colorKeyed := source.(engine.ColorKeySource)
	presentation := false
	if c.session.scene != nil && c.session.scene.PresentationAssets != nil {
		presentation = c.session.scene.PresentationAssets[strings.ToLower(strings.TrimSpace(id))]
	}
	if presentation {
		// [COMPILER] VideoN objects are installed into the original scene's
		// dedicated presentation slot (FUN_0044a330), whose draw path is
		// unkeyed.  They are still represented as temporary Layers in this
		// port, but Presentation keeps them out of the normal keyed layer pass.
		colorKeyed = false
	}
	layer := &engine.Layer{
		ID:           id,
		Source:       source,
		X:            x,
		Y:            y,
		Z:            z,
		Zoom:         zoom,
		FPS:          fps,
		Mode:         engine.AnimLoop,
		Enabled:      true,
		Visible:      false,
		Playing:      false,
		ZBuffered:    zBuffered,
		DirtyAlpha:   dirtyAlpha,
		ColorKeyed:   colorKeyed,
		Presentation: presentation,
	}

	if _, exists := c.session.scene.Layers[id]; !exists {
		c.session.scene.LayerOrder = append(c.session.scene.LayerOrder, id)
	}
	c.session.scene.Layers[id] = layer
	return layer, real, nil
}

func (c *Context) layer(id string) (*engine.Layer, bool) {
	if c.session.scene == nil {
		return nil, false
	}
	if layer, ok := c.session.scene.Layers[id]; ok {
		return layer, layer != nil
	}
	for _, name := range c.session.scene.LayerOrder {
		if !strings.EqualFold(name, id) {
			continue
		}
		layer := c.session.scene.Layers[name]
		return layer, layer != nil
	}
	return nil, false
}

func (c *Context) actor(id string) (*engine.Actor, bool) {
	a, _, ok := c.session.actorByNameOrAsset(id)
	if !ok {
		log.Printf("game: unknown actor %q", id)
	}
	return a, ok
}

// WalkTo moves actor to (x, y), pathfinding and snapping as needed.
func (c *Context) WalkTo(actorID string, x, y float64) engine.Task {
	return &deferredWalkTask{ctx: c, actorID: actorID, x: x, y: y}
}

func (c *Context) WalkToPerspective(actorID string, x, y float64) engine.Task {
	return &deferredWalkTask{ctx: c, actorID: actorID, x: x, y: y, perspective: true}
}

func (c *Context) WalkStraightTo(actorID string, x, y float64) engine.Task {
	return &deferredWalkTask{ctx: c, actorID: actorID, x: x, y: y, straight: true}
}

func (c *Context) WalkToFacing(actorID string, x, y float64, direction int) engine.Task {
	return engine.Sequence(c.WalkTo(actorID, x, y), c.SetActorOrientation(actorID, direction))
}

func (c *Context) WalkToFacingPerspective(actorID string, x, y float64, direction int) engine.Task {
	return engine.Sequence(c.WalkToPerspective(actorID, x, y), c.SetActorOrientation(actorID, direction))
}

type deferredWalkTask struct {
	ctx         *Context
	actorID     string
	x           float64
	y           float64
	straight    bool
	perspective bool
	actor       *engine.Actor
	inner       engine.Task
	started     bool
	epoch       uint64
}

func (t *deferredWalkTask) Update(dt float64) bool {
	if !t.started {
		t.started = true
		actor, ok := t.ctx.actor(t.actorID)
		if !ok {
			return true
		}
		t.actor = actor
		if t.ctx.session.walkEpoch == nil {
			t.ctx.session.walkEpoch = make(map[string]uint64)
		}
		t.ctx.session.walkEpoch[t.actorID]++
		t.epoch = t.ctx.session.walkEpoch[t.actorID]
		if t.straight {
			t.inner = engine.NewStraightWalkTo(actor, t.x, t.y, nil)
		} else {
			t.inner = engine.NewWalkTo(t.ctx.session.scene, actor, t.x, t.y, nil)
		}
	}
	if t.inner == nil {
		return true
	}
	if t.ctx.session.walkEpoch[t.actorID] != t.epoch {
		return true
	}
	done := t.inner.Update(dt)
	if t.perspective && t.actor != nil {
		engine.UpdateActorPerspective(t.ctx.session.scene, t.actor)
	}
	return done
}

// Face orients actor to face the given direction immediately.
func (c *Context) Face(actorID string, facing engine.FacingDirection) engine.Task {
	actor, ok := c.actor(actorID)
	if !ok {
		return engine.Immediate(func() {})
	}
	return engine.Face(actor, facing)
}

// Say plays a real recorded line (basename.WAV + optional basename.ACS)
// with a localized subtitle, per agents/IMPLEMENTATION.md "Speech". The
// subtitle becomes available via Session.CurrentSubtitle while the line
// plays. Playback starts on the returned Task's first Update, not at
// construction -- so a Sequence of (video, then Say) does not fire the
// line under the still-playing cutscene (see
// agents/scenes/116_TO_7A_FIRST_ENCOUNTER_SCRIPT.MD section 4).
func (c *Context) Say(actorID, baseName, subtitle string) engine.Task {
	actor, ok := c.actor(actorID)
	if !ok {
		return engine.Immediate(func() {})
	}

	return &dialogueTask{session: c.session, actor: actor, baseName: baseName, subtitle: subtitle}
}

// PlayVoiceover plays a recorded line (baseName+".WAV") with a localized
// subtitle, for scenes with no on-screen Actor to attribute it to -- e.g.
// Location 35's glider sequence (agents/scenes/02_DRAGON_BLASTER_DELUXE.MD),
// whose presentation is entirely full-screen video layers. Unlike Say, this
// never touches an Actor's talk state or mouth animation, and has no ACS
// lip-sync track.
func (c *Context) PlayVoiceover(baseName, subtitle string) engine.Task {
	sound, err := engine.LoadSoundByBaseName(c.session.idx, baseName+".WAV")
	if err != nil {
		log.Printf("game: PlayVoiceover(%s): %v", baseName, err)
		return engine.Immediate(func() {})
	}

	return &voiceoverTask{session: c.session, sound: sound, subtitle: subtitle}
}

// voiceoverTask plays sound to completion, showing subtitle via
// Session.CurrentSubtitle while it plays (mirroring dialogueTask, minus the
// Actor-driven mouth animation Say's SpeechTask has).
type voiceoverTask struct {
	session  *Session
	sound    *audio.Sound
	subtitle string

	started  bool
	handle   audio.Handle
	elapsed  float64
	duration float64
}

func (t *voiceoverTask) Update(dt float64) bool {
	if !t.started {
		t.started = true
		t.handle = t.session.audioEngine.Play(t.sound, audio.CategorySpeech)
		t.duration = t.sound.Duration()
		t.session.currentSubtitle = t.subtitle
	}

	t.elapsed += dt
	if t.duration <= 0 || t.elapsed >= t.duration {
		t.handle.Stop()
		t.session.currentSubtitle = ""
		return true
	}
	return false
}

// PlaySpeechBoundToLayer plays baseName as speech while looping layerID's
// frames cyclically within [startFrame, endFrame] for exactly as long as
// the speech lasts, per agents/scenes/S1_ALPHA_AND_SYNC.MD sections 15-20:
// the original's generic speech system can "drive/own an associated
// animation object+range" rather than always being an independent, unbound
// voice line -- e.g. Location 35's close-up lines
// (011_ROD_01/02/03), explicitly bound to S1_RodToCloseup's own talk range.
// Completion is gated on the speech, not the animation (there is no
// authored "end" to a talk loop otherwise). layerID's own normal per-tick
// Advance is suspended for the duration (engine.Layer.TaskDriven) so
// nothing double-advances it (section 17's "do not double-advance
// speech-bound animation").
func (c *Context) PlaySpeechBoundToLayer(layerID, baseName, subtitle string, startFrame, endFrame int) engine.Task {
	return &deferredSpeechBoundAnimationTask{
		ctx:        c,
		layerID:    layerID,
		baseName:   baseName,
		subtitle:   subtitle,
		startFrame: startFrame,
		endFrame:   endFrame,
	}
}

// deferredSpeechBoundAnimationTask resolves the associated animation when the
// speech actually starts. Some original scene controllers keep direct pointers
// to animation objects that are not necessarily emitted as ordinary parsed SZN
// layers; resolving at task construction time would incorrectly turn those
// lines into permanent voiceovers before preceding scene setup has run.
type deferredSpeechBoundAnimationTask struct {
	ctx        *Context
	layerID    string
	baseName   string
	subtitle   string
	startFrame int
	endFrame   int
	inner      engine.Task
	started    bool
}

func (t *deferredSpeechBoundAnimationTask) Update(dt float64) bool {
	if !t.started {
		t.started = true

		layer, ok := t.ctx.layer(t.layerID)
		if !ok {
			var err error
			layer, _, err = t.ctx.ensureAssetLayer(t.layerID)
			if err != nil {
				log.Printf("game: PlaySpeechBoundToLayer(%s): unknown layer %q: %v", t.baseName, t.layerID, err)
				t.inner = t.ctx.PlayVoiceover(t.baseName, t.subtitle)
				return t.inner.Update(dt)
			}
		}

		var track *acs.Track
		if acsPath, err := t.ctx.session.idx.ResolveBaseName(t.baseName + ".ACS"); err == nil {
			if data, err := os.ReadFile(acsPath); err == nil {
				if parsed, err := acs.Parse(data); err == nil {
					track = parsed
				}
			}
		}

		t.inner = &speechBoundAnimationTask{
			layer:   layer,
			start:   t.startFrame,
			end:     t.endFrame,
			voice:   t.ctx.PlayVoiceover(t.baseName, t.subtitle),
			track:   track,
			natural: t.startFrame == -1 && t.endFrame == -1,
		}
	}

	if t.inner == nil {
		return true
	}
	return t.inner.Update(dt)
}

// speechBoundAnimationTask is PlaySpeechBoundToLayer's driver: it owns
// layer.Frame for its own duration, cycling [start,end] at the layer's own
// authored FPS, and completes exactly when voice does.
type speechBoundAnimationTask struct {
	layer   *engine.Layer
	start   int
	end     int
	voice   engine.Task
	track   *acs.Track
	natural bool

	started bool
	elapsed float64
}

func (t *speechBoundAnimationTask) Update(dt float64) bool {
	if !t.started {
		t.started = true
		t.layer.Visible = true
		t.layer.Enabled = true
		t.layer.TaskDriven = true
		if !t.natural {
			t.layer.Frame = t.start
			t.layer.Accumulator = 0
		}
	}

	t.elapsed += dt
	start, end := t.start, t.end
	if t.natural {
		start = 0
		end = 0
		if t.layer.Source != nil {
			end = t.layer.Source.Frames() - 1
		}
	}
	if t.track != nil && !t.natural {
		fps := t.layer.FPS
		if fps <= 0 {
			fps = int(math.Round(t.track.Rate))
		}
		if fps <= 0 {
			fps = 20
		}
		t.layer.Accumulator += dt
		frameDuration := 1.0 / float64(fps)
		if t.layer.Accumulator >= frameDuration {
			for t.layer.Accumulator >= frameDuration {
				t.layer.Accumulator -= frameDuration
			}
			mouth := t.track.MouthStateAt(t.elapsed)
			frame := start
			if mouth != 0xffff {
				frame = int(mouth)
			}
			if t.layer.Source == nil || (frame >= 0 && frame < t.layer.Source.Frames()) {
				t.layer.Frame = frame
			}
		}
	} else {
		fps := t.layer.FPS
		if fps <= 0 && end > start {
			fps = 20
		}
		if fps > 0 && end > start {
			frameDuration := 1.0 / float64(fps)
			t.layer.Accumulator += dt
			frames := 0
			if t.natural && t.layer.Source != nil {
				frames = t.layer.Source.Frames()
			}
			for t.layer.Accumulator >= frameDuration {
				t.layer.Accumulator -= frameDuration
				t.layer.Frame++
				if t.layer.Frame > end || (t.natural && frames > 0 && t.layer.Frame >= frames) || t.layer.Frame < start {
					t.layer.Frame = start
				}
			}
		}
	}

	if t.voice.Update(dt) {
		reset := t.start
		if reset < 0 {
			reset = 0
		}
		t.layer.Frame = reset
		t.layer.Accumulator = 0
		t.layer.TaskDriven = false
		return true
	}
	return false
}

// CompleteLocation signals that this session's location has finished (see
// Session.Done): the caller driving the session is responsible for noticing
// and moving on to whatever comes next.
func (c *Context) CompleteLocation() engine.Task {
	return engine.Immediate(func() { c.session.done = true })
}

// SayText shows a subtitle for a fixed duration without audio. Used for
// scripted lines that have no corresponding recorded WAV (matching a real
// line to invented puzzle dialogue requires the original dialogue
// script/disassembly, which is Milestone 6 content-porting work).
func (c *Context) SayText(actorID, text string, seconds float64) engine.Task {
	actor, _ := c.actor(actorID)

	return engine.Sequence(
		engine.Immediate(func() {
			if actor != nil {
				actor.State = engine.ActorTalking
			}
			c.session.currentSubtitle = text
		}),
		engine.Wait(seconds),
		engine.Immediate(func() {
			if actor != nil {
				actor.State = engine.ActorIdle
			}
			c.session.currentSubtitle = ""
		}),
	)
}

// PlaySFX starts name (resolved through the current scene's asset index) on
// the SFX category and returns immediately (fire-and-forget).
func (c *Context) PlaySFX(name string) engine.Task {
	sound, err := engine.LoadSound(c.session.idx, name)
	if err != nil {
		log.Printf("game: PlaySFX(%s): %v", name, err)
		return engine.Immediate(func() {})
	}

	return engine.Immediate(func() {
		c.session.audioEngine.Play(sound, audio.CategorySFX)
	})
}

func (c *Context) PlaySFXVolume(name string, volume int) engine.Task {
	sound, err := engine.LoadSound(c.session.idx, name)
	if err != nil {
		log.Printf("game: PlaySFXVolume(%s): %v", name, err)
		return engine.Immediate(func() {})
	}
	if volume < 1 {
		volume = 1
	}
	if volume > 200 {
		volume = 200
	}
	mono := make([]int16, len(sound.Mono))
	for i, sample := range sound.Mono {
		v := int32(sample) * int32(volume) / 100
		if v > 32767 {
			v = 32767
		}
		if v < -32768 {
			v = -32768
		}
		mono[i] = int16(v)
	}
	scaled := &audio.Sound{SampleRate: sound.SampleRate, Mono: mono}
	return engine.Immediate(func() {
		c.session.audioEngine.Play(scaled, audio.CategorySFX)
	})
}

// PlayMusic starts name (resolved through the current scene's asset index)
// looping on the Music category and returns immediately (fire-and-forget).
// Used for a location's ambient background bed (e.g. Location 35's
// Loc35_RodrigoOnGlider.wav): it must keep playing for as long as the
// player stays there, restarting from the beginning every time it reaches
// the end, not fall silent once the clip's own (much shorter) runtime
// elapses -- see audio.Engine.PlayLooping.
func (c *Context) PlayMusic(name string) engine.Task {
	sound, err := engine.LoadSound(c.session.idx, name)
	if err != nil {
		log.Printf("game: PlayMusic(%s): %v", name, err)
		return engine.Immediate(func() {})
	}

	return engine.Immediate(func() {
		s := c.session
		if s.musicHandle != nil && strings.EqualFold(s.musicName, name) {
			s.musicLocation = s.state.Location
			return
		}
		if s.musicHandle != nil {
			s.musicHandle.Stop()
		}
		s.musicHandle = s.audioEngine.PlayLooping(sound, audio.CategoryMusic)
		s.musicName = name
		s.musicLocation = s.state.Location
	})
}

// PlayLayer plays layerID's animation once and waits for it to finish, per
// agents/GAMEPLAY.md's PlayLayerOnce example.
func (c *Context) PlayLayer(layerID string) engine.Task {
	return &deferredPlayLayerTask{ctx: c, layerID: layerID}
}

type deferredPlayLayerTask struct {
	ctx     *Context
	layerID string
	inner   engine.Task
	started bool
}

func (t *deferredPlayLayerTask) Update(dt float64) bool {
	if !t.started {
		t.started = true
		layer, ok := t.ctx.layer(t.layerID)
		if !ok {
			log.Printf("game: unknown layer %q", t.layerID)
			return true
		}
		t.inner = engine.PlayLayerOnce(layer)
	}
	if t.inner == nil {
		return true
	}
	return t.inner.Update(dt)
}

// PlayLayerSplit plays layerID's animation once, like PlayLayer, but pauses
// exactly at splitFrame to fire sfxName before continuing on to the last
// frame. Used for an authored two-part animation with a sound cue in the
// middle -- e.g. Scene 7B's elevator lever, which plays frames 0x00-0x13,
// Sfx_ClickelDiClack fires, then 0x14-end plays (see
// agents/scenes/7B.MD "Elevator UP"/"Elevator DOWN").
func (c *Context) PlayLayerSplit(layerID string, splitFrame int, sfxName string) engine.Task {
	layer, ok := c.layer(layerID)
	if !ok {
		log.Printf("game: unknown layer %q", layerID)
		return engine.Immediate(func() {})
	}

	return engine.Sequence(
		engine.Immediate(func() {
			layer.Mode = engine.AnimOnce
			layer.Frame = 0
			layer.Accumulator = 0
			layer.Playing = true
			layer.TaskDriven = true
		}),
		&layerRangeTask{layer: layer, target: splitFrame, freeze: false},
		c.PlaySFX(sfxName),
		&layerRangeTask{layer: layer, target: -1, freeze: true},
	)
}

// PlayLayerFrames plays layerID's animation once from frame `from` through
// frame `to` inclusive -- unlike PlayLayer/PlayLayerSplit, not necessarily
// starting at 0 -- for an authored partial-range cue, e.g. Scene 46's
// KanalTalk conversation animation played across frames 5-9 while a line
// plays (see agents/scenes/46.MD "animation range 5 -> 9").
// PlayLayerFrames supports a reverse range (to < from, and to >= 0) exactly
// like a forward one, per
// agents/scenes/S1_CLOSEUP_ALPHA_FLICKER_FIXES.MD sections 4/5/26: e.g.
// Location 35's close-up return plays S1_RodToCloseup backward from frame
// 40 to frame 1. Direction is inferred from comparing the two endpoints,
// never sorted/normalized -- doing so would silently turn an authored
// reverse range into a forward one. to==-1 keeps its existing meaning,
// "play forward to the source's real last frame", regardless of from.
func (c *Context) PlayLayerFrames(layerID string, from, to int) engine.Task {
	layer, ok := c.layer(layerID)
	if !ok {
		log.Printf("game: unknown layer %q", layerID)
		return engine.Immediate(func() {})
	}

	reverse := to >= 0 && to < from

	return engine.Sequence(
		engine.Immediate(func() {
			layer.Visible = true
			layer.Enabled = true
			if reverse {
				layer.Mode = engine.AnimOnceReverse
			} else {
				layer.Mode = engine.AnimOnce
			}
			layer.Frame = from
			layer.Accumulator = 0
			layer.Playing = true
			layer.TaskDriven = true
		}),
		&layerRangeTask{layer: layer, target: to, freeze: true, reverse: reverse},
	)
}

// layerRangeTask advances layer (already started, mirroring
// engine.PlayLayerOnce's own setup) until it reaches target frame or the
// source's first/last frame (depending on direction), whichever comes
// first; target < 0 means "play to the last frame" (forward only -- a
// reverse range always has a real target, see PlayLayerFrames). reverse
// mirrors layer.Mode being engine.AnimOnceReverse, decrementing toward 0
// instead of incrementing toward frames-1. freeze matches PlayLayerOnce's
// completion behavior (setting Playing = false once done) for a sequence's
// final range, but is left false for an earlier range that immediately
// continues into more of the same animation right after (see
// PlayLayerSplit).
type layerRangeTask struct {
	layer   *engine.Layer
	target  int
	freeze  bool
	reverse bool
}

func (t *layerRangeTask) Update(dt float64) bool {
	if t.layer.Source == nil {
		return true
	}

	frames := t.layer.Source.Frames()
	if frames <= 1 {
		return true
	}

	t.layer.AdvanceScripted(dt)

	var done bool
	if t.reverse {
		done = t.layer.Frame <= 0 || (t.target >= 0 && t.layer.Frame <= t.target)
	} else {
		done = t.layer.Frame >= frames-1 || (t.target >= 0 && t.layer.Frame >= t.target)
	}

	if done && t.freeze {
		t.layer.Playing = false
		t.layer.TaskDriven = false
	}
	return done
}

// PlaceActor immediately sets actorID's position with no walking animation
// -- for an arrival cinematic where the character is already at a fixed
// spot when the scene becomes current, e.g. stepping out of an elevator
// that arrived while the destination scene's Enter hook runs (see
// agents/scenes/7B.MD "Returning from Scene 9/46 to 7B").
func (c *Context) PlaceActor(actorID string, x, y float64) engine.Task {
	actor, ok := c.actor(actorID)
	if !ok {
		return engine.Immediate(func() {})
	}
	return engine.Immediate(func() {
		actor.X, actor.Y = x, y
	})
}

func (c *Context) PlaceActorPerspective(actorID string, x, y float64) engine.Task {
	actor, ok := c.actor(actorID)
	if !ok {
		return engine.Immediate(func() {})
	}
	return engine.Immediate(func() {
		actor.X, actor.Y = x, y
		engine.UpdateActorPerspective(c.session.scene, actor)
	})
}

// FreezeLayer pins layerID at frame with no animation, for a "special
// action" prop (e.g. a lever) that must stay dormant until a script plays
// it via PlayLayer, per agents/GAMEPLAY.md "Scene entry state
// reconstruction".
func (c *Context) FreezeLayer(layerID string, frame int) engine.Task {
	layer, ok := c.layer(layerID)
	if !ok {
		log.Printf("game: unknown layer %q", layerID)
		return engine.Immediate(func() {})
	}
	return engine.FreezeLayer(layer, frame)
}

// ShowLayer makes layerID visible immediately.
func (c *Context) ShowLayer(layerID string) engine.Task {
	return engine.Immediate(func() {
		if l, ok := c.layer(layerID); ok {
			l.Visible = true
			l.Enabled = true
		}
	})
}

// HideLayer hides layerID immediately.
func (c *Context) HideLayer(layerID string) engine.Task {
	return engine.Immediate(func() {
		if l, ok := c.layer(layerID); ok {
			l.Visible = false
		}
	})
}

// SetLayerZ overrides layerID's authored Z immediately, for the rare case
// where a scene's own SZN data conflicts with this engine's established
// layer-vs-layer painter-order convention (ascending Z, drawn back to
// front) -- see agents/scenes/S1.MD: Location 35's full-screen background
// videos (S1_Back/S1_Back2/S1_BackToCloseup/S1_CloseupToBack) are declared
// with Z=255, the highest in the scene, which would draw them *last* (in
// front of, and so hiding, every character-animation layer at Z=1-2) under
// that convention -- backwards for what are unambiguously background
// layers. Rather than invert the convention globally (unproven and risky
// for every other scene that already relies on it, e.g. the Scene 7B
// railing occlusion fix), this corrects just the specific layers whose
// authored Z is demonstrably wrong for this one scene.
func (c *Context) SetLayerZ(layerID string, z int) engine.Task {
	return engine.Immediate(func() {
		if l, ok := c.layer(layerID); ok {
			l.Z = z
		}
	})
}

// HideActor takes actorID out of normal scene rendering immediately: it is
// skipped by both drawing and depth sorting (see engine.SceneDrawOrder)
// until ShowActor. Used before a scripted sequence that already depicts the
// character itself (a lever animation, a cinematic) so the live sprite and
// that scripted representation don't both render at once -- a duplicate/
// "ghost" actor -- per
// agents/techniques/ELEVATOR-TRANSITIONS-AND-ZBUFFER.MD sections 1/2/11.
// Since a scene transition always instantiates a fresh Scene (and thus a
// fresh, visible-by-default Actor) for the destination, a script that hides
// the departing scene's actor and then changes scenes never needs to call
// ShowActor itself.
func (c *Context) HideActor(actorID string) engine.Task {
	return engine.Immediate(func() {
		if a, ok := c.actor(actorID); ok {
			a.Visible = false
		}
	})
}

// ShowActor restores actorID to normal scene rendering immediately, undoing
// HideActor.
func (c *Context) ShowActor(actorID string) engine.Task {
	return engine.Immediate(func() {
		if a, ok := c.actor(actorID); ok {
			a.Visible = true
		}
	})
}

// SetActorDirection sets actorID's raw Direction value immediately. This is
// metadata only: nothing in the renderer currently reads Actor.Direction
// (on-screen facing/animation comes from the separately engine-computed
// Actor.Facing instead -- see engine.ClassifyFacing), so this has no
// visible effect today. It exists to preserve an authored numeric
// direction/state value a transition anchor documents -- e.g.
// agents/techniques/ELEVATOR-TRANSITIONS-AND-ZBUFFER.MD's elevator anchors
// -- whose exact cardinal meaning and consumer are not yet source-proven;
// per that doc, such values must be preserved verbatim rather than guessed
// into engine.FacingDirection.
func (c *Context) SetActorDirection(actorID string, direction int) engine.Task {
	return engine.Immediate(func() {
		if a, ok := c.actor(actorID); ok {
			a.Direction = direction
			a.SetFacingImmediate(engine.FacingFromDirectionValue(direction))
		}
	})
}

func (c *Context) SetActorOrientation(actorID string, direction int) engine.Task {
	actor, ok := c.actor(actorID)
	if !ok {
		return engine.Immediate(func() {})
	}
	return &orientActorTask{actor: actor, direction: direction}
}

type orientActorTask struct {
	actor     *engine.Actor
	direction int
}

func (t *orientActorTask) Update(dt float64) bool {
	t.actor.Direction = t.direction
	return t.actor.StepFacing(engine.FacingFromDirectionValue(t.direction))
}

// EnableArea enables areaID's hotspot immediately.
func (c *Context) EnableArea(areaID string) engine.Task {
	return engine.Immediate(func() {
		if a, ok := c.session.scene.Areas[areaID]; ok {
			a.Enabled = true
		}
	})
}

// DisableArea disables areaID's hotspot immediately.
func (c *Context) DisableArea(areaID string) engine.Task {
	return engine.Immediate(func() {
		if a, ok := c.session.scene.Areas[areaID]; ok {
			a.Enabled = false
		}
	})
}

// MakeLayerClickable registers layerID as a clickable hotspot spanning its
// current on-screen rectangle (X, Y and its sprite's Width/Height at Zoom),
// for interactive scene-dressing that has no author-declared Area of its own
// in the SZN data -- e.g. Scene 7A's GliderLayer, clicked to start the
// glider departure (see agents/scenes/7A-TRANSITIONS-OFFICE-GLIDER.MD
// section 5). Reuses the Area mechanism wholesale (Controller.Click dispatch,
// DisableArea/EnableArea, AreaCenter, debug-area rendering) rather than
// inventing a parallel "clickable layer" hit-test path. The rectangle is
// computed once, at registration time -- fine for GliderLayer, whose bob
// animation (SMOOTH-TRANSITIONS-ZBUFFER-GLIDER.MD) only ever moves it by a
// couple of pixels, but a layer that moves substantially afterward would
// need this called again to stay accurate.
func (c *Context) MakeLayerClickable(layerID string) engine.Task {
	return engine.Immediate(func() {
		layer, ok := c.session.scene.Layers[layerID]
		if !ok || layer.Source == nil {
			log.Printf("game: MakeLayerClickable(%s): layer not found", layerID)
			return
		}

		w := layer.Source.Width() * layer.Zoom / 100
		h := layer.Source.Height() * layer.Zoom / 100

		area := &engine.Area{
			ID:         layerID,
			X1:         layer.X,
			Y1:         layer.Y,
			X2:         layer.X + w,
			Y2:         layer.Y + h,
			CursorType: 10,
			Enabled:    true,
		}
		c.session.localizeArea(area, c.session.state.Location)
		c.session.scene.Areas[layerID] = area
		c.session.scene.AreaOrder = append(c.session.scene.AreaOrder, layerID)
	})
}

// AreaCenter returns the geometric center of areaID's rectangle, a
// convenient WalkTo target for "walk up to this hotspot" sequences.
func (c *Context) AreaCenter(areaID string) (x, y float64, ok bool) {
	area, exists := c.session.scene.Areas[areaID]
	if !exists {
		return 0, 0, false
	}
	minX, minY, maxX, maxY := area.Normalized()
	return float64(minX+maxX) / 2, float64(minY+maxY) / 2, true
}

// HasItem reports whether the player holds item id.
func (c *Context) HasItem(id int) bool {
	return c.session.state.HasItem(id)
}

// AddItem adds item id to the player's inventory.
func (c *Context) AddItem(id int) engine.Task {
	return engine.Immediate(func() { c.session.state.AddItem(id) })
}

// RemoveItem removes item id from the player's inventory.
func (c *Context) RemoveItem(id int) engine.Task {
	return engine.Immediate(func() { c.session.state.RemoveItem(id) })
}

// GetFlag returns the named flag's value (0 if unset).
func (c *Context) GetFlag(name string) int {
	return c.session.state.Flags[name]
}

// SetFlag sets the named flag's value immediately.
func (c *Context) SetFlag(name string, value int) engine.Task {
	return engine.Immediate(func() {
		if c.session.state.Flags == nil {
			c.session.state.Flags = map[string]int{}
		}
		c.session.state.Flags[name] = value
	})
}

// SetBackground swaps the current scene's background image in place (e.g.
// selecting an _MK/_OK state alternative per agents/SZN.md), keeping its
// depth/walk rasters untouched.
func (c *Context) SetBackground(filename string) engine.Task {
	return engine.Immediate(func() {
		src, err := engine.LoadSpriteSource(c.session.idx, filename)
		if err != nil {
			log.Printf("game: SetBackground(%s): %v", filename, err)
			return
		}
		c.session.scene.Background.Source = src
	})
}

func (c *Context) SetBackgroundByStem(stem string) engine.Task {
	return engine.Immediate(func() {
		src, _, err := engine.LoadSpriteSourceByStem(c.session.idx, stem)
		if err != nil {
			log.Printf("game: SetBackgroundByStem(%s): %v", stem, err)
			return
		}
		if c.session.scene == nil || c.session.scene.Background == nil {
			return
		}
		c.session.scene.Background.Source = src
	})
}

// Wait completes after seconds have elapsed.
func (c *Context) Wait(seconds float64) engine.Task {
	return engine.Wait(seconds)
}

// RunAmbient registers task to run on every Session.Update alongside
// whatever else is happening, regardless of Locked() -- unlike a task
// returned from Enter/Click/UseItem, it is never awaited for completion and
// never gates the interaction lock. Used for a scene behavior that must run
// continuously and indefinitely once started, independent of any scripted
// action that happens to be locking normal input at the time -- e.g. Scene
// 7A's GliderLayer bob loop (see
// agents/techniques/SMOOTH-TRANSITIONS-ZBUFFER-GLIDER.MD sections 15-21).
// Ambient tasks are cleared automatically on every scene transition; call
// this again from the new scene's Enter hook if it needs the same behavior.
// RunAmbient itself completes immediately (it only registers task); it is
// safe to include in an ordinary Sequence without blocking it.
func (c *Context) RunAmbient(task engine.Task) engine.Task {
	return engine.Immediate(func() {
		c.session.ambientTasks = append(c.session.ambientTasks, task)
	})
}

// ChangeScene transitions to sceneID, preserving PreviousScene, then runs
// the destination controller's Enter hook.
func (c *Context) ChangeScene(sceneID string) engine.Task {
	return &changeSceneTask{session: c.session, target: sceneID}
}

// ChangeLocation transitions to sceneID in an entirely different location --
// a different asset directory and Controller, not just a different scene
// within the current one (see ChangeScene) -- via the session's configured
// location resolver (see Session.SetLocationResolver). Used for
// doors/exits that cross location boundaries, e.g. Scene 9's 009_Tuer
// leading to Location 5 / Scene S10 (agents/scenes/9.MD section 9). Logs
// and does nothing if no resolver was configured, or if it fails.
func (c *Context) ChangeLocation(location int, sceneID string) engine.Task {
	return &changeLocationTask{session: c.session, location: location, sceneID: sceneID}
}

// PlayVideo plays an AVI cutscene (see internal/video and
// agents/IMPLEMENTATION.md "AVI strategy") full-screen over the current
// scene until it finishes. aviFilename is resolved through the scene's
// asset index, e.g. "LIFTMITTERAUF.AVI".
//
// The cutscene is implemented as a temporary top-Z scene Layer played once
// via PlayLayerOnce, reusing the existing render pipeline rather than
// adding a separate video-drawing path. Like every other Context method,
// all of this is deferred until the returned Task's first Update rather
// than done at construction time -- a caller building a Sequence of
// several PlayVideo calls up front (see intro.go) must not have every one
// of them fight over the same fixed layer ID before the first one even
// starts playing.
func (c *Context) PlayVideo(aviFilename string) engine.Task {
	return &videoTask{session: c.session, aviFilename: aviFilename}
}

// PlayVideoSplit plays aviFilename like PlayVideo, but pauses exactly at
// splitFrame to fire sfxName before continuing to the last frame -- e.g.
// Scene 46's canal-crossing camera clips, which play part of the clip, a
// clothes-scratching sound, then the rest (see agents/scenes/46.MD
// sections 4.2 and 10).
func (c *Context) PlayVideoSplit(aviFilename string, splitFrame int, sfxName string) engine.Task {
	return &videoTask{session: c.session, aviFilename: aviFilename, split: true, splitFrame: splitFrame, splitAction: c.PlaySFX(sfxName)}
}

// videoTask starts aviFilename playing (creating and attaching its layer)
// on its first Update, drives it, and removes it from the scene once
// playback finishes (or immediately, if ChangeScene has already replaced
// the scene entirely by then).
type videoTask struct {
	session     *Session
	aviFilename string

	// split and splitFrame configure a mid-playback pause at an exact
	// frame; split is false for a plain PlayVideo. splitAction runs once
	// the video reaches splitFrame, and playback resumes to the last frame
	// only once splitAction itself completes -- for PlayVideoSplit this is
	// a one-tick PlaySFX, but it can be any Task, e.g. Scene 116's own
	// speech-completion gate (see intro.go's scene116 handling of
	// agents/scenes/116.MD section 6, "004_PRI_01 is completion-gated").
	split       bool
	splitFrame  int
	splitAction engine.Task

	started  bool
	finished bool
	inner    engine.Task // nil if it never started, or already failed/finished
}

func (t *videoTask) Update(dt float64) bool {
	if !t.started {
		t.started = true
		t.inner = t.start()
	}

	if t.inner == nil {
		return true
	}

	if t.inner.Update(dt) {
		t.finish()
		return true
	}
	return false
}

// finishNow immediately completes playback as if it had reached its last
// frame, without needing to actually feed it more time. Used by the
// intro's skip handling (see skippableTask): fast-forwarding by repeatedly
// calling Update would otherwise let a concurrently-running, time-based
// side effect (e.g. the ocean ambience loop) fire further real audio
// before "catching up", audibly bleeding into the next part.
func (t *videoTask) finishNow() {
	if !t.started || t.inner == nil {
		return // never actually started; nothing to clean up
	}
	t.finish()
}

// finish removes this video's temporary layer from the scene. Guarded by
// finished so calling both Update (to its natural end) and finishNow is
// safe and idempotent.
func (t *videoTask) finish() {
	if t.finished {
		return
	}
	t.finished = true

	delete(t.session.scene.Layers, videoLayerID)
	for i, id := range t.session.scene.LayerOrder {
		if id == videoLayerID {
			t.session.scene.LayerOrder = append(t.session.scene.LayerOrder[:i], t.session.scene.LayerOrder[i+1:]...)
			break
		}
	}
}

// start resolves and loads aviFilename, attaches its layer to the scene,
// and begins playback, returning the inner Task that drives it -- or nil
// if it could not be started (logged, and treated as already complete).
func (t *videoTask) start() engine.Task {
	path, err := t.session.idx.Resolve(t.aviFilename)
	if err != nil {
		log.Printf("game: PlayVideo(%s): %v", t.aviFilename, err)
		return nil
	}

	src, err := video.Load(path)
	if err != nil {
		log.Printf("game: PlayVideo(%s): %v", t.aviFilename, err)
		return nil
	}

	layer := &engine.Layer{
		ID:      videoLayerID,
		Source:  src,
		X:       0,
		Y:       0,
		Z:       1 << 30, // always on top, covering the whole scene viewport
		Zoom:    100,
		FPS:     int(math.Round(src.FPS())),
		Visible: true,
		Enabled: true,
		// Active presentation object (ReturnHomeCam, StairHomeUpCam, …): high
		// Z within the normal depth sort until playback completes — not a
		// separate hide-actor hack (see
		// agents/scenes/116_TO_7A_CORRECT_DEC2_RUN_PATH.MD sections 5/6).
		// config.ini's stretch_video: fill the whole logical window
		// (including the area normally reserved for the inventory strip)
		// instead of the cutscene's native size within the scene viewport.
		FillWindow:   t.session.stretchVideo,
		Presentation: true,
	}

	t.session.scene.Layers[layer.ID] = layer
	t.session.scene.LayerOrder = append(t.session.scene.LayerOrder, layer.ID)

	if !t.split {
		return engine.PlayLayerOnce(layer)
	}

	return engine.Sequence(
		engine.Immediate(func() {
			layer.Mode = engine.AnimOnce
			layer.Playing = true
			layer.TaskDriven = true
		}),
		&layerRangeTask{layer: layer, target: t.splitFrame, freeze: false},
		t.splitAction,
		&layerRangeTask{layer: layer, target: -1, freeze: true},
	)
}

// dialogueTask wraps a SpeechTask so its subtitle is visible through
// Session.CurrentSubtitle while it plays. If inner is nil, NewSpeech is
// deferred until the first Update (see Context.Say); Session.Speak sets
// inner at construction because that CLI path starts the line immediately.
type dialogueTask struct {
	session  *Session
	actor    *engine.Actor
	baseName string
	subtitle string
	inner    *engine.SpeechTask
}

func (t *dialogueTask) Update(dt float64) bool {
	if t.inner == nil {
		if t.actor == nil {
			return true
		}
		speech, err := engine.NewSpeech(t.session.idx, t.session.audioEngine, t.actor, t.baseName, t.subtitle)
		if err != nil {
			log.Printf("game: Say(%s): %v", t.baseName, err)
			return true
		}
		t.inner = speech
	}

	done := t.inner.Update(dt)
	if done {
		t.session.currentSubtitle = ""
	} else {
		t.session.currentSubtitle = t.inner.Subtitle()
	}
	return done
}

// changeSceneTask performs the scene swap on its first Update, then
// delegates to the destination controller's Enter task (if any) until it
// completes.
type changeSceneTask struct {
	session *Session
	target  string
	started bool
	inner   engine.Task
}

func (t *changeSceneTask) Update(dt float64) bool {
	if !t.started {
		t.started = true

		inner, err := t.session.transitionTo(t.target)
		if err != nil {
			log.Printf("game: ChangeScene(%s): %v", t.target, err)
			return true
		}

		t.inner = inner
	}

	if t.inner == nil {
		return true
	}

	return t.inner.Update(dt)
}

// changeLocationTask resolves location's asset index/Controller via the
// session's configured resolver on its first Update, performs the
// cross-location scene swap (see Session.TransitionToLocation), then
// delegates to the destination controller's Enter task (if any) until it
// completes.
type changeLocationTask struct {
	session  *Session
	location int
	sceneID  string
	started  bool
	inner    engine.Task
}

func (t *changeLocationTask) Update(dt float64) bool {
	if !t.started {
		t.started = true

		var idx *assets.Index
		var controller Controller
		var err error
		if t.session.resolveLocation != nil {
			idx, controller, err = t.session.resolveLocation(t.location)
		}
		if t.session.resolveLocation == nil || err != nil {
			fallbackIdx, fallbackController, supported, fallbackErr := resolveBuiltinLocation(t.location)
			if supported {
				if fallbackErr != nil {
					log.Printf("game: ChangeLocation(location=%d, scene=%s): %v", t.location, t.sceneID, fallbackErr)
					return true
				}
				idx, controller, err = fallbackIdx, fallbackController, nil
			} else if t.session.resolveLocation == nil {
				log.Printf("game: ChangeLocation(location=%d, scene=%s): no location resolver configured (see Session.SetLocationResolver)", t.location, t.sceneID)
				return true
			}
		}
		if err != nil {
			log.Printf("game: ChangeLocation(location=%d, scene=%s): %v", t.location, t.sceneID, err)
			return true
		}

		inner, err := t.session.TransitionToLocation(idx, controller, t.location, t.sceneID)
		if err != nil {
			log.Printf("game: ChangeLocation(location=%d, scene=%s): %v", t.location, t.sceneID, err)
			return true
		}

		t.inner = inner
	}

	if t.inner == nil {
		return true
	}

	return t.inner.Update(dt)
}
