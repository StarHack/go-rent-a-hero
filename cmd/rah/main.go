// Command rah is the game executable: scene transitions and location
// scripting (Milestone 5) over click-to-walk movement, speech and
// inventory, growing into the full point-and-click engine.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jupiterrider/purego-sdl3/sdl"
	"github.com/wok/rent-a-hero/internal/assets"
	"github.com/wok/rent-a-hero/internal/audio"
	"github.com/wok/rent-a-hero/internal/config"
	"github.com/wok/rent-a-hero/internal/datapath"
	"github.com/wok/rent-a-hero/internal/engine"
	"github.com/wok/rent-a-hero/internal/game"
	"github.com/wok/rent-a-hero/internal/recorder"
	"github.com/wok/rent-a-hero/internal/render"
	"github.com/wok/rent-a-hero/internal/text"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "rah:", err)
		os.Exit(1)
	}
}

func run() error {
	args := os.Args[1:]
	if len(args) > 1 {
		return fmt.Errorf("usage: rah [savegame]")
	}

	saveGamePath := ""
	if len(args) == 1 {
		saveGamePath = args[0]
	}

	configFile, err := datapath.Resolve("config.ini")
	if err != nil {
		return err
	}

	paths, err := loadGamePaths("data/LOC01", "data/Common")
	if err != nil {
		return err
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		return err
	}

	scene := cfg.InitScene
	if scene == "" {
		scene = "114"
	}

	var audioEngine audio.Engine
	if !cfg.EnableSound {
		log.Printf("audio: disabled via config.ini's enable_sound = 0, running silent")
		audioEngine = audio.NewNullEngine()
	} else if otoEngine, err := audio.NewOtoEngine(44100); err != nil {
		log.Printf("audio: no device available (%v), running silent", err)
		audioEngine = audio.NewNullEngine()
	} else {
		audioEngine = otoEngine
	}
	defer audioEngine.Close()

	windowWidth := int32(render.LogicalWidth * 2)
	windowHeight := int32(render.LogicalHeight * 2)

	r, err := render.New("Rent-A-Hero", windowWidth, windowHeight, cfg.FullScreen)
	if err != nil {
		return err
	}
	defer r.Close()

	r.SetDebugAreas(cfg.DebugHotSpots)
	r.SetDebugArea(cfg.DebugArea)

	itemText, err := loadInventoryText(paths.Common, "eng")
	if err != nil {
		return err
	}

	idx, err := assets.NewIndex(paths.Location, paths.Common, paths.CommonCD)
	if err != nil {
		return fmt.Errorf("build asset index: %w", err)
	}

	initialLocation := 1
	var initialController game.Controller = game.LOC01Controller{}
	if strings.EqualFold(filepath.Base(filepath.Clean(paths.Location)), "LOC02") {
		initialLocation = 2
		initialController = game.LOC02Controller{}
	}
	session := game.NewSessionWithSaveGamePath(idx, audioEngine, initialController, initialLocation, saveGamePath)
	session.SetLocationResolver(paths.locationResolver())
	session.SetStretchVideo(cfg.StretchVideo)

	phase := game.PhaseGameplay
	if !session.HasLoadedSave() {
		phase = game.ResolveLaunchPhase(scene)
	}

	if phase <= game.PhaseIntro116 {
		quit, err := playLaunchCinematics(r, audioEngine, paths, itemText, cfg.StretchVideo, phase, scene)
		audioEngine.StopAll()
		if err != nil {
			return fmt.Errorf("play intro: %w", err)
		}
		if quit {
			return nil
		}
	}

	gameplayScene := scene
	if phase != game.PhaseGameplay {
		gameplayScene = postIntroLandingScene
		session.PrepareReturnHomeFirstEncounter()
	}
	if err := session.LoadInitialScene(gameplayScene); err != nil {
		return fmt.Errorf("load initial scene %q: %w", gameplayScene, err)
	}

	drainSession(session)

	ui := &uiState{session: session, idx: idx, itemText: itemText, hoverSlot: -1}
	_, err = ui.runInteractive(r)
	return err
}

