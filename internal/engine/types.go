// Package engine holds the renderer-independent runtime scene model:
// backgrounds, layers, areas and actors instantiated from a parsed SZN
// definition. Nothing in this file touches SDL, so gameplay/composition
// logic can be tested without a display.
package engine

import "github.com/wok/rent-a-hero/internal/formats/zbf"

// SpriteSource is the common interface over TCC, A16 and BMP visual
// sources, so the renderer does not need to know which original format
// backs a given layer/actor/background.
type SpriteSource interface {
	Width() int
	Height() int
	Frames() int
	RGBA(frame int) ([]byte, error)
}

// ColorKeySource is implemented by sprite/video sources that expose the
// fixed source color key used by the original DirectDraw keyed-blit path.
// Decoding remains untouched; the renderer applies the key only when a Layer
// is marked ColorKeyed.
type ColorKeySource interface {
	ColorKey() (r, g, b byte, ok bool)
}

// ColorZone is one numbered local color-correction zone
// (see agents/IMPLEMENTATION.md "Character lighting").
type ColorZone struct {
	R, G, B int
	X, Y    int
	Range   int
}

// ColorCorrection is the global additive RGB correction plus any local
// correction zones for a character.
type ColorCorrection struct {
	R, G, B int
	Zones   []ColorZone
}

// Background is the scene's base image plus its depth/walk rasters.
type Background struct {
	ID     string
	Source SpriteSource
	Depth  *zbf.Raster
	Walk   *zbf.Raster

	navGrid *NavGrid
}

// DepthAt returns the background's authored per-pixel Z value at (x, y)
// and whether that coordinate is covered by the raster.
//
// Important: this function only exposes the scene ZBF byte. It does NOT
// resolve an actor's current depth. The original executable updates actor
// depth through additional character-anchor/zoom transforms before the
// per-pixel renderer consumes it. The reconstructed engine must therefore
// not substitute DepthAt(actor.X, actor.Y) for Actor.ZPos.
func (b *Background) DepthAt(x, y int) (int, bool) {
	if b == nil || b.Depth == nil {
		return 0, false
	}
	r := b.Depth
	if x < 0 || y < 0 || x >= r.Width || y >= r.Height {
		return 0, false
	}
	return int(r.Pixels[y*r.Width+x]), true
}

// NavGrid returns (building and caching on first use) the coarse
// navigation grid derived from this background's walk raster. It returns
// nil if the background has no walk raster.
func (b *Background) NavGrid() *NavGrid {
	if b.Walk == nil {
		return nil
	}
	if b.navGrid == nil {
		b.navGrid = BuildNavGrid(b.Walk)
	}
	return b.navGrid
}

// AnimMode controls how a Layer's Frame advances over time.
type AnimMode int

const (
	AnimLoop AnimMode = iota
	AnimOnce
	AnimHoldLast
	AnimPingPong

	// AnimOnceReverse decrements Frame toward 0, per
	// agents/scenes/S1_CLOSEUP_ALPHA_FLICKER_FIXES.MD sections 4/5: some
	// authored ranges play backward (e.g. Location 35's close-up return,
	// S1_RodToCloseup frames 40->1) through the same per-tick FPS-timed
	// scheduler as forward playback, not an instant jump. Never wraps or
	// goes negative; Frame simply stops at 0.
	AnimOnceReverse
)

