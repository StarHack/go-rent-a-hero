// Package szn parses the game's `.SZN` scene definition files.
//
// SZN files are Windows-INI-style text, not bytecode: they describe which
// background/character/layer/area sections make up a scene. Puzzle logic
// lives in the (to-be-ported) location controllers, not in this format.
//
// Parsing here only produces data: it never touches the asset filesystem or
// creates renderer resources (see agents/SZN.md, "Runtime instantiation").
package szn

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/wok/rent-a-hero/internal/formats/ini"
)

// General holds the [GENERAL] section: ordered references to Background,
// Character, Layer and Area sections, keyed by their numeric suffix.
type General struct {
	Backgrounds []string
	Characters  []string
	Layers      []string
	Areas       []string
	Extra       map[string]string
}

// Compiler holds controller-owned video resources declared by an optional
// [COMPILER] section.  These are not ordinary depth-sorted scene layers:
// the original controller installs them into the scene's dedicated
// presentation-video slot before playback (FUN_0044a330).
type Compiler struct {
	Videos []string
	Extra  map[string]string
}

// BackgroundDef is a parsed background section.
type BackgroundDef struct {
	ID       string
	Filename string
	ZBuf     string
	WBuf     string
	FPS      int
	Extra    map[string]string
}

// ColorZone is one numbered local color-correction zone
// (RedAddN/GreenAddN/BlueAddN/ColorAddXN/ColorAddYN/ColorAddRangeN).
type ColorZone struct {
	Index int
	Red   *int
	Green *int
	Blue  *int
	X     *int
	Y     *int
	Range *int
}

// CharacterDef is a parsed character section.
type CharacterDef struct {
	ID        string
	Filename  string
	Type      *int
	X         int
	Y         int
	ZPos      int
	ZoomVal   int
	Direction int
	ZBuffered *int
	SFXLeft   string
	SFXRight  string
	SFXAlt    string
	RedAdd    *int
	GreenAdd  *int
	BlueAdd   *int
	Zones     []ColorZone
	Extra     map[string]string
}

// LayerDef is a parsed layer section.
type LayerDef struct {
	ID         string
	Filename   string
	X          int
	Y          int
	Z          *int
	ZoomVal    *int
	FPS        *int
	LoadCond   *int
	ZBuffered  *int
	DirtyAlpha *int
	Extra      map[string]string
}

// AreaDef is a parsed area (hotspot) section.
type AreaDef struct {
	ID         string
	X1         int
	Y1         int
	X2         int
	Y2         int
	CursorType *int
	Type       *int
	Extra      map[string]string
}

// Definition is a fully parsed SZN scene: raw data only, no SDL/renderer
// resources.
type Definition struct {
	Path        string
	General     General
	Compiler    Compiler
	Backgrounds map[string]*BackgroundDef
	Characters  map[string]*CharacterDef
	Layers      map[string]*LayerDef
	Areas       map[string]*AreaDef
}

var generalRefPattern = regexp.MustCompile(`(?i)^(Background|Character|Layer|Area)([0-9]+)$`)
var compilerVideoPattern = regexp.MustCompile(`(?i)^Video([0-9]+)$`)