func loadInventoryText(commonDir, lang string) (*text.InventoryText, error) {
	data, err := os.ReadFile(filepath.Join(commonDir, "DefInvent."+lang))
	if err != nil {
		return nil, fmt.Errorf("read DefInvent.%s: %w", lang, err)
	}

	itemText, err := text.ParseInventoryText(data)
	if err != nil {
		return nil, fmt.Errorf("parse DefInvent.%s: %w", lang, err)
	}

	return itemText, nil
}

const fixedTimestep = 1.0 / 60.0

// gamePaths holds resolved absolute asset directories (repo cwd or next to
// the .app bundle — see internal/datapath).
type gamePaths struct {
	Location string
	Common   string
	CommonCD string
	Intro    string
	Loc35    string
	Loc37    string
	Loc02    string
	Loc03    string
	Loc04    string
	Loc05    string
	Loc06    string
	Loc07    string
	Loc10    string
	Loc15    string
	Loc28    string
	Loc29    string
	Loc30    string
	Loc31    string
}

func loadGamePaths(locationFlag, commonFlag string) (gamePaths, error) {
	var p gamePaths
	var err error

	if p.Location, err = datapath.Resolve(locationFlag); err != nil {
		return p, fmt.Errorf("location: %w", err)
	}
	if p.Common, err = datapath.Resolve(commonFlag); err != nil {
		return p, fmt.Errorf("common: %w", err)
	}

	resolve := func(name, rel string) (string, error) {
		abs, err := datapath.Resolve(rel)
		if err != nil {
			return "", fmt.Errorf("%s: %w", name, err)
		}
		return abs, nil
	}

	if p.CommonCD, err = resolve("commonCD", commonCDDir); err != nil {
		return p, err
	}
	if p.Intro, err = resolve("intro", introLocationDir); err != nil {
		return p, err
	}
	if p.Loc35, err = resolve("loc35", loc35Dir); err != nil {
		return p, err
	}
	if p.Loc37, err = resolve("loc37", loc37Dir); err != nil {
		return p, err
	}
	if p.Loc02, err = resolve("loc02", loc02Dir); err != nil {
		return p, err
	}
	if p.Loc03, err = resolve("loc03", loc03Dir); err != nil {
		return p, err
	}
	if p.Loc04, err = resolve("loc04", loc04Dir); err != nil {
		return p, err
	}
	if p.Loc05, err = resolve("loc05", loc05Dir); err != nil {
		return p, err
	}
	if p.Loc06, err = resolve("loc06", loc06Dir); err != nil {
		return p, err
	}
	if p.Loc07, err = resolve("loc07", loc07Dir); err != nil {
		return p, err
	}
	if p.Loc10, err = resolve("loc10", loc10Dir); err != nil {
		return p, err
	}
	if p.Loc15, err = resolve("loc15", loc15Dir); err != nil {
		return p, err
	}
	if p.Loc28, err = resolve("loc28", loc28Dir); err != nil {
		return p, err
	}
	if p.Loc29, err = resolve("loc29", loc29Dir); err != nil {
		return p, err
	}
	if p.Loc30, err = resolve("loc30", loc30Dir); err != nil {
		return p, err
	}
	if p.Loc31, err = resolve("loc31", loc31Dir); err != nil {
		return p, err
	}

	return p, nil
}

func (p gamePaths) locationResolver() func(location int) (*assets.Index, game.Controller, error) {
	return func(location int) (*assets.Index, game.Controller, error) {
		var dir string
		var controller game.Controller

		switch location {
		case 1:
			dir, controller = p.Location, game.LOC01Controller{}
		case 2:
			dir, controller = p.Loc02, game.LOC02Controller{}
		case 4:
			dir, controller = p.Loc04, game.LOC04Controller{}
		case 5:
			dir, controller = p.Loc05, game.LOC05Controller{}
		case 7:
			dir, controller = p.Loc07, game.LOC07Controller{}
		case 10:
			dir, controller = p.Loc10, game.LOC10Controller{}
		case 15:
			dir, controller = p.Loc15, game.LOC15Controller{}
		case 28:
			dir, controller = p.Loc28, game.LOC28Controller{}
		case 29:
			dir, controller = p.Loc29, game.LOC29Controller{}
		case 30:
			dir, controller = p.Loc30, game.LOC30Controller{}
		case 3:
			dir, controller = p.Loc03, game.LOC03Controller{}
		case 6:
			dir, controller = p.Loc06, game.LOC06Controller{}
		case 31:
			dir, controller = p.Loc31, game.LOC31Controller{}
		default:
			return nil, nil, fmt.Errorf("no known asset directory for location %d", location)
		}

		idx, err := assets.NewIndex(dir, p.Common, p.CommonCD)
		if err != nil {
			return nil, nil, err
		}
		return idx, controller, nil
	}
}

