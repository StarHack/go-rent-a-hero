package render

import (
	"fmt"

	"github.com/jupiterrider/purego-sdl3/sdl"
	"github.com/wok/rent-a-hero/internal/engine"
)

// DrawScene renders one frame of scene: background, then layers and actors
// composited back-to-front per engine.SceneDrawOrder (see
// agents/scenes/Z-DEPTH.MD), then hotspot debug rectangles if enabled. It
// does not call Present; callers control frame pacing.
func (r *Renderer) DrawScene(scene *engine.Scene) error {
	r.Clear(0, 0, 0, 255)

	if scene == nil {
		return nil
	}

	for _, item := range engine.SceneDrawOrder(scene) {
		if item.Layer == nil || !item.Layer.Presentation {
			continue
		}
		l := item.Layer
		// Presentation video is the original unkeyed presentation/background
		// path, not the normal scene-layer DDBLT_KEYSRC path.  Never inherit
		// a scene layer's source-key semantics here merely because the source
		// happens to be an AVI.
		if l.FillWindow {
			return r.drawStretched(l.Source, l.Frame, 0, 0, LogicalWidth, LogicalHeight)
		}
		return r.drawTopLeft(l.Source, l.Frame, l.X, l.Y, l.Zoom)
	}

	if scene.Background != nil && scene.Background.Source != nil {
		if err := r.drawTopLeft(scene.Background.Source, 0, 0, 0, 100); err != nil {
			return err
		}
	}

	for _, item := range engine.SceneDrawOrder(scene) {
		switch {
		case item.Layer != nil:
			l := item.Layer
			var err error
			if l.FillWindow {
				if l.ColorKeyed {
					err = r.drawStretchedKeyed(l.Source, l.Frame, 0, 0, LogicalWidth, LogicalHeight)
				} else {
					err = r.drawStretched(l.Source, l.Frame, 0, 0, LogicalWidth, LogicalHeight)
				}
			} else if l.ZBuffered && item.Z != 255 && scene.Background != nil && scene.Background.Depth != nil {
				err = r.drawTopLeftMaskedKeyed(l.Source, l.Frame, l.X, l.Y, l.Zoom, scene.Background, item.Z, l.ColorKeyed)
			} else if l.ColorKeyed {
				err = r.drawTopLeftKeyed(l.Source, l.Frame, l.X, l.Y, l.Zoom)
			} else {
				err = r.drawTopLeft(l.Source, l.Frame, l.X, l.Y, l.Zoom)
			}
			if err != nil {
				return err
			}
		case item.Actor != nil:
			a := item.Actor
			_, colorKeyed := a.Source.(engine.ColorKeySource)
			zoom := a.Zoom
			if zoom == 0 {
				zoom = 100
			}
			red, green, blue := actorColorCorrection(a)

			var err error
			if a.ZBuffered && scene.Background != nil && scene.Background.Depth != nil {
				err = r.drawFeetAnchoredMaskedCorrectedKeyed(a.Source, a.Frame, a.X, a.Y, zoom, a.AnchorX, a.AnchorY, scene.Background, item.Z, red, green, blue, colorKeyed)
			} else {
				err = r.drawFeetAnchoredCorrectedKeyed(a.Source, a.Frame, a.X, a.Y, zoom, a.AnchorX, a.AnchorY, red, green, blue, colorKeyed)
			}
			if err != nil {
				return err
			}

			if a.State == engine.ActorTalking && a.TalkSource != nil {
				_, talkColorKeyed := a.TalkSource.(engine.ColorKeySource)
				if a.ZBuffered && scene.Background != nil && scene.Background.Depth != nil {
					err = r.drawFeetAnchoredMaskedCorrectedKeyed(a.TalkSource, a.TalkFrame, a.X, a.Y, zoom, a.AnchorX, a.AnchorY, scene.Background, item.Z, red, green, blue, talkColorKeyed)
				} else {
					err = r.drawFeetAnchoredCorrectedKeyed(a.TalkSource, a.TalkFrame, a.X, a.Y, zoom, a.AnchorX, a.AnchorY, red, green, blue, talkColorKeyed)
				}
				if err != nil {
					return err
				}
			}
		}
	}

	if !scene.InventoryHidden {
		r.drawInventoryPanel()
		if err := r.drawSceneInventory(scene); err != nil {
			return err
		}
	}

	if r.debugAreas {
		r.drawDebugAreas(scene)
	}

	r.updateSceneCursor(scene)
	return nil
}