// Load parses raw SZN bytes (any of the recognized text encodings) into a
// Definition.
func Load(path string, data []byte) (*Definition, error) {
	text := ini.Decode(data)
	f := ini.Parse(text)

	generalSection, ok := f.Section("GENERAL")
	if !ok {
		return nil, fmt.Errorf("szn: %s: missing [GENERAL] section", path)
	}

	type ref struct {
		index int
		value string
	}

	refs := map[string][]ref{"Background": nil, "Character": nil, "Layer": nil, "Area": nil}
	extra := map[string]string{}

	for _, key := range generalSection.Keys {
		m := generalRefPattern.FindStringSubmatch(key)
		if m == nil {
			v, _ := generalSection.Get(key)
			extra[key] = v
			continue
		}

		prefix := canonicalPrefix(m[1])
		index, err := strconv.Atoi(m[2])
		if err != nil {
			return nil, fmt.Errorf("szn: %s: [GENERAL] key %q: %w", path, key, err)
		}

		value, _ := generalSection.Get(key)
		refs[prefix] = append(refs[prefix], ref{index: index, value: value})
	}

	for _, list := range refs {
		sort.Slice(list, func(i, j int) bool { return list[i].index < list[j].index })
	}

	orderedNames := func(prefix string) []string {
		out := make([]string, 0, len(refs[prefix]))
		for _, r := range refs[prefix] {
			out = append(out, r.value)
		}
		return out
	}

	compiler := Compiler{Extra: map[string]string{}}
	if compilerSection, ok := f.Section("COMPILER"); ok {
		videos := make([]ref, 0)
		for _, key := range compilerSection.Keys {
			value, _ := compilerSection.Get(key)
			m := compilerVideoPattern.FindStringSubmatch(key)
			if m == nil {
				compiler.Extra[key] = value
				continue
			}
			index, err := strconv.Atoi(m[1])
			if err != nil {
				return nil, fmt.Errorf("szn: %s: [COMPILER] key %q: %w", path, key, err)
			}
			videos = append(videos, ref{index: index, value: value})
		}
		sort.Slice(videos, func(i, j int) bool { return videos[i].index < videos[j].index })
		for _, video := range videos {
			compiler.Videos = append(compiler.Videos, video.value)
		}
	}

	def := &Definition{
		Path: path,
		General: General{
			Backgrounds: orderedNames("Background"),
			Characters:  orderedNames("Character"),
			Layers:      orderedNames("Layer"),
			Areas:       orderedNames("Area"),
			Extra:       extra,
		},
		Compiler:    compiler,
		Backgrounds: map[string]*BackgroundDef{},
		Characters:  map[string]*CharacterDef{},
		Layers:      map[string]*LayerDef{},
		Areas:       map[string]*AreaDef{},
	}

	for _, name := range def.General.Backgrounds {
		bg, err := parseBackground(path, f, name)
		if err != nil {
			return nil, err
		}
		def.Backgrounds[name] = bg
	}

	for _, name := range def.General.Characters {
		ch, err := parseCharacter(path, f, name)
		if err != nil {
			return nil, err
		}
		def.Characters[name] = ch
	}

	for _, name := range def.General.Layers {
		l, err := parseLayer(path, f, name)
		if err != nil {
			return nil, err
		}
		def.Layers[name] = l
	}

	for _, name := range def.General.Areas {
		a, err := parseArea(path, f, name)
		if err != nil {
			return nil, err
		}
		def.Areas[name] = a
	}

	return def, nil
}

func canonicalPrefix(p string) string {
	switch strings.ToLower(p) {
	case "background":
		return "Background"
	case "character":
		return "Character"
	case "layer":
		return "Layer"
	case "area":
		return "Area"
	}
	return p
}

func requireSection(path string, f *ini.File, name, kind string) (*ini.Section, error) {
	s, ok := f.Section(name)
	if !ok {
		return nil, fmt.Errorf("szn: %s: referenced %s section %q not found", path, kind, name)
	}
	return s, nil
}

// knownKeys tracks which keys a section-specific parser consumed so the rest
// can be captured into Extra.
type keyTracker struct {
	section *ini.Section
	used    map[string]bool
}

func newKeyTracker(s *ini.Section) *keyTracker {
	return &keyTracker{section: s, used: map[string]bool{}}
}

func (t *keyTracker) getString(key string) (string, bool) {
	t.used[strings.ToLower(key)] = true
	return t.section.Get(key)
}

func (t *keyTracker) getInt(key string) (int, bool, error) {
	v, ok := t.getString(key)
	if !ok {
		return 0, false, nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0, false, fmt.Errorf("key %q: value %q is not a decimal integer: %w", key, v, err)
	}
	return n, true, nil
}

func (t *keyTracker) getIntPtr(key string) (*int, error) {
	n, ok, err := t.getInt(key)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return &n, nil
}

// extra returns all section keys not consumed by getString/getInt/getIntPtr.
func (t *keyTracker) extra() map[string]string {
	out := map[string]string{}
	for _, key := range t.section.Keys {
		if t.used[strings.ToLower(key)] {
			continue
		}
		v, _ := t.section.Get(key)
		out[key] = v
	}
	return out
}

