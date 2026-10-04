package game

import (
	"math"
	"math/rand/v2"

	"github.com/wok/rent-a-hero/internal/engine"
)

// LOC01Controller implements Controller for LOC01: the "Hero Discounter"
// shop landing (scene 7B), the upper landing (scene 9), the lower-floor
// canal area (scene 46, connected to 7B by a lift), and the office-front
// approach (scene 7A, connected to 7B by the stairs -- and Scene 116's
// first-encounter return, see enter7AReturnHomeMarker5).
//
// All four scenes' hotspot/entry-positioning logic is a verified
// transcription of the decompiled FUN_00405530/FUN_00405dc0/FUN_004065c0,
// per agents/scenes/7B.MD, agents/scenes/9.MD, agents/scenes/46.MD and
// agents/scenes/1.MD respectively: the shared fixed walk-to points, each
// lever's two-part animation split around Sfx_ClickelDiClack, Scene 7B
// owning every elevator travel video on arrival (never the departure
// scene), the real recorded lines and their walk targets,
// FlagCanalStoryFlag's effect on Scene 46's resources/interactions (and,
// inferred from the same shared state field, both the elevator and
// stair-camera video variants -- see liftVariant/stairCamVariant), and the
// cross-location exits into Location 4 (007a_Tuer, see LOC04Controller),
// Location 5 (009_Tuer, see LOC05Controller) and Location 7 (046_Kanal, see
// LOC07Controller) are all source-proven there, not inferred -- except
// where a doc's own decompile is itself incomplete or ambiguous (Scene
// 46's exact KanalTalk stage-transition table, its
// documented-but-unclear reverse-range animation call, and the stair
// cameras' _OK/_MK selector), which is called out at each such site's own
// doc comment (kanalTalkConversation, departScene46Elevator,
// stairCamVariant).
//
// The 7A <-> 7B stairs are explicitly asymmetric (7B.MD section 18.3), not
// a generic bidirectional edge, and every anchor point below is a distinct
// pixel-exact coordinate rather than one shared "the stairs" position:
// 7B -> 7A (goToOfficeFront) walks in 7B, switches scenes immediately with
// no video of its own, then Scene 7A hard-places Rodrigo at its own
// cinematic anchor (loc7ARunterCamAnchorX/Y, not the same point
// 007b_Treppe departs from) and plays StairLiftRunterCam_* with its own
// footstep-SFX choreography (loc7ARunterCamCues) before settling him at yet
// another final spot. 7A -> 7B (leaveOfficeFront) is the mirror image but
// video-before-switch instead of video-after-switch: it walks to its own
// exact departure point in 7A, plays StairLiftRaufCam_* (loc7ARaufCamCues)
// there while still in 7A, and only then switches to 7B, which places
// Rodrigo through its own two-step arrival sequence.
//
// The plant/matches puzzle in scene 7B remains a looser reconstruction
// grounded only in real asset names, not a decompiled scene doc: it is not
// a verified transcription of the original compiled game logic, which
// requires disassembly and playtesting per agents/GAMEPLAY.md's
// reverse-engineering workflow (Milestone 6 content-porting work).
type LOC01Controller struct{}