// introLocationDir is Location 40's fixed asset directory, per
// agents/scenes/01_INTRO.MD's dispatch of the launch transition to
// "Location: 0x28 = 40". Unlike --location (the location under active
// development), this is the original game's own hardcoded launch target,
// so it is not exposed as a flag.
const introLocationDir = "data/LOC40"

// commonCDDir holds assets that ship on the game CD rather than under the
// installed --common directory: many of Scene 114's cue-sheet SFX (cannon
// fire, explosions, bombs, fire, sword, mobile-down, gong -- see
// 01_INTRO.MD "Complete Scene-114 SFX cue sheet"), and Location 35's
// Gen_Black.bmp backdrop.
const commonCDDir = "data/COMMONCD"

// loc35Dir is Location 35's fixed asset directory: the "Dragon Blaster
// Deluxe" glider interlude the real launch path inserts between Scene 114
// and Scene 115 (see agents/scenes/02_DRAGON_BLASTER_DELUXE.MD). Like
// introLocationDir, this is the original game's own hardcoded target, not
// something under active development, so it is not exposed as a flag.
const loc35Dir = "data/LOC35"

// loc37Dir is Location 37's fixed asset directory: the walkable Scene S3
// interlude between intro Scenes 115 and 116 (see agents/scenes/3.MD).
const loc37Dir = "data/LOC37"

// postIntroLandingScene is where the real launch path drops the player once
// its cinematic (Scenes 114/S1/115/116) finishes: Location 1 / Scene 7A
// with previousScene marker 5 (dec(2).c), triggering FUN_004072e0's
// first-return path — see agents/scenes/116_TO_7A_CORRECT_DEC2_RUN_PATH.MD.
const postIntroLandingScene = "7A"

// loc04Dir is Location 4's fixed asset directory: Rodrigo's office interior,
// reached through Scene 7A's 007a_Tuer door (see agents/scenes/7B.MD
// sections 5/9). Like loc35Dir, this is one of the game's fixed original
// directories, not something under active development, so it is not
// exposed as a flag.
const loc04Dir = "data/LOC04"

// loc05Dir is Location 5's fixed asset directory: Loui's office, reached
// through Scene 9's 009_Tuer door (see agents/scenes/9.MD section 9). Like
// loc35Dir, this is one of the game's fixed original directories, not
// something under active development, so it is not exposed as a flag.
const loc05Dir = "data/LOC05"

// loc07Dir is Location 7's fixed asset directory: the canal tunnel reached
// through Scene 46's 046_Kanal (see agents/scenes/46.MD section 4.2). Like
// loc05Dir, this is one of the game's fixed original directories, not
// something under active development, so it is not exposed as a flag.
const loc07Dir = "data/LOC07"

// loc03Dir/loc06Dir/loc31Dir are Scene 8's own three glider-departure
// destinations (agents/scenes/08.MD sections 6-8): the tavern (008_Kneipe),
// the monastery (008_Kloster), and "outside" (008_Raus), respectively. Like
// loc05Dir, these are fixed original directories, not exposed as flags.
const (
	loc02Dir = "data/LOC02"
	loc03Dir = "data/LOC03"
	loc06Dir = "data/LOC06"
	loc10Dir = "data/LOC10"
	loc15Dir = "data/LOC15"
	loc28Dir = "data/LOC28"
	loc29Dir = "data/LOC29"
	loc30Dir = "data/LOC30"
	loc31Dir = "data/LOC31"
)

