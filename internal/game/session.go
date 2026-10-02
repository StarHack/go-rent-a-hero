package game

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/wok/rent-a-hero/internal/assets"
	"github.com/wok/rent-a-hero/internal/audio"
	"github.com/wok/rent-a-hero/internal/config"
	"github.com/wok/rent-a-hero/internal/datapath"
	"github.com/wok/rent-a-hero/internal/engine"
	"github.com/wok/rent-a-hero/internal/formats/szn"
	gametext "github.com/wok/rent-a-hero/internal/text"
)

// Session owns one play session's mutable state: the current scene, the
// player's persistent GameState, and whichever task is currently animating
// the world (a plain click-to-walk, or a locked scripted controller
// action).
type Session struct {
	idx         *assets.Index
	audioEngine audio.Engine
	controller  Controller
	state       *engine.GameState
	scene       *engine.Scene

	activeTask engine.Task
	locked     bool

	musicHandle   audio.Handle
	musicName     string
	musicLocation int
	walkEpoch     map[string]uint64

	selectedItem    *int
	currentSubtitle string

	stretchVideo         bool
	inventoryText        *gametext.InventoryText
	localizationExt      string
	localizationDetected bool
	areaTextDB           map[int]gametext.DB

	done bool

	initialLocation      int
	saveGamePath         string
	loadedSave           bool
	savedPlayerX         int
	savedPlayerY         int
	savedPlayerDirection int
	haveSavedPlayer      bool
	savedFlash           float64
	loc04Runtime         *loc04Runtime

	resolveLocation func(location int) (*assets.Index, Controller, error)

	// ambientTasks run every Update alongside activeTask, regardless of
	// Locked() -- unlike activeTask, they are never awaited for completion
	// and never gate the interaction lock. Used for a scene behavior that
	// must run continuously and indefinitely once started, independent of
	// whatever scripted action happens to be locking normal input at any
	// given moment -- e.g. Scene 7A's GliderLayer bob loop (see
	// Context.RunAmbient and agents/techniques/
	// SMOOTH-TRANSITIONS-ZBUFFER-GLIDER.MD sections 15-21). Cleared on every
	// scene transition; a Controller's Enter hook re-registers whatever it
	// needs for the new scene.
	ambientTasks []engine.Task
}

// SetStretchVideo configures whether Context.PlayVideo fills the whole
// logical window (config.ini's stretch_video) instead of drawing cutscenes
// at their native size within the scene viewport (the default, matching
// how in-game cutscenes like a lift ride have always looked).
func (s *Session) SetStretchVideo(stretch bool) {
	s.stretchVideo = stretch
}

// SetLocationResolver configures how Context.ChangeLocation looks up a
// different location's asset index and Controller by location number, for
// Controllers that lead somewhere outside the location this session was
// built for -- e.g. Scene 9's 009_Tuer door into Location 5 (see
// agents/scenes/9.MD section 9). A session whose Controllers never call
// ChangeLocation (the launch intro, Location 35) need not set this.
func (s *Session) SetLocationResolver(resolve func(location int) (*assets.Index, Controller, error)) {
	s.resolveLocation = resolve
}

// NewSession creates a session scoped to idx (a location's own directory
// plus shared asset roots) using controller for all scenes loaded from it.
func NewSession(idx *assets.Index, audioEngine audio.Engine, controller Controller, location int) *Session {
	return newSession(idx, audioEngine, controller, location, "", true)
}

func NewSessionWithSaveGamePath(idx *assets.Index, audioEngine audio.Engine, controller Controller, location int, saveGamePath string) *Session {
	return newSession(idx, audioEngine, controller, location, saveGamePath, true)
}

func NewSessionWithoutSaveGame(idx *assets.Index, audioEngine audio.Engine, controller Controller, location int) *Session {
	return newSession(idx, audioEngine, controller, location, "", false)
}

func newSession(idx *assets.Index, audioEngine audio.Engine, controller Controller, location int, saveGamePath string, loadSave bool) *Session {
	s := &Session{
		idx:         idx,
		audioEngine: audioEngine,
		controller:  controller,
		state: &engine.GameState{Location: location, Flags: map[string]int{
			FlagCanalStoryFlag: 1,
		}, OriginalState: newOriginalStateBlock()},
		initialLocation: location,
		saveGamePath:    strings.TrimSpace(saveGamePath),
	}
	if loadSave {
		s.autoLoadGame()
	}
	s.applyInventoryOverride()
	return s
}