func actorColorCorrection(a *engine.Actor) (int, int, int) {
	red, green, blue := a.Color.R, a.Color.G, a.Color.B
	x, y := int(a.X), int(a.Y)
	for _, zone := range a.Color.Zones {
		if zone.Range <= 0 {
			continue
		}
		dx := x - zone.X
		if dx < 0 {
			dx = -dx
		}
		dy := y - zone.Y
		if dy < 0 {
			dy = -dy
		}
		if dx > zone.Range || dy > zone.Range {
			continue
		}
		factor := ((zone.Range-dy)*100)/zone.Range - 100 + ((zone.Range-dx)*100)/zone.Range
		if factor < 0 {
			factor = 0
		}
		red += zone.R * factor / 100
		green += zone.G * factor / 100
		blue += zone.B * factor / 100
	}
	return red, green, blue
}

// DrawDebugPath draws small markers at each remaining path waypoint and
// connecting lines, letting a developer visually confirm pathfinding
// output against the background art. Intended to be called after DrawScene
// and before Present.
func (r *Renderer) DrawDebugPath(waypoints []engine.Point) {
	if len(waypoints) == 0 {
		return
	}

	sdl.SetRenderDrawColor(r.renderer, 255, 255, 0, 255)

	for i, p := range waypoints {
		const half = 2.0
		rect := sdl.FRect{X: float32(p.X) - half, Y: float32(p.Y) - half, W: half * 2, H: half * 2}
		sdl.RenderFillRect(r.renderer, &rect)

		if i > 0 {
			prev := waypoints[i-1]
			sdl.RenderLine(r.renderer, float32(prev.X), float32(prev.Y), float32(p.X), float32(p.Y))
		}
	}
}

// debugAreaLabelScale is the SDL debug-font scale used by DrawDebugArea:
// 2x the built-in 8x8 glyphs, matching DrawSubtitle's size so the
// top-left S/L label stays readable over scene art.
const (
	debugAreaLabelScale  = 2.0
	debugAreaLabelMargin = 8.0
)

// DrawDebugArea draws "S: <sectionID> | L: <locationID>" in yellow at the
// top-left of the logical screen when debug_area is enabled (see
// SetDebugArea). Callers should invoke it after DrawScene (and any other
// overlays that should sit underneath) and before Present.
func (r *Renderer) DrawDebugArea(sectionID string, locationID int) {
	if !r.debugArea {
		return
	}

	label := fmt.Sprintf("S: %s | L: %d", sectionID, locationID)

	sdl.SetRenderScale(r.renderer, debugAreaLabelScale, debugAreaLabelScale)
	sdl.SetRenderDrawColor(r.renderer, 255, 255, 0, 255)
	renderDebugTextExtended(r.renderer, debugAreaLabelMargin/debugAreaLabelScale, debugAreaLabelMargin/debugAreaLabelScale, label)
	sdl.SetRenderScale(r.renderer, 1, 1)
}

// drawDebugAreas outlines every enabled hotspot rectangle, letting a
// developer visually confirm areas align with the background art.
func (r *Renderer) drawDebugAreas(scene *engine.Scene) {
	var windowX, windowY float32
	sdl.GetMouseState(&windowX, &windowY)
	mouseX, mouseY := r.WindowToLogical(windowX, windowY)

	hovered := ""
	for _, name := range scene.AreaOrder {
		area := scene.Areas[name]
		if area != nil && area.Enabled && area.Contains(int(mouseX), int(mouseY)) {
			hovered = name
			break
		}
	}

	if hovered != "" {
		area := scene.Areas[hovered]
		minX, minY, maxX, maxY := area.Normalized()
		rect := sdl.FRect{
			X: float32(minX),
			Y: float32(minY),
			W: float32(maxX - minX),
			H: float32(maxY - minY),
		}
		sdl.SetRenderDrawBlendMode(r.renderer, sdl.BlendModeBlend)
		sdl.SetRenderDrawColor(r.renderer, 0, 255, 0, 72)
		sdl.RenderFillRect(r.renderer, &rect)
		sdl.SetRenderDrawBlendMode(r.renderer, sdl.BlendModeNone)
	}

	sdl.SetRenderDrawColor(r.renderer, 0, 255, 0, 255)
	for _, name := range scene.AreaOrder {
		area := scene.Areas[name]
		if area == nil || !area.Enabled {
			continue
		}
		minX, minY, maxX, maxY := area.Normalized()
		rect := sdl.FRect{
			X: float32(minX),
			Y: float32(minY),
			W: float32(maxX - minX),
			H: float32(maxY - minY),
		}
		sdl.RenderRect(r.renderer, &rect)
	}
}

const cursorTextDelayNS = uint64(250 * 1000 * 1000)

