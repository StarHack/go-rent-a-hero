package game

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wok/rent-a-hero/internal/datapath"
)

const (
	saveGameFileName  = "SaveGame01.rah"
	saveGameVersion   = uint32(1)
	saveGameNameSize  = 0x1e
	saveGameItems     = 100
	saveGameStateSize = 0x4110
)

var originalStateDefaults = map[int]uint32{
	0x3e80: 0, 0x3e84: 0, 0x3e88: 0, 0x3e8c: 0, 0x3e90: 0, 0x3e94: 1, 0x3e98: 1, 0x3e9c: 0,
	0x3ea0: 0, 0x3ea4: 1, 0x3ea8: 1, 0x3eac: 1, 0x3eb0: 1, 0x3eb4: 1, 0x3eb8: 1, 0x3ebc: 1,
	0x3ec0: 1, 0x3ec4: 0, 0x3ec8: 0, 0x3ecc: 0, 0x3ed0: 0, 0x3ed4: 0, 0x3ed8: 1, 0x3edc: 1,
	0x3ee0: 1, 0x3ee4: 0, 0x3ee8: 0, 0x3eec: 1, 0x3ef0: 1, 0x3ef4: 0, 0x3ef8: 1, 0x3efc: 1,
	0x3f00: 0, 0x3f04: 1, 0x3f08: 0, 0x3f0c: 1, 0x3f10: 1, 0x3f14: 0, 0x3f18: 0, 0x3f1c: 0,
	0x3f20: 0, 0x3f24: 0, 0x3f28: 0, 0x3f2c: 0, 0x3f30: 0, 0x3f34: 1, 0x3f38: 0, 0x3f3c: 0,
	0x3f40: 0, 0x3f44: 0, 0x3f48: 0, 0x3f4c: 0, 0x3f50: 1, 0x3f54: 0, 0x3f58: 0, 0x3f5c: 1,
	0x3f60: 0, 0x3f64: 1, 0x3f68: 1, 0x3f6c: 0, 0x3f70: 1, 0x3f74: 1, 0x3f78: 1, 0x3f7c: 0,
	0x3f80: 0, 0x3f84: 0, 0x3f88: 1, 0x3f8c: 1, 0x3f90: 1, 0x3f94: 1, 0x3f98: 0, 0x3f9c: 0,
	0x3fa0: 0, 0x3fa4: 1, 0x3fa8: 0, 0x3fac: 1, 0x3fb0: 1, 0x3fb4: 1, 0x3fb8: 0, 0x3fbc: 0,
	0x3fc0: 0, 0x3fc4: 0, 0x3fc8: 0, 0x3fcc: 1, 0x3fd0: 0, 0x3fd4: 1, 0x3fd8: 1, 0x3fdc: 1,
	0x3fe0: 0, 0x3fe4: 0, 0x3fe8: 1, 0x3fec: 0, 0x3ff0: 0, 0x3ff4: 0, 0x3ff8: 0, 0x3ffc: 0,
	0x4000: 0, 0x4004: 1, 0x4008: 1, 0x400c: 1, 0x4010: 1, 0x4014: 0, 0x4018: 0, 0x401c: 0xffffffff,
	0x4020: 1, 0x4024: 0, 0x4028: 0, 0x402c: 0, 0x4030: 1, 0x4034: 1, 0x4038: 0, 0x403c: 0,
	0x4040: 1, 0x4044: 1, 0x4048: 0, 0x404c: 1, 0x4050: 1, 0x4054: 1, 0x4058: 1, 0x405c: 0,
	0x4060: 1, 0x4064: 0, 0x4068: 0, 0x406c: 0, 0x4070: 0, 0x4074: 0, 0x4078: 0, 0x407c: 1,
	0x4080: 0, 0x4084: 0, 0x4088: 0, 0x408c: 0, 0x4090: 0, 0x4094: 0, 0x4098: 0, 0x409c: 1,
	0x40a0: 1, 0x40a4: 1, 0x40a8: 0, 0x40ac: 1, 0x40b0: 1, 0x40b4: 1, 0x40b8: 0, 0x40bc: 0,
	0x40c0: 1, 0x40c4: 1, 0x40c8: 1, 0x40cc: 1, 0x40d0: 1, 0x40d4: 1, 0x40d8: 1, 0x40dc: 1,
	0x40e0: 0, 0x40e4: 0, 0x40e8: 1, 0x40ec: 1, 0x40f0: 0, 0x40f4: 0, 0x40f8: 1, 0x40fc: 0,
	0x4100: 0, 0x4104: 1, 0x4108: 1, 0x410c: 1,
}

