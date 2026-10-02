package game

import (
	"log"

	"github.com/wok/rent-a-hero/internal/assets"
	"github.com/wok/rent-a-hero/internal/audio"
	"github.com/wok/rent-a-hero/internal/engine"
)

// introPart is one cinematic segment of the Location 40 launch intro, as
// reconstructed from _game/dec.c's FUN_00431920 (see
// agents/scenes/01_INTRO.MD). scene is the original scene ID (logged the
// same way as any other scene change, see Session.transitionTo). video is
// the full-screen cutscene; ambient is the looping background sound object
// the original loads alongside it (FUN_004400c0, named the same as the
// scene: e.g. "Loc40_IntroPart1" for Scene 114).
type introPart struct {
	scene   string
	video   string
	ambient string
}

// introSequence is the launch path's pending scene chain: the normal
// startup transition (case 3 in FUN_004045c0) sets Location 40 / Scene 114
// ("S114_IntroPart1"), and FUN_00431920 shows the same constructor
// continuing through Scene 115 ("S115_IntroPart2") and Scene 116
// ("S116_IntroPart3"). Scene 114 has its own voice-over/soundtrack timeline
// (see scene114Timeline); Scene 116 does too, per
// agents/scenes/116.MD (see scene116VoiceCues) -- that doc supersedes this
// package's earlier assumption that only Scene 114 had any. Scene 115 has
// none.
//
// Scenes 115 and 116 both name "Loc36_IntroDragonAttack" as their ambient
// object. FUN_004400c0 is a no-op when asked to (re)load the name it
// already has loaded (it only tears down and reloads on a *different*
// name), so the ambient track is not restarted between those two parts --
// it keeps playing continuously across the cut. LoadIntroTask reproduces
// that by only starting a new ambient when the name changes (this holds
// even though Scene 114's own ambient is itself replaced mid-scene, at
// 01:47.7, by scene114Timeline -- either way it's still a different name
// than "Loc36_IntroDragonAttack" once Scene 115 starts).
//
// The large FUN_00442c70/FUN_00442c10/FUN_00442bb0 tables that follow each
// branch in FUN_00431920 are per-frame SFX-cue scheduling (birds, surf,
// explosions, ...) layered on top of these cutscenes. Per 01_INTRO.MD they
// are "important for reproducing the complete intro sequence, but ... not
// the core low-level rendering primitives" and are not reproduced here. The
// one exception is Scene 114's "Sfx_Ocean" ambience, which is reproduced at
// a coarser grain by oceanPhases below, since 01_INTRO.MD calls out that
// its two on/off phases (not just one continuous loop) is a real, audible
// part of the scene rather than a single low-level cue.
var introSequence = []introPart{
	{scene: "114", video: "S114_IntroPart1.avi", ambient: "Loc40_IntroPart1.wav"},
	{scene: "115", video: "S115_IntroPart2.avi", ambient: "Loc36_IntroDragonAttack.wav"},
	{scene: "116", video: "S116_IntroPart3.avi", ambient: "Loc36_IntroDragonAttack.wav"},
}

// introFPS is S114_IntroPart1's native frame rate: 01_INTRO.MD's recovered
// cue sheet ("FUN_00442c70(scene, 0x78, ocean, ...)" etc.) times events by
// frame number into this same video, so a cue at hex frame F falls at
// F/introFPS seconds.
const introFPS = 10.0

// oceanPhases are the two "Sfx_Ocean" ambience spans 01_INTRO.MD recovers
// from Scene 114's cue sheet: an initial ambience explicitly stopped at
// frame 0x1a9, and "a second ambient/ocean playback" starting at 0x3c0 and
// stopped at 0x486 -- i.e. the ambience audibly changes partway through the
// scene rather than looping unchanged for its whole length. The many
// individual bird/explosion/etc. cues around them are not reproduced (see
// introSequence's doc comment).
var oceanPhases = [2]struct{ startFrame, endFrame int }{
	{0x000, 0x1a9},
	{0x3c0, 0x486},
}