const (
	loc01Actor = "RodrigoSmall"

	loc01StateCanalStory      = 0x3e94
	loc01StateCanalTalkStage  = 0x3e98
	loc01StateCanalTalkBranch = 0x3e9c
	loc01StateCanalTalkUnlock = 0x3ec4

	// FlagCanalStoryFlag mirrors the game state field at state+0x3E94,
	// which -- per agents/scenes/46.MD section 1 and 7B.MD section 2 --
	// gates both Scene 46's own resources/interactions and the choice of
	// _OK/_MK elevator video variant between 7B and 46 (see liftVariant).
	// It is NOT "whether the player has opened the canal": 0 (the default,
	// unset zero value) is the normal/starting configuration, where
	// back46_ok is the active background and 046_Kanal is a real exit to
	// Location 7; a nonzero value is a KanalTalk conversation mode instead
	// (see Enter's "46" case and kanalTalkBanter). This flag's true
	// external trigger (whatever unrelated story event sets it away from 0
	// during normal play) is not covered by any scene doc yet -- 46.MD only
	// documents that returning from Scene 47 always clears it back to 0.
	FlagCanalStoryFlag = "loc01.canal_story_flag"

	// FlagCanalTalkStage approximates the game state fields at state+0x3E98
	// (progression) referenced by 46.MD section 9's KanalTalk conversation.
	// 46.MD lists every line the conversation references (046_ROD_01..07,
	// 046_KAN_01..05) and says it "advances state+0x3E98 through successive
	// stages", but does not give the exact per-stage transition table or
	// what state+0x3EC4 (set partway through) unlocks elsewhere -- so this
	// is a reasonable, clearly-approximate reconstruction (advancing
	// linearly through the referenced lines each time the hotspot is used),
	// not a verified transcription of the original stage machine.
	FlagCanalTalkStage  = "loc01.canal_talk_stage"
	FlagCanalTalkBranch = "loc01.canal_talk_branch"
	FlagCanalTalkUnlock = "loc01.canal_talk_unlock"

	// FlagPlantBurned records whether the plant in scene 7B has been set
	// alight.
	FlagPlantBurned = "loc01.plant_burned"

	// ItemMatches is DefInvent item 26, "A box of matches".
	ItemMatches = 26

	// loc7BElevatorX/Y is the fixed spot Rodrigo walks to (departing) or is
	// placed at (arriving) for every elevator ride through Scene 7B --
	// 007b_Lift, 007b_LiftRunter and 007b_UntererStock alike, not each
	// hotspot's own area center -- per agents/scenes/7B.MD "Step 1 — walk
	// Rodrigo to the elevator": FUN_00434d70(Rodrigo, 0x154, 0xB6, event, 5).
	// loc7BElevatorDirection (5) is that same call's forced final
	// direction/state, per
	// agents/techniques/ELEVATOR-TRANSITIONS-AND-ZBUFFER.MD section 3 --
	// preserved verbatim via Context.SetActorDirection (see ride7BElevator),
	// its exact cardinal meaning not yet source-proven.
	loc7BElevatorX         = 0x154
	loc7BElevatorY         = 0xB6
	loc7BElevatorDirection = 5

	// loc7BLeverSplitFrame is where PressLeverMitteLayer's authored
	// press animation pauses for Sfx_ClickelDiClack before continuing to
	// its last frame (0x14, of 42 frames total), per 7B.MD "Step 3 — lever
	// animation".
	loc7BLeverSplitFrame = 0x13

	loc9LiftX         = 0x98
	loc9LiftY         = 0xF1
	loc9LiftDirection = 0
	loc9LiftArrivalX  = 0xA1
	loc9LiftArrivalY  = 0x10F

	// loc9LeverSplitFrame is where PressLeverObenLayer's press animation
	// pauses for Sfx_ClickelDiClack before continuing to its last frame,
	// per 9.MD "The upper lever animation is split into two pieces": frames
	// 0-6, then 7-end.
	loc9LeverSplitFrame = 0x06

	// loc9PflanzeX/Y, loc9SchildLinksX/Y, loc9SchildRechtsX/Y,
	// loc9KlingelX/Y and loc9BesenX/Y are Scene 9's remaining hotspots'
	// documented walk/interact targets, per 9.MD sections 6-11.
	loc9PflanzeX, loc9PflanzeY           = 0x114, 0x107
	loc9SchildLinksX, loc9SchildLinksY   = 0x187, 0x10B
	loc9SchildRechtsX, loc9SchildRechtsY = 0x206, 0x11B
	loc9KlingelX, loc9KlingelY           = 0x206, 0x11B
	loc9BesenX, loc9BesenY               = 0x234, 0x111

	loc9TuerX, loc9TuerY               = 0x216, 0x111
	loc9DoorArrivalX, loc9DoorArrivalY = 0x1D5, 0x115

	// loc05Location is Location 5's location number (agents/scenes/9.MD
	// section 9, "Location 5 / Scene 10"); loc9DoorTargetScene is the
	// actual scene resource on disk (LOC05/S10.SZN), matching the "S<N>"
	// convention also seen for Location 35's S1 rather than the doc's bare
	// "10".
	loc05Location       = 5
	loc9DoorTargetScene = "S10"

	// loc46GangX/Y is 046_Gang's walk target when FlagCanalStoryFlag is 0
	// (the ordinary "look down the corridor" interaction), per
	// agents/scenes/46.MD section 3.
	loc46GangX, loc46GangY = 0xB8, 0xA0

	// loc46KanalX/Y is 046_Kanal's shared walk target for both of its
	// branches (the real exit and the KanalTalk banter), per 46.MD
	// section 4.
	loc46KanalX, loc46KanalY = 0xD4, 0x100

	// loc46KanalReinSplitFrame is where KANALREINCAM.AVI (the outbound
	// canal-crossing camera, played when FlagCanalStoryFlag == 0) pauses
	// for Sfx_Clothes_Scratched before continuing to its last frame, per
	// 46.MD section 4.2: frames 0-8, then 9-end.
	loc46KanalReinSplitFrame = 0x08

	// loc46LiftX/Y is 046_Lift's walk target -- also Scene 46's own normal
	// (non-Scene-47) entry placement: arriving via the elevator hard-places
	// Rodrigo here and leaves him put, rather than walking him any further
	// (see Enter's "46" case, and the "no walk right after a transition"
	// policy documented on arriveAt7BFromAbove) -- per 46.MD sections 5/11.
	// loc46LiftDirection (1) is the departure call's forced final
	// direction/state, per ELEVATOR-TRANSITIONS-AND-ZBUFFER.MD section 8.
	loc46LiftX, loc46LiftY                 = 0x1A2, 0x89
	loc46LiftDirection                     = 1
	loc46EntrySettleX, loc46EntrySettleY   = 0x18C, 0xB4
	loc46ReturnSettleX, loc46ReturnSettleY = 0x108, 0x11C

	// loc46SchildX/Y and loc46Schild2X/Y are the two signs' independent
	// walk targets (they share the same randomized speech pool -- see
	// randomLoc1SignLine), per 46.MD sections 7/8.
	loc46SchildX, loc46SchildY   = 0x1A0, 0x117
	loc46Schild2X, loc46Schild2Y = 0x18A, 0x153

	// loc46KanalTalkX/Y is 046_Gang's walk target specifically when it is
	// repurposed into the KanalTalk conversation trigger (FlagCanalStoryFlag
	// != 0) -- 46.MD section 9's own FUN_00407b80 gives this different
	// coordinate for that same physical hotspot, and section 9 explicitly
	// says the ordinary 046_Gang state is disabled in this configuration,
	// evidence this is the same area repurposed rather than a distinct one
	// (46.SZN itself declares no separate "KanalTalk" area rectangle to use
	// instead).
	loc46KanalTalkX, loc46KanalTalkY = 0xB0, 0x100

	// loc46KanalRausEntryX/Y is where Rodrigo is positioned when Scene 46
	// is entered from Scene 47, before KANALRAUSCAM.AVI plays, per 46.MD
	// section 10.
	loc46KanalRausEntryX, loc46KanalRausEntryY = 0xD1, 0x104

	// loc46KanalRausSplitFrame is where KANALRAUSCAM.AVI (the return trip)
	// pauses for Sfx_Clothes_Scratched, per 46.MD section 10: frames
	// 0-0x19, then 0x1A-end.
	loc46KanalRausSplitFrame = 0x19

	// loc07Location is Location 7's location number (46.MD section 4.2,
	// "Location 7 / Scene 47"); the scene resource on disk is "47" (no
	// "S" prefix, unlike Locations 35/5's S1/S10).
	loc07Location        = 7
	loc46DoorTargetScene = "47"

	// loc7BTreppeX/Y is 007b_Treppe's documented walk target, per
	// agents/scenes/7B.MD section 17 ("Exact hotspot transition:
	// 007b_Treppe -> office-front Scene 7"): FUN_00434d70(Rodrigo, 0x12A,
	// 0xC7, event, 1). This is a plain walk-to-exit, not an elevator
	// action: no lever, no travel AVI played by 7B itself.
	loc7BTreppeX, loc7BTreppeY = 0x12A, 0xC7

	// loc7ASceneID is the office-front scene's actual resource name on disk
	// (LOC01/7A.SZN) -- the docs call it "Scene 7", but that numeral alone
	// collides with nothing else here; "7A" is simply this location's own
	// file-naming convention, matching how 7B/9/46 are named after their
	// own scene numbers.
	loc7ASceneID = "7A"

	// loc06Location is Location 4's location number (7B.MD section 5,
	// "Location 4 / Scene 6", Rodrigo's office interior); the scene
	// resource on disk is "006" (LOC04/006.SZN).
	loc06Location        = 4
	loc7ADoorTargetScene = "006"

	// loc7ATreppeX/Y is 007a_Treppe's exact documented departure walk
	// target in Scene 7A, per 7B.MD section 18.2 ("exact departure walk in
	// Scene 7A"): FUN_00434d70(Rodrigo, 0x1D7, 0xB7, token, 5).
	loc7ATreppeX, loc7ATreppeY = 0x1D7, 0xB7

	// loc7ARunterCamAnchorX/Y is Scene 7A's own cinematic hard-place anchor
	// when arriving from 7B (7B.MD section 18.1 phase 3) -- deliberately
	// not the same point as 007b_Treppe's departure walk target in 7B: the
	// two directions are asymmetric (section 18.3). Rodrigo simply stays put
	// here once StairLiftRunterCam_* finishes and Scene 7A's walkable
	// control resumes -- see the "no walk right after a transition" policy
	// documented on arriveAt7BFromAbove, which supersedes
	// SMOOTH-TRANSITIONS-ZBUFFER-GLIDER.MD sections 1/2/4's original
	// "walk to a further, separate final spot" reconstruction.
	loc7ARunterCamAnchorX, loc7ARunterCamAnchorY = 0x1D8, 0xB7

	// loc7BStairsArrivalAnchorX/Y is Scene 7B's hard-place cinematic anchor
	// when arriving from Scene 7A, after StairLiftRaufCam_* has already
	// finished in 7A -- distinct from 007b_Treppe's own departure point in
	// 7B (SMOOTH-TRANSITIONS-ZBUFFER-GLIDER.MD section 3.3).
	loc7BStairsArrivalAnchorX, loc7BStairsArrivalAnchorY = 0x124, 0xCC

	// loc7BElevatorArrivalAnchorX/Y is Scene 7B's hard-place cinematic
	// anchor when arriving via the elevator (from Scene 9 or Scene 46) --
	// distinct from both the elevator's shared *departure* anchor
	// (loc7BElevatorX/Y) and the stairs-arrival anchor above
	// (SMOOTH-TRANSITIONS-ZBUFFER-GLIDER.MD sections 3.1/3.2).
	loc7BElevatorArrivalAnchorX, loc7BElevatorArrivalAnchorY = 0x161, 0xA5

	loc7BArrivalSettleX, loc7BArrivalSettleY = 0x134, 0xC6

	// loc7AStairCamFPS is both stair-camera clips' native frame rate
	// (confirmed 10fps, matching every other AVI cue table in this
	// project), used to convert loc7ARunterCamCues/loc7ARaufCamCues' frame
	// numbers into wait durations. StairHomeUpCam and FlyWegCam (below) are
	// confirmed 10fps too, so this same constant covers all of Scene 7A's
	// camera transitions.
	loc7AStairCamFPS = 10.0

	// loc7AOfficeDoorX/Y is 007a_Tuer's exact documented departure walk
	// target in Scene 7A, per
	// agents/scenes/7A-TRANSITIONS-OFFICE-GLIDER.MD section 3.
	// loc7AOfficeDoorDirection (6) is that leg's forced final direction/state.
	loc7AOfficeDoorX, loc7AOfficeDoorY     = 104, 142
	loc7AOfficeDoorDirection               = 6
	loc7AOfficeReturnX, loc7AOfficeReturnY = 0x67, 0x8B
	loc7AOfficeSettleX, loc7AOfficeSettleY = 0x7B, 0x9C

	// loc7AGliderDepartX/Y is GliderLayer's documented departure walk target
	// in Scene 7A, per 7A-TRANSITIONS-OFFICE-GLIDER.MD section 5.
	// loc7AGliderDepartDirection (0) is that leg's forced final
	// direction/state.
	loc7AGliderDepartX, loc7AGliderDepartY = 188, 170
	loc7AGliderDepartDirection             = 0

	// loc7AGliderSplitFrame is where FlyWegCam_*.avi (both variants: 65
	// frames, confirmed identical) pauses for Sfx_Glider_PassingBy before
	// continuing to its last frame, per 7A-TRANSITIONS-OFFICE-GLIDER.MD
	// section 5: frames 0-34, then 35-end.
	loc7AGliderSplitFrame = 34

	// loc7AGliderDestinationScene is where the glider departure leads: Scene
	// 8 in this same location (LOC01/8.SZN, a full-screen "FlyWegSelect.avi"
	// location-select screen per agents/scenes/08.MD, whose own four
	// destination hotspots are implemented below).
	loc7AGliderDestinationScene = "8"

	// loc8MonasteryLocation/Scene is 008_Kloster's glider-departure
	// destination when inventory item 0x0E is absent, per
	// agents/scenes/08.MD sections 6.1/14. Item 0x0E has no recovered
	// human-readable name (section 16: "do not invent names"), so it is
	// used as a raw numeric literal at the call site rather than a guessed
	// ItemXxx constant.
	loc8MonasteryLocation = 6
	loc8MonasteryScene    = "026"

	// loc8TavernLocation/Scene is 008_Kneipe's glider-departure destination,
	// per 08.MD section 7. The scene resource on disk is "S12" (LOC03/
	// S12.SZN), matching Locations 5/35's own "S"-prefixed naming.
	loc8TavernLocation = 3
	loc8TavernScene    = "S12"

	// loc8OutsideLocation/Scene is 008_Raus's glider-departure destination,
	// per 08.MD section 8. The scene resource on disk is "S113" (LOC31/
	// S113.SZN).
	loc8OutsideLocation = 31
	loc8OutsideScene    = "S113"
)