func (s *Session) applyInventoryOverride() {
	path, err := datapath.Resolve("config.ini")
	if err != nil {
		return
	}
	cfg, err := config.Load(path)
	if err != nil || cfg.OverrideInventory == nil {
		return
	}
	ids, err := engine.ResolveInventoryNames(s.idx, cfg.OverrideInventory)
	if err != nil {
		log.Printf("game: %v", err)
		return
	}
	s.state.Inventory = ids
}

// Scene returns the currently instantiated scene.

func (s *Session) localizationExtension() string {
	if s.localizationDetected {
		return s.localizationExt
	}
	s.localizationDetected = true
	if s.idx == nil {
		return ""
	}

	defInvent := map[string]bool{}
	locCounts := map[string]int{}
	for _, key := range s.idx.Keys() {
		base := strings.ToLower(filepath.Base(key))
		if strings.HasPrefix(base, "definvent.") {
			ext := strings.TrimPrefix(base, "definvent.")
			if ext != "" {
				defInvent[ext] = true
			}
			continue
		}
		if len(base) < len("loc00.x") || !strings.HasPrefix(base, "loc") || base[5] != '.' {
			continue
		}
		if _, err := strconv.Atoi(base[3:5]); err != nil {
			continue
		}
		ext := base[6:]
		if ext != "" {
			locCounts[ext]++
		}
	}

	candidates := make([]string, 0, len(defInvent))
	for ext := range defInvent {
		candidates = append(candidates, ext)
	}
	if len(candidates) == 0 {
		for ext := range locCounts {
			candidates = append(candidates, ext)
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	sort.Slice(candidates, func(i, j int) bool {
		ci, cj := locCounts[candidates[i]], locCounts[candidates[j]]
		if ci != cj {
			return ci > cj
		}
		return candidates[i] < candidates[j]
	})
	s.localizationExt = candidates[0]
	return s.localizationExt
}

func (s *Session) resolveLocalizedFile(base string) string {
	if s.idx == nil {
		return ""
	}
	if ext := s.localizationExtension(); ext != "" {
		if p, err := s.idx.ResolveBaseName(base + "." + ext); err == nil {
			return p
		}
	}

	want := strings.ToLower(base + ".")
	for _, key := range s.idx.Keys() {
		if strings.HasPrefix(strings.ToLower(filepath.Base(key)), want) {
			if p, err := s.idx.Resolve(key); err == nil {
				return p
			}
		}
	}
	return ""
}

func (s *Session) loadInventoryText() *gametext.InventoryText {
	if s.inventoryText != nil {
		return s.inventoryText
	}
	path := s.resolveLocalizedFile("DefInvent")
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	parsed, err := gametext.ParseInventoryText(data)
	if err != nil {
		return nil
	}
	s.inventoryText = parsed
	return parsed
}

func (s *Session) areaText(location int, id string) (string, bool) {
	if s.idx == nil {
		return "", false
	}
	if s.areaTextDB == nil {
		s.areaTextDB = map[int]gametext.DB{}
	}
	db, loaded := s.areaTextDB[location]
	if !loaded {
		path := s.resolveLocalizedFile(fmt.Sprintf("Loc%02d", location))
		if path == "" {
			s.areaTextDB[location] = nil
			return "", false
		}
		data, err := os.ReadFile(path)
		if err != nil {
			s.areaTextDB[location] = nil
			return "", false
		}
		db = &gametext.IniDB{File: gametext.LoadIni(data)}
		s.areaTextDB[location] = db
	}
	if db == nil {
		return "", false
	}
	return db.Get("AREA_TEXT", id)
}

func (s *Session) localizeArea(area *engine.Area, location int) {
	if area == nil {
		return
	}
	if value, ok := s.areaText(location, area.ID); ok {
		area.Text = value
	}
}

func (s *Session) localizeSceneAreas(scene *engine.Scene, location int) {
	if scene == nil {
		return
	}
	for _, id := range scene.AreaOrder {
		s.localizeArea(scene.Areas[id], location)
	}
}

func (s *Session) syncSceneInventory() {
	if s.scene == nil {
		return
	}
	if len(s.scene.InventoryItems) == len(s.state.Inventory) {
		same := true
		for i, id := range s.state.Inventory {
			if s.scene.InventoryItems[i] == nil || s.scene.InventoryItems[i].ID != id {
				same = false
				break
			}
		}
		if same {
			return
		}
	}
	itemText := s.loadInventoryText()
	if itemText == nil {
		s.scene.InventoryItems = nil
		s.scene.InventoryAnims = nil
		return
	}
	items := make([]*engine.InventoryItem, 0, len(s.state.Inventory))
	anims := make([]*engine.Layer, 0, len(s.state.Inventory))
	for _, id := range s.state.Inventory {
		item, err := engine.LoadInventoryItem(s.idx, itemText, id)
		if err != nil {
			log.Printf("game: inventory item %d: %v", id, err)
			continue
		}
		items = append(items, item)
		anims = append(anims, &engine.Layer{Source: item.Sprite, FPS: 10, Mode: engine.AnimLoop, Playing: true})
	}
	s.scene.InventoryItems = items
	s.scene.InventoryAnims = anims
	s.scene.InventoryHoverIndex = -1
	s.scene.InventoryClickIndex = -1
}

func (s *Session) Scene() *engine.Scene { return s.scene }

// State returns the persistent game state (flags, inventory, current/previous scene).
func (s *Session) State() *engine.GameState { return s.state }

// CurrentSubtitle returns the subtitle text to display this frame, or "" if
// none.
func (s *Session) CurrentSubtitle() string { return s.currentSubtitle }

// Locked reports whether a scripted controller action currently owns
// input (see agents/GAMEPLAY.md "Interaction lock").
func (s *Session) Locked() bool { return s.locked }

// Done reports whether this session's Controller has signaled that its
// location is finished (see Context.CompleteLocation), e.g. Location 35's
// Dragon Blaster Deluxe sequence handing off to Location 40 / Scene 115
// per agents/scenes/02_DRAGON_BLASTER_DELUXE.MD. Ordinary locations never
// set this; the caller driving the session (cmd/rah) is responsible for
// checking it and moving on to whatever comes next, since that next step
// may be an entirely different location/asset index that this package has
// no general mechanism for transitioning to on its own.
func (s *Session) Done() bool { return s.done }

func (s *Session) actorByNameOrAsset(name string) (*engine.Actor, string, bool) {
	if s.scene == nil {
		return nil, "", false
	}
	if a, ok := s.scene.Characters[name]; ok {
		return a, name, true
	}
	target := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	for _, id := range s.scene.CharacterOrder {
		a, ok := s.scene.Characters[id]
		if !ok || a == nil || a.AssetName == "" {
			continue
		}
		assetStem := strings.TrimSuffix(filepath.Base(a.AssetName), filepath.Ext(a.AssetName))
		if strings.EqualFold(assetStem, target) {
			return a, id, true
		}
	}
	return nil, "", false
}

func (s *Session) PlayerActor() *engine.Actor {
	if s.scene == nil {
		return nil
	}
	for _, id := range s.scene.CharacterOrder {
		if a, ok := s.scene.Characters[id]; ok {
			return a
		}
	}
	return nil
}

func (s *Session) ctx() *Context { return &Context{session: s} }

// loc7AFirstEncounterFrom is Scene 116's outgoing scene marker (not
// previousScene == "116"): dec(2).c sets marker 5 before transitioning to
// Location 1 / Scene 7, and Scene 7A's return-home helper (FUN_004072e0)
// keys the first-encounter branch on previousScene == 5 — see
// agents/scenes/116_TO_7A_CORRECT_DEC2_RUN_PATH.MD sections 1/2.
const loc7AFirstEncounterFrom = "5"

// PrepareReturnHomeFirstEncounter sets the outgoing marker Scene 116 writes
// so LoadInitialScene("7A") receives previousScene == 5 and runs
// FUN_004072e0's marker-5 branch (see
// agents/scenes/116_TO_7A_CORRECT_DEC2_RUN_PATH.MD).
func (s *Session) PrepareReturnHomeFirstEncounter() {
	s.state.Scene = loc7AFirstEncounterFrom
}

// LoadInitialScene loads sceneID as the session's first scene (no Exit hook
// runs, since there is no prior scene).

func normalizeLocationSceneID(location int, sceneID string) string {
	switch location {
	case 1:
		switch strings.ToUpper(strings.TrimSpace(sceneID)) {
		case "7", "7A":
			return "7A"
		}
	case 2:
		switch strings.ToUpper(strings.TrimSpace(sceneID)) {
		case "13", "0013", "S0013":
			return "S0013"
		case "14", "0014", "S0014":
			return "S0014"
		case "15", "0015", "S0015":
			return "S0015"
		case "1013", "S1013":
			return "S1013"
		case "2013", "S2013":
			return "S2013"
		}
	case 12:
		if loc12SceneID(sceneID) == "S91" {
			return "091"
		}
	case 14:
		switch strings.ToUpper(strings.TrimSpace(sceneID)) {
		case "90", "090", "S90", "S090":
			return "S90"
		}
	case 15:
		switch strings.ToUpper(strings.TrimSpace(sceneID)) {
		case "57", "S57", "570", "571", "572":
			return "S57"
		}
	case 16:
		switch strings.ToUpper(strings.TrimSpace(sceneID)) {
		case "58", "S58":
			return "S58"
		}
	case 17:
		switch strings.ToUpper(strings.TrimSpace(sceneID)) {
		case "59", "S59":
			return "S59"
		case "60", "S60":
			return "S60"
		}
	case 18:
		switch strings.ToUpper(strings.TrimSpace(sceneID)) {
		case "61", "S61":
			return "S61"
		}
	case 19:
		switch strings.ToUpper(strings.TrimSpace(sceneID)) {
		case "68", "S68":
			return "S68"
		case "69", "S69":
			return "S69"
		case "101", "S101":
			return "S101"
		}
	case 20:
		switch strings.ToUpper(strings.TrimSpace(sceneID)) {
		case "70", "S70":
			return "S70"
		}
	case 21:
		value := strings.ToUpper(strings.TrimSpace(sceneID))
		value = strings.TrimPrefix(value, "S")
		if n, err := strconv.Atoi(value); err == nil && n >= 71 && n <= 80 {
			return fmt.Sprintf("S%d", n)
		}
	case 22:
		switch strings.ToUpper(strings.TrimSpace(sceneID)) {
		case "81", "S81":
			return "S81"
		case "1081", "S1081":
			return "S1081"
		}
	case 28:
		switch strings.ToUpper(strings.TrimSpace(sceneID)) {
		case "30", "31", "36", "S30", "S31", "S36":
			return "S31"
		case "32", "S32":
			return "S32"
		case "35", "S35":
			return "S35"
		}
	case 29:
		switch strings.ToUpper(strings.TrimSpace(sceneID)) {
		case "33", "S33":
			return "S33"
		}
	case 30:
		switch strings.ToUpper(strings.TrimSpace(sceneID)) {
		case "34", "S34":
			return "S34"
		}
	case 31:
		switch strings.ToUpper(strings.TrimSpace(sceneID)) {
		case "113", "S113":
			return "S113"
		}
	case 6:
		switch strings.ToUpper(strings.TrimSpace(sceneID)) {
		case "26", "026", "S26", "S026":
			return "026"
		case "27", "027", "S27", "S027":
			return "027"
		}
	case 9:
		switch strings.ToUpper(strings.TrimSpace(sceneID)) {
		case "89", "089", "S89", "S089":
			return "S89"
		}
	case 10:
		switch strings.ToUpper(strings.TrimSpace(sceneID)) {
		case "37", "S37":
			return "S37"
		case "38", "S38":
			return "S38"
		case "39", "S39":
			return "S39"
		case "40", "S40":
			return "S40"
		case "41", "S41":
			return "S41"
		case "42", "S42":
			return "S42"
		case "43", "S43":
			return "S43"
		case "44", "S44":
			return "S44"
		case "66", "S66":
			return "S66"
		}
	}
	return sceneID
}

func (s *Session) applyLocationEntryState(location int, sceneID string) {
	if location != 15 {
		return
	}
	switch strings.ToUpper(strings.TrimSpace(sceneID)) {
	case "570":
		putOriginalFlag(s.state.OriginalState, loc15StateNight, 0)
	case "571":
		putOriginalFlag(s.state.OriginalState, loc15StateNight, 1)
	case "572":
		putOriginalFlag(s.state.OriginalState, loc15StatePiratePeek, 1)
	}
}

func (s *Session) HasLoadedSave() bool {
	return s.loadedSave
}

func (s *Session) LoadInitialScene(sceneID string) error {
	if s.loadedSave {
		sceneID = normalizeLocationSceneID(s.state.Location, s.state.Scene)
		s.state.Scene = s.state.PreviousScene
		if s.state.Location != s.initialLocation {
			var idx *assets.Index
			var controller Controller
			var err error
			if s.resolveLocation != nil {
				idx, controller, err = s.resolveLocation(s.state.Location)
			}
			if s.resolveLocation == nil || err != nil {
				fallbackIdx, fallbackController, supported, fallbackErr := resolveBuiltinLocation(s.state.Location)
				if supported {
					if fallbackErr != nil {
						return fallbackErr
					}
					idx, controller, err = fallbackIdx, fallbackController, nil
				} else if s.resolveLocation == nil {
					return fmt.Errorf("game: save location %d requires a location resolver", s.state.Location)
				}
			}
			if err != nil {
				return err
			}
			s.idx = idx
			s.controller = controller
		}
	} else {
		s.applyLocationEntryState(s.state.Location, sceneID)
		sceneID = normalizeLocationSceneID(s.state.Location, sceneID)
	}
	task, err := s.transitionTo(sceneID)
	if err != nil {
		return err
	}

	s.activeTask = task
	s.locked = task != nil
	s.loadedSave = false

	return nil
}

// transitionTo runs the current scene's Exit hook (if any), loads and
// instantiates sceneID, updates State.Scene/PreviousScene, and returns the
// destination controller's Enter task (which may be nil).
func (s *Session) transitionTo(sceneID string) (engine.Task, error) {
	sceneID = normalizeLocationSceneID(s.state.Location, sceneID)
	if s.controller != nil && s.scene != nil {
		exitTask := s.controller.Exit(s.ctx(), s.state.Scene, sceneID)
		if exitTask != nil && !exitTask.Update(0) {
			log.Printf("game: Exit(%s -> %s) did not complete synchronously; Exit hooks must not block", s.state.Scene, sceneID)
		}
	}

	def, err := s.loadSceneDef(s.state.Location, sceneID)
	if err != nil {
		return nil, err
	}

	newScene, err := engine.InstantiateScene(def, s.idx, nil, s.loadConditionMask(sceneID))
	if err != nil {
		return nil, fmt.Errorf("game: instantiate scene %q: %w", sceneID, err)
	}

	for _, w := range newScene.Warnings {
		log.Printf("game: scene %q: %s", sceneID, w)
	}
	s.localizeSceneAreas(newScene, s.state.Location)

	previous := s.state.Scene
	if !s.loadedSave || s.scene != nil {
		s.state.PreviousLocation = s.state.Location
		s.state.PreviousScene = previous
	}
	s.state.Scene = sceneID
	s.scene = newScene
	s.ambientTasks = nil
	s.walkEpoch = nil
	s.syncSceneInventory()
	if s.haveSavedPlayer {
		if actor := s.PlayerActor(); actor != nil {
			actor.X = float64(s.savedPlayerX)
			actor.Y = float64(s.savedPlayerY)
			actor.Direction = s.savedPlayerDirection
			engine.UpdateActorPerspective(s.scene, actor)
		}
		s.haveSavedPlayer = false
	}

	log.Printf("SCENE: %s | LOCATION: %d", sceneID, s.state.Location)

	if s.controller == nil {
		return nil, nil
	}

	return s.controller.Enter(s.ctx(), sceneID, previous), nil
}

// loadConditionMask asks s.controller for sceneID's scene-condition
// bitmask (see Controller.LoadConditionMask), or 0 if there is no
// controller yet.
func (s *Session) loadConditionMask(sceneID string) int {
	if s.controller == nil {
		return 0
	}
	return s.controller.LoadConditionMask(s.ctx(), sceneID)
}

// TransitionToLocation switches this session to a different location
// entirely -- a different asset index and Controller, not just a different
// scene within the current one (see transitionTo) -- while preserving
// persistent GameState (inventory, flags): only Location/Scene/
// PreviousScene and the live idx/controller change. idx/controller are
// built by the caller (see Context.ChangeLocation, driven by
// SetLocationResolver), since Session itself has no notion of asset
// directory layout. Used for cross-location exits, e.g. Scene 9's
// 009_Tuer leading to Location 5 (agents/scenes/9.MD section 9).
func (s *Session) TransitionToLocation(idx *assets.Index, controller Controller, location int, sceneID string) (engine.Task, error) {
	s.applyLocationEntryState(location, sceneID)
	sceneID = normalizeLocationSceneID(location, sceneID)
	if s.controller != nil && s.scene != nil {
		exitTask := s.controller.Exit(s.ctx(), s.state.Scene, sceneID)
		if exitTask != nil && !exitTask.Update(0) {
			log.Printf("game: Exit(%s -> location %d scene %s) did not complete synchronously; Exit hooks must not block", s.state.Scene, location, sceneID)
		}
	}

	s.audioEngine.StopCategory(audio.CategorySpeech)
	s.audioEngine.StopCategory(audio.CategorySFX)

	s.idx = idx
	s.controller = controller

	def, err := s.loadSceneDef(location, sceneID)
	if err != nil {
		if (location == 10 && loc10SceneID(sceneID) == "S66") || (location == 9 && loc09SceneID(sceneID) == "S89") {
			def = &szn.Definition{Path: sceneID + ".SZN"}
		} else {
			return nil, err
		}
	}

	newScene, err := engine.InstantiateScene(def, s.idx, nil, s.loadConditionMask(sceneID))
	if err != nil {
		return nil, fmt.Errorf("game: instantiate scene %q: %w", sceneID, err)
	}

	for _, w := range newScene.Warnings {
		log.Printf("game: scene %q: %s", sceneID, w)
	}
	s.localizeSceneAreas(newScene, location)

	previous := s.state.Scene
	if !s.loadedSave || s.scene != nil {
		s.state.PreviousLocation = s.state.Location
		s.state.PreviousScene = previous
	}
	s.state.Scene = sceneID
	s.state.Location = location
	s.scene = newScene
	s.ambientTasks = nil
	s.walkEpoch = nil
	s.inventoryText = nil
	s.syncSceneInventory()

	log.Printf("SCENE: %s | LOCATION: %d", sceneID, location)

	if s.controller == nil {
		return nil, nil
	}

	return s.controller.Enter(s.ctx(), sceneID, previous), nil
}

func sceneAssetID(location int, sceneID string) string {
	switch location {
	case 9:
		if loc09SceneID(sceneID) == "S89" {
			return "084"
		}
	case 12:
		if loc12SceneID(sceneID) == "S91" {
			return "091"
		}
	case 14:
		switch strings.ToUpper(strings.TrimSpace(sceneID)) {
		case "90", "090", "S90", "S090":
			return "090"
		}
	case 19:
		if loc19SceneID(sceneID) == "S101" {
			return "68"
		}
	case 23:
		if loc23SceneID(sceneID) == "S102" {
			return "102"
		}
	case 25:
		// Script scene 104 is the on-disk scene S105 (LOC25/S105.SZN).
		// loc25SceneID applies the same mapping after the file is loaded.
		switch strings.ToUpper(strings.TrimSpace(sceneID)) {
		case "104", "S104", "0104", "S0104", "105", "S105":
			return "S105"
		}
	}
	return sceneID
}

func sceneAssetCandidates(location int, sceneID string) []string {
	seen := map[string]bool{}
	var candidates []string
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		key := strings.ToUpper(value)
		if seen[key] {
			return
		}
		seen[key] = true
		candidates = append(candidates, value)
	}

	add(sceneAssetID(location, sceneID))
	add(sceneID)
	normalized := normalizeLocationSceneID(location, sceneID)
	add(normalized)

	for _, value := range append([]string(nil), candidates...) {
		upper := strings.ToUpper(strings.TrimSpace(value))
		if strings.HasPrefix(upper, "S") {
			add(value[1:])
		} else {
			add("S" + value)
		}
		digits := upper
		if strings.HasPrefix(digits, "S") {
			digits = digits[1:]
		}
		if digits == "" {
			continue
		}
		n, err := strconv.Atoi(digits)
		if err != nil {
			continue
		}
		add(strconv.Itoa(n))
		add(fmt.Sprintf("%02d", n))
		add(fmt.Sprintf("%03d", n))
		add(fmt.Sprintf("%04d", n))
		add("S" + strconv.Itoa(n))
		add(fmt.Sprintf("S%02d", n))
		add(fmt.Sprintf("S%03d", n))
		add(fmt.Sprintf("S%04d", n))
	}

	return candidates
}

func (s *Session) loadSceneDef(location int, sceneID string) (*szn.Definition, error) {
	candidates := sceneAssetCandidates(location, sceneID)
	var ambiguous error
	for _, assetID := range candidates {
		name := assetID + ".SZN"
		path, err := s.idx.Resolve(name)
		if err != nil {
			path, err = s.idx.ResolveBaseName(name)
		}
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "ambiguous") && ambiguous == nil {
				ambiguous = err
			}
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("game: read %s: %w", path, err)
		}
		return szn.Load(path, data)
	}
	if ambiguous != nil {
		return nil, fmt.Errorf("game: scene %q: %w", sceneID, ambiguous)
	}
	return nil, fmt.Errorf("game: scene %q: no SZN asset found for candidates %v", sceneID, candidates)
}