func newOriginalStateBlock() []byte {
	data := make([]byte, saveGameStateSize)
	for offset, value := range originalStateDefaults {
		binary.LittleEndian.PutUint32(data[offset:offset+4], value)
	}
	return data
}

func saveGameRootPath() (string, error) {
	if configPath, err := datapath.Resolve("config.ini"); err == nil {
		return filepath.Join(filepath.Dir(configPath), saveGameFileName), nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(cwd, saveGameFileName), nil
}

func encodeOriginalScene(scene string) (int32, error) {
	switch strings.ToUpper(strings.TrimSpace(scene)) {
	case "7A":
		return 7, nil
	case "7B":
		return 0x3ef, nil
	}
	value := strings.TrimSpace(scene)
	if len(value) > 1 && (value[0] == 'S' || value[0] == 's') {
		value = value[1:]
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("game: scene %q has no original numeric scene code", scene)
	}
	return int32(n), nil
}

func decodeOriginalScene(location int32, scene int32) string {
	if location == 1 {
		switch scene {
		case 7:
			return "7A"
		case 0x3ef:
			return "7B"
		}
	}
	switch location {
	case 2:
		switch scene {
		case 13:
			return "S0013"
		case 14:
			return "S0014"
		case 15:
			return "S0015"
		case 1013:
			return "S1013"
		case 2013:
			return "S2013"
		}
	case 14:
		if scene == 90 {
			return "S90"
		}
	case 10:
		switch scene {
		case 37, 38, 39, 40, 41, 42, 43, 44, 66:
			return fmt.Sprintf("S%d", scene)
		}
	case 3:
		if scene == 12 {
			return "S12"
		}
	case 4:
		if scene == 6 {
			return "006"
		}
	case 5:
		if scene == 10 {
			return "S10"
		}
	case 6:
		switch scene {
		case 26:
			return "026"
		case 27:
			return "027"
		}
	case 19:
		switch scene {
		case 68, 69, 101:
			return fmt.Sprintf("S%d", scene)
		}
	case 20:
		if scene == 70 {
			return "S70"
		}
	case 21:
		if scene >= 71 && scene <= 80 {
			return fmt.Sprintf("S%d", scene)
		}
	case 22:
		switch scene {
		case 81, 1081:
			return fmt.Sprintf("S%d", scene)
		}
	case 28:
		switch scene {
		case 30, 31, 36:
			return "S31"
		case 32:
			return "S32"
		case 35:
			return "S35"
		}
	case 29:
		if scene == 33 {
			return "S33"
		}
	case 30:
		if scene == 34 {
			return "S34"
		}
	case 31:
		if scene == 113 {
			return "S113"
		}
	case 35:
		if scene == 1 {
			return "S1"
		}
	case 37:
		if scene == 3 {
			return "S3"
		}
	}
	return strconv.Itoa(int(scene))
}

func putOriginalFlag(data []byte, offset int, value int) {
	if offset < 0 || offset+4 > len(data) {
		return
	}
	binary.LittleEndian.PutUint32(data[offset:offset+4], uint32(value))
}

func getOriginalFlag(data []byte, offset int) int {
	if offset < 0 || offset+4 > len(data) {
		return 0
	}
	return int(int32(binary.LittleEndian.Uint32(data[offset : offset+4])))
}

func syncOptionalOriginalFlag(data []byte, flags map[string]int, offset int, name string) {
	if value, ok := flags[name]; ok {
		putOriginalFlag(data, offset, value)
	}
}

func migrateLegacyLoc05State(data []byte) {
	if getOriginalFlag(data, 0x3ed8) != 0 || getOriginalFlag(data, 0x3edc) != 0 || getOriginalFlag(data, 0x3ee0) != 0 {
		return
	}
	putOriginalFlag(data, 0x3ed8, 1)
	putOriginalFlag(data, 0x3edc, 1)
	putOriginalFlag(data, 0x3ee0, 1)
}

func (s *Session) syncFlagsToOriginalState() {
	if len(s.state.OriginalState) != saveGameStateSize {
		s.state.OriginalState = newOriginalStateBlock()
	}
	syncOptionalOriginalFlag(s.state.OriginalState, s.state.Flags, 0x3e84, flagLoc05QuestA)
	putOriginalFlag(s.state.OriginalState, 0x3e88, s.state.Flags[FlagLoc04CustomerStory])
	putOriginalFlag(s.state.OriginalState, 0x3e94, s.state.Flags[FlagCanalStoryFlag])
	syncOptionalOriginalFlag(s.state.OriginalState, s.state.Flags, 0x3ea0, flagLoc05QuestB)
	syncOptionalOriginalFlag(s.state.OriginalState, s.state.Flags, 0x3ec8, FlagLoc05PirateAttackSeen)
	putOriginalFlag(s.state.OriginalState, 0x3ecc, s.state.Flags[FlagLoc03RaidersGone])
	putOriginalFlag(s.state.OriginalState, 0x3ed0, s.state.Flags[FlagLoc04VisitCount])
	putOriginalFlag(s.state.OriginalState, 0x3ed4, s.state.Flags[FlagLoc04PaymentOnDesk])
	syncOptionalOriginalFlag(s.state.OriginalState, s.state.Flags, 0x3ed8, flagLoc05GreetingPending)
	syncOptionalOriginalFlag(s.state.OriginalState, s.state.Flags, 0x3edc, flagLoc05Conversation)
	syncOptionalOriginalFlag(s.state.OriginalState, s.state.Flags, 0x3ee0, flagLoc05Flowers)
	syncOptionalOriginalFlag(s.state.OriginalState, s.state.Flags, 0x3ee4, flagLoc05LouiDepartureDone)
}

func (s *Session) syncOriginalStateToFlags() {
	if len(s.state.OriginalState) != saveGameStateSize {
		return
	}
	s.state.Flags[flagLoc05DefaultsInitialized] = 1
	s.state.Flags[flagLoc05QuestA] = getOriginalFlag(s.state.OriginalState, 0x3e84)
	s.state.Flags[FlagLoc04CustomerStory] = getOriginalFlag(s.state.OriginalState, 0x3e88)
	s.state.Flags[FlagCanalStoryFlag] = getOriginalFlag(s.state.OriginalState, 0x3e94)
	s.state.Flags[FlagCanalTalkStage] = getOriginalFlag(s.state.OriginalState, 0x3e98)
	s.state.Flags[flagLoc05QuestB] = getOriginalFlag(s.state.OriginalState, 0x3ea0)
	s.state.Flags[FlagCanalTalkUnlock] = getOriginalFlag(s.state.OriginalState, 0x3ec4)
	s.state.Flags[FlagLoc05PirateAttackSeen] = getOriginalFlag(s.state.OriginalState, 0x3ec8)
	s.state.Flags[FlagLoc03RaidersGone] = getOriginalFlag(s.state.OriginalState, 0x3ecc)
	s.state.Flags[FlagLoc04VisitCount] = getOriginalFlag(s.state.OriginalState, 0x3ed0)
	s.state.Flags[FlagLoc04PaymentOnDesk] = getOriginalFlag(s.state.OriginalState, 0x3ed4)
	s.state.Flags[flagLoc05GreetingPending] = getOriginalFlag(s.state.OriginalState, 0x3ed8)
	s.state.Flags[flagLoc05Conversation] = getOriginalFlag(s.state.OriginalState, 0x3edc)
	s.state.Flags[flagLoc05Flowers] = getOriginalFlag(s.state.OriginalState, 0x3ee0)
	s.state.Flags[flagLoc05LouiDepartureDone] = getOriginalFlag(s.state.OriginalState, 0x3ee4)
}

func (s *Session) SaveGame() error {
	path := s.saveGamePath
	if path == "" {
		var err error
		path, err = saveGameRootPath()
		if err != nil {
			return err
		}
	}
	return s.SaveGameTo(path)
}

func (s *Session) SaveGameTo(path string) error {
	if s.scene == nil {
		return fmt.Errorf("game: cannot save without a loaded scene")
	}
	currentScene, err := encodeOriginalScene(s.state.Scene)
	if err != nil {
		return err
	}
	previousScene := int32(0)
	if s.state.PreviousScene != "" {
		previousScene, err = encodeOriginalScene(s.state.PreviousScene)
		if err != nil {
			return err
		}
	}

	s.syncFlagsToOriginalState()

	playerX, playerY, playerDirection := int32(0), int32(0), int32(0)
	if actor := s.PlayerActor(); actor != nil {
		playerX = int32(actor.X)
		playerY = int32(actor.Y)
		playerDirection = int32(actor.Direction)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("game: create save %s: %w", path, err)
	}
	defer file.Close()

	if err := binary.Write(file, binary.LittleEndian, saveGameVersion); err != nil {
		return err
	}
	var name [saveGameNameSize]byte
	copy(name[:], "Game 1")
	if _, err := file.Write(name[:]); err != nil {
		return err
	}

	values := []int32{
		int32(s.state.PreviousLocation),
		previousScene,
		int32(s.state.Location),
		currentScene,
		playerX,
		playerY,
		playerDirection,
	}
	for _, value := range values {
		if err := binary.Write(file, binary.LittleEndian, value); err != nil {
			return err
		}
	}

	present := make(map[int]struct{}, len(s.state.Inventory))
	for _, id := range s.state.Inventory {
		present[id] = struct{}{}
	}
	for id := 0; id < saveGameItems; id++ {
		value := int32(0)
		if _, ok := present[id]; ok {
			value = 1
		}
		if err := binary.Write(file, binary.LittleEndian, value); err != nil {
			return err
		}
	}

	if err := binary.Write(file, binary.LittleEndian, uint32(saveGameStateSize)); err != nil {
		return err
	}
	if _, err := file.Write(s.state.OriginalState); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}

	s.savedFlash = 2
	return nil
}