// Layer is one scene-composited visual element (an A16/TCC/BMP source
// placed at a fixed X/Y with an explicit Z and optional animation).
type Layer struct {
	ID     string
	Source SpriteSource
	X      int
	Y      int
	Z      int
	Zoom   int // percent; 100 == 1.0
	FPS    int

	// FillWindow, if true, overrides X/Y/Zoom entirely: the renderer draws
	// this layer stretched (independent X/Y scale, ignoring aspect ratio)
	// to fill its whole logical presentation area. Used only for cutscene
	// playback's optional "fill the whole window" presentation (see
	// Context.PlayVideo and config.ini's stretch_video); engine deliberately
	// has no idea what that logical size actually is (that's the
	// renderer's own constant), so an ordinary sprite/background layer
	// just leaves this false.
	FillWindow   bool
	Presentation bool
	Visible      bool
	Enabled      bool
	LoadCond     int
	Mode         AnimMode
	LoopStart    int
	Frame        int
	Accumulator  float64

	// ZBuffered records the SZN's ZBuffered property (default true, per
	// agents/scenes/Z-DEPTH.MD section 4): whether this layer participates
	// in dynamic depth comparison against a walking character's own
	// per-pixel depth (see Background.DepthAt) or, when false, keeps a
	// fixed painter-order position regardless of where the character
	// stands (Z-DEPTH.MD section 5, "Only use static painter order for
	// objects explicitly behaving as non-Z-buffered overlays"). See
	// render.DrawScene's sort for where this is actually used.
	ZBuffered bool

	// DirtyAlpha records the SZN's authored DirtyAlpha property. In the
	// original engine this toggles dirty-rectangle/update bookkeeping on the
	// sprite/video object; it is not a chroma-key instruction and does not
	// make AVI pixels transparent. The reconstructed renderer redraws the
	// scene each frame, so retaining the flag is sufficient until dirty-rect
	// optimization itself is reproduced.
	DirtyAlpha bool

	// ColorKeyed requests the original source-color-key compositing path for
	// this scene layer. Controller-owned and SZN AVI layers are drawn this way;
	// full-screen PlayVideo is not.
	ColorKeyed bool

	// Playing gates whether Advance does anything. It defaults to true for
	// every SZN-declared layer (agents/IMPLEMENTATION.md: "Default
	// scene-layer mode should be Loop"), but many layers with fps>0 are not
	// actually ambient animations -- they are dormant "special action" props
	// (e.g. a lever) that a controller's Enter hook must explicitly freeze
	// with Playing=false until a script plays them once via PlayLayerOnce.
	// FPS itself is left untouched so the original speed is preserved for
	// when the layer does play.
	Playing bool

	// TaskDriven is true while a scripted one-shot task (engine.PlayLayerOnce
	// or game's layerRangeTask, underlying PlayLayer/PlayLayerSplit/
	// PlayLayerFrames/PlayVideo/PlayVideoSplit) owns this layer's Advance
	// calls for the duration of its own Update. Those tasks each call
	// Advance(dt) themselves once per tick to detect "reached target frame"
	// precisely; the scene-wide per-tick Advance loop (see
	// game.Session.Update) must skip any layer with TaskDriven set, or the
	// layer gets Advance'd twice in the same tick -- once by the task, once
	// by the generic loop -- and its whole scripted animation (and anything
	// timed against it, e.g. a cutscene AVI's own footstep-SFX cue sheet)
	// plays at roughly double its intended speed. Ordinary ambient/looping
	// layers (no owning task) are never marked TaskDriven and keep being
	// advanced solely by that generic loop, exactly as before.
	TaskDriven bool

	// FrameEvents contains callbacks fired when playback enters an authored
	// frame. This mirrors the original engine's per-animation frame-event
	// mechanism (e.g. FUN_00442bb0 for frame-bound SFX).
	FrameEvents map[int][]func()

	pingDir int
}

// Advance moves the layer's animation clock forward by dt seconds,
// advancing Frame according to FPS and Mode. A frozen layer (Playing ==
// false), fps<=0, or a single-frame source is static and never advances.
func (l *Layer) Advance(dt float64) {
	l.advance(dt, l.FPS)
}

func (l *Layer) AdvanceScripted(dt float64) {
	fps := l.FPS
	if fps <= 0 {
		fps = 10
	}
	l.advance(dt, fps)
}

func (l *Layer) advance(dt float64, fps int) {
	if !l.Playing || fps <= 0 || l.Source == nil {
		return
	}

	frames := l.Source.Frames()
	if frames <= 1 {
		return
	}

	frameDuration := 1.0 / float64(fps)
	l.Accumulator += dt

	for l.Accumulator >= frameDuration {
		l.Accumulator -= frameDuration
		l.step(frames)
	}
}

func (l *Layer) step(frames int) {
	previous := l.Frame
	switch l.Mode {
	case AnimOnce:
		if l.Frame < frames-1 {
			l.Frame++
		}
	case AnimHoldLast:
		if l.Frame < frames-1 {
			l.Frame++
		}
	case AnimOnceReverse:
		if l.Frame > 0 {
			l.Frame--
		}
	case AnimPingPong:
		if l.pingDir == 0 {
			l.pingDir = 1
		}
		l.Frame += l.pingDir
		if l.Frame >= frames-1 {
			l.Frame = frames - 1
			l.pingDir = -1
		} else if l.Frame <= 0 {
			l.Frame = 0
			l.pingDir = 1
		}
	default: // AnimLoop
		loopStart := l.LoopStart
		if loopStart < 0 || loopStart >= frames {
			loopStart = 0
		}
		if l.Frame >= frames-1 {
			l.Frame = loopStart
		} else {
			l.Frame++
		}
	}

	if l.Frame != previous {
		for _, fn := range l.FrameEvents[l.Frame] {
			if fn != nil {
				fn()
			}
		}
	}
}