func parseBackground(path string, f *ini.File, name string) (*BackgroundDef, error) {
	s, err := requireSection(path, f, name, "background")
	if err != nil {
		return nil, err
	}

	t := newKeyTracker(s)

	filename, _ := t.getString("Filename")
	zbuf, _ := t.getString("zBuf")
	wbuf, _ := t.getString("wBuf")
	fps, _, err := t.getInt("fps")
	if err != nil {
		return nil, fmt.Errorf("szn: %s: background %q: %w", path, name, err)
	}

	return &BackgroundDef{
		ID:       name,
		Filename: filename,
		ZBuf:     zbuf,
		WBuf:     wbuf,
		FPS:      fps,
		Extra:    t.extra(),
	}, nil
}

func parseCharacter(path string, f *ini.File, name string) (*CharacterDef, error) {
	s, err := requireSection(path, f, name, "character")
	if err != nil {
		return nil, err
	}

	t := newKeyTracker(s)

	wrap := func(err error) error {
		if err == nil {
			return nil
		}
		return fmt.Errorf("szn: %s: character %q: %w", path, name, err)
	}

	filename, _ := t.getString("Filename")
	typ, err := t.getIntPtr("Type")
	if err != nil {
		return nil, wrap(err)
	}

	x, _, err := t.getInt("X")
	if err != nil {
		return nil, wrap(err)
	}
	y, _, err := t.getInt("Y")
	if err != nil {
		return nil, wrap(err)
	}
	zpos, _, err := t.getInt("ZPos")
	if err != nil {
		return nil, wrap(err)
	}
	zoom, _, err := t.getInt("ZoomVal")
	if err != nil {
		return nil, wrap(err)
	}
	direction, _, err := t.getInt("Direction")
	if err != nil {
		return nil, wrap(err)
	}
	zbuffered, err := t.getIntPtr("ZBuffered")
	if err != nil {
		return nil, wrap(err)
	}

	sfxLeft, _ := t.getString("SFXLeft")
	sfxRight, _ := t.getString("SFXRight")
	sfxAlt, _ := t.getString("SFXAlt")

	redAdd, err := t.getIntPtr("RedAdd")
	if err != nil {
		return nil, wrap(err)
	}
	greenAdd, err := t.getIntPtr("GreenAdd")
	if err != nil {
		return nil, wrap(err)
	}
	blueAdd, err := t.getIntPtr("BlueAdd")
	if err != nil {
		return nil, wrap(err)
	}

	zones, err := parseColorZones(t)
	if err != nil {
		return nil, wrap(err)
	}

	return &CharacterDef{
		ID:        name,
		Filename:  filename,
		Type:      typ,
		X:         x,
		Y:         y,
		ZPos:      zpos,
		ZoomVal:   zoom,
		Direction: direction,
		ZBuffered: zbuffered,
		SFXLeft:   sfxLeft,
		SFXRight:  sfxRight,
		SFXAlt:    sfxAlt,
		RedAdd:    redAdd,
		GreenAdd:  greenAdd,
		BlueAdd:   blueAdd,
		Zones:     zones,
		Extra:     t.extra(),
	}, nil
}

var zoneKeyPattern = regexp.MustCompile(`(?i)^(RedAdd|GreenAdd|BlueAdd|ColorAddX|ColorAddY|ColorAddRange)([0-9]+)$`)

// parseColorZones scans all section keys for the ColorAddN/RedAddN/etc.
// family and groups them by numeric suffix N.
func parseColorZones(t *keyTracker) ([]ColorZone, error) {
	byIndex := map[int]*ColorZone{}
	var order []int

	for _, key := range t.section.Keys {
		m := zoneKeyPattern.FindStringSubmatch(key)
		if m == nil {
			continue
		}

		idx, err := strconv.Atoi(m[2])
		if err != nil {
			return nil, fmt.Errorf("zone key %q: %w", key, err)
		}

		zone, ok := byIndex[idx]
		if !ok {
			zone = &ColorZone{Index: idx}
			byIndex[idx] = zone
			order = append(order, idx)
		}

		n, _, err := t.getInt(key)
		if err != nil {
			return nil, err
		}

		switch strings.ToLower(m[1]) {
		case "redadd":
			zone.Red = intPtr(n)
		case "greenadd":
			zone.Green = intPtr(n)
		case "blueadd":
			zone.Blue = intPtr(n)
		case "coloraddx":
			zone.X = intPtr(n)
		case "coloraddy":
			zone.Y = intPtr(n)
		case "coloraddrange":
			zone.Range = intPtr(n)
		}
	}

	sort.Ints(order)

	zones := make([]ColorZone, 0, len(order))
	for _, idx := range order {
		zones = append(zones, *byIndex[idx])
	}

	return zones, nil
}