// scene114Timeline is Scene 114's recovered voice-over/soundtrack cue
// sheet -- Ghidra's decompiled UndefinedFunction_00432ef0, the callback
// FUN_00432ca0 registers via FUN_0043fa30(param_1, LAB_00432ef0) -- which
// reads the current S114_IntroPart1.avi frame position and fires these
// events (see 01_INTRO.MD "Recovered UndefinedFunction_00432ef0: exact
// Scene-114 speech/music timeline"). Every entry has exactly one of
// speech (a one-shot recorded line) or ambient (replaces the current
// long-form/ambient audio, same as introPart.ambient) set. Scene 115 has no
// voice-over of its own; Scene 116 does (see scene116VoiceCues), per
// agents/scenes/116.MD.
var scene114Timeline = []struct {
	atFrame int
	speech  string
	ambient string
}{
	{0x00A, "016_SPR_01.wav", ""},
	{0x096, "016_SPR_02.wav", ""},
	{0x212, "016_SPR_03.wav", ""},
	{0x2E4, "021_SPR_01.wav", ""},
	{0x352, "021_SPR_02.wav", ""},
	{0x3C0, "023_SPR_01.wav", ""},
	{0x435, "", "Loc09_SmashvilleAttack.wav"}, // ~01:47.7: replaces Loc40_IntroPart1.wav
	{0x4E0, "024_ST2_01.wav", ""},
}

// scene114TimelineTask fires every scene114Timeline entry at its recorded
// frame position (S114_IntroPart1.avi runs at introFPS), in order.
func scene114TimelineTask(ia *introAudio) engine.Task {
	var tasks []engine.Task
	clock := 0.0

	for _, e := range scene114Timeline {
		at := float64(e.atFrame) / introFPS
		if gap := at - clock; gap > 0 {
			tasks = append(tasks, engine.Wait(gap))
		}

		tasks = append(tasks, engine.Immediate(func() {
			if e.speech != "" {
				ia.playNarration(e.speech)
			}
			if e.ambient != "" {
				ia.playAmbient(e.ambient)
			}
		}))

		clock = at
	}

	return engine.Sequence(tasks...)
}

// scene114SFXCues is every one-shot SFX event explicitly scheduled by the
// Scene-114 constructor (see 01_INTRO.MD "Complete Scene-114 SFX cue
// sheet"), excluding the three Sfx_Ocean state-change cues at
// 0x1a9/0x3c0/0x486 -- those are already reproduced by oceanPhases's loop
// start/stop boundaries. Per the doc's "Recreation rule": every one of
// these fires, repeated samples and alternate variants (Work_Hack vs
// Work_Hack2, BombApproaching vs BombApproaching2) are preserved rather
// than deduplicated, and "Sfx_Gong" is deliberately excluded -- it is
// loaded by the constructor but never actually scheduled in the original
// code, so playing it would be inventing a cue that was never there.
var scene114SFXCues = []struct {
	atFrame int
	clip    string
}{
	{0x0FF, "Sfx_Seagull.wav"},
	{0x109, "Sfx_Seagull.wav"},
	{0x19A, "Sfx_Bird.wav"},
	{0x1A4, "Sfx_Bird2.wav"},
	{0x1B8, "Sfx_Bird3.wav"},
	{0x1DE, "Sfx_Work_Hack.wav"},
	{0x1E4, "Sfx_Work_Hack2.wav"},
	{0x1EE, "Sfx_Work_Hack.wav"},
	{0x1F2, "Sfx_Work_Hack2.wav"},
	{0x1FA, "Sfx_Work_Hack.wav"},
	{0x204, "Sfx_Work_Hack2.wav"},
	{0x20D, "Sfx_Work_Hack.wav"},
	{0x216, "Sfx_Work_Hack2.wav"},
	{0x225, "Sfx_Work_Hack.wav"},
	{0x228, "Sfx_Work_Hack2.wav"},
	{0x235, "Sfx_Work_Hack.wav"},
	{0x23A, "Sfx_Work_Hack2.wav"},
	{0x241, "Sfx_Sword2.wav"},
	{0x256, "Sfx_Work_Hack2.wav"},
	{0x268, "Sfx_Work_Hack2.wav"},
	{0x270, "Sfx_MobileDown.wav"},
	{0x29E, "Sfx_Owl.wav"},
	{0x2BC, "Sfx_Cricket.wav"},
	{0x2C6, "Sfx_Cricket.wav"},
	{0x460, "Sfx_Seagull.wav"},
	{0x474, "Sfx_Seagull.wav"},
	{0x476, "Sfx_CannonFire.wav"},
	{0x480, "Sfx_BombApproaching.wav"},
	{0x486, "Sfx_Fire.wav"},
	{0x490, "Sfx_Explosion_Bass2.wav"},
	{0x490, "Sfx_Explosion_Misc2.wav"},
	{0x49B, "Sfx_CannonFire2.wav"},
	{0x4A5, "Sfx_BombApproaching2.wav"},
	{0x4B7, "Sfx_Explosion_Misc2.wav"},
	{0x4D2, "Sfx_CannonFire.wav"},
	{0x4DC, "Sfx_BombApproaching.wav"},
	{0x4EC, "Sfx_Explosion_Bass.wav"},
	{0x4F6, "Sfx_Fire.wav"},
}