// loc7AOfficeCues is StairHomeUpCam_*.avi's documented footstep
// choreography, per 7A-TRANSITIONS-OFFICE-GLIDER.MD section 3. The two
// variants' real frame counts differ by one (confirmed via ffprobe: 32
// frames for _OK, 31 for _MK, both 10fps) -- unlike every other _OK/_MK pair
// in this location, which match exactly. The final cue below uses frame 30
// (_MK's last frame) rather than _OK's 31, so this choreography never
// outlasts whichever variant actually plays and exposes the bare scene
// behind it (the exact class of bug fixed for STAIRLIFTRAUFCAM_OK.AVI
// elsewhere in this file); finishing up to 0.1s before _OK's own last frame
// is a harmless, one-directional asymmetry by comparison.
var loc7AOfficeCues = []loc7AStepCue{
	{6, "Gen_StepLeft.wav"},
	{11, "Gen_StepRight.wav"},
	{18, "Gen_StepLeft.wav"},
	{24, "Gen_StepRight.wav"},
	{30, "Gen_StepLeft.wav"},
}

var loc7AOfficeDownCues = []loc7AStepCue{
	{4, "Gen_StepRight.wav"},
	{10, "Gen_StepLeft.wav"},
	{16, "Gen_StepLeft.wav"},
	{22, "Gen_StepLeft.wav"},
}

// loc7AStepCue is one frame-timed footstep SFX cue during a stair-camera
// clip (see stairCamStepCues). atFrame is when the cue fires, not a range
// start -- matching 7B.MD section 18's own "frames A -> B, then Sfx" cue
// sheet, where the SFX fires once frame B is reached.
type loc7AStepCue struct {
	atFrame int
	sfx     string
}

// loc7ARunterCamCues is StairLiftRunterCam_*.avi's recovered footstep
// choreography (7B.MD section 18.1, 7A arriving from 7B): 28 frames total,
// so the final cue's "frames 0x16 -> end" fires at the last frame, 27.
var loc7ARunterCamCues = []loc7AStepCue{
	{0x06, "Gen_StepRight.wav"},
	{0x0A, "Gen_StepLeft.wav"},
	{0x10, "Gen_StepRight.wav"},
	{0x16, "Gen_StepRight.wav"},
	{27, "Gen_StepLeft.wav"},
}

// loc7ARaufCamCues is StairLiftRaufCam_*.avi's recovered footstep
// choreography (7B.MD section 18.2, 7A departing to 7B): 30 frames total,
// so the final cue's "frames 0x19 -> end" fires at the last frame, 29.
var loc7ARaufCamCues = []loc7AStepCue{
	{0x06, "Gen_StepLeft.wav"},
	{0x0D, "Gen_StepRight.wav"},
	{0x12, "Gen_StepLeft.wav"},
	{0x19, "Gen_StepRight.wav"},
	{29, "Gen_StepLeft.wav"},
}

// stairCamStepCues fires cues against a stair-camera clip running at
// loc7AStairCamFPS, for use in Parallel alongside ctx.PlayVideo. The
// choreography plays *during* continuous, uninterrupted video playback
// (7B.MD's cue table covers each clip end to end with no gaps) -- unlike
// the elevator lever's genuine pause-then-resume split (see
// Context.PlayLayerSplit), this never pauses the video itself.
func (c LOC01Controller) stairCamStepCues(ctx *Context, cues []loc7AStepCue) engine.Task {
	var tasks []engine.Task
	clock := 0.0

	for _, cue := range cues {
		at := float64(cue.atFrame) / loc7AStairCamFPS
		if gap := at - clock; gap > 0 {
			tasks = append(tasks, ctx.Wait(gap))
		}
		tasks = append(tasks, ctx.PlaySFX(cue.sfx))
		clock = at
	}

	return engine.Sequence(tasks...)
}

// loc7AGliderLayer is Scene 7A's hovercraft prop that bobs continuously
// once the scene activates, per
// agents/techniques/SMOOTH-TRANSITIONS-ZBUFFER-GLIDER.MD sections 15-21.
const loc7AGliderLayer = "GliderLayer"

// loc7AGliderBobTickSeconds approximates one "tick" of the original's
// non-video object update scheduler (sections 18/21: "5/10/15/20 FPS
// object update classes... rather than the host monitor's refresh rate").
// GliderLayer's own SZN-declared fps is 0 (a static single-frame prop, no
// ordinary frame animation of its own) and its exact scheduler bucket
// (object+0x54, per ELEVATOR-TRANSITIONS-AND-ZBUFFER.MD section 12) has not
// been recovered, so this reuses the 10Hz tick already established
// throughout this project's other frame-timed cue sheets (the intro, the
// stair cameras) as the most defensible available default -- a
// reconstruction choice, not a proven original rate.
const loc7AGliderBobTickSeconds = 1.0 / 10.0

// loc7AGliderBobLowY/HighY are the glider's two authored interpolation
// targets (section 16); GliderLayer's own declared Y (64, see 7A.SZN) sits
// exactly between them.
const (
	loc7AGliderBobLowY  = 63.0
	loc7AGliderBobHighY = 65.0
)

// newGliderBobTask reproduces FUN_00407dc0's self-callback interpolation
// loop (sections 15-21): GliderLayer continuously bobs between Y=63 (a
// 3-tick leg) and Y=65 (a 4-tick leg), forever, starting the very first leg
// toward 63.0 to match the recovered pseudocode's initial (zero-value)
// toggle state. Run via Context.RunAmbient rather than as a normal blocking
// step, since it never completes on its own and must keep animating
// independently of the interaction lock and of whatever the player does
// afterward.
func (c LOC01Controller) newGliderBobTask(ctx *Context) engine.Task {
	return &glideBobTask{ctx: ctx}
}

type glideBobTask struct {
	ctx *Context

	active   bool
	toggle   bool // mirrors FUN_00407dc0's own toggle field; starts false (0)
	fromY    float64
	targetY  float64
	legTicks float64
	elapsed  float64
}

func (t *glideBobTask) Update(dt float64) bool {
	layer, ok := t.ctx.session.scene.Layers[loc7AGliderLayer]
	if !ok || layer == nil {
		return true // scene changed (or layer missing); nothing left to animate
	}

	if !t.active {
		t.startLeg(layer)
	}

	t.elapsed += dt
	duration := t.legTicks * loc7AGliderBobTickSeconds
	progress := 1.0
	if duration > 0 {
		progress = t.elapsed / duration
		if progress > 1 {
			progress = 1
		}
	}

	layer.Y = int(math.Round(t.fromY + (t.targetY-t.fromY)*progress))

	if progress >= 1 {
		t.startLeg(layer)
	}

	return false
}

func (t *glideBobTask) startLeg(layer *engine.Layer) {
	t.active = true
	t.fromY = float64(layer.Y)
	t.elapsed = 0

	if t.toggle {
		t.targetY = loc7AGliderBobHighY
		t.legTicks = 4
	} else {
		t.targetY = loc7AGliderBobLowY
		t.legTicks = 3
	}
	t.toggle = !t.toggle
}