// playLaunchCinematics runs whichever of the real launch sequence's
// cinematic/interactive phases -- Scene 114, Location 35, Scene 115,
// Location 37 / Scene S3, Scene 116 (see agents/scenes/01_INTRO.MD,
// agents/scenes/115_TRANSITION.MD, agents/scenes/3.MD) -- come at or after
// startPhase, in order, full-screen on r in real time.
// Every phase always plays in full and cannot be skipped outright, but
// pressing SPACE fast-forwards to the next part within a cinematic phase
// (see game.LoadIntroPart1Task's skip parameter). sceneID is the originally
// requested scene (config.ini's init_scene or --scene): if startPhase is
// PhaseIntroPart2, it also picks which of Scenes 115/116 to start at when
// jumping mid-chain (see game.LoadIntroPart2TaskFrom); for PhaseIntro116
// only Scene 116 runs. For any other startPhase it has no further effect
// returns quit=true if the window was closed during playback.
func playLaunchCinematics(r *render.Renderer, audioEngine audio.Engine, paths gamePaths, itemText *text.InventoryText, stretchVideo bool, startPhase game.LaunchPhase, sceneID string) (quit bool, err error) {
	introScene := game.NewIntroScene()

	idx40, err := assets.NewIndex(paths.Intro, paths.Common, paths.CommonCD)
	if err != nil {
		return false, fmt.Errorf("build intro asset index: %w", err)
	}

	if startPhase <= game.PhaseIntroPart1 {
		skip := new(bool)
		task := game.LoadIntroPart1Task(introScene, idx40, audioEngine, stretchVideo, skip)
		if quit, err := runIntroLoop(r, introScene, task, skip); quit || err != nil {
			return quit, err
		}
		// Location 35 starts its own music/ambient SFX (see
		// LOC35Controller.Enter); nothing must still be playing from Part
		// 1's own cue sheet underneath it.
		audioEngine.StopAll()
	}

	if startPhase <= game.PhaseLoc35 {
		quit, err := playLoc35(r, audioEngine, paths, itemText, stretchVideo)
		// playLoc35 runs its own Session (game.NewSession, not the one
		// TransitionToLocation guards), so its music/ambient SFX/voiceover
		// need the same explicit cleanup on the way out, regardless of how
		// it returned -- otherwise it keeps playing into Part 2.
		audioEngine.StopAll()
		if quit || err != nil {
			return quit, err
		}
	}

	if startPhase <= game.PhaseIntro115 {
		skip115 := new(bool)
		task := game.LoadIntroPart115Task(introScene, idx40, audioEngine, stretchVideo, skip115)
		if quit, err := runIntroLoop(r, introScene, task, skip115); quit || err != nil {
			return quit, err
		}
		audioEngine.StopAll()
	}

	if startPhase <= game.PhaseLoc37 {
		quit, err := playLoc37(r, audioEngine, paths, itemText, stretchVideo)
		audioEngine.StopAll()
		if quit || err != nil {
			return quit, err
		}
	}

	if startPhase <= game.PhaseIntro116 {
		skip := new(bool)
		var task engine.Task
		if startPhase <= game.PhaseLoc37 {
			task = game.LoadIntroPart116Task(introScene, idx40, audioEngine, stretchVideo, skip)
		} else {
			task = game.LoadIntroPart2TaskFrom(introScene, idx40, audioEngine, stretchVideo, skip, sceneID)
		}
		return runIntroLoop(r, introScene, task, skip)
	}

	return false, nil
}

// playLoc37 runs Location 37 / Scene S3 to completion (see game.LOC37Controller):
// the walkable interlude between intro Scenes 115 and 116. It returns
// quit=true if the window was closed during play.
func playLoc37(r *render.Renderer, audioEngine audio.Engine, paths gamePaths, itemText *text.InventoryText, stretchVideo bool) (quit bool, err error) {
	idx, err := assets.NewIndex(paths.Loc37, paths.Common, paths.CommonCD)
	if err != nil {
		return false, fmt.Errorf("build Location 37 asset index: %w", err)
	}

	session := game.NewSessionWithoutSaveGame(idx, audioEngine, game.LOC37Controller{}, 37)
	session.SetStretchVideo(stretchVideo)
	if err := session.LoadInitialScene(game.Loc37InitialSceneID()); err != nil {
		return false, fmt.Errorf("load Location 37 scene: %w", err)
	}

	ui := &uiState{session: session, idx: idx, itemText: itemText, hoverSlot: -1}

	return ui.runInteractive(r)
}

