package game

import "github.com/wok/rent-a-hero/internal/engine"

// LOC35Controller implements Controller for Location 35's "S1" scene: the
// Dragon Blaster Deluxe glider interlude the real launch path inserts
// between the intro's Scene 114 and Scene 115 (see agents/scenes/S1.MD,
// which supersedes the earlier, simplified
// agents/scenes/02_DRAGON_BLASTER_DELUXE.MD reconstruction).
//
// Per S1.MD's own framing, this is not "background video + speech +
// eventual scene change": it is three concurrent systems --
//
//  1. a layered visual composition (S1_Back plus a foreground
//     hero/hovercraft layer, never one exclusive fullscreen video),
//  2. the inventory UI (item 0x1B / Dragon Blaster Deluxe, granted on
//     entry and visible/selectable throughout), and
//  3. a timer-driven progression state machine plus an
//     animation-completion-driven action state machine, running
//     concurrently and communicating only through shared state -- never a
//     direct function call from one into the other.
//
// See loc35SceneTask for the state machine itself.
type LOC35Controller struct{}

// ItemDragonBlasterDeluxe is inventory item 0x1B (27), the item this scene
// specially enables; see S1.MD section 9 ("Observed UI identity: Dragon
// Blaster Deluxe").
const ItemDragonBlasterDeluxe = 27

// loc35SceneID is the only scene this location has (S1.SZN).
const loc35SceneID = "S1"

// loc35DragonBlasterStateFlag preserves the scene's own item-state
// assignment (S1.MD section 10: "item 0x1B state/value = 3") without
// discarding it, even though no rendering behavior is wired to it yet --
// the doc is explicit that its visual meaning should come from separately
// proven inventory-asset behavior, not a scene-specific guess.
const loc35DragonBlasterStateFlag = "Loc35DragonBlasterDeluxeState"

// loc35ActionStateFlag holds the action state machine's current state
// (S1.MD sections 7/13/20/43). It lives in Session.State().Flags, not a
// private loc35SceneTask field, specifically so LOC35Controller.SelectItem
// -- invoked completely separately from the running ambient task, on a
// discrete player click -- can request action state 10 without needing a
// reference to that task instance (S1.MD section 11: "the click only
// selects action state 10... the normal animation-completion event then
// executes the choreography").
const loc35ActionStateFlag = "Loc35ActionState"

// Action states (S1.MD sections 13, 21-31).
const (
	loc35ActionNormalFlight    = 1
	loc35ActionFirstItemCheck  = 2
	loc35ActionSecondItemCheck = 3
	loc35ActionReturnToFlight  = 4
	loc35ActionCloseup         = 5
	loc35ActionAutoBlaster     = 9
	loc35ActionPlayerBlaster   = 10
)

// loc35CloseupTalkStartFrame/EndFrame is S1_RodToCloseup's authored talk
// range that the close-up speech lines (011_ROD_01/02/03) are bound to.
// Both S1_ALPHA_AND_SYNC.MD and S1_CLOSEUP_ALPHA_FLICKER_FIXES.MD document
// this range as "41..44" using this project's usual 1-indexed frame
// authoring convention (matching e.g. "S1_BackToCloseup frame 1 -> end"
// elsewhere in the same docs, where frame 1 is the first frame) -- so as a
// 0-indexed engine.Layer.Frame value this is 40..43, one less each. The
// entrance transition-in is frames 0..39 (1-indexed 1..40); the close-up
// return (see buildCloseupCycle) explicitly repositions to frame 39 and
// plays that same entrance range backward to 0, per
// S1_CLOSEUP_ALPHA_FLICKER_FIXES.MD sections 3/7 ("frame 40 -> frame 1,
// REVERSE") -- it does not continue forward from wherever the talk loop
// left off.
const (
	loc35CloseupTalkStartFrame = 41
	loc35CloseupTalkEndFrame   = 44
)

// loc35InactiveLayers are S1.SZN's layers that must start hidden AND frozen:
// only S1_Back (the base flying background) and S1_RodOnGlider (Rodrigo's
// normal glider loop) play from scene entry (S1.MD section 4: "S1_Back and
// S1_RodOnGlider are started separately during activation"). Every layer
// defaults to visible+looping+*playing* on instantiation (see
// engine.instantiateLayer), so Enter must explicitly both hide AND freeze
// the rest -- hiding alone (as an earlier version of this file did) leaves
// Playing=true, so these "dormant" layers kept silently cycling their own
// frames in the background the whole time, invisibly, completely detached
// from the action state machine's own scripted control of them (the exact
// class of bug agents/scenes/Z-DEPTH.MD's "dormant special action prop"
// guidance -- see e.g. LOC01Controller's own lever layers -- already
// establishes FreezeLayer for).
var loc35InactiveLayers = []string{
	"S1_Back2",
	"S1_BackToCloseup",
	"S1_CloseupToBack",
	"S1_RodChecksItems",
	"S1_Schild",
	"S1_RodDrawsBlaster",
	"S1_RodToCloseup",
}