func (c LOC01Controller) Enter(ctx *Context, scene string, from string) engine.Task {
	var tasks []engine.Task

	tasks = append(tasks, ctx.PlayMusic("Loc01_Smashville.wav"))

	switch scene {
	case "8":
		if from != loc7ASceneID {
			tasks = append(tasks, ctx.ChangeScene(loc7ASceneID))
		}

	case "7B":
		tasks = append(tasks,
			ctx.FreezeLayer("PressLeverMitteLayer", 0),
			ctx.HideLayer("PressLeverMitteLayer"),
		)

		// The elevator's travel video is owned by Scene 7B itself, played
		// on arrival according to which floor the player came from --
		// never by scene 9's/46's own lift button, which just transitions
		// straight here (see agents/scenes/7B.MD sections 10/11 and
		// departScene9Elevator/departScene46Elevator). Arriving from 7A is
		// different again (section 18.2 phase 4): 7A already played its own
		// departure video before switching, so 7B only needs its own
		// hard-place arrival, no video of its own. Every arrival route -- 9,
		// 46 or 7A alike -- simply hard-places Rodrigo at its own arrival
		// anchor and leaves him there: no further walk right after a
		// transition (see arriveAt7BFromAbove's doc comment for why this
		// supersedes SMOOTH-TRANSITIONS-ZBUFFER-GLIDER.MD sections 1/2/3's
		// original "walk to a further final spot" reconstruction).
		switch from {
		case "9":
			tasks = append(tasks, c.arriveAt7BFromAbove(ctx))
		case "46":
			tasks = append(tasks, c.arriveAt7BFromBelow(ctx))
		case loc7ASceneID:
			tasks = append(tasks,
				ctx.PlaceActor(loc01Actor, loc7BStairsArrivalAnchorX, loc7BStairsArrivalAnchorY),
				ctx.WalkTo(loc01Actor, loc7BArrivalSettleX, loc7BArrivalSettleY),
			)
		default:
			tasks = append(tasks,
				ctx.PlaceActor(loc01Actor, loc7BStairsArrivalAnchorX, loc7BStairsArrivalAnchorY),
				ctx.WalkTo(loc01Actor, loc7BArrivalSettleX, loc7BArrivalSettleY),
			)
		}

		if ctx.GetFlag(FlagPlantBurned) != 0 {
			tasks = append(tasks, ctx.DisableArea("007b_Pflanze"))
		}

	case "9":
		tasks = append(tasks,
			ctx.FreezeLayer("PressLeverObenLayer", 0),
			ctx.HideLayer("PressLeverObenLayer"),
		)

		if from == loc9DoorTargetScene {
			tasks = append(tasks,
				ctx.PlaceActor(loc01Actor, loc9TuerX, loc9TuerY),
				ctx.WalkTo(loc01Actor, loc9DoorArrivalX, loc9DoorArrivalY),
			)
		} else {
			tasks = append(tasks,
				ctx.PlaceActor(loc01Actor, loc9LiftX, loc9LiftY),
				ctx.WalkTo(loc01Actor, loc9LiftArrivalX, loc9LiftArrivalY),
			)
		}

	case "46":
		tasks = append(tasks, c.enterScene46(ctx, from)...)

	case loc7ASceneID:
		tasks = append(tasks, c.resetScene7AQuestLayers(ctx))
		if ctx.GetFlag(FlagCanalStoryFlag) != 0 {
			tasks = append(tasks, c.showScene7AWorker(ctx))
		}

		// GliderLayer bobs continuously from the moment Scene 7A activates,
		// regardless of which direction it was entered from and without
		// requiring player interaction (SMOOTH-TRANSITIONS-ZBUFFER-GLIDER.MD
		// section 20) -- registered as an ambient task specifically so it
		// keeps running independently of the arrival cinematic's own
		// interaction lock below, and of whatever the player does
		// afterward.
		tasks = append(tasks, ctx.RunAmbient(c.newGliderBobTask(ctx)))

		// GliderLayer is also the click target for departing to Scene 8
		// (agents/scenes/7A-TRANSITIONS-OFFICE-GLIDER.MD section 5): 7A.SZN
		// declares no Area of its own for it, so it's registered as a
		// clickable hotspot matching its own on-screen rectangle, exactly
		// like every other Scene-7A entry, regardless of arrival direction.
		tasks = append(tasks, ctx.MakeLayerClickable(loc7AGliderLayer))

		switch scene7AEntry(from) {
		case entry7AFromOffice:
			tasks = append(tasks,
				ctx.PlaceActor(loc01Actor, loc7AOfficeReturnX, loc7AOfficeReturnY),
				engine.Parallel(
					ctx.PlayVideo(c.stairCamVariant(ctx, "STAIRHOMEDOWNCAM")),
					c.stairCamStepCues(ctx, loc7AOfficeDownCues),
				),
			)
			if ctx.HasItem(7) {
				tasks = append(tasks, c.dropScene7AMoneyBag(ctx))
			} else {
				tasks = append(tasks, ctx.WalkTo(loc01Actor, loc7AOfficeSettleX, loc7AOfficeSettleY))
			}
		case entry7AFrom7B:
			// The stair-camera clip is owned by Scene 7A on both legs (unlike
			// the elevator, where 7B always owns the arrival video): 7B.MD
			// section 18.2 has Scene 7A play the outbound StairLiftRaufCam_*
			// itself before switching away (see leaveOfficeFront), and section
			// 18.1 phases 3-5 have it play the complementary
			// StairLiftRunterCam_* arrival clip here, on entering from 7B --
			// with its own cinematic hard-place anchor (asymmetric from
			// 007b_Treppe's own departure point in 7B) and its own authored
			// footstep-SFX choreography (loc7ARunterCamCues). The live actor is
			// hidden for the video's duration and shown again once it finishes,
			// simply staying at the cinematic anchor -- no further walk right
			// after a transition (see arriveAt7BFromAbove's doc comment). Unlike
			// ride7BElevator/departScene9Elevator, this leg does not end in a
			// ChangeScene (Scene 7A is already current), so there is no fresh
			// Actor instance to rely on for visibility recovery
			// (ELEVATOR-TRANSITIONS-AND-ZBUFFER.MD's "ghost actor" bug, section
			// 11, applies here just as much as to the elevator lever).
			tasks = append(tasks,
				ctx.PlaceActor(loc01Actor, loc7ARunterCamAnchorX, loc7ARunterCamAnchorY),
				ctx.HideActor(loc01Actor),
				engine.Parallel(
					ctx.PlayVideo(c.stairCamVariant(ctx, "STAIRLIFTRUNTERCAM")),
					c.stairCamStepCues(ctx, loc7ARunterCamCues),
				),
				ctx.ShowActor(loc01Actor),
			)
		case entry7AReturnHomeMarker5:
			tasks = append(tasks, c.enter7AReturnHomeMarker5(ctx))
		case entry7AReturnHomeFrom8:
			tasks = append(tasks, c.enter7AReturnHomeFrom8(ctx))
		}
	}

	if len(tasks) == 0 {
		return nil
	}

	return engine.Sequence(tasks...)
}

func isLOC01Scene(scene string) bool {
	switch scene {
	case "7A", "7B", "8", "9", "46":
		return true
	default:
		return false
	}
}

func (LOC01Controller) Exit(ctx *Context, scene string, to string) engine.Task {
	return nil
}

func (c LOC01Controller) Click(ctx *Context, area string) engine.Task {
	switch area {
	case "007b_Lift":
		return c.ride7BElevator(ctx, []string{"007b_Lift"}, "LIFTMITTERAUF.AVI", "9")
	case "007b_LiftRunter", "007b_UntererStock":
		// Both hotspots trigger the same elevator-DOWN action (event 10 for
		// both, per 7B.MD section 3), so both are disabled together for the
		// duration of the ride.
		return c.ride7BElevator(ctx, []string{"007b_LiftRunter", "007b_UntererStock"}, c.liftVariant(ctx, "LIFTMITTERUNTER"), "46")
	case "046_Gang":
		return c.sayLoc1LineFacingPerspective(ctx, loc46GangX, loc46GangY, 0, "046_ROD_08")
	case "KanalTalk":
		if ctx.GetFlag(FlagCanalStoryFlag) != 0 {
			return c.kanalTalkConversation(ctx)
		}
		return nil
	case "046_Kanal":
		if ctx.GetFlag(FlagCanalStoryFlag) != 0 {
			return c.kanalTalkBanter(ctx)
		}
		return c.enterCanalToLocation7(ctx)
	case "046_Lift":
		return c.departScene46Elevator(ctx)
	case "046_Schild":
		return c.randomLoc1SignLine(ctx, loc46SchildX, loc46SchildY, 6)
	case "046_Schild2":
		return c.randomLoc1SignLine(ctx, loc46Schild2X, loc46Schild2Y, 6)
	case "009_Lift":
		return c.departScene9Elevator(ctx)
	case "009_Pflanze":
		return c.sayLoc1Line(ctx, loc9PflanzeX, loc9PflanzeY, "009_ROD_03")
	case "009_SchildLinks":
		return c.sayLoc1Line(ctx, loc9SchildLinksX, loc9SchildLinksY, "009_ROD_01")
	case "009_SchildRechts":
		return c.sayLoc1Line(ctx, loc9SchildRechtsX, loc9SchildRechtsY, "009_ROD_02")
	case "009_Klingel":
		return c.sayLoc1Line(ctx, loc9KlingelX, loc9KlingelY, "009_ROD_05")
	case "009_Besen":
		return c.sayLoc1Line(ctx, loc9BesenX, loc9BesenY, "009_ROD_04")
	case "009_Tuer":
		return c.goThroughScene9Door(ctx)
	case "007b_Pflanze":
		return engine.Sequence(
			ctx.WalkTo(loc01Actor, 0x182, 0xE3),
			ctx.SetActorDirection(loc01Actor, 6),
			ctx.Say(loc01Actor, "007_ROD_05", "[007_ROD_05]"),
		)
	case "007b_Treppe":
		return c.goToOfficeFront(ctx)
	case "007a_Pflanze":
		return engine.Sequence(
			ctx.WalkTo(loc01Actor, 0x13B, 0x8C),
			ctx.SetActorDirection(loc01Actor, 3),
			ctx.Say(loc01Actor, "007_ROD_05", "[007_ROD_05]"),
		)
	case "007a_Schild":
		line := []string{"007_ROD_02", "007_ROD_03", "007_ROD_04"}[rand.IntN(3)]
		return engine.Sequence(
			ctx.WalkTo(loc01Actor, 0x1BD, 0xBF),
			ctx.SetActorDirection(loc01Actor, 7),
			ctx.Say(loc01Actor, line, "["+line+"]"),
		)
	case "007a_Treppe":
		return c.leaveOfficeFront(ctx)
	case "007a_Tuer":
		return c.goThroughOfficeDoor(ctx)
	case loc7AGliderLayer:
		return c.departViaGlider(ctx)
	case "008_Office":
		// 08.MD section 5: direct, no FlyGanzWegCam playback on this leg.
		return ctx.ChangeScene(loc7ASceneID)
	case "008_Kloster":
		return c.departScene8Kloster(ctx)
	case "008_Kneipe":
		return c.beginScene8GliderDeparture(ctx, loc8TavernLocation, loc8TavernScene)
	case "008_Raus":
		return c.beginScene8GliderDeparture(ctx, loc8OutsideLocation, loc8OutsideScene)
	default:
		return nil
	}
}