// playLoc35 runs the interactive Location 35 glider sequence to completion
// (see game.LOC35Controller): the player selects the Dragon Blaster Deluxe
// (granted automatically on entry) to proceed. Per agents/scenes/S1.MD
// section 9 ("The toolbar is visible during Location 35"), the inventory
// overlay starts open rather than requiring the "I" key -- it can still be
// toggled normally from there. It returns quit=true if the window was
// closed during play.
func playLoc35(r *render.Renderer, audioEngine audio.Engine, paths gamePaths, itemText *text.InventoryText, stretchVideo bool) (quit bool, err error) {
	idx, err := assets.NewIndex(paths.Loc35, paths.Common, paths.CommonCD)
	if err != nil {
		return false, fmt.Errorf("build Location 35 asset index: %w", err)
	}

	session := game.NewSessionWithoutSaveGame(idx, audioEngine, game.LOC35Controller{}, 35)
	session.SetStretchVideo(stretchVideo)
	if err := session.LoadInitialScene("S1"); err != nil {
		return false, fmt.Errorf("load Location 35 scene: %w", err)
	}

	ui := &uiState{session: session, idx: idx, itemText: itemText, hoverSlot: -1, showInventory: true}

	return ui.runInteractive(r)
}

// runIntroLoop drives task (a *videoTask-driving Sequence built by
// game.LoadIntroPart1Task/LoadIntroPart2Task et al.) against scene in real
// time until it completes, rendering each frame. Pressing SPACE
// (non-repeat) sets *skip, which task interprets as "fast-forward to the
// next part" (see those functions' skip parameter); skip must be the exact
// pointer given to whichever Load*Task built task. It returns quit=true if
// the window was closed or Escape was pressed, matching uiState.
// runInteractive's own Escape-quits-the-application behavior -- video
// playback (cinematics, Location 35's own runIntroLoop-driven parts) must
// be quittable the same way as ordinary gameplay.
func runIntroLoop(r *render.Renderer, scene *engine.Scene, task engine.Task, skip *bool) (quit bool, err error) {
	var accumulator float64
	lastTicks := sdl.GetTicksNS()

	for {
		var event sdl.Event
		for sdl.PollEvent(&event) {
			switch event.Type() {
			case sdl.EventQuit:
				return true, nil
			case sdl.EventKeyDown:
				switch event.Key().Scancode {
				case sdl.ScancodeEscape:
					return true, nil
				case sdl.ScancodeSpace:
					if !event.Key().Repeat {
						*skip = true
					}
				}
			}
		}

		now := sdl.GetTicksNS()
		frameTime := float64(now-lastTicks) / 1e9
		lastTicks = now
		accumulator += min(frameTime, maxFrameTime)

		done := false
		for accumulator >= fixedTimestep {
			if task.Update(fixedTimestep) {
				done = true
			}
			accumulator -= fixedTimestep
		}

		if err := r.DrawScene(scene); err != nil {
			return false, fmt.Errorf("draw intro scene: %w", err)
		}
		r.DrawDebugArea(scene.ID, 40)
		r.Present()

		if done {
			return false, nil
		}
	}
}

// drainSession flushes Immediate Enter setup (FreezeLayer, PlaceActor,
// HideActor, starting a video) so a --click/--speak/--screenshot is not
// swallowed by a one-tick lock. It must not play a blocking cinematic,
// walk, or speech through to completion — doing that silently consumed
// Scene 116's ReturnHomeCam before any frame was presented (see
// agents/scenes/116_TO_7A_CORRECT_DEC2_RUN_PATH.MD section 3). After an
// Update that leaves the session still locked, a real timed action owns the
// lock and belongs to the render loop.
func drainSession(session *game.Session) {
	const maxSteps = 8
	for i := 0; i < maxSteps && session.Locked(); i++ {
		session.Update(fixedTimestep)
		if session.Locked() {
			return
		}
	}
}