func (s *Session) hitArea(x, y float32) (string, bool) {
	for _, id := range s.scene.AreaOrder {
		area := s.scene.Areas[id]
		if area.Enabled && area.Contains(int(x), int(y)) {
			return id, true
		}
	}
	return "", false
}

func (s *Session) inventoryItemAt(x, y float32) (int, bool) {
	const (
		slotSize    = float32(56)
		slotPadding = float32(10)
		panelTop    = float32(360)
		panelHeight = float32(120)
	)

	if y < panelTop || y >= panelTop+panelHeight {
		return 0, false
	}

	slotY := panelTop + (panelHeight-slotSize)/2
	if y < slotY || y >= slotY+slotSize {
		return 0, false
	}

	for i, id := range s.state.Inventory {
		slotX := slotPadding + float32(i)*(slotSize+slotPadding)
		if x >= slotX && x < slotX+slotSize {
			return id, true
		}
	}

	return 0, false
}

// SelectItem handles the player directly selecting item id in the
// inventory overlay: it first offers the controller a chance to handle
// direct activation (see Controller.SelectItem), running that task
// immediately if given one; otherwise it falls back to arming id for a
// "use item on hotspot" click, per agents/GAMEPLAY.md "Item use".
func (s *Session) SelectItem(id int) {
	if s.controller != nil {
		if task := s.controller.SelectItem(s.ctx(), id); task != nil {
			s.startControllerTask(task)
			return
		}
	}
	s.selectedItem = &id
}

