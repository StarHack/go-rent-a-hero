package render

import (
	"strings"

	"github.com/jupiterrider/purego-sdl3/sdl"
)

const (
	subtitleCharSize   = float32(sdl.DebugTextFontCharacterSize)
	subtitleScale      = 2.0
	subtitleMarginY    = 8.0
	subtitleLineGap    = 4.0
	subtitleSideMargin = 16.0
)

// wrapText breaks s into lines of at most maxChars runes, breaking on word
// boundaries where possible (falling back to a hard break for a single word
// longer than maxChars).
func wrapText(s string, maxChars int) []string {
	if maxChars <= 0 {
		return []string{s}
	}

	var lines []string
	var current strings.Builder

	for word := range strings.FieldsSeq(s) {
		for len(word) > maxChars {
			if current.Len() > 0 {
				lines = append(lines, current.String())
				current.Reset()
			}
			lines = append(lines, word[:maxChars])
			word = word[maxChars:]
		}

		candidate := word
		if current.Len() > 0 {
			candidate = current.String() + " " + word
		}

		if len(candidate) > maxChars && current.Len() > 0 {
			lines = append(lines, current.String())
			current.Reset()
			candidate = word
		}

		current.Reset()
		current.WriteString(candidate)
	}

	if current.Len() > 0 {
		lines = append(lines, current.String())
	}

	if len(lines) == 0 {
		lines = []string{""}
	}

	return lines
}

// DrawSubtitle draws a centered, word-wrapped subtitle near the bottom of
// the scene viewport with a translucent backing panel for readability.
//
// Text rendering uses SDL's built-in fixed 8x8 debug font as a placeholder:
// no agents/*.md document describes the original localized text/font asset
// format, so this is intentionally a stand-in, easy to replace once that
// format is identified.
func (r *Renderer) DrawSubtitle(text string) {
	if text == "" {
		return
	}

	maxWidth := float32(LogicalWidth) - 2*subtitleSideMargin
	maxChars := int(maxWidth / (subtitleCharSize * subtitleScale))

	lines := wrapText(text, maxChars)

	lineHeight := subtitleCharSize*subtitleScale + subtitleLineGap
	blockHeight := float32(len(lines))*lineHeight - subtitleLineGap

	top := float32(SceneHeight) - subtitleMarginY - blockHeight

	longest := 0
	for _, l := range lines {
		longest = max(longest, len(l))
	}
	blockWidth := float32(longest) * subtitleCharSize * subtitleScale

	panel := sdl.FRect{
		X: (LogicalWidth-blockWidth)/2 - 6,
		Y: top - 4,
		W: blockWidth + 12,
		H: blockHeight + 8,
	}
	sdl.SetRenderDrawColor(r.renderer, 0, 0, 0, 180)
	sdl.RenderFillRect(r.renderer, &panel)

	sdl.SetRenderScale(r.renderer, subtitleScale, subtitleScale)
	sdl.SetRenderDrawColor(r.renderer, 255, 255, 255, 255)

	for i, line := range lines {
		width := float32(len(line)) * subtitleCharSize * subtitleScale
		x := (LogicalWidth - width) / 2
		y := top + float32(i)*lineHeight
		sdl.RenderDebugText(r.renderer, x/subtitleScale, y/subtitleScale, line)
	}

	sdl.SetRenderScale(r.renderer, 1, 1)
}