// uiState holds presentation-layer state that sits outside the gameplay
// session: inventory icon slots/animation and hover tracking. All
// gameplay orchestration (scenes, tasks, flags) lives in *game.Session.
type uiState struct {
	session  *game.Session
	idx      *assets.Index
	itemText *text.InventoryText

	showInventory  bool
	inventorySlots []render.InventorySlot
	hoverSlot      int

	// recorder is non-nil while the "8" key toggle (see runInteractive) is
	// actively capturing the window's canvas to recording.mp4 in the
	// project root. recordAccum throttles capture to recordingFPS
	// independently of the render/simulation rates.
	recorder    *recorder.Recorder
	recordAccum float64

	// screenshotPending is set by the "0" key (see runInteractive) and
	// consumed on the next captureScreenshotIfRequested call, which runs
	// right after the frame is fully drawn but before anything else (in
	// particular the recording indicator) is drawn over it, so
	// recording.png is a clean capture. screenshotFlash then counts down
	// from screenshotFlashSeconds so DrawScreenshotIndicator shows on
	// screen for exactly that long afterward.
	screenshotPending bool
	screenshotFlash   float64
}

// recordingFPS is the frame rate recording.mp4 is captured and muxed at.
const recordingFPS = 30

// recordingPath is where the "8" key toggle always saves to, per the
// project-root requirement.
const recordingPath = "recording.mp4"

// toggleRecording starts or stops the "8" key screen recording. Stopping
// finalizes recording.mp4; starting always begins a fresh recording,
// overwriting any previous one at the same path.
func (u *uiState) toggleRecording() {
	if u.recorder != nil {
		if err := u.recorder.Close(); err != nil {
			log.Printf("recording: %v", err)
		} else {
			log.Printf("recording: saved %s", recordingPath)
		}
		u.recorder = nil
		return
	}

	rec, err := recorder.New(recordingPath, recordingFPS)
	if err != nil {
		log.Printf("recording: %v", err)
		return
	}

	u.recorder = rec
	u.recordAccum = 0
	log.Printf("recording: started, saving to %s", recordingPath)
}

// captureRecordingFrame draws the recording indicator (so the saved video
// itself shows it was on) and, once enough real time has passed for
// recordingFPS, appends the current canvas as the next frame. Called after
// the rest of the scene/UI has been drawn but before Present, per
// Renderer.DrawRecordingIndicator's contract.
func (u *uiState) captureRecordingFrame(r *render.Renderer, frameTime float64) {
	if u.recorder == nil {
		return
	}

	r.DrawRecordingIndicator()

	const recordingStep = 1.0 / recordingFPS
	u.recordAccum += min(frameTime, maxFrameTime)
	if u.recordAccum < recordingStep {
		return
	}
	u.recordAccum -= recordingStep

	img, err := r.CaptureRGBA()
	if err != nil {
		log.Printf("recording: capture frame: %v", err)
		return
	}
	if err := u.recorder.AddFrame(img); err != nil {
		log.Printf("recording: %v", err)
	}
}

// screenshotPath is where the "0" key screenshot always saves to: the
// project root, alongside recording.mp4, per the same project-root
// requirement -- overwriting any previous screenshot at the same path.
const screenshotPath = "recording.png"

// screenshotFlashSeconds is how long DrawScreenshotIndicator stays on
// screen after each "0" key capture before disappearing automatically.
const screenshotFlashSeconds = 1.0

// captureScreenshotIfRequested saves the current canvas to screenshotPath
// if the "0" key was pressed this frame (see runInteractive), then starts
// the on-screen flash timer. Called right after the rest of the scene/UI
// has been drawn but before anything else (recording indicator, screenshot
// indicator) is drawn over it, so the saved PNG is a clean capture with
// neither indicator baked in.
func (u *uiState) captureScreenshotIfRequested(r *render.Renderer) {
	if !u.screenshotPending {
		return
	}
	u.screenshotPending = false

	if err := r.SavePNG(screenshotPath); err != nil {
		log.Printf("screenshot: %v", err)
		return
	}
	log.Printf("screenshot: saved %s", screenshotPath)
	u.screenshotFlash = screenshotFlashSeconds
}