// SelectedItem returns the currently armed item ID, if any.
func (s *Session) SelectedItem() (int, bool) {
	if s.selectedItem == nil {
		return 0, false
	}
	return *s.selectedItem, true
}

// ClearSelectedItem disarms any selected item without using it.
func (s *Session) ClearSelectedItem() {
	s.selectedItem = nil
}

// Speak starts a real recorded line (basename.WAV + optional
// basename.ACS) for the player actor outside of any controller dispatch,
// replacing any current action. Intended for developer/CLI verification
// (see cmd/rah --speak); scripted dialogue in normal play goes through a
// Controller returning ctx.Say(...) instead.
func (s *Session) Speak(baseName, subtitle string) error {
	actor := s.PlayerActor()
	if actor == nil {
		return fmt.Errorf("game: no actor to speak")
	}

	speech, err := engine.NewSpeech(s.idx, s.audioEngine, actor, baseName, subtitle)
	if err != nil {
		return err
	}

	s.activeTask = &dialogueTask{session: s, inner: speech}
	s.locked = true

	return nil
}

// HandleClick dispatches a logical-space world click (y must already be
// known to be within the scene viewport, not the reserved UI strip).
// While locked (a scripted action is running), clicks are ignored per the
// interaction lock.
func (s *Session) HandleClick(x, y float32) {
	if s.locked {
		return
	}

	if item, ok := s.inventoryItemAt(x, y); ok {
		s.SelectItem(item)
		return
	}

	areaID, hit := s.hitArea(x, y)

	if item, armed := s.SelectedItem(); armed {
		s.selectedItem = nil
		if hit {
			task := s.controller.UseItem(s.ctx(), item, areaID)
			if task == nil {
				task = s.defaultAreaResponse(areaID)
			}
			s.startControllerTask(task)
		}
		return
	}

	if hit {
		task := s.controller.Click(s.ctx(), areaID)
		if task == nil {
			// A hotspot with no scripted behavior yet must still feel
			// responsive: walk there and acknowledge the click, rather than
			// silently doing nothing (which reads as "this is broken"
			// rather than "this puzzle isn't ported yet").
			task = s.defaultAreaResponse(areaID)
		}
		s.startControllerTask(task)
		return
	}

	actor := s.PlayerActor()
	if actor == nil {
		return
	}

	s.activeTask = s.ctx().WalkTo(actor.ID, float64(x), float64(y))
	s.locked = false
}

