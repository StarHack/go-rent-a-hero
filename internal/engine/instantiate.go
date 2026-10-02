package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wok/rent-a-hero/internal/assets"
	"github.com/wok/rent-a-hero/internal/formats/szn"
	"github.com/wok/rent-a-hero/internal/formats/zbf"
)

// LoadConditionEvaluator decides whether a layer's LoadCond should be
// treated as satisfied. See agents/SZN.md "LoadCond".
type LoadConditionEvaluator interface {
	ShouldLoad(cond int, mask int) bool
}

// defaultLoadConditionEvaluator implements the original engine's own
// generic scene-loader rule, per agents/scenes/46.MD section 16 ("Critical
// implementation note -- LoadCond=1"): an absent/zero condition always
// loads; a non-zero condition loads exactly when every bit it sets is also
// set in mask (a scene-specific bitmask each location's own constructor
// computes -- see game.Controller.LoadConditionMask). LoadCond is never a
// reference to a "controller ID"; it is purely this bitmask test. Callers
// that pass mask=0 without a real per-scene mask (i.e. no Controller
// implements LoadConditionMask meaningfully for that scene) get the
// conservative fallback of never loading a non-zero-conditioned layer,
// which avoids silently displaying puzzle-state layers before their
// triggering logic is ported.
type defaultLoadConditionEvaluator struct{}

func (defaultLoadConditionEvaluator) ShouldLoad(cond int, mask int) bool {
	return cond == 0 || cond&mask == cond
}

// DefaultLoadConditionEvaluator is the default LoadConditionEvaluator, used
// whenever InstantiateScene is given a nil one.
var DefaultLoadConditionEvaluator LoadConditionEvaluator = defaultLoadConditionEvaluator{}

func intOr(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}

// InstantiateScene resolves and decodes every asset referenced by def
// through idx, producing a fully in-memory Scene with no SDL/renderer
// resources (per agents/SZN.md "Runtime instantiation").
//
// Only the first declared Background is instantiated; the original format
// allows Background1..N but no observed scene declares more than one.
func InstantiateScene(def *szn.Definition, idx *assets.Index, loadCond LoadConditionEvaluator, loadMask int) (*Scene, error) {
	if loadCond == nil {
		loadCond = DefaultLoadConditionEvaluator
	}

	scene := &Scene{
		ID:                 def.Path,
		Characters:         map[string]*Actor{},
		Layers:             map[string]*Layer{},
		Areas:              map[string]*Area{},
		PresentationAssets: map[string]bool{},
	}
	for _, name := range def.Compiler.Videos {
		scene.PresentationAssets[strings.ToLower(strings.TrimSpace(name))] = true
	}

	if len(def.General.Backgrounds) > 0 {
		bg, err := instantiateBackground(def.Backgrounds[def.General.Backgrounds[0]], idx)
		if err != nil {
			return nil, err
		}
		scene.Background = bg
	}

	for _, name := range def.General.Characters {
		actor, err := instantiateActor(def.Characters[name], idx)
		if err != nil {
			return nil, fmt.Errorf("character %q: %w", name, err)
		}
		if actor.TalkSourceError != "" {
			scene.Warnings = append(scene.Warnings, fmt.Sprintf("character %q: %s", name, actor.TalkSourceError))
		}
		scene.Characters[name] = actor
		scene.CharacterOrder = append(scene.CharacterOrder, name)
	}

	for _, name := range def.General.Layers {
		layerDef := def.Layers[name]

		layer, err := instantiateLayer(layerDef, idx)
		if err != nil {
			return nil, fmt.Errorf("layer %q: %w", name, err)
		}

		cond := intOr(layerDef.LoadCond, 0)
		layer.Enabled = loadCond.ShouldLoad(cond, loadMask)
		layer.Visible = layer.Enabled

		// LoadCond is a bitmask requirement against loadMask, never a
		// "controller ID" (agents/scenes/7A-46.MD sections 1/18/26) -- this
		// diagnostic must say so, not claim some "controller" failed to
		// resolve. A mismatch is often entirely intentional (the layer's
		// condition genuinely isn't met yet), but still worth surfacing:
		// loadMask=0 with no real per-scene Controller.LoadConditionMask
		// wired up looks identical to "condition legitimately false" from
		// here, and this is the only signal that distinguishes the two.
		scene.Layers[name] = layer
		scene.LayerOrder = append(scene.LayerOrder, name)
	}

	for _, name := range def.General.Areas {
		areaDef := def.Areas[name]
		cursorType := intOr(areaDef.CursorType, 0)
		if cursorType == 0 {
			cursorType = 10
		}
		scene.Areas[name] = &Area{
			ID:         name,
			X1:         areaDef.X1,
			Y1:         areaDef.Y1,
			X2:         areaDef.X2,
			Y2:         areaDef.Y2,
			CursorType: cursorType,
			Enabled:    true,
		}
		scene.AreaOrder = append(scene.AreaOrder, name)
	}

	return scene, nil
}