func (r *Renderer) updateSceneCursor(scene *engine.Scene) {
	var windowX, windowY float32
	sdl.GetMouseState(&windowX, &windowY)
	x, y := r.WindowToLogical(windowX, windowY)

	var hovered *engine.Area
	if scene != nil && y >= 0 && y < SceneHeight {
		for _, id := range scene.AreaOrder {
			area := scene.Areas[id]
			if area != nil && area.Enabled && area.Contains(int(x), int(y)) {
				hovered = area
				break
			}
		}
	}

	if hovered == nil {
		r.hoverArea = ""
		r.hoverSince = 0
		r.setCursorType(0)
		return
	}

	cursorType := hovered.CursorType
	if cursorType == 0 {
		cursorType = 10
	}
	r.setCursorType(cursorType)
	if hovered.ID != r.hoverArea {
		r.hoverArea = hovered.ID
		r.hoverSince = sdl.GetTicksNS()
		r.hoverX = x
		if y < 330 {
			r.hoverY = y + 25
		} else {
			r.hoverY = y - 20
		}
		return
	}
	if r.hoverSince == 0 || sdl.GetTicksNS()-r.hoverSince < cursorTextDelayNS {
		return
	}

	label := hovered.Text
	if label == "" {
		label = hovered.ID
	}
	if label == "" {
		return
	}
	r.drawCursorText(label, r.hoverX, r.hoverY)
}

func (r *Renderer) drawCursorText(label string, x, y float32) {
	const scale = float32(2)
	width := float32(len([]rune(label))) * 8 * scale
	drawX := x - width/2
	if drawX < 0 {
		drawX = x
	} else if drawX+width > LogicalWidth {
		drawX = x - width
	}

	sdl.SetRenderScale(r.renderer, scale, scale)
	x0 := drawX / scale
	y0 := y / scale
	sdl.SetRenderDrawColor(r.renderer, 0, 0, 0, 255)
	renderDebugTextExtended(r.renderer, x0-0.5, y0-0.5, label)
	renderDebugTextExtended(r.renderer, x0-0.5, y0+0.5, label)
	renderDebugTextExtended(r.renderer, x0+0.5, y0-0.5, label)
	renderDebugTextExtended(r.renderer, x0+0.5, y0+0.5, label)
	sdl.SetRenderDrawColor(r.renderer, 255, 255, 255, 255)
	renderDebugTextExtended(r.renderer, x0, y0, label)
	sdl.SetRenderScale(r.renderer, 1, 1)
}

var debugTextExtendedGlyphs = map[rune][8]byte{
	'ä': {0x24, 0x00, 0x3c, 0x02, 0x3e, 0x42, 0x3e, 0x00},
	'ö': {0x24, 0x00, 0x3c, 0x42, 0x42, 0x42, 0x3c, 0x00},
	'ü': {0x24, 0x00, 0x42, 0x42, 0x42, 0x46, 0x3a, 0x00},
	'Ä': {0x24, 0x00, 0x18, 0x24, 0x42, 0x7e, 0x42, 0x42},
	'Ö': {0x24, 0x00, 0x3c, 0x42, 0x42, 0x42, 0x3c, 0x00},
	'Ü': {0x24, 0x00, 0x42, 0x42, 0x42, 0x42, 0x3c, 0x00},
	'ß': {0x3c, 0x42, 0x42, 0x5c, 0x42, 0x42, 0x5c, 0x40},
}

func renderDebugTextExtended(renderer *sdl.Renderer, x, y float32, text string) {
	start := 0
	xPos := x
	for i, ch := range text {
		if _, ok := debugTextExtendedGlyphs[ch]; !ok && ch >= 0x20 && ch <= 0x7e {
			continue
		}
		if start < i {
			part := text[start:i]
			sdl.RenderDebugText(renderer, xPos, y, part)
			xPos += float32(len(part) * 8)
		}
		if glyph, ok := debugTextExtendedGlyphs[ch]; ok {
			renderDebugGlyph(renderer, xPos, y, glyph)
		} else {
			sdl.RenderDebugText(renderer, xPos, y, "?")
		}
		xPos += 8
		start = i + len(string(ch))
	}
	if start < len(text) {
		sdl.RenderDebugText(renderer, xPos, y, text[start:])
	}
}

func renderDebugGlyph(renderer *sdl.Renderer, x, y float32, glyph [8]byte) {
	for row, bits := range glyph {
		for col := 0; col < 8; col++ {
			if bits&(0x80>>col) == 0 {
				continue
			}
			rect := sdl.FRect{X: x + float32(col), Y: y + float32(row), W: 1, H: 1}
			sdl.RenderFillRect(renderer, &rect)
		}
	}
}