// departScene8Kloster reproduces agents/scenes/08.MD section 6: with
// inventory item 0x0E present, Rodrigo just comments (008_ROD_01) and stays
// in Scene 8 -- played via PlayVoiceover rather than Say, since Scene 8 has
// no on-screen Actor of its own to attribute the line to or animate (8.SZN
// declares no Character at all, per 08.MD section 2). Otherwise, the glider
// departure to Location 6's monastery runs as normal.
func (c LOC01Controller) departScene8Kloster(ctx *Context) engine.Task {
	if ctx.HasItem(0x0E) {
		return ctx.PlayVoiceover("008_ROD_01", "[008_ROD_01]")
	}
	return c.beginScene8GliderDeparture(ctx, loc8MonasteryLocation, loc8MonasteryScene)
}

// beginScene8GliderDeparture reproduces agents/scenes/08.MD section 9's
// shared Scene-8 travel sequence -- Sfx_Glider_PassingBy, then
// FlyGanzWegCam.avi to completion, only then switching to destination --
// used identically by 008_Kloster (when allowed), 008_Kneipe and 008_Raus.
// The doc's "authored sound parameter/fade" step (section 10) has no
// recovered concrete mechanism in this engine, so this plays a plain
// fire-and-forget SFX instead of guessing at a fade implementation. Scene 8
// has no on-screen Actor (08.MD section 2), so unlike every LOC01Actor
// transition elsewhere in this location, there is nothing to hide/show here.
func (c LOC01Controller) beginScene8GliderDeparture(ctx *Context, location int, scene string) engine.Task {
	return engine.Sequence(
		ctx.PlaySFX("Sfx_Glider_PassingBy.wav"),
		ctx.PlayVideo("FLYGANZWEGCAM.AVI"),
		ctx.ChangeLocation(location, scene),
	)
}

// liftVariant selects the _OK or _MK cut of a lift-shaft video between 7B
// and 46, keyed by the same FlagCanalStoryFlag that drives Scene 46's own
// background/interaction choice (see agents/scenes/46.MD section 1 and
// 7B.MD section 2): _OK when the flag is 0 (the default/normal
// configuration, matching 46.MD's "state 0x3E94==0" loading back46_ok), _MK
// otherwise. This polarity is inferred from that shared naming convention
// (back46_ok/back46_mk, and _OK/_MK on the videos themselves) rather than
// separately proven for the video files specifically, but it is the only
// reading consistent with both docs describing the same state field.
func (c LOC01Controller) liftVariant(ctx *Context, base string) string {
	if ctx.GetFlag(FlagCanalStoryFlag) != 0 {
		return base + "_MK.AVI"
	}
	return base + "_OK.AVI"
}

// SelectItem: none of LOC01's items are directly activated, only used on
// hotspots (see UseItem).
func (c LOC01Controller) SelectItem(ctx *Context, item int) engine.Task {
	if item != 0x0D {
		return nil
	}

	switch ctx.session.state.Scene {
	case loc7ASceneID:
		if ctx.GetFlag(FlagCanalStoryFlag) != 0 {
			return c.activateScene7AKeystone(ctx)
		}
	case "46":
		return ctx.Say(loc01Actor, "046_ROD_09", "[046_ROD_09]")
	}

	return nil
}

func (c LOC01Controller) activateScene7AKeystone(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.WalkToFacing(loc01Actor, 0x7B, 0x9C, 7),
		ctx.HideActor(loc01Actor),
		ctx.ShowLayer("TrootLayer"),
		ctx.PlayLayerFrames("TrootLayer", 0, 0x36),
		ctx.PlaySFX("Sfx_SignalHorn_Modern.wav"),
		ctx.PlayLayerFrames("TrootLayer", 0x37, 0x44),
		ctx.ShowLayer("KanalLowTalk"),
		ctx.RunAmbient(ctx.PlayLayerFrames("TrootLayer", 0x45, 0x50)),
		ctx.PlayLayerFrames("KanalLowTalk", 0, 5),
		ctx.PlaySpeechBoundToLayer("KanalLowTalk", "007_KAN_01", "[007_KAN_01]", 5, 9),
		ctx.HideLayer("TrootLayer"),
		ctx.ShowActor(loc01Actor),
		ctx.HideLayer("KanalLowTalk"),
		ctx.ShowLayer("KanalLowRun"),
		ctx.PlayLayerFrames("KanalLowRun", 0, 0x0F),
		ctx.PlaySFX("Sfx_Woosh.wav"),
		ctx.PlayLayerFrames("KanalLowRun", 0x10, -1),
		ctx.PlaySFX("Sfx_Tock.wav"),
		ctx.ShowLayer("KanalWegLayer"),
		ctx.SetFlag(FlagCanalStoryFlag, 0),
	)
}

// LoadConditionMask reproduces 46.MD section 16's / agents/scenes/7A-46.MD's
// scene condition masks. LoadCond is a bitmask requirement compared against
// this mask -- never a reference to a "controller ID" (7A-46.MD sections
// 1/18/26) -- so every case here returns the actual bits the original
// scene's own constructor would have computed, not an object lookup.
func (LOC01Controller) LoadConditionMask(ctx *Context, scene string) int {
	switch scene {
	case loc7ASceneID:
		return loc7AConditionMask(ctx)
	case "46":
		// 7A-46.MD section 14: returning from Scene 47 resets the shared
		// story state *before* this scene's own mask is computed (and thus
		// before KanalTalk's LoadCond=1 is evaluated below) -- doing the
		// reset later, as an Enter-task side effect (see enterScene46),
		// would compute this mask against the stale, not-yet-cleared flag,
		// letting KanalTalk load for one scene instance despite the
		// intended reset. ctx.session.state.Scene still holds the
		// *departing* scene here: Session.transitionTo computes this mask
		// before overwriting it with the destination.
		if ctx.session.state.Scene == loc46DoorTargetScene {
			ctx.session.state.Flags[FlagCanalStoryFlag] = 0
		}
		if ctx.GetFlag(FlagCanalStoryFlag) != 0 {
			return 1
		}
		return 0
	default:
		return 0
	}
}

// loc7AConditionMask reproduces agents/scenes/7A-46.MD section 3's
// three-bit Scene-7A condition mask, gating TrootLayer/SackAnimLayer/
// KanalLowRun/KanalLowTalk (section 9's mapping table). Item IDs 7 and 0x0D
// have no recovered human-readable name (section 25: "do not invent
// names"), so they appear here as raw numeric literals rather than guessed
// ItemXxx constants; FlagCanalStoryFlag is the same shared story-state field
// Scene 46's own mask above reads (7A-46.MD's "storyState_3E94").
func loc7AConditionMask(ctx *Context) int {
	mask := 0
	storyActive := ctx.GetFlag(FlagCanalStoryFlag) != 0

	if storyActive {
		mask |= 1
	}
	if ctx.HasItem(7) {
		mask |= 2
	}
	if storyActive && ctx.HasItem(0x0D) {
		mask |= 4
	}

	return mask
}

func (LOC01Controller) resetScene7AQuestLayers(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		for _, id := range []string{"SackAnimLayer", "TrootLayer", "KanalLowRun"} {
			if layer, ok := ctx.session.scene.Layers[id]; ok {
				layer.Frame = 0
				layer.Accumulator = 0
				layer.Playing = false
				layer.Visible = false
				if id == "TrootLayer" || id == "KanalLowRun" {
					layer.Z = 0
					layer.ZBuffered = false
				}
			}
		}
	})
}

func (LOC01Controller) showScene7AWorker(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		layer, ok := ctx.session.scene.Layers["KanalLowTalk"]
		if !ok {
			return
		}
		layer.Enabled = true
		layer.Visible = true
		layer.Playing = false
		layer.Frame = 0
		layer.Accumulator = 0
		layer.Z = 0
		layer.ZBuffered = false
	})
}

func (c LOC01Controller) UseItem(ctx *Context, item int, area string) engine.Task {
	if area == "007b_Pflanze" {
		if item == ItemMatches {
			return c.burnPlant(ctx)
		}
		return ctx.SayText(loc01Actor, "That doesn't seem like the right thing to use here.", 1.2)
	}

	return ctx.SayText(loc01Actor, "That doesn't seem like the right thing to use here.", 1.2)
}