func (LOC35Controller) Enter(ctx *Context, scene string, from string) engine.Task {
	if scene != loc35SceneID {
		return nil
	}

	for _, id := range []string{"S1_Back", "S1_Back2"} {
		if layer, ok := ctx.layer(id); ok {
			layer.LoopStart = 1
			if layer.Frame < 1 {
				layer.Frame = 1
			}
		}
	}
	if layer, ok := ctx.layer("S1_RodOnGlider"); ok && layer.Frame < 1 {
		layer.Frame = 1
	}

	tasks := []engine.Task{
		ctx.AddItem(ItemDragonBlasterDeluxe),
		ctx.SetFlag(loc35DragonBlasterStateFlag, 3),
		ctx.SetFlag(loc35ActionStateFlag, loc35ActionNormalFlight),
	}

	for _, id := range loc35InactiveLayers {
		frame := 0
		if id == "S1_Back2" {
			frame = 1
		}
		tasks = append(tasks, ctx.HideLayer(id), ctx.FreezeLayer(id, frame))
	}

	tasks = append(tasks,
		// Loc35_RodrigoOnGlider is a long-form/3D audio bed, not a
		// background resource -- see S1.MD section 6 -- so it plays the
		// same way the intro's per-part ambient beds do (see intro.go's
		// introAudio): fired once, expected to comfortably outlast the
		// sequence.
		ctx.PlayMusic("Loc35_RodrigoOnGlider.wav"),
		ctx.PlaySFX("Sfx_Glider_Flying.wav"),
		ctx.RunAmbient(newLoc35SceneTask(ctx)),
	)

	return engine.Sequence(tasks...)
}

func (LOC35Controller) Exit(ctx *Context, scene string, to string) engine.Task { return nil }

func (LOC35Controller) Click(ctx *Context, area string) engine.Task { return nil }

func (LOC35Controller) UseItem(ctx *Context, item int, area string) engine.Task { return nil }

// LoadConditionMask: S1.SZN declares no conditional (LoadCond != 0) layers.
func (LOC35Controller) LoadConditionMask(ctx *Context, scene string) int { return 0 }

// SelectItem implements event type 3 (S1.MD sections 11/28/41): selecting
// the Dragon Blaster Deluxe removes it from the inventory, plays
// Sfx_Door_Opened, and sets the shared action state to 10 -- nothing more.
// It deliberately does not touch any layer or start the blaster animation
// itself: the running loc35SceneTask (registered as an ambient task by
// Enter, so it keeps driving the scene independently of whatever this
// returns) only ever acts on actionState at its current action's own
// animation-completion boundary, never interrupted mid-flight. This task
// itself runs via the normal activeTask slot -- harmless here since nothing
// else occupies it during Location 35 (the scene's real work is the
// ambient task), and it always completes within a tick since every step is
// immediate.
func (LOC35Controller) SelectItem(ctx *Context, item int) engine.Task {
	if item != ItemDragonBlasterDeluxe {
		return nil
	}
	return engine.Sequence(
		ctx.RemoveItem(ItemDragonBlasterDeluxe),
		ctx.PlaySFX("Sfx_Door_Opened.wav"),
		ctx.SetFlag(loc35ActionStateFlag, loc35ActionPlayerBlaster),
	)
}

// loc35SceneTask is Location 35's entire concurrent state machine, run as
// one ambient task (Context.RunAmbient) for the scene's whole lifetime: a
// real-time progression timer (event type 1, S1.MD sections 14-19) plus an
// animation-completion-driven action dispatcher (event type 2, sections
// 20-31). Both read/write actionState through Session.State().Flags (see
// loc35ActionStateFlag) rather than a private field, since
// LOC35Controller.SelectItem (event type 3) must be able to request action
// state 10 from a completely separate call. progressionState and the
// in-flight action/voice tasks stay private fields: nothing outside this
// task ever needs to observe or mutate them.
type loc35SceneTask struct {
	ctx *Context

	progressionState int
	progressionTimer float64
	progressionArmed bool

	action engine.Task // the current action's own task; rebuilt once it completes
	voice  engine.Task // an in-flight fire-and-forget PlayVoiceover from handleProgression; nil when idle
}