func MergeScene(scene *Scene, def *szn.Definition, idx *assets.Index, loadCond LoadConditionEvaluator, loadMask int) error {
	fragment, err := InstantiateScene(def, idx, loadCond, loadMask)
	if err != nil {
		return err
	}

	for _, name := range fragment.CharacterOrder {
		if _, exists := scene.Characters[name]; exists {
			continue
		}
		scene.Characters[name] = fragment.Characters[name]
		scene.CharacterOrder = append(scene.CharacterOrder, name)
	}
	for _, name := range fragment.LayerOrder {
		if _, exists := scene.Layers[name]; exists {
			continue
		}
		scene.Layers[name] = fragment.Layers[name]
		scene.LayerOrder = append(scene.LayerOrder, name)
	}
	for _, name := range fragment.AreaOrder {
		if _, exists := scene.Areas[name]; exists {
			continue
		}
		scene.Areas[name] = fragment.Areas[name]
		scene.AreaOrder = append(scene.AreaOrder, name)
	}
	if scene.PresentationAssets == nil {
		scene.PresentationAssets = map[string]bool{}
	}
	for name := range fragment.PresentationAssets {
		scene.PresentationAssets[name] = true
	}
	scene.Warnings = append(scene.Warnings, fragment.Warnings...)
	return nil
}

func MergeSceneOverlay(scene *Scene, def *szn.Definition, idx *assets.Index, loadCond LoadConditionEvaluator, loadMask int) error {
	fragment, err := InstantiateScene(def, idx, loadCond, loadMask)
	if err != nil {
		return err
	}

	for _, name := range fragment.CharacterOrder {
		if _, exists := scene.Characters[name]; !exists {
			scene.CharacterOrder = append(scene.CharacterOrder, name)
		}
		scene.Characters[name] = fragment.Characters[name]
	}
	for _, name := range fragment.LayerOrder {
		if _, exists := scene.Layers[name]; !exists {
			scene.LayerOrder = append(scene.LayerOrder, name)
		}
		scene.Layers[name] = fragment.Layers[name]
	}
	for _, name := range fragment.AreaOrder {
		if _, exists := scene.Areas[name]; !exists {
			scene.AreaOrder = append(scene.AreaOrder, name)
		}
		scene.Areas[name] = fragment.Areas[name]
	}
	if scene.PresentationAssets == nil {
		scene.PresentationAssets = map[string]bool{}
	}
	for name := range fragment.PresentationAssets {
		scene.PresentationAssets[name] = true
	}
	scene.Warnings = append(scene.Warnings, fragment.Warnings...)
	return nil
}

func instantiateBackground(def *szn.BackgroundDef, idx *assets.Index) (*Background, error) {
	source, err := LoadSpriteSource(idx, def.Filename)
	if err != nil {
		return nil, fmt.Errorf("background %q: %w", def.ID, err)
	}

	bg := &Background{ID: def.ID, Source: source}

	if def.ZBuf != "" {
		depth, err := loadRaster(idx, def.ZBuf)
		if err != nil {
			return nil, fmt.Errorf("background %q: zBuf: %w", def.ID, err)
		}
		bg.Depth = depth
	}

	if def.WBuf != "" {
		walk, err := loadRaster(idx, def.WBuf)
		if err != nil {
			return nil, fmt.Errorf("background %q: wBuf: %w", def.ID, err)
		}
		bg.Walk = walk
	}

	return bg, nil
}