// ride7BElevator reproduces Scene 7B's own elevator-departure sequence
// (agents/scenes/7B.MD sections 8/9/12, "Both elevator directions share the
// same preparation", refined by
// agents/techniques/ELEVATOR-TRANSITIONS-AND-ZBUFFER.MD sections 1-4): walk
// to the fixed elevator spot (not areaIDs' own centers), force the
// documented direction/state, WAIT for that movement to actually finish
// (guaranteed here simply by these being sequential Sequence steps -- the
// lever can never start early), then hide the live actor before the lever
// animation starts (so it isn't rendered doubled against the lever/video's
// own depiction of Rodrigo -- ELEVATOR-TRANSITIONS-AND-ZBUFFER.MD's "ghost
// actor" bug), play PressLeverMitteLayer's authored two-part press
// animation around Sfx_ClickelDiClack, start Sfx_Lift, play the
// direction-specific travel AVI, and only then change scenes. areaIDs are
// every hotspot that triggers this same ride (e.g. both 007b_LiftRunter and
// 007b_UntererStock for DOWN); all are disabled together immediately on
// click. The actor never needs to be shown again here: ChangeScene always
// instantiates a fresh destination Scene (and thus a fresh, visible
// Actor).
func (c LOC01Controller) ride7BElevator(ctx *Context, areaIDs []string, videoFile, targetScene string) engine.Task {
	tasks := make([]engine.Task, 0, len(areaIDs)+7)
	for _, id := range areaIDs {
		tasks = append(tasks, ctx.DisableArea(id))
	}

	tasks = append(tasks,
		ctx.WalkTo(loc01Actor, loc7BElevatorX, loc7BElevatorY),
		ctx.SetActorDirection(loc01Actor, loc7BElevatorDirection),
		ctx.HideActor(loc01Actor),
		ctx.HideLayer("Hebel"),
		ctx.ShowLayer("PressLeverMitteLayer"),
		ctx.PlayLayerSplit("PressLeverMitteLayer", loc7BLeverSplitFrame, "Sfx_ClickelDiClack.wav"),
		ctx.PlaySFX("Sfx_Lift.wav"),
		ctx.PlayVideo(videoFile),
		ctx.ChangeScene(targetScene),
	)

	return engine.Sequence(tasks...)
}

// departScene9Elevator reproduces 9.MD sections 4/14 ("a source-scene lever
// animation followed by a destination-scene elevator AVI"), refined by
// ELEVATOR-TRANSITIONS-AND-ZBUFFER.MD section 7: walk to the fixed elevator
// spot, force the documented direction/state, hide the live actor before
// the lever starts (same "ghost actor" reasoning as ride7BElevator), play
// PressLeverObenLayer's own two-part press animation split around
// Sfx_ClickelDiClack, then switch straight to Scene 7B with no travel video
// here -- 7B's own Enter plays LiftRaufMitte.avi on arrival (see
// arriveAt7BFromAbove), never Scene 9 itself.
func (c LOC01Controller) departScene9Elevator(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.DisableArea("009_Lift"),
		ctx.WalkTo(loc01Actor, loc9LiftX, loc9LiftY),
		ctx.SetActorDirection(loc01Actor, loc9LiftDirection),
		ctx.HideActor(loc01Actor),
		ctx.HideLayer("HebelOben"),
		ctx.ShowLayer("PressLeverObenLayer"),
		ctx.PlayLayerSplit("PressLeverObenLayer", loc9LeverSplitFrame, "Sfx_ClickelDiClack.wav"),
		ctx.ChangeScene("7B"),
	)
}

// sayLoc1Line walks to (x, y) then plays baseName's real recorded line, per
// 9.MD sections 6-11 (the plant, both signs, the bell and the broom) and
// 46.MD section 3 (046_Gang). The subtitle is a bracketed placeholder
// rather than invented English text: matching a real recorded line to its
// actual translated subtitle requires the original dialogue script, which
// is Milestone 6 content-porting work (see agents/GAMEPLAY.md) -- this at
// least plays the authentic recorded audio the docs specify rather than
// substituting made-up dialogue for it.
func (c LOC01Controller) sayLoc1Line(ctx *Context, x, y float64, baseName string) engine.Task {
	return engine.Sequence(
		ctx.WalkTo(loc01Actor, x, y),
		ctx.Say(loc01Actor, baseName, "["+baseName+"]"),
	)
}

func (c LOC01Controller) sayLoc1LineFacingPerspective(ctx *Context, x, y float64, direction int, baseName string) engine.Task {
	return engine.Sequence(
		ctx.WalkToFacingPerspective(loc01Actor, x, y, direction),
		ctx.Say(loc01Actor, baseName, "["+baseName+"]"),
	)
}

// goThroughScene9Door reproduces 9.MD section 9: walk to the door, then
// leave Location 1 entirely for Location 5 / Scene S10 (Loui's office) -- a
// real, source-proven cross-location exit (see LOC05Controller for why
// Scene S10 itself is only a placeholder beyond that).
func (c LOC01Controller) goThroughScene9Door(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.WalkTo(loc01Actor, loc9TuerX, loc9TuerY),
		ctx.ChangeLocation(loc05Location, loc9DoorTargetScene),
	)
}

// goToOfficeFront reproduces 7B.MD section 17 (007b_Treppe): walk to the
// documented exit point and switch straight to the office-front scene, with
// no lever or elevator AVI on this leg at all -- the doc is explicit that
// "the Scene-7B callback itself does not play a lever animation or elevator
// AVI on this path". The complementary StairLiftRaufCam_* clip belongs to
// Scene 7's own departure the other way (see leaveOfficeFront), and its
// StairLiftRunterCam_* arrival counterpart to Scene 7's own Enter (see the
// loc7ASceneID case above) -- never to 7B.
func (c LOC01Controller) goToOfficeFront(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.WalkTo(loc01Actor, loc7BTreppeX, loc7BTreppeY),
		ctx.ChangeScene(loc7ASceneID),
	)
}

// leaveOfficeFront reproduces 7B.MD section 18.2 ("7A / Scene 7 -> 7B"):
// walk to the exact documented departure point, then -- while still in
// Scene 7A, before switching -- play the outbound StairLiftRaufCam_* clip
// with its own authored footstep-SFX choreography (loc7ARaufCamCues).
// Unlike the elevator, this leg's video is owned by the *departure* scene;
// only once it finishes does the scene actually switch (7B.MD section
// 18.3's documented directional asymmetry). The live actor is hidden for the
// video's duration, exactly like ride7BElevator/departScene9Elevator's own
// "ghost actor" fix (ELEVATOR-TRANSITIONS-AND-ZBUFFER.MD section 11):
// ChangeScene right after gives Scene 7B a fresh, visible Actor, so no
// ShowActor is needed here.
func (c LOC01Controller) leaveOfficeFront(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.DisableArea("007a_Treppe"),
		ctx.WalkTo(loc01Actor, loc7ATreppeX, loc7ATreppeY),
		ctx.HideActor(loc01Actor),
		engine.Parallel(
			ctx.PlayVideo(c.stairCamVariant(ctx, "STAIRLIFTRAUFCAM")),
			c.stairCamStepCues(ctx, loc7ARaufCamCues),
		),
		ctx.ChangeScene("7B"),
	)
}

// stairCamVariant selects the _OK or _MK cut of a Scene 7/7B stair-camera
// clip. Neither 7B.MD nor 1.MD document what selects between them (unlike
// the elevator's explicit state+0x3E94, see liftVariant) -- this reuses the
// same FlagCanalStoryFlag/liftVariant convention for consistency with every
// other _OK/_MK pair in this location, not because it is proven to be the
// same selector here.
func (c LOC01Controller) stairCamVariant(ctx *Context, base string) string {
	return c.liftVariant(ctx, base)
}

// scene7AEntry selects Scene 7A's Enter branch from the destination's
// previous-scene marker (Enter's `from` argument), per dec(2).c: marker 5
// → FUN_004072e0 first-return; "7B" → stair arrival clip; everything else
// → generic spawn (see agents/scenes/116_TO_7A_CORRECT_DEC2_RUN_PATH.MD).
type scene7AEntryReason int

const (
	entry7ANormal scene7AEntryReason = iota
	entry7AFromOffice
	entry7AFrom7B
	entry7AReturnHomeMarker5 // previousScene == "5", FUN_004072e0 marker-5 branch
	entry7AReturnHomeFrom8
)

func scene7AEntry(from string) scene7AEntryReason {
	switch from {
	case loc7ADoorTargetScene:
		return entry7AFromOffice
	case "7B":
		return entry7AFrom7B
	case loc7AFirstEncounterFrom, "":
		return entry7AReturnHomeMarker5
	case "8", "S113":
		return entry7AReturnHomeFrom8
	default:
		return entry7ANormal
	}
}

// enter7AReturnHomeMarker5 is FUN_004072e0 for previousScene == 5, per
// agents/scenes/116_TO_7A_CORRECT_DEC2_RUN_PATH.MD: PlaceImmediate
// (188,170), Sfx_Glider_PassingBy, active presentation ReturnHomeCam_ok/mk
// (no reconstruction-only HideActor — the original leaves Rodrigo placed and
// lets the camera object own presentation), orientation 0 + wait, Loc01/
// 005_ROD_01 + wait, then FUN_00407500 (office door: walk (104,142) dir 6,
// StairHomeUpCam, Location 4 / Scene 006).