func (u *uiState) giveItem(id int) error {
	item, err := engine.LoadInventoryItem(u.idx, u.itemText, id)
	if err != nil {
		return err
	}

	u.session.State().AddItem(id)
	u.inventorySlots = append(u.inventorySlots, render.NewInventorySlot(item))

	return nil
}

func (u *uiState) update(dt float64) {
	u.session.Update(dt)
	u.syncInventorySlots()

	for _, slot := range u.inventorySlots {
		slot.Anim.Advance(dt)
	}
}

// syncInventorySlots adds a render.InventorySlot for every item in the
// session's persistent inventory that doesn't have one yet -- not just
// items given via --give, but any a Controller granted directly (e.g.
// LOC35Controller.Enter's ctx.AddItem(ItemDragonBlasterDeluxe)), so the
// inventory overlay actually reflects Session.State().Inventory rather than
// only ever showing what --give listed. A missing icon/text is logged and
// skipped rather than aborting the session over one bad asset.
func (u *uiState) syncInventorySlots() {
	have := make(map[int]bool, len(u.inventorySlots))
	for _, slot := range u.inventorySlots {
		have[slot.Item.ID] = true
	}

	for _, id := range u.session.State().Inventory {
		if have[id] {
			continue
		}
		if err := u.giveItem(id); err != nil {
			log.Printf("game: sync inventory item %d: %v", id, err)
		}
	}
}

func (u *uiState) draw(r *render.Renderer) error {
	if err := r.DrawScene(u.session.Scene()); err != nil {
		return err
	}

	if subtitle := u.session.CurrentSubtitle(); subtitle != "" {
		r.DrawSubtitle(subtitle)
	}

	if u.showInventory {
		if err := r.DrawInventory(u.inventorySlots, u.hoverSlot); err != nil {
			return err
		}
	}

	state := u.session.State()
	r.DrawDebugArea(state.Scene, state.Location)

	return nil
}

// handleClick routes a logical-space click to either the inventory overlay
// (when open, per agents/GAMEPLAY.md "Inventory UI" pausing world
// interaction) or the gameplay session.
func (u *uiState) handleClick(x, y float32) {
	if u.showInventory {
		if i := render.HitTestInventory(u.inventorySlots, x, y); i >= 0 {
			u.session.SelectItem(u.inventorySlots[i].Item.ID)
			log.Printf("selected inventory item %q", u.inventorySlots[i].Item.Name)
			u.showInventory = false
		}
		return
	}

	if y >= render.SceneHeight {
		return
	}

	u.session.HandleClick(x, y)
}

func (u *uiState) handleMouseMove(x, y float32) {
	if u.showInventory {
		u.hoverSlot = render.HitTestInventory(u.inventorySlots, x, y)
	}
}

// maxFrameTime caps how much real elapsed time a single rendered frame can
// feed into the accumulator, so a debugger pause, window drag, or other
// stall doesn't cause a burst of catch-up simulation steps on the next
// frame (the classic "spiral of death").
const maxFrameTime = 0.25