func (s *Session) startControllerTask(task engine.Task) {
	if task == nil {
		return
	}
	s.activeTask = task
	s.locked = true
}

// defaultAreaResponse walks the player to areaID's center and shows a
// generic acknowledgement. Used when a Controller returns nil for a Click
// or UseItem it doesn't specifically handle, so every hotspot in a scene
// gives the player some feedback even before its real puzzle logic is
// ported (see agents/GAMEPLAY.md's equivalent guidance for UseItem:
// "If unhandled, play a generic refusal line").
func (s *Session) defaultAreaResponse(areaID string) engine.Task {
	actor := s.PlayerActor()
	if actor == nil {
		return nil
	}

	x, y, ok := s.ctx().AreaCenter(areaID)
	if !ok {
		return nil
	}

	return engine.Sequence(
		s.ctx().WalkTo(actor.ID, x, y),
		s.ctx().SayText(actor.ID, "Nothing interesting happens.", 1.0),
	)
}

// Update advances the active task (if any) and the scene's layer
// animations. Any layer currently marked engine.Layer.TaskDriven is skipped
// here -- some step of the active task chain (or an ambient task) already
// calls Advance on it directly this same tick, e.g. engine.PlayLayerOnce or
// this package's layerRangeTask (see PlayLayer/PlayLayerSplit/
// PlayLayerFrames/PlayVideo/PlayVideoSplit); Advancing it again here would
// double its effective frame rate, playing that whole scripted animation
// (and anything timed against it, like a cutscene's footstep-SFX cue sheet)
// at roughly 2x its intended speed.
func (s *Session) Update(dt float64) {
	if s.savedFlash > 0 {
		s.savedFlash -= dt
		if s.savedFlash < 0 {
			s.savedFlash = 0
		}
	}
	if s.activeTask != nil {
		if s.activeTask.Update(dt) {
			s.activeTask = nil
			s.locked = false
		}
	}

	if len(s.ambientTasks) > 0 {
		kept := s.ambientTasks[:0]
		for _, t := range s.ambientTasks {
			if !t.Update(dt) {
				kept = append(kept, t)
			}
		}
		s.ambientTasks = kept
	}

	if s.scene == nil {
		return
	}

	s.syncSceneInventory()
	if s.scene.InventoryHidden {
		s.scene.InventoryHoverIndex = -1
		s.scene.InventoryClickIndex = -1
	} else {
		if s.locked {
			s.scene.InventoryClickIndex = -1
		} else if s.scene.InventoryClickIndex >= 0 && s.scene.InventoryClickIndex < len(s.scene.InventoryItems) {
			index := s.scene.InventoryClickIndex
			s.scene.InventoryClickIndex = -1
			if item := s.scene.InventoryItems[index]; item != nil {
				s.SelectItem(item.ID)
			}
		}
		for _, anim := range s.scene.InventoryAnims {
			anim.Advance(dt)
		}
	}

	for _, name := range s.scene.LayerOrder {
		if layer := s.scene.Layers[name]; !layer.TaskDriven {
			layer.Advance(dt)
		}
	}
}
