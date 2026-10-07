package render

import (
	"strings"

	"github.com/jupiterrider/purego-sdl3/sdl"
)

func (r *Renderer) DrawMenuText(text string, x, y float32) {
	sdl.SetRenderDrawColor(r.renderer, 255, 255, 255, 255)
	for i, line := range strings.Split(text, "\n") {
		renderDebugTextExtended(r.renderer, x, y+float32(i*12), line)
	}
}

func (r *Renderer) DrawMenuSelection(x1, y1, x2, y2 int) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	sdl.SetRenderDrawColor(r.renderer, 255, 255, 255, 255)
	rect := sdl.FRect{X: float32(x1), Y: float32(y1), W: float32(x2 - x1), H: float32(y2 - y1)}
	sdl.RenderRect(r.renderer, &rect)
}