// scene114SFXTask fires every scene114SFXCues entry at its recorded frame
// position, in order (two entries sharing a frame, e.g. 0x490's pair, fire
// together in the same tick). These are short one-shot effects -- unlike
// the ocean loop or voice-over, they are not tracked for an explicit stop
// on skip: skippableTask simply stops calling this task's Update at all
// once skipped, so no not-yet-fired cue can ever fire late, and letting an
// already-fired one-shot finish naturally is inaudible compared to a
// multi-second loop or voice line bleeding over.
func scene114SFXTask(idx *assets.Index, audioEngine audio.Engine) engine.Task {
	var tasks []engine.Task
	clock := 0.0

	for _, cue := range scene114SFXCues {
		at := float64(cue.atFrame) / introFPS
		if gap := at - clock; gap > 0 {
			tasks = append(tasks, engine.Wait(gap))
		}

		tasks = append(tasks, engine.Immediate(func() {
			sound, err := engine.LoadSound(idx, cue.clip)
			if err != nil {
				log.Printf("game: intro SFX %s: %v", cue.clip, err)
				return
			}
			audioEngine.Play(sound, audio.CategorySFX)
		}))

		clock = at
	}

	return engine.Sequence(tasks...)
}

// scene116SplitFrame is where S116_IntroPart3 must stop and hold (see
// agents/scenes/116.MD sections 2/6): the scene's own voice-over turned out
// not to belong to Scene 114 after all -- this doc supersedes introSequence
// and scene114Timeline's earlier "Scenes 115/116 have none" assumption for
// 116 specifically. Confirmed via the asset itself: S116_IntroPart3.avi is
// 140 frames at 10fps (introFPS), and 004_PRI_01.wav -- the line that gates
// the resume -- is 17.94s long, far longer than the 3.3s of video remaining
// between when it starts (frame 87) and this split point, so the video
// really does sit frozen at frame 120 waiting on it most of the time; this
// is not a guessed wall-clock delay (116.MD section 11's explicit warning
// against that), it's a real speech-completion gate.
const scene116SplitFrame = 120

// scene116VoiceCues is Scene 116's own voice-over cue sheet (116.MD section
// 3): 003_PRI_05 and 003_PRI_06 are fire-and-forget, like Scene 114's own
// lines -- "the video continues while this line plays" -- but 004_PRI_01 is
// not: nothing after it fires until it completes (see scene116SplitFrame
// and the videoTask.splitAction wired up in loadIntroPartsTask).
var scene116VoiceCues = []struct {
	atFrame int
	speech  string
}{
	{1, "003_PRI_05.wav"},
	{21, "003_PRI_06.wav"},
	{87, "004_PRI_01.wav"},
}

// scene116BackingSwapFrame is when Scene 116 replaces its ambient bed with
// Loc35_RodrigoOnGlider (116.MD section 7) -- while 004_PRI_01 (started at
// frame 87) is most likely still playing, since this frame arrives only
// 1.1s later, nowhere near that line's own 17.94s length. ia.playAmbient
// only ever touches the tracked ambient handle, never narration, so this
// swap cannot cut the active speech off (116.MD: "This backing-audio change
// must not stop the active speech channel").
const scene116BackingSwapFrame = 98

