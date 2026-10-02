package render

import (
	"github.com/jupiterrider/purego-sdl3/sdl"
)

// topRightLabelGlyphs is a 5-wide x 7-tall bitmap font, one row per byte
// (bit 4 is the leftmost pixel, bit 0 the rightmost), covering exactly the
// letters needed to spell every top-right on-screen label this package
// draws: "RECORDING" (the "8" key screen-recording toggle) and
// "SCREENSHOT" (the "0" key screenshot indicator, see
// DrawScreenshotIndicator).
var topRightLabelGlyphs = map[byte][7]byte{
	'R': {0x1e, 0x11, 0x11, 0x1e, 0x14, 0x12, 0x11},
	'E': {0x1f, 0x10, 0x10, 0x1e, 0x10, 0x10, 0x1f},
	'C': {0x0f, 0x10, 0x10, 0x10, 0x10, 0x10, 0x0f},
	'O': {0x0e, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0e},
	'D': {0x1e, 0x11, 0x11, 0x11, 0x11, 0x11, 0x1e},
	'I': {0x1f, 0x04, 0x04, 0x04, 0x04, 0x04, 0x1f},
	'N': {0x11, 0x19, 0x19, 0x15, 0x13, 0x13, 0x11},
	'G': {0x0f, 0x10, 0x10, 0x13, 0x11, 0x11, 0x0f},
	'V': {0x11, 0x11, 0x11, 0x11, 0x11, 0x0a, 0x04},
	'S': {0x0e, 0x10, 0x10, 0x0e, 0x01, 0x01, 0x1e},
	'H': {0x11, 0x11, 0x11, 0x1f, 0x11, 0x11, 0x11},
	'T': {0x1f, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04},
}

// recordingText is what DrawRecordingIndicator spells; screenshotText is
// what DrawScreenshotIndicator spells. Every character in either must have
// an entry in topRightLabelGlyphs.
const (
	recordingText  = "RECORDING"
	screenshotText = "SCREENSHOT"
	savedText      = "SAVED"
)

const (
	glyphCols       = 5
	glyphRows       = 7
	glyphPixelSize  = 2 // logical pixels per glyph bit
	glyphColSpacing = 1 // blank glyph-bit-sized columns between characters
	recordingMargin = 8 // logical pixels from the top/right edges
)

// DrawRecordingIndicator draws a red "RECORDING" label near the top-right
// of the logical screen, for cmd/rah's "8" key screen-recording toggle. It
// draws directly over whatever DrawScene already composited, so callers
// should call it last, right before Present.
func (r *Renderer) DrawRecordingIndicator() {
	r.drawTopRightLabel(recordingText, 0)
}

// DrawScreenshotIndicator draws a red "SCREENSHOT" label near the top-right
// of the logical screen, for cmd/rah's "0" key screenshot capture: shown
// for a fixed short duration after each capture (see uiState.screenshotFlash
// in cmd/rah/main.go), then automatically cleared. It draws directly over
// whatever DrawScene already composited, so callers should call it last,
// right before Present. It draws one row below DrawRecordingIndicator's own
// line so the two never overlap if a screenshot happens to be taken while
// a recording is also in progress. Unlike DrawRecordingIndicator (which is
// drawn before recording.mp4's own frame capture, so the recording visibly
// shows it was on), cmd/rah calls this only after recording.png has already
// been saved, so the indicator itself is never baked into the saved image.
func (r *Renderer) DrawScreenshotIndicator() {
	r.drawTopRightLabel(screenshotText, 1)
}

func (r *Renderer) DrawSavedIndicator() {
	r.drawTopRightLabel(savedText, 0)
}

// drawTopRightLabel draws text right-aligned near the top-right of the
// logical screen in topRightLabelGlyphs' bitmap font, in solid red. row
// stacks multiple labels vertically without overlapping (0 is the topmost).
func (r *Renderer) drawTopRightLabel(text string, row int) {
	glyphWidth := (glyphCols + glyphColSpacing) * glyphPixelSize
	rowHeight := glyphRows*glyphPixelSize + recordingMargin
	totalWidth := glyphWidth * len(text)

	startX := LogicalWidth - recordingMargin - totalWidth
	startY := recordingMargin + row*rowHeight

	sdl.SetRenderDrawColor(r.renderer, 255, 0, 0, 255)

	for i := range len(text) {
		glyph, ok := topRightLabelGlyphs[text[i]]
		if !ok {
			continue
		}

		gx := startX + i*glyphWidth

		for glyphRow := range glyphRows {
			bits := glyph[glyphRow]
			for col := range glyphCols {
				if (bits>>uint(glyphCols-1-col))&1 == 0 {
					continue
				}

				rect := sdl.FRect{
					X: float32(gx + col*glyphPixelSize),
					Y: float32(startY + glyphRow*glyphPixelSize),
					W: float32(glyphPixelSize),
					H: float32(glyphPixelSize),
				}
				sdl.RenderFillRect(r.renderer, &rect)
			}
		}
	}
}