func intPtr(n int) *int { return &n }

func parseLayer(path string, f *ini.File, name string) (*LayerDef, error) {
	s, err := requireSection(path, f, name, "layer")
	if err != nil {
		return nil, err
	}

	t := newKeyTracker(s)

	wrap := func(err error) error {
		if err == nil {
			return nil
		}
		return fmt.Errorf("szn: %s: layer %q: %w", path, name, err)
	}

	filename, _ := t.getString("Filename")

	x, _, err := t.getInt("X")
	if err != nil {
		return nil, wrap(err)
	}
	y, _, err := t.getInt("Y")
	if err != nil {
		return nil, wrap(err)
	}

	z, err := t.getIntPtr("Z")
	if err != nil {
		return nil, wrap(err)
	}
	zoom, err := t.getIntPtr("ZoomVal")
	if err != nil {
		return nil, wrap(err)
	}
	fps, err := t.getIntPtr("fps")
	if err != nil {
		return nil, wrap(err)
	}
	loadCond, err := t.getIntPtr("LoadCond")
	if err != nil {
		return nil, wrap(err)
	}
	zbuffered, err := t.getIntPtr("ZBuffered")
	if err != nil {
		return nil, wrap(err)
	}
	dirtyAlpha, err := t.getIntPtr("DirtyAlpha")
	if err != nil {
		return nil, wrap(err)
	}

	return &LayerDef{
		ID:         name,
		Filename:   filename,
		X:          x,
		Y:          y,
		Z:          z,
		ZoomVal:    zoom,
		FPS:        fps,
		LoadCond:   loadCond,
		ZBuffered:  zbuffered,
		DirtyAlpha: dirtyAlpha,
		Extra:      t.extra(),
	}, nil
}

func parseArea(path string, f *ini.File, name string) (*AreaDef, error) {
	s, err := requireSection(path, f, name, "area")
	if err != nil {
		return nil, err
	}

	t := newKeyTracker(s)

	wrap := func(err error) error {
		if err == nil {
			return nil
		}
		return fmt.Errorf("szn: %s: area %q: %w", path, name, err)
	}

	x1, _, err := t.getInt("X1")
	if err != nil {
		return nil, wrap(err)
	}
	x2, _, err := t.getInt("X2")
	if err != nil {
		return nil, wrap(err)
	}
	y1, _, err := t.getInt("Y1")
	if err != nil {
		return nil, wrap(err)
	}
	y2, _, err := t.getInt("Y2")
	if err != nil {
		return nil, wrap(err)
	}

	cursorType, err := t.getIntPtr("CursorType")
	if err != nil {
		return nil, wrap(err)
	}
	typ, err := t.getIntPtr("Type")
	if err != nil {
		return nil, wrap(err)
	}

	return &AreaDef{
		ID:         name,
		X1:         x1,
		Y1:         y1,
		X2:         x2,
		Y2:         y2,
		CursorType: cursorType,
		Type:       typ,
		Extra:      t.extra(),
	}, nil
}

// Normalized returns the rectangle with min/max ordered, without mutating
// the original raw values (which are retained for diagnostics).
func (a *AreaDef) Normalized() (minX, minY, maxX, maxY int) {
	minX, maxX = a.X1, a.X2
	if minX > maxX {
		minX, maxX = maxX, minX
	}

	minY, maxY = a.Y1, a.Y2
	if minY > maxY {
		minY, maxY = maxY, minY
	}

	return minX, minY, maxX, maxY
}