// scene116CueTask fires scene116VoiceCues in order, then the frame-98
// backing-audio swap alongside a timed Sfx_Glider_Flying cue -- the
// original's own "timed parameter/fade change" on that SFX (116.MD section
// 7) is not reproduced at that fine a grain, matching this file's existing
// treatment of the large per-frame SFX cue tables (see introSequence's own
// doc comment); firing the cue once, alongside the backing swap, is enough
// to have it present rather than silently missing.
func scene116CueTask(ia *introAudio, ctx *Context) engine.Task {
	var tasks []engine.Task
	clock := 0.0

	for _, e := range scene116VoiceCues {
		at := float64(e.atFrame) / introFPS
		if gap := at - clock; gap > 0 {
			tasks = append(tasks, engine.Wait(gap))
		}
		tasks = append(tasks, engine.Immediate(func() { ia.playNarration(e.speech) }))
		clock = at
	}

	at := float64(scene116BackingSwapFrame) / introFPS
	if gap := at - clock; gap > 0 {
		tasks = append(tasks, engine.Wait(gap))
	}
	tasks = append(tasks, engine.Sequence(
		engine.Immediate(func() { ia.playAmbient("Loc35_RodrigoOnGlider.wav") }),
		ctx.PlaySFX("Sfx_Glider_Flying.wav"),
	))

	return engine.Sequence(tasks...)
}

// waitForNarrationTask completes once introAudio's currently tracked
// narration handle (see introAudio.playNarration) finishes -- the
// speech-completion gate scene116SplitFrame's videoTask.splitAction runs,
// per agents/scenes/116.MD section 6 ("004_PRI_01 is completion-gated").
// By the time the video naturally reaches scene116SplitFrame, 004_PRI_01
// has already started (it fires at frame 87, well before frame 120's
// 12.0s), so ia.narration is always the right handle to poll here.
type waitForNarrationTask struct{ ia *introAudio }

func (t waitForNarrationTask) Update(dt float64) bool {
	return t.ia.narration == nil || !t.ia.narration.IsPlaying()
}

// NewIntroScene builds the blank scene used to host the launch intro's
// full-screen video layers. It has no background/characters/areas of its
// own -- each cutscene AVI is already the full 640x360 frame -- so it
// reuses the normal layer-composition render pipeline (render.DrawScene)
// purely as a way to play a sequence of full-screen videos, instead of a
// separate video-drawing path.
func NewIntroScene() *engine.Scene {
	return &engine.Scene{
		ID:         "LOC40",
		Characters: map[string]*engine.Actor{},
		Layers:     map[string]*engine.Layer{},
		Areas:      map[string]*engine.Area{},
	}
}

// LoadIntroTask builds the task that plays the original game's launch
// cinematic into scene (see NewIntroScene): Location 40's Scenes 114, 115
// and 116 in order, each with its named ambient sound per introSequence.
// idx must resolve LOC40's assets (the AVIs) and the shared Common
// directory (the ambient WAVs). stretchVideo is config.ini's
// stretch_video, forwarded to Context.PlayVideo.
//
// skip, if non-nil, lets the caller (see cmd/rah's SPACE-key handling) skip
// forward to the next part: setting *skip to true is consumed (reset to
// false) the next Update, ending the current part immediately so the outer
// sequence moves straight on to the next part's own SCENE log/ambient
// swap/video. It never skips past the final part -- there is nothing after
// it to skip to. Either way -- skipped or played out in full -- a part's
// audio (Scene 114's ocean ambience and its whole voice-over/soundtrack
// timeline, see scene114Timeline) is explicitly stopped the moment it
// ends, mid-line or not, and the whole intro's ambient bed is stopped once
// the last part ends, so nothing bleeds into the next part (or into real
// gameplay); see introAudio.
//
// The real launch path does not actually run Scenes 114/115/116 back to
// back: between Scene 114 and Scene 115 it drops into the interactive
// Location 35 "Dragon Blaster Deluxe" glider sequence (see
// agents/scenes/02_DRAGON_BLASTER_DELUXE.MD), and between Scene 115 and
// Scene 116 into Location 37 / Scene S3 (agents/scenes/3.MD) — both are
// different locations entirely (their own asset index, Session/Controller) and
// so cannot be represented as just another engine.Task inside this one
// sequence. LoadIntroTask therefore stays as a convenience wrapper over
// the whole (114, 115, 116) chain for callers -- and tests -- that don't
// care about those interruptions; cmd/rah's real launch flow instead calls
// LoadIntroPart1Task, runs Location 35 to completion, LoadIntroPart115Task,
// runs Location 37 to completion, then LoadIntroPart116Task.
func LoadIntroTask(scene *engine.Scene, idx *assets.Index, audioEngine audio.Engine, stretchVideo bool, skip *bool) engine.Task {
	return loadIntroPartsTask(scene, idx, audioEngine, stretchVideo, skip, introSequence)
}