// newLoc35SceneTask starts the timer at progression state 1's initial
// 2-second delay (S1.MD section 8, activation step 8: "schedule event type
// 1 after 2000 ms").
func newLoc35SceneTask(ctx *Context) *loc35SceneTask {
	return &loc35SceneTask{ctx: ctx, progressionState: 1, progressionArmed: true, progressionTimer: 2.0}
}

func (t *loc35SceneTask) Update(dt float64) bool {
	if t.progressionArmed {
		t.progressionTimer -= dt
		if t.progressionTimer <= 0 {
			t.progressionArmed = false
			t.handleProgression()
		}
	}

	if t.voice != nil && t.voice.Update(dt) {
		t.voice = nil
	}

	if t.action == nil {
		t.action = t.buildAction(t.actionState())
	}
	// S1_CLOSEUP_ALPHA_FLICKER_FIXES.MD sections 19/22/24: the completion
	// event is dispatched (the next action built and immediately given a
	// tick) within this same Update call, before this function returns and
	// the frame gets presented -- not deferred to the next Update call,
	// which would leave the just-completed action's stale last frame
	// visible for one extra render with nothing yet replacing it. Every
	// action ends in a real time-consuming step (an animation range or
	// speech), so this cannot loop more than a couple of times per tick in
	// practice; ctx.CompleteLocation ends the session, and nothing calls
	// Update again afterward (see cmd/rah's playLoc35 checking
	// Session.Done()), so state 10's own action is never re-dispatched from
	// scratch once the blaster finishes.
	for t.action.Update(dt) {
		t.action = t.buildAction(t.actionState())
	}

	// This ambient task never completes on its own: ctx.CompleteLocation
	// (reached via the blaster finalization) is what ends Location 35, per
	// cmd/rah's playLoc35 checking Session.Done().
	return false
}

func (t *loc35SceneTask) actionState() int { return t.ctx.GetFlag(loc35ActionStateFlag) }

// setActionStateUnlessBlasterChosen implements the "if actionState != 10"
// guard repeated throughout S1.MD sections 16-23/42: an asynchronous item
// selection must never be overwritten by the scripted progression/action
// sequence that was already in flight when it happened.
func (t *loc35SceneTask) setActionStateUnlessBlasterChosen(next int) {
	if t.actionState() == loc35ActionPlayerBlaster {
		return
	}
	t.setActionState(next)
}

func (t *loc35SceneTask) setActionState(v int) {
	if t.ctx.session.state.Flags == nil {
		t.ctx.session.state.Flags = map[string]int{}
	}
	t.ctx.session.state.Flags[loc35ActionStateFlag] = v
}

// armProgression schedules the next event-type-1 callback after seconds
// real-time seconds.
func (t *loc35SceneTask) armProgression(seconds float64) {
	t.progressionArmed = true
	t.progressionTimer = seconds
}

// playVoiceover starts baseName as a fire-and-forget line (tracked via t.voice,
// see Update) -- used only from handleProgression, a plain synchronous
// function that (unlike buildAction's returned Task) cannot itself await a
// Sequence step. Actions that need to play a line as part of their own
// paced sequence just use ctx.PlayVoiceover directly as a Sequence step.
func (t *loc35SceneTask) playVoiceover(baseName string) {
	t.voice = t.ctx.PlayVoiceover(baseName, "["+baseName+"]")
}

// handleProgression is the event-type-1 (timer) callback: S1.MD sections
// 15-19's exact progression state machine.
func (t *loc35SceneTask) handleProgression() {
	switch t.progressionState {
	case 1:
		t.playVoiceover("001_ROD_01")
		t.armProgression(5.0)
		t.progressionState = 2

	case 2:
		t.setActionStateUnlessBlasterChosen(loc35ActionFirstItemCheck)
		t.armProgression(20.0)
		t.progressionState = 3

	case 3:
		t.setActionStateUnlessBlasterChosen(loc35ActionCloseup)
		t.progressionState = 4

	case 4, 5:
		// Sections 18: "if actionState != 10: actionState = 5" and
		// "advances: 4->5, 5->6". No new timer is armed here -- the
		// close-up action itself schedules the next one when it returns to
		// normal flight (section 27), since these two progression events
		// are only ever reached via that return, never a bare timer of
		// their own.
		t.setActionStateUnlessBlasterChosen(loc35ActionCloseup)
		t.progressionState++

	case 6:
		// Automatic fallback (section 19): the scene may not wait forever.
		t.setActionStateUnlessBlasterChosen(loc35ActionAutoBlaster)
	}
}

