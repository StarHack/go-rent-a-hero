package engine

import (
	"testing"

	"github.com/wok/rent-a-hero/internal/formats/zbf"
)

func TestSceneDrawOrderNilSceneReturnsNil(t *testing.T) {
	if got := SceneDrawOrder(nil); got != nil {
		t.Errorf("SceneDrawOrder(nil) = %v, want nil", got)
	}
}

func TestSceneDrawOrderSkipsHiddenAndMissingLayers(t *testing.T) {
	scene := &Scene{
		Layers: map[string]*Layer{
			"visible": {ID: "visible", Visible: true, ZBuffered: true},
			"hidden":  {ID: "hidden", Visible: false, ZBuffered: true},
		},
		LayerOrder: []string{"visible", "hidden", "missing"},
		Characters: map[string]*Actor{},
	}

	items := SceneDrawOrder(scene)
	if len(items) != 1 || items[0].Layer == nil || items[0].Layer.ID != "visible" {
		t.Fatalf("items = %+v, want exactly the visible layer", items)
	}
}

// TestSceneDrawOrderSkipsInvisibleActor locks in
// agents/techniques/ELEVATOR-TRANSITIONS-AND-ZBUFFER.MD sections 1/2/11:
// an actor with Visible == false must be excluded from both drawing and
// depth comparison entirely, e.g. while a scripted lever/cinematic sequence
// already depicts the character itself -- otherwise the live sprite and
// that scripted depiction both render, producing a duplicate/"ghost" actor.
func TestSceneDrawOrderSkipsInvisibleActor(t *testing.T) {
	layer := &Layer{ID: "layer", Visible: true, ZBuffered: true, Z: 10}
	hidden := &Actor{ID: "Rodrigo", X: 1, Y: 1, ZPos: 999} // Visible left false

	scene := &Scene{
		Layers:         map[string]*Layer{"layer": layer},
		LayerOrder:     []string{"layer"},
		Characters:     map[string]*Actor{"Rodrigo": hidden},
		CharacterOrder: []string{"Rodrigo"},
	}

	items := SceneDrawOrder(scene)
	if len(items) != 1 || items[0].Layer == nil {
		t.Fatalf("items = %+v, want only the layer (the invisible actor excluded entirely)", items)
	}
}

// TestSceneDrawOrderCharacterUsesDynamicDepth locks in Z-DEPTH.MD sections
// 6/10: a character's draw-order position is sampled per frame from the
// background's own zBuf raster at its current feet position, not fixed for
// the whole scene at its SZN-declared ZPos.
func TestSceneDrawOrderCharacterUsesDynamicDepth(t *testing.T) {
	// A 4x4 depth raster: value 200 at (1,1) (the actor will stand there),
	// everywhere else 0.
	depth := &zbf.Raster{Width: 4, Height: 4, Pixels: make([]byte, 16)}
	depth.Pixels[1*4+1] = 200

	foreground := &Layer{ID: "foreground", Visible: true, ZBuffered: true, Z: 100}
	background := &Layer{ID: "background", Visible: true, ZBuffered: true, Z: 10}

	actor := &Actor{ID: "Rodrigo", X: 1, Y: 1, ZPos: 50, Visible: true}

	scene := &Scene{
		Background: &Background{Depth: depth},
		Layers: map[string]*Layer{
			"foreground": foreground,
			"background": background,
		},
		LayerOrder:     []string{"background", "foreground"},
		Characters:     map[string]*Actor{"Rodrigo": actor},
		CharacterOrder: []string{"Rodrigo"},
	}

	items := SceneDrawOrder(scene)
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3", len(items))
	}

	// Depth at (1,1) is 200, higher than the foreground layer's Z=100, so
	// the actor -- despite its static ZPos=50, which would place it behind
	// both layers -- should now draw *last* (in front of everything).
	last := items[len(items)-1]
	if last.Actor == nil || last.Actor.ID != "Rodrigo" {
		t.Errorf("last item = %+v, want the actor drawn last (in front)", last)
	}

	// Move the actor to a pixel with depth 0 (behind both layers) and
	// confirm the order flips accordingly.
	actor.X, actor.Y = 2, 2
	items = SceneDrawOrder(scene)
	first := items[0]
	if first.Actor == nil || first.Actor.ID != "Rodrigo" {
		t.Errorf("first item = %+v, want the actor drawn first (behind) once its depth drops", first)
	}
}