func (c LOC01Controller) dropScene7AMoneyBag(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.WalkToFacing(loc01Actor, loc7AOfficeSettleX, loc7AOfficeSettleY, 7),
		ctx.HideActor(loc01Actor),
		ctx.ShowLayer("SackAnimLayer"),
		ctx.PlayLayerFrames("SackAnimLayer", 0, 0x21),
		ctx.PlaySFX("Sfx_MoneyBag.wav"),
		ctx.PlayLayerFrames("SackAnimLayer", 0x22, 0x32),
		ctx.PlaySFX("Sfx_MoneyBag.wav"),
		ctx.PlayLayerFrames("SackAnimLayer", 0x33, 0x46),
		ctx.ShowLayer("SackLayer"),
		engine.Immediate(func() {
			putOriginalFlag(ctx.session.state.OriginalState, 0x3e98, 1)
		}),
		ctx.RemoveItem(7),
		ctx.PlayLayerFrames("SackAnimLayer", 0x47, -1),
		ctx.HideLayer("SackLayer"),
		ctx.PlaySFX("Sfx_Bubble.wav"),
		ctx.HideLayer("SackAnimLayer"),
		ctx.ShowActor(loc01Actor),
		engine.Immediate(func() {
			putOriginalFlag(ctx.session.state.OriginalState, 0x3e9c, 1)
		}),
		ctx.Say(loc01Actor, "007_ROD_01", "[007_ROD_01]"),
	)
}

func (c LOC01Controller) enter7AReturnHomeFrom8(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.PlaceActor(loc01Actor, loc7AGliderDepartX, loc7AGliderDepartY),
		ctx.PlaySFX("Sfx_Glider_PassingBy.wav"),
		ctx.PlayVideo(c.stairCamVariant(ctx, "RETURNHOMECAM")),
		ctx.WalkTo(loc01Actor, 0x13b, 0x9d),
	)
}

func (c LOC01Controller) enter7AReturnHomeMarker5(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.PlaceActor(loc01Actor, loc7AGliderDepartX, loc7AGliderDepartY),
		ctx.PlaySFX("Sfx_Glider_PassingBy.wav"),
		ctx.PlayVideo(c.stairCamVariant(ctx, "RETURNHOMECAM")),
		ctx.SetActorOrientation(loc01Actor, loc7AGliderDepartDirection),
		ctx.Say(loc01Actor, "005_ROD_01", "[005_ROD_01]"),
		c.goStraightThroughOfficeDoor(ctx),
	)
}

// goThroughOfficeDoor reproduces agents/scenes/7A-TRANSITIONS-OFFICE-GLIDER.MD
// section 3 (007a_Tuer): walk to the exact documented departure anchor,
// force the documented direction/state, hide the live actor before the
// StairHomeUpCam_* clip plays (section 6's "ghost actor" policy -- matching
// ELEVATOR-TRANSITIONS-AND-ZBUFFER.MD's own fix elsewhere in this location),
// then leave Location 1 entirely for Location 4 / Scene 006, Rodrigo's
// office interior, only once the clip finishes. ChangeLocation gives Scene
// 006 a fresh, visible Actor, so no ShowActor is needed here (see
// arriveAt7BFromAbove's "no walk right after a transition" doc comment --
// the same reasoning: nothing further happens to Rodrigo in Scene 7A after
// this). LOC04Controller's own Enter continues the visual handoff with
// RodRein (section 4).
func (c LOC01Controller) goThroughOfficeDoor(ctx *Context) engine.Task {
	return c.goThroughOfficeDoorWithWalk(ctx, ctx.WalkTo(loc01Actor, loc7AOfficeDoorX, loc7AOfficeDoorY))
}

func (c LOC01Controller) goStraightThroughOfficeDoor(ctx *Context) engine.Task {
	return c.goThroughOfficeDoorWithWalk(ctx, ctx.WalkStraightTo(loc01Actor, loc7AOfficeDoorX, loc7AOfficeDoorY))
}

func (c LOC01Controller) goThroughOfficeDoorWithWalk(ctx *Context, walk engine.Task) engine.Task {
	return engine.Sequence(
		walk,
		ctx.SetActorDirection(loc01Actor, loc7AOfficeDoorDirection),
		ctx.HideActor(loc01Actor),
		engine.Parallel(
			ctx.PlayVideo(c.stairCamVariant(ctx, "STAIRHOMEUPCAM")),
			c.stairCamStepCues(ctx, loc7AOfficeCues),
		),
		ctx.ChangeLocation(loc06Location, loc7ADoorTargetScene),
	)
}

// departViaGlider reproduces agents/scenes/7A-TRANSITIONS-OFFICE-GLIDER.MD
// section 5: walk to the documented departure anchor, force the documented
// direction/state, hide the live actor (section 6), then play
// FlyWegCam_ok/mk split at its documented frame boundary for
// Sfx_Glider_PassingBy -- never fired on click, only at that authored video
// boundary -- switching to Scene 8 only once the whole clip finishes. No
// ShowActor is needed: ChangeScene gives Scene 8 a fresh Actor (which, per
// 8.SZN, that scene doesn't even declare -- it's a full-screen AVI
// location-select screen per agents/techniques/05_SMASHVILLE_OFFICE.MD,
// whose own hotspots are out of scope here).
func (c LOC01Controller) departViaGlider(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.WalkTo(loc01Actor, loc7AGliderDepartX, loc7AGliderDepartY),
		ctx.SetActorDirection(loc01Actor, loc7AGliderDepartDirection),
		ctx.HideActor(loc01Actor),
		ctx.PlayVideoSplit(c.stairCamVariant(ctx, "FLYWEGCAM"), loc7AGliderSplitFrame, "Sfx_Glider_PassingBy.wav"),
		ctx.ChangeScene(loc7AGliderDestinationScene),
	)
}

func (c LOC01Controller) arriveAt7BFromAbove(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.PlaceActor(loc01Actor, loc7BElevatorArrivalAnchorX, loc7BElevatorArrivalAnchorY),
		ctx.HideActor(loc01Actor),
		ctx.PlaySFX("Sfx_Lift.wav"),
		ctx.PlayVideo("LIFTRAUFMITTE.AVI"),
		ctx.ShowActor(loc01Actor),
		ctx.WalkTo(loc01Actor, loc7BArrivalSettleX, loc7BArrivalSettleY),
	)
}

func (c LOC01Controller) arriveAt7BFromBelow(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.PlaceActor(loc01Actor, loc7BElevatorArrivalAnchorX, loc7BElevatorArrivalAnchorY),
		ctx.HideActor(loc01Actor),
		ctx.PlaySFX("Sfx_ClickelDiClack.wav"),
		ctx.PlaySFX("Sfx_Lift.wav"),
		ctx.PlayVideo(c.liftVariant(ctx, "LIFTRUNTERMITTE")),
		ctx.ShowActor(loc01Actor),
		ctx.WalkTo(loc01Actor, loc7BArrivalSettleX, loc7BArrivalSettleY),
	)
}

// enterScene46 builds Scene 46's own entry reconstruction (agents/scenes/
// 46.MD sections 1/10/11), returned as a slice so Enter can append it
// alongside whatever else that switch case ever needs.
//
// Arriving from Scene 47 is the documented special case: FlagCanalStoryFlag
// has already been reset to 0 by LOC01Controller.LoadConditionMask, before
// Scene 46's own condition mask was even computed (7A-46.MD section 14 --
// doing the reset here instead, after instantiation, would be one scene
// instance too late for KanalTalk's own LoadCond=1 check). This continuation
// positions Rodrigo at the canal-return spot and plays KANALRAUSCAM.AVI
// split around Sfx_Clothes_Scratched (46.MD section 10). Any other entry
// (the elevator, or the session's first load) places Rodrigo at the lift and
// leaves him there -- no further walk right after a transition (see
// arriveAt7BFromAbove's doc comment). The KanalTalk layer itself needs no
// explicit ShowLayer/HideLayer here: its LoadCond=1 is resolved correctly at
// instantiation time via LoadConditionMask, so it loads visible exactly when
// FlagCanalStoryFlag is active and stays hidden otherwise, on its own. The
// Scene-47 branch hides the live actor for KANALRAUSCAM.AVI and shows it
// again immediately after, since nothing else in this leg (no further
// WalkTo, no ChangeScene) would otherwise undo the hide
// (ELEVATOR-TRANSITIONS-AND-ZBUFFER.MD section 11's "ghost actor" bug).
func (c LOC01Controller) enterScene46(ctx *Context, from string) []engine.Task {
	if from == loc46DoorTargetScene {
		return []engine.Task{
			ctx.SetBackground("back46_ok.bmp"),
			ctx.PlaceActorPerspective(loc01Actor, loc46KanalRausEntryX, loc46KanalRausEntryY),
			ctx.HideActor(loc01Actor),
			ctx.PlayVideoSplit("KANALRAUSCAM.AVI", loc46KanalRausSplitFrame, "Sfx_Clothes_Scratched.wav"),
			ctx.ShowActor(loc01Actor),
			ctx.WalkToPerspective(loc01Actor, loc46ReturnSettleX, loc46ReturnSettleY),
		}
	}

	tasks := []engine.Task{
		ctx.PlaceActorPerspective(loc01Actor, loc46LiftX, loc46LiftY),
	}

	if ctx.GetFlag(FlagCanalStoryFlag) == 0 {
		tasks = append(tasks, ctx.SetBackground("back46_ok.bmp"))
	} else {
		tasks = append(tasks,
			ctx.DisableArea("046_Gang"),
			ctx.MakeLayerClickable("KanalTalk"),
			ctx.RunAmbient(ctx.PlayLayerFrames("KanalTalk", 0, 5)),
		)
	}

	tasks = append(tasks, ctx.WalkToPerspective(loc01Actor, loc46EntrySettleX, loc46EntrySettleY))
	return tasks
}