// LoadIntroPart1Task plays only Scene 114, the launch cinematic's first
// part -- everything before the real game's Location 35 interlude (see
// LoadIntroTask's doc comment).
func LoadIntroPart1Task(scene *engine.Scene, idx *assets.Index, audioEngine audio.Engine, stretchVideo bool, skip *bool) engine.Task {
	return loadIntroPartsTask(scene, idx, audioEngine, stretchVideo, skip, introSequence[:1])
}

// LoadIntroPart2Task plays Scene 115 only — the cinematic that precedes
// Location 37 / Scene S3 (see agents/scenes/115_TRANSITION.MD). Scene 116
// is played separately after that interlude (see LoadIntroPart116Task).
func LoadIntroPart2Task(scene *engine.Scene, idx *assets.Index, audioEngine audio.Engine, stretchVideo bool, skip *bool) engine.Task {
	return loadIntroPartsTask(scene, idx, audioEngine, stretchVideo, skip, introSequence[1:2])
}

// LoadIntroPart115Task is an alias for LoadIntroPart2Task.
func LoadIntroPart115Task(scene *engine.Scene, idx *assets.Index, audioEngine audio.Engine, stretchVideo bool, skip *bool) engine.Task {
	return LoadIntroPart2Task(scene, idx, audioEngine, stretchVideo, skip)
}

// LoadIntroPart116Task plays Scene 116 after Location 37 hands control back.
func LoadIntroPart116Task(scene *engine.Scene, idx *assets.Index, audioEngine audio.Engine, stretchVideo bool, skip *bool) engine.Task {
	return loadIntroPartsTask(scene, idx, audioEngine, stretchVideo, skip, introSequence[2:3])
}

// LoadIntroPart2TaskFrom is LoadIntroPart2Task starting at sceneID ("115" or
// "116") instead of always at 115 -- for jumping directly into the intro's
// tail (see ResolveLaunchPhase / config.ini's init_scene). Any sceneID not
// found among Part 2's own scenes (in particular, one belonging to an
// earlier phase, since this is also called when falling through normally
// from Scene 114/Location 35) falls back to the full Part 2 (115 then 116).
func LoadIntroPart2TaskFrom(scene *engine.Scene, idx *assets.Index, audioEngine audio.Engine, stretchVideo bool, skip *bool, sceneID string) engine.Task {
	if sceneID == loc37SceneID || sceneID == "3" {
		return LoadIntroPart116Task(scene, idx, audioEngine, stretchVideo, skip)
	}
	part2 := introSequence[1:]
	for i, part := range part2 {
		if part.scene == sceneID {
			return loadIntroPartsTask(scene, idx, audioEngine, stretchVideo, skip, part2[i:])
		}
	}
	return loadIntroPartsTask(scene, idx, audioEngine, stretchVideo, skip, part2)
}