// buildAction is the event-type-2 (animation-completion) dispatcher: S1.MD
// sections 20-31's exact action state machine. Each returned Task owns the
// scene's visual layers until it reaches its own completion, at which point
// Update rebuilds this dispatch against whatever actionState now is --
// possibly changed asynchronously by handleProgression or SelectItem while
// the previous action was still playing.
func (t *loc35SceneTask) buildAction(state int) engine.Task {
	ctx := t.ctx

	switch state {
	case loc35ActionNormalFlight, loc35ActionReturnToFlight: // 1, 4
		// S1_ALPHA_AND_SYNC.MD sections 21/24: both states just keep
		// Rodrigo visibly flying; state 4 additionally represents "just
		// returned from an item check", but the visual behavior is
		// identical. S1_CLOSEUP_ALPHA_FLICKER_FIXES.MD section 23: show
		// S1_RodOnGlider *before* hiding S1_RodChecksItems, so there is
		// always a foreground hero representation available during the
		// handoff.
		return engine.Sequence(
			ctx.ShowLayer("S1_RodOnGlider"),
			ctx.HideLayer("S1_RodChecksItems"),
			ctx.PlayLayerFrames("S1_RodOnGlider", 1, -1),
		)

	case loc35ActionFirstItemCheck: // 2
		// S1_ALPHA_AND_SYNC.MD sections 12/14: 001_ROD_02 starts immediately
		// after S1_RodChecksItems, running concurrently with it -- not
		// "play the animation, then play the line". Event type 2 (the next
		// dispatch) is driven solely by the *animation's* own completion, so
		// the voice is fired via the fire-and-forget t.playVoiceover (like
		// handleProgression's own lines) rather than sequenced as a
		// Sequence step, which would make this action wait for whichever of
		// the two takes longer.
		return engine.Sequence(
			ctx.ShowLayer("S1_RodChecksItems"),
			ctx.HideLayer("S1_RodOnGlider"),
			engine.Immediate(func() { t.playVoiceover("001_ROD_02") }),
			ctx.PlayLayerFrames("S1_RodChecksItems", 1, -1),
			engine.Immediate(func() { t.setActionStateUnlessBlasterChosen(loc35ActionSecondItemCheck) }),
		)

	case loc35ActionSecondItemCheck: // 3
		// Section 13: same concurrent pattern as state 2, for 001_ROD_03.
		return engine.Sequence(
			engine.Immediate(func() { t.playVoiceover("001_ROD_03") }),
			ctx.PlayLayerFrames("S1_RodChecksItems", 1, -1),
			engine.Immediate(func() { t.setActionStateUnlessBlasterChosen(loc35ActionReturnToFlight) }),
		)

	case loc35ActionCloseup: // 5
		return t.buildCloseupCycle()

	case loc35ActionAutoBlaster: // 9
		// Section 21: unlike the player branch, this one plays 011_ROD_04
		// and removes the item itself first (the player never claimed it).
		// The line is not bound to any animation and must not block the
		// blaster animation from starting -- "allow the audio/animation
		// systems to overlap naturally" -- so it's fire-and-forget too.
		return engine.Sequence(
			ctx.HideLayer("S1_RodOnGlider"),
			engine.Immediate(func() { t.playVoiceover("011_ROD_04") }),
			ctx.RemoveItem(ItemDragonBlasterDeluxe),
			t.buildBlasterFinalization(),
		)

	case loc35ActionPlayerBlaster: // 10
		return engine.Sequence(
			ctx.HideLayer("S1_RodOnGlider"),
			t.buildBlasterFinalization(),
		)

	default:
		// Never reached in practice (actionState only ever holds one of the
		// values above), but avoids ever leaving t.action permanently nil.
		return engine.Immediate(func() {})
	}
}