// TestSceneDrawOrderFallsBackToStaticZPosWithoutDepthRaster covers a scene
// with no zBuf at all (Background.Depth == nil, or the actor's position
// falls outside the raster): the actor must fall back to its SZN-declared
// ZPos rather than silently defaulting to 0, which would put it behind
// everything regardless of its authored position.
func TestSceneDrawOrderFallsBackToStaticZPosWithoutDepthRaster(t *testing.T) {
	layer := &Layer{ID: "layer", Visible: true, ZBuffered: true, Z: 10}
	actor := &Actor{ID: "Rodrigo", X: 5, Y: 5, ZPos: 20, Visible: true}

	scene := &Scene{
		Background:     &Background{}, // no Depth raster
		Layers:         map[string]*Layer{"layer": layer},
		LayerOrder:     []string{"layer"},
		Characters:     map[string]*Actor{"Rodrigo": actor},
		CharacterOrder: []string{"Rodrigo"},
	}

	items := SceneDrawOrder(scene)
	if len(items) != 2 || items[0].Layer == nil || items[1].Actor == nil {
		t.Fatalf("items = %+v, want [layer(Z=10), actor(ZPos=20)] in that order", items)
	}
}

// TestSceneDrawOrderNonZBufferedLayerIgnoresDynamicDepth locks in
// Z-DEPTH.MD section 5: a layer with ZBuffered=false must keep a fixed
// painter-order position relative to the character regardless of the
// character's per-pixel dynamic depth.
func TestSceneDrawOrderNonZBufferedLayerIgnoresDynamicDepth(t *testing.T) {
	depth := &zbf.Raster{Width: 2, Height: 2, Pixels: []byte{0, 0, 0, 255}}

	overlay := &Layer{ID: "overlay", Visible: true, ZBuffered: false, Z: 30}
	actor := &Actor{ID: "Rodrigo", X: 1, Y: 1, ZPos: 10, Visible: true}

	scene := &Scene{
		Background:     &Background{Depth: depth},
		Layers:         map[string]*Layer{"overlay": overlay},
		LayerOrder:     []string{"overlay"},
		Characters:     map[string]*Actor{"Rodrigo": actor},
		CharacterOrder: []string{"Rodrigo"},
	}

	// Dynamic depth at (1,1) is 255, far above the overlay's Z=30 -- if the
	// dynamic depth were used, the actor would now draw in front. But since
	// the overlay is not Z-buffered, it must stay in front regardless.
	items := SceneDrawOrder(scene)
	if len(items) != 2 || items[0].Actor == nil || items[1].Layer == nil {
		t.Fatalf("items = %+v, want the actor still behind the non-Z-buffered overlay", items)
	}
}

// TestSceneDrawOrderNonZBufferedLayerAlwaysRendersInFrontOfActor locks in
// the fix from agents/scenes/7A-46.MD: a non-Z-buffered layer always draws
// in front of every actor, even when its own authored Z is numerically far
// below the actor's ZPos -- exactly Scene 7A's GliderLayer (Z=1) against
// Rodrigo (ZPos=106), which an earlier reconstruction got backwards by
// comparing those two numbers directly.
func TestSceneDrawOrderNonZBufferedLayerAlwaysRendersInFrontOfActor(t *testing.T) {
	glider := &Layer{ID: "GliderLayer", Visible: true, ZBuffered: false, Z: 1}
	actor := &Actor{ID: "Rodrigo", X: 0, Y: 0, ZPos: 106, Visible: true}

	scene := &Scene{
		Layers:         map[string]*Layer{"GliderLayer": glider},
		LayerOrder:     []string{"GliderLayer"},
		Characters:     map[string]*Actor{"Rodrigo": actor},
		CharacterOrder: []string{"Rodrigo"},
	}

	items := SceneDrawOrder(scene)
	if len(items) != 2 || items[0].Actor == nil || items[1].Layer == nil {
		t.Fatalf("items = %+v, want the actor drawn first (behind), the low-Z non-Z-buffered layer drawn last (in front)", items)
	}
}