// AddFrameEvent registers fn to fire whenever animation playback enters frame.
// Direct frame assignment does not fire events; only playback advancement does.
func (l *Layer) AddFrameEvent(frame int, fn func()) {
	if l == nil || fn == nil || frame < 0 {
		return
	}
	if l.FrameEvents == nil {
		l.FrameEvents = make(map[int][]func())
	}
	l.FrameEvents[frame] = append(l.FrameEvents[frame], fn)
}

// Area is an axis-aligned hotspot rectangle.
type Area struct {
	ID         string
	Text       string
	X1, Y1     int
	X2, Y2     int
	CursorType int
	Enabled    bool
}

// Normalized returns the rectangle with min/max ordered.
func (a *Area) Normalized() (minX, minY, maxX, maxY int) {
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

// Contains reports whether (x, y) falls within the normalized rectangle.
func (a *Area) Contains(x, y int) bool {
	minX, minY, maxX, maxY := a.Normalized()
	return x >= minX && x < maxX && y >= minY && y < maxY
}

// ActorState is the actor's current high-level activity.
type ActorState int

const (
	ActorIdle ActorState = iota
	ActorWalking
	ActorTalking
)

// Actor is a character in the scene.
//
// X/Y are the logical authored character position. They are not the sprite's
// literal bottom-center. The original default Rodrigo character uses native
// frame anchor (55,162); the renderer applies that anchor at the current
// zoom when converting X/Y into a sprite destination rectangle.
//
// ZPos is the authored/runtime actor depth value currently available to this
// reconstruction. Do not overwrite it by sampling Background.DepthAt(X,Y):
// the original engine's dynamic depth update uses an additional transformed
// character sample point which is not represented by raw X/Y.
type Actor struct {
	ID              string
	AssetName       string
	X               float64
	Y               float64
	ZPos            int
	Zoom            int // percent; 100 == 1.0
	AuthoredZPos    int
	AuthoredZoom    int
	Direction       int // raw SZN value, preserved as-is; see agents/SZN.md
	Type            int
	AnchorX         int
	AnchorY         int
	ZBuffered       bool
	Source          SpriteSource
	TalkSource      SpriteSource
	TalkSourceError string
	Frame           int
	State           ActorState
	StepLeft        string
	StepRight       string
	StepAlt         string
	Color           ColorCorrection

	// Visible gates whether this actor is drawn (and participates in the
	// scene's depth sort, see SceneDrawOrder) at all. Defaults to true for
	// every instantiated actor; a Controller sets it false to take the live
	// character out of normal scene handling during a scripted transition
	// that already depicts them itself (a lever animation, a cinematic),
	// per agents/techniques/ELEVATOR-TRANSITIONS-AND-ZBUFFER.MD sections
	// 1/2/11 -- without this, the live sprite and the scripted
	// representation both render at once, producing a duplicate/"ghost"
	// actor.
	Visible bool

	// Facing, WalkAccumulator and MouthState are engine-computed
	// rendering/animation state, transient and never serialized (see
	// agents/GAMEPLAY.md "Save/load correctness").
	Facing          FacingDirection
	WalkAccumulator float64
	WalkPhase       int
	MouthState      uint16
	TalkFrame       int
}

// Scene is one fully instantiated SZN scene: raw data plus decoded visual
// sources, but no SDL/renderer resources.
type Scene struct {
	ID         string
	Background *Background
	Characters map[string]*Actor
	Layers     map[string]*Layer
	Areas      map[string]*Area

	// Declared order from the SZN [GENERAL] section, preserved for stable
	// draw order and diagnostics.
	CharacterOrder []string
	LayerOrder     []string
	AreaOrder      []string

	// PresentationAssets are controller-owned videos declared in the SZN's
	// [COMPILER] VideoN list.  The original scene keeps these outside the
	// ordinary layer array and installs one into its dedicated presentation
	// slot (FUN_0044a330) while it plays, so they must use the unkeyed video
	// path rather than the normal DDBLT_KEYSRC scene-layer path.  Keys are
	// stored lower-case for case-insensitive asset lookup.
	PresentationAssets map[string]bool

	// Warnings collects non-fatal instantiation notes (e.g. an unresolved
	// LoadCond), for debug logging rather than silent behavior.
	Warnings []string

	InventoryItems      []*InventoryItem
	InventoryAnims      []*Layer
	InventoryHoverIndex int
	InventoryClickIndex int
	InventoryHidden     bool
}

// GameState is the persistent, serializable state of a play session.
type GameState struct {
	Location         int
	PreviousLocation int
	Scene            string
	PreviousScene    string
	Flags            map[string]int
	Inventory        []int
	OriginalState    []byte
}