func loadIntroPartsTask(scene *engine.Scene, idx *assets.Index, audioEngine audio.Engine, stretchVideo bool, skip *bool, parts []introPart) engine.Task {
	session := &Session{
		idx:          idx,
		audioEngine:  audioEngine,
		scene:        scene,
		stretchVideo: stretchVideo,
	}
	ctx := session.ctx()
	ia := &introAudio{idx: idx, engine: audioEngine}

	var tasks []engine.Task
	lastAmbient := ""

	for _, part := range parts {
		ambientChanges := part.ambient != lastAmbient
		lastAmbient = part.ambient

		tasks = append(tasks, engine.Immediate(func() {
			scene.ID = part.scene
			log.Printf("SCENE: %s | LOCATION: 40", part.scene)
			// FUN_004400c0 is a no-op when asked to (re)load the name it
			// already has loaded -- it only tears down and reloads on a
			// *different* name -- so the ambient bed only restarts here
			// when it's actually changing (see introSequence's doc
			// comment); otherwise it just keeps playing across the cut.
			if ambientChanges {
				ia.playAmbient(part.ambient)
			}
		}))

		// PlayVideo always hands back a *videoTask (see Context.PlayVideo);
		// skippableTask needs the concrete type to force it to finish
		// immediately on skip, rather than fast-forwarding it.
		var video *videoTask
		var ambience engine.Task

		switch part.scene {
		case "114":
			video, _ = ctx.PlayVideo(part.video).(*videoTask)
			// scene114TimelineTask's own mid-scene ambient swap (see
			// scene114Timeline) means "the next part's own ambient
			// differs" (ambientChanges above) stays true regardless, so
			// no extra bookkeeping is needed here for that.
			ambience = engine.Parallel(oceanAmbienceTask(ia), scene114TimelineTask(ia), scene114SFXTask(idx, audioEngine))

		case "116":
			// See agents/scenes/116.MD: unlike a plain PlayVideo, this
			// scene's video must stop and hold at scene116SplitFrame until
			// 004_PRI_01 (fired by scene116CueTask, running concurrently
			// as this part's own ambience) completes.
			video = &videoTask{
				session:     session,
				aviFilename: part.video,
				split:       true,
				splitFrame:  scene116SplitFrame,
				splitAction: waitForNarrationTask{ia: ia},
			}
			ambience = scene116CueTask(ia, ctx)

		default:
			video, _ = ctx.PlayVideo(part.video).(*videoTask)
		}

		tasks = append(tasks, &skippableTask{video: video, ambience: ambience, ia: ia, skip: skip})
	}

	tasks = append(tasks, engine.Immediate(ia.stopAmbient))

	return engine.Sequence(tasks...)
}

// introAudio tracks the audio.Handles the running intro part has started
// (ambient bed, voice-over, ocean loop) so they can be explicitly stopped:
// a part's narration and ocean loop the instant that part ends (skipped or
// not, see LoadIntroTask), the ambient bed when the next part's differs,
// and the ambient bed again once the whole intro ends. Without this,
// skipping ahead -- or even just a short line finishing early on its own
// -- would otherwise leave narration or ambient audio playing on into the
// next part, or past the intro into real gameplay.
type introAudio struct {
	idx    *assets.Index
	engine audio.Engine

	ambient   audio.Handle
	narration audio.Handle
	ocean     audio.Handle
}

func (a *introAudio) playAmbient(name string) {
	a.stopAmbient()

	sound, err := engine.LoadSound(a.idx, name)
	if err != nil {
		log.Printf("game: intro ambient %s: %v", name, err)
		return
	}

	// A long-form ambient bed must keep playing for as long as this part
	// does, restarting from the beginning if its own runtime is shorter --
	// not fall silent partway through (see audio.Engine.PlayLooping).
	a.ambient = a.engine.PlayLooping(sound, audio.CategoryMusic)
}

func (a *introAudio) playNarration(name string) {
	if name == "" {
		return
	}

	// Stop whatever narration line is still going rather than just
	// overwriting the tracked handle -- see loopSoundTask's identical
	// reasoning for the ocean ambience loop. Real cue spacing never
	// overlaps, but this keeps that an assumption skipping/timing drift
	// can't violate rather than a hard requirement.
	if a.narration != nil {
		a.narration.Stop()
	}

	sound, err := engine.LoadSound(a.idx, name)
	if err != nil {
		log.Printf("game: intro narration %s: %v", name, err)
		a.narration = nil
		return
	}

	a.narration = a.engine.Play(sound, audio.CategorySpeech)
}

func (a *introAudio) stopPart() {
	if a.narration != nil {
		a.narration.Stop()
		a.narration = nil
	}
	if a.ocean != nil {
		a.ocean.Stop()
		a.ocean = nil
	}
}