// buildCloseupCycle is action state 5 (S1.MD sections 25-27): the full
// visual close-up transition, its progression-mapped speech line (section
// 26), and the return to normal flight -- which also re-arms the next
// 6-second progression timer (section 18's "these progression events are
// scheduled by the close-up action itself") and advances actionState to 4
// unless the player has since chosen the blaster.
func (t *loc35SceneTask) buildCloseupCycle() engine.Task {
	ctx := t.ctx

	line := "011_ROD_03" // progressionState 6 and beyond
	switch t.progressionState {
	case 4:
		line = "011_ROD_01"
	case 5:
		line = "011_ROD_02"
	}

	return engine.Sequence(
		// S1_ALPHA_AND_SYNC.MD section 18: enter close-up.
		// S1_RodToCloseup's own initial transition-in (frames 0..39, right
		// up to its speech-bound talk range) runs concurrently with the
		// S1_BackToCloseup transition -- the doc's own WAIT boundary here is
		// only for S1_BackToCloseup finishing.
		ctx.ShowLayer("S1_RodToCloseup"),
		ctx.HideLayer("S1_RodOnGlider"),
		ctx.ShowLayer("S1_BackToCloseup"),
		ctx.HideLayer("S1_Back"),
		engine.Parallel(
			ctx.PlayLayerFrames("S1_RodToCloseup", 1, loc35CloseupTalkStartFrame-1),
			ctx.PlayLayerFrames("S1_BackToCloseup", 1, -1),
		),
		ctx.HideLayer("S1_BackToCloseup"),
		ctx.ShowLayer("S1_Back2"),
		engine.Immediate(func() {
			if layer, ok := ctx.layer("S1_Back2"); ok {
				layer.Mode = engine.AnimLoop
				layer.LoopStart = 1
				layer.Frame = 1
				layer.Accumulator = 0
				layer.Playing = true
			}
		}),

		// S1_ALPHA_AND_SYNC.MD sections 15/19/26: the close-up line is
		// explicitly bound to S1_RodToCloseup's own authored talk range, not
		// an independent voice line -- and the return sequence below only
		// begins once the *speech* completes, not a fixed animation
		// duration.
		ctx.PlaySpeechBoundToLayer("S1_RodToCloseup", line, "["+line+"]", loc35CloseupTalkStartFrame, loc35CloseupTalkEndFrame),

		// S1_CLOSEUP_ALPHA_FLICKER_FIXES.MD sections 3/6/7: once speech
		// ownership of the layer ends, the scene explicitly repositions
		// S1_RodToCloseup to the end of its entrance transition (frame 39 --
		// not wherever the talk loop happened to leave it) and plays that
		// same range *backward* to frame 0, undoing the entrance rather than
		// continuing forward into unused trailing frames. PlayLayerFrames
		// detects the reverse direction from from > to.
		ctx.PlayLayerFrames("S1_RodToCloseup", loc35CloseupTalkStartFrame-1, 1),
		ctx.HideLayer("S1_Back2"),
		ctx.ShowLayer("S1_CloseupToBack"),
		ctx.PlayLayerFrames("S1_CloseupToBack", 1, -1),
		ctx.HideLayer("S1_CloseupToBack"),
		ctx.HideLayer("S1_RodToCloseup"),
		ctx.ShowLayer("S1_Back"),
		engine.Immediate(func() {
			if layer, ok := ctx.layer("S1_Back"); ok {
				layer.Mode = engine.AnimLoop
				layer.LoopStart = 1
				layer.Frame = 1
				layer.Accumulator = 0
				layer.Playing = true
			}
		}),
		ctx.ShowLayer("S1_RodOnGlider"),

		// The 6-second timer is scheduled right as the scene restarts, not
		// after an extra full S1_RodOnGlider pass -- the very next action
		// dispatch (state 4, ReturnToFlight) is what actually plays it.
		engine.Immediate(func() {
			t.armProgression(6.0)
			t.setActionStateUnlessBlasterChosen(loc35ActionReturnToFlight)
		}),
	)
}

// buildBlasterFinalization is the common success path shared by action
// states 9 and 10 (S1.MD section 31): S1_RodDrawsBlaster frames 1-0x3C
// (0-indexed 0-59), then Sfx_SignFlyingBy, S1_Schild, and the frame-0x3D
// Sfx_Glider_Flying cue all together (section 32), then frames
// 0x3D-end (0-indexed 60-end), and finally the transition to Location 40 /
// Scene 115 (via Session.Done, since a cross-location jump needs a
// different asset index/Session entirely -- see cmd/rah's playLoc35).
func (t *loc35SceneTask) buildBlasterFinalization() engine.Task {
	ctx := t.ctx

	return engine.Sequence(
		ctx.HideLayer("S1_RodChecksItems"),
		ctx.HideLayer("S1_RodToCloseup"),
		ctx.HideLayer("S1_Back2"),
		ctx.ShowLayer("S1_Back"),
		ctx.ShowLayer("S1_RodDrawsBlaster"),
		ctx.PlayLayerFrames("S1_RodDrawsBlaster", 1, 60),
		engine.Parallel(
			ctx.PlaySFX("Sfx_SignFlyingBy.wav"),
			ctx.PlaySFX("Sfx_Glider_Flying.wav"),
			engine.Sequence(ctx.ShowLayer("S1_Schild"), ctx.PlayLayer("S1_Schild")),
			ctx.PlayLayerFrames("S1_RodDrawsBlaster", 61, -1),
		),
		ctx.CompleteLocation(),
	)
}
