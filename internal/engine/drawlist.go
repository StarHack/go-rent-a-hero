package engine

import (
	"math"
	"sort"
)

// DrawItem is one entry in a scene's depth-sorted draw list: either a Layer
// or an Actor, in the order it should be drawn (back to front).
type DrawItem struct {
	Layer *Layer
	Actor *Actor

	// Z is the depth value actually used to place this item in the sort --
	// a character's dynamic, per-pixel-sampled depth (see Background.
	// DepthAt), or a layer's own static Z. The renderer reuses this (rather
	// than resampling) for per-pixel actor/background occlusion, per
	// agents/techniques/SMOOTH-TRANSITIONS-ZBUFFER-GLIDER.MD sections
	// 10-13.
	Z int
}

// drawEntry is a DrawItem plus the bookkeeping SceneDrawOrder's sort needs,
// kept unexported since callers only need the final ordered []DrawItem.
type drawEntry struct {
	DrawItem
	z        int
	orderTie int
}

// SceneDrawOrder returns the scene's visible layers and characters in the
// original back-to-front bucket order. Layers use their authored static Z.
// Characters use the non-zero wBuf value under their logical path position;
// the original engine updates this value whenever a character is positioned
// or walks. Actor.ZPos is the authored reference depth used as a fallback, not
// the character's permanent painter-order value.
func SceneDrawOrder(scene *Scene) []DrawItem {
	if scene == nil {
		return nil
	}

	entries := make([]drawEntry, 0, len(scene.LayerOrder)+len(scene.CharacterOrder))

	for i, name := range scene.LayerOrder {
		layer := scene.Layers[name]
		if layer == nil || !layer.Visible {
			continue
		}
		entries = append(entries, drawEntry{DrawItem: DrawItem{Layer: layer}, z: layer.Z, orderTie: i})
	}

	for i, name := range scene.CharacterOrder {
		actor := scene.Characters[name]
		if actor == nil || !actor.Visible {
			continue
		}

		z := actor.ZPos
		if scene.Background != nil && scene.Background.Walk != nil {
			x := int(math.Round(actor.X))
			y := int(math.Round(actor.Y))
			walk := scene.Background.Walk
			if walk.InBounds(x, y) {
				if sampled := int(walk.At(x, y)); sampled != 0 {
					z = sampled
				}
			}
		}

		entries = append(entries, drawEntry{
			DrawItem: DrawItem{Actor: actor},
			z:        z,
			orderTie: len(scene.LayerOrder) + i,
		})
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].z != entries[j].z {
			return entries[i].z > entries[j].z
		}
		return entries[i].orderTie < entries[j].orderTie
	})

	items := make([]DrawItem, len(entries))
	for i, e := range entries {
		item := e.DrawItem
		item.Z = e.z
		items[i] = item
	}
	return items
}