// runInteractive drives the session (mouse/keyboard input, fixed-timestep
// update, draw) until the window is closed or Escape is pressed (quit=true
// either way) or -- see Session.Done -- the session's own Controller
// signals its location is finished (quit=false; e.g. LOC35Controller
// handing off to Scene 115, see cmd/rah's playLoc35). The top-level game
// session set up by run() never sets Done, so for it this always ends via
// quit.
func (u *uiState) runInteractive(r *render.Renderer) (quit bool, err error) {
	running := true

	// The render loop is not rate-limited by SDL (no vsync is configured,
	// and even if it were, refresh rates vary across displays), so it can
	// run at any speed. Simulation must therefore be driven by an
	// accumulator over real elapsed time and stepped at a fixed 60Hz, per
	// agents/IMPLEMENTATION.md "Main loop": rendering as fast as possible
	// while gameplay/animation always advances at the same rate regardless
	// of how many (or few) frames get drawn per second.
	var accumulator float64
	lastTicks := sdl.GetTicksNS()

	for running {
		var event sdl.Event
		for sdl.PollEvent(&event) {
			switch event.Type() {
			case sdl.EventQuit:
				running = false
				quit = true

			case sdl.EventKeyDown:
				switch event.Key().Scancode {
				case sdl.ScancodeEscape:
					running = false
					quit = true
				case sdl.ScancodeTab, sdl.ScancodeF1:
					if !event.Key().Repeat {
						r.SetDebugAreas(!r.DebugAreas())
					}
				case sdl.ScancodeI:
					if !event.Key().Repeat {
						u.showInventory = !u.showInventory
						u.hoverSlot = -1
					}
				case sdl.Scancode0:
					if !event.Key().Repeat {
						u.screenshotPending = true
					}
				case sdl.Scancode1:
					if !event.Key().Repeat {
						if err := u.session.SaveGame(); err != nil {
							log.Printf("savegame: %v", err)
						}
					}
				}

			case sdl.EventKeyUp:
				switch event.Key().Scancode {
				case sdl.Scancode8:
					if !event.Key().Repeat {
						u.toggleRecording()
					}
				}

			case sdl.EventMouseButtonDown:
				btn := event.Button()
				if btn.Down && btn.Button == uint8(sdl.ButtonLeft) {
					lx, ly := r.WindowToLogical(btn.X, btn.Y)
					u.handleClick(lx, ly)
				}

			case sdl.EventMouseMotion:
				m := event.Motion()
				lx, ly := r.WindowToLogical(m.X, m.Y)
				u.handleMouseMove(lx, ly)
			}
		}

		now := sdl.GetTicksNS()
		frameTime := float64(now-lastTicks) / 1e9
		lastTicks = now
		accumulator += min(frameTime, maxFrameTime)

		for accumulator >= fixedTimestep {
			u.update(fixedTimestep)
			accumulator -= fixedTimestep
			if u.session.Done() {
				running = false
			}
		}

		if err := u.draw(r); err != nil {
			return quit, fmt.Errorf("draw scene: %w", err)
		}

		u.captureScreenshotIfRequested(r)
		u.captureRecordingFrame(r, frameTime)

		if u.screenshotFlash > 0 {
			r.DrawScreenshotIndicator()
			u.screenshotFlash -= frameTime
		}

		if u.session.SavedVisible() {
			r.DrawSavedIndicator()
		}

		r.Present()
	}

	if u.recorder != nil {
		u.toggleRecording()
	}

	return quit, nil
}

// runHeadless runs the simulation in real time (audio playback and thus
// speech-driven tasks cannot be fast-forwarded) until the session is idle,
// or -- if captureAfterSeconds > 0 -- for exactly that long regardless of
// lock state, then saves one screenshot and exits. The fixed-duration mode
// is for catching a mid-action frame (e.g. a playing cutscene) that would
// otherwise be gone by the time the action completes. Used for automated
// verification without a human clicking.
func (u *uiState) runHeadless(r *render.Renderer, screenshotPath string, captureAfterSeconds float64) error {
	const maxSteps = 60 * 60 // 60 seconds of real time, safety cap

	step := fixedTimestep // copy into a variable to force a runtime (not constant) duration conversion below
	sleepDuration := time.Duration(step * float64(time.Second))

	steps := maxSteps
	if captureAfterSeconds > 0 {
		steps = int(captureAfterSeconds / fixedTimestep)
	}

	for i := 0; i < steps; i++ {
		if captureAfterSeconds <= 0 && !u.session.Locked() {
			break
		}
		u.update(fixedTimestep)
		time.Sleep(sleepDuration)
	}

	if err := u.draw(r); err != nil {
		return fmt.Errorf("draw scene: %w", err)
	}

	if err := r.SavePNG(screenshotPath); err != nil {
		return fmt.Errorf("save screenshot: %w", err)
	}

	return nil
}