func (a *introAudio) stopAmbient() {
	if a.ambient != nil {
		a.ambient.Stop()
		a.ambient = nil
	}
}

// skippableTask drives one intro part's video to completion, running
// ambience (narration is already fired-and-forgotten before this task
// starts; see LoadIntroTask) alongside it purely for its own audio side
// effects. Setting *skip to true ends the part immediately: it stops this
// part's audio via ia.stopPart and forces video to its last frame directly
// (video.finishNow), rather than fast-forwarding simulated time through
// it. Fast-forwarding by feeding many more ticks would let a
// time-based side effect that's still "behind" -- e.g. the ocean ambience
// loop's own restart timer -- fire further real Play calls while
// catching up, which is exactly what audibly bled the ocean ambience into
// the next part before this was fixed to stop things immediately instead.
type skippableTask struct {
	video    *videoTask
	ambience engine.Task // nil if this part has none (only Scene 114 does)
	ia       *introAudio
	skip     *bool
}

func (t *skippableTask) Update(dt float64) bool {
	if t.skip != nil && *t.skip {
		*t.skip = false
		t.ia.stopPart()
		if t.video != nil {
			t.video.finishNow()
		}
		return true
	}

	done := true
	if t.video != nil {
		done = t.video.Update(dt)
	}
	if t.ambience != nil {
		t.ambience.Update(dt)
	}

	if done {
		t.ia.stopPart()
	}

	return done
}

// oceanAmbienceTask reproduces oceanPhases: silence, then Sfx_Ocean looping
// (it is a short ~8s clip) for the first phase's span, silence again, then
// looping again for the second phase's span. Each (re)start's Handle is
// tracked on ia so stopPart can silence a loop still running mid-clip.
func oceanAmbienceTask(ia *introAudio) engine.Task {
	var tasks []engine.Task
	clock := 0.0

	for _, phase := range oceanPhases {
		start := float64(phase.startFrame) / introFPS
		end := float64(phase.endFrame) / introFPS

		if gap := start - clock; gap > 0 {
			tasks = append(tasks, engine.Wait(gap))
		}
		tasks = append(tasks, loopSound(ia, "Sfx_Ocean.wav", end-start))
		clock = end
	}

	return engine.Sequence(tasks...)
}

// loopSound restarts name every time its own playback duration elapses,
// for exactly totalSeconds, then completes. name is loaded once up front;
// a resolve/load failure degrades to silently waiting out totalSeconds
// rather than failing the whole intro over one missing ambience clip.
func loopSound(ia *introAudio, name string, totalSeconds float64) engine.Task {
	sound, err := engine.LoadSound(ia.idx, name)
	if err != nil || sound.Duration() <= 0 {
		log.Printf("game: intro ambience %s: %v", name, err)
		return engine.Wait(totalSeconds)
	}

	return &loopSoundTask{ia: ia, sound: sound, interval: sound.Duration(), remaining: totalSeconds}
}

type loopSoundTask struct {
	ia        *introAudio
	sound     *audio.Sound
	interval  float64
	remaining float64
	untilNext float64 // <= 0 means "(re)start now"
}

func (t *loopSoundTask) Update(dt float64) bool {
	if t.untilNext <= 0 {
		// Explicitly stop the previous instance rather than just
		// overwriting the tracked handle: our restart timing is driven by
		// simulated dt, which is not guaranteed to line up exactly with
		// the previous instance's real playback position (a frame hitch,
		// or a burst of catch-up ticks, is enough to drift them apart).
		// Without this, a still-technically-playing earlier instance
		// would keep going for real, on its own, for up to another
		// interval seconds -- audible as the ambience bleeding into
		// whatever comes next, which is the actual bug this guards
		// against (see TestIntroSkipStopsEveryOceanRestartNotJustTheLast).
		if t.ia.ocean != nil {
			t.ia.ocean.Stop()
		}
		t.ia.ocean = t.ia.engine.Play(t.sound, audio.CategoryMusic)
		t.untilNext = t.interval
	}
	t.untilNext -= dt
	t.remaining -= dt

	return t.remaining <= 0
}