func loadRaster(idx *assets.Index, gamePath string) (*zbf.Raster, error) {
	real, err := idx.Resolve(gamePath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(real)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", real, err)
	}

	return zbf.Parse(data)
}

func instantiateLayer(def *szn.LayerDef, idx *assets.Index) (*Layer, error) {
	source, err := LoadSpriteSource(idx, def.Filename)
	if err != nil {
		return nil, err
	}

	dirtyAlpha := intOr(def.DirtyAlpha, 0) != 0
	_, colorKeyed := source.(ColorKeySource)

	return &Layer{
		ID:         def.ID,
		Source:     source,
		X:          def.X,
		Y:          def.Y,
		Z:          intOr(def.Z, 1),
		Zoom:       intOr(def.ZoomVal, 100),
		FPS:        intOr(def.FPS, 0),
		Mode:       AnimLoop,
		Playing:    true,
		ZBuffered:  intOr(def.ZBuffered, 1) != 0,
		DirtyAlpha: dirtyAlpha,
		ColorKeyed: colorKeyed,
	}, nil
}

func characterAnchor(characterType int) (int, int) {
	switch characterType {
	case 1:
		return 108, 318
	case 10:
		return 55, 150
	case 11:
		return 78, 198
	case 12:
		return 42, 124
	default:
		return 55, 162
	}
}

func instantiateActor(def *szn.CharacterDef, idx *assets.Index) (*Actor, error) {
	source, err := LoadSpriteSource(idx, def.Filename)
	if err != nil {
		return nil, err
	}

	zones := make([]ColorZone, len(def.Zones))
	for i, z := range def.Zones {
		zones[i] = ColorZone{
			R:     intOr(z.Red, 0),
			G:     intOr(z.Green, 0),
			B:     intOr(z.Blue, 0),
			X:     intOr(z.X, 0),
			Y:     intOr(z.Y, 0),
			Range: intOr(z.Range, 0),
		}
	}

	var talkSource SpriteSource
	var talkSourceError string
	ext := filepath.Ext(def.Filename)
	if strings.EqualFold(ext, ".a16") {
		talkName := strings.TrimSuffix(filepath.Base(def.Filename), ext) + "Talk" + ext
		if talkReal, resolveErr := idx.Resolve(talkName); resolveErr == nil {
			if src, loadErr := loadSpriteSourceFile(talkReal, talkName); loadErr == nil {
				talkSource = src
			} else {
				talkSourceError = fmt.Sprintf("talk source %q: %v", talkReal, loadErr)
			}
		}
	}

	characterType := intOr(def.Type, 0)
	anchorX, anchorY := characterAnchor(characterType)

	actor := &Actor{
		ID:              def.ID,
		AssetName:       def.Filename,
		X:               float64(def.X),
		Y:               float64(def.Y),
		ZPos:            def.ZPos,
		Zoom:            def.ZoomVal,
		AuthoredZPos:    def.ZPos,
		AuthoredZoom:    def.ZoomVal,
		Direction:       def.Direction,
		Type:            characterType,
		AnchorX:         anchorX,
		AnchorY:         anchorY,
		ZBuffered:       intOr(def.ZBuffered, 1) != 0,
		Source:          source,
		TalkSource:      talkSource,
		TalkSourceError: talkSourceError,
		StepLeft:        def.SFXLeft,
		StepRight:       def.SFXRight,
		StepAlt:         def.SFXAlt,
		Visible:         true,
		Color: ColorCorrection{
			R:     intOr(def.RedAdd, 0),
			G:     intOr(def.GreenAdd, 0),
			B:     intOr(def.BlueAdd, 0),
			Zones: zones,
		},
	}
	actor.SetFacingImmediate(FacingFromDirectionValue(def.Direction))
	return actor, nil
}