// enterCanalToLocation7 reproduces 46.MD section 4.2 (FlagCanalStoryFlag ==
// 0, the default): walk to the canal, play KANALREINCAM.AVI split around
// Sfx_Clothes_Scratched, then leave Location 1 entirely for Location 7 /
// Scene 47 -- a real, source-proven cross-location exit (see LOC07Controller
// for why Scene 47 itself is only a placeholder beyond that). The live actor
// is hidden for the video's duration; ChangeLocation right after gives the
// destination location's Scene 47 a fresh, visible Actor, so no ShowActor is
// needed here (same reasoning as ride7BElevator/departScene9Elevator).
func (c LOC01Controller) enterCanalToLocation7(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.WalkToFacingPerspective(loc01Actor, loc46KanalX, loc46KanalY, 3),
		ctx.HideActor(loc01Actor),
		ctx.PlayVideoSplit("KANALREINCAM.AVI", loc46KanalReinSplitFrame, "Sfx_Clothes_Scratched.wav"),
		ctx.ChangeLocation(loc07Location, loc46DoorTargetScene),
	)
}

// kanalTalkBanter reproduces 46.MD section 4.1 (FlagCanalStoryFlag != 0):
// the canal stays an in-scene interaction, playing part of the KanalTalk
// animation (frames 5-9) alongside one randomly chosen line.
func (c LOC01Controller) kanalTalkBanter(ctx *Context) engine.Task {
	line := []string{"046_KAN_06", "046_KAN_07", "046_KAN_08"}[rand.IntN(3)]

	return engine.Sequence(
		ctx.WalkToFacingPerspective(loc01Actor, loc46KanalX, loc46KanalY, 3),
		ctx.PlaySpeechBoundToLayer("KanalTalk", line, "["+line+"]", 5, 9),
	)
}

// randomLoc1SignLine reproduces 46.MD sections 7/8: both Scene-46 signs
// walk to their own (different) target, then share the same randomized
// pool of three generic Location-1 Rodrigo lines.
func (c LOC01Controller) randomLoc1SignLine(ctx *Context, x, y float64, direction int) engine.Task {
	line := []string{"007_ROD_02", "007_ROD_03", "007_ROD_04"}[rand.IntN(3)]

	return engine.Sequence(
		ctx.WalkToFacingPerspective(loc01Actor, x, y, direction),
		ctx.Say(loc01Actor, line, "["+line+"]"),
	)
}

// departScene46Elevator reproduces 46.MD section 5 (refined by
// ELEVATOR-TRANSITIONS-AND-ZBUFFER.MD section 8): walk to the lift, force
// the documented direction/state, then switch straight to Scene 7B, with no
// travel video here -- 7B's own Enter plays LiftRunterMitte_ok/mk.avi on
// arrival (see arriveAt7BFromBelow), never Scene 46 itself. Unlike 7B's own
// elevator or Scene 9's, no lever animation is documented for this leg, so
// the live actor is not hidden here -- ELEVATOR-TRANSITIONS-AND-ZBUFFER.MD's
// own "Scene 46 UP" regression case (section 19) has no such step either.
// If FlagCanalStoryFlag is active, the doc also documents a KanalTalk
// animation call with an unusual range (start=5, end=0) that it explicitly
// flags as unclear, possibly reverse-direction playback -- not supported by
// this engine's forward-only layer stepping. Rather than guess at
// unsupported semantics, this approximates it as a brief pause on the
// layer's frame-5 pose.
func (c LOC01Controller) departScene46Elevator(ctx *Context) engine.Task {
	tasks := []engine.Task{
		ctx.DisableArea("046_Lift"),
		ctx.WalkToFacingPerspective(loc01Actor, loc46LiftX, loc46LiftY, loc46LiftDirection),
	}

	if ctx.GetFlag(FlagCanalStoryFlag) != 0 {
		tasks = append(tasks, ctx.PlayLayerFrames("KanalTalk", 5, 0))
	}

	tasks = append(tasks, ctx.ChangeScene("7B"))

	return engine.Sequence(tasks...)
}

func loc01SetOriginalFlag(ctx *Context, offset, value int) engine.Task {
	return engine.Immediate(func() {
		putOriginalFlag(ctx.session.state.OriginalState, offset, value)
	})
}

// kanalTalkConversation reproduces the original stateful KanalTalk dialogue.
// KanalTalk owns event 0x2D while 046_Gang is disabled. state+0x3E98 selects
// the dialogue stage, state+0x3E9C selects the branch, and state+0x3EC4 is
// set after 046_KAN_04. Worker speech drives frames 5..9 through its ACS track.
func (c LOC01Controller) kanalTalkConversation(ctx *Context) engine.Task {
	stage := getOriginalFlag(ctx.session.state.OriginalState, loc01StateCanalTalkStage)
	if stage == 0 {
		stage = 1
	}

	tasks := []engine.Task{ctx.WalkToFacingPerspective(loc01Actor, loc46KanalTalkX, loc46KanalTalkY, 3)}
	branch := getOriginalFlag(ctx.session.state.OriginalState, loc01StateCanalTalkBranch)

	if branch == 0 {
		if stage == 1 {
			tasks = append(tasks,
				ctx.Say(loc01Actor, "046_ROD_01", "[046_ROD_01]"),
				ctx.PlaySpeechBoundToLayer("KanalTalk", "046_KAN_01", "[046_KAN_01]", 5, 9),
				loc01SetOriginalFlag(ctx, loc01StateCanalTalkStage, 2),
			)
		} else {
			tasks = append(tasks, ctx.Say(loc01Actor, "046_ROD_02", "[046_ROD_02]"))
		}
		return engine.Sequence(tasks...)
	}

	switch stage {
	case 1:
		tasks = append(tasks,
			ctx.Say(loc01Actor, "046_ROD_03", "[046_ROD_03]"),
			ctx.PlaySpeechBoundToLayer("KanalTalk", "046_KAN_02", "[046_KAN_02]", 5, 9),
			loc01SetOriginalFlag(ctx, loc01StateCanalTalkStage, 2),
		)
	case 2:
		tasks = append(tasks,
			ctx.Say(loc01Actor, "046_ROD_04", "[046_ROD_04]"),
			ctx.PlaySpeechBoundToLayer("KanalTalk", "046_KAN_03", "[046_KAN_03]", 5, 9),
			loc01SetOriginalFlag(ctx, loc01StateCanalTalkStage, 3),
		)
	case 3:
		tasks = append(tasks,
			ctx.Say(loc01Actor, "046_ROD_05", "[046_ROD_05]"),
			ctx.PlaySpeechBoundToLayer("KanalTalk", "046_KAN_04", "[046_KAN_04]", 5, 9),
			loc01SetOriginalFlag(ctx, loc01StateCanalTalkStage, 4),
			loc01SetOriginalFlag(ctx, loc01StateCanalTalkUnlock, 1),
		)
	case 4:
		tasks = append(tasks,
			ctx.Say(loc01Actor, "046_ROD_06", "[046_ROD_06]"),
			ctx.PlaySpeechBoundToLayer("KanalTalk", "046_KAN_05", "[046_KAN_05]", 5, 9),
			loc01SetOriginalFlag(ctx, loc01StateCanalTalkStage, 5),
		)
	default:
		tasks = append(tasks, ctx.Say(loc01Actor, "046_ROD_07", "[046_ROD_07]"))
	}

	return engine.Sequence(tasks...)
}

func (c LOC01Controller) burnPlant(ctx *Context) engine.Task {
	if ctx.GetFlag(FlagPlantBurned) != 0 {
		return ctx.SayText(loc01Actor, "There's not much left of it to burn.", 1.2)
	}

	x, y, _ := ctx.AreaCenter("007b_Pflanze")

	return engine.Sequence(
		ctx.WalkTo(loc01Actor, x, y),
		ctx.SayText(loc01Actor, "Whoops. Probably shouldn't have done that.", 1.2),
		ctx.SetFlag(FlagPlantBurned, 1),
		ctx.DisableArea("007b_Pflanze"),
	)
}