func (s *Session) loadGameFrom(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var version uint32
	if err := binary.Read(file, binary.LittleEndian, &version); err != nil {
		return err
	}
	if version != saveGameVersion {
		return fmt.Errorf("game: unsupported save version %d", version)
	}

	var name [saveGameNameSize]byte
	if _, err := io.ReadFull(file, name[:]); err != nil {
		return err
	}

	var previousLocation, previousScene, location, scene int32
	var playerX, playerY, playerDirection int32
	for _, target := range []any{&previousLocation, &previousScene, &location, &scene, &playerX, &playerY, &playerDirection} {
		if err := binary.Read(file, binary.LittleEndian, target); err != nil {
			return err
		}
	}

	inventory := make([]int, 0, saveGameItems)
	for id := 0; id < saveGameItems; id++ {
		var value int32
		if err := binary.Read(file, binary.LittleEndian, &value); err != nil {
			return err
		}
		if value != 0 {
			inventory = append(inventory, id)
		}
	}

	var stateSize uint32
	if err := binary.Read(file, binary.LittleEndian, &stateSize); err != nil {
		return err
	}
	if stateSize != saveGameStateSize {
		return fmt.Errorf("game: save state size 0x%x, want 0x%x", stateSize, saveGameStateSize)
	}
	stateData := make([]byte, stateSize)
	if _, err := io.ReadFull(file, stateData); err != nil {
		return err
	}

	s.state.PreviousLocation = int(previousLocation)
	s.state.PreviousScene = decodeOriginalScene(previousLocation, previousScene)
	s.state.Location = int(location)
	s.state.Scene = decodeOriginalScene(location, scene)
	s.state.Inventory = inventory
	migrateLegacyLoc05State(stateData)
	s.state.OriginalState = stateData
	s.savedPlayerX = int(playerX)
	s.savedPlayerY = int(playerY)
	s.savedPlayerDirection = int(playerDirection)
	s.haveSavedPlayer = true
	s.loadedSave = true
	s.syncOriginalStateToFlags()
	return nil
}

func (s *Session) autoLoadGame() {
	path := s.saveGamePath
	if path == "" {
		var err error
		path, err = saveGameRootPath()
		if err != nil {
			return
		}
	}
	if err := s.loadGameFrom(path); err != nil && !os.IsNotExist(err) {
		return
	}
}

func (s *Session) SavedVisible() bool {
	return s.savedFlash > 0
}
