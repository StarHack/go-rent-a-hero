package render

import (
	"github.com/jupiterrider/purego-sdl3/sdl"
	"github.com/wok/rent-a-hero/internal/engine"
)

// InventorySlot pairs an inventory item with a Layer used purely to drive
// its Item%03d.a16 icon animation clock (X/Y/Z on the Layer are unused).
type InventorySlot struct {
	Item *engine.InventoryItem
	Anim *engine.Layer
}

// NewInventorySlot builds a slot with a looping icon animation.
func NewInventorySlot(item *engine.InventoryItem) InventorySlot {
	return InventorySlot{
		Item: item,
		Anim: &engine.Layer{Source: item.Sprite, FPS: 10, Mode: engine.AnimLoop, Playing: true},
	}
}

const (
	inventorySlotSize    = 56
	inventorySlotPadding = 10
)

func inventorySlotRect(index int) sdl.FRect {
	x := float32(inventorySlotPadding + index*(inventorySlotSize+inventorySlotPadding))
	y := float32(SceneHeight) + float32(LogicalHeight-SceneHeight-inventorySlotSize)/2
	return sdl.FRect{X: x, Y: y, W: inventorySlotSize, H: inventorySlotSize}
}

// HitTestInventory returns the slot index under logical point (x, y), or -1
// if none.
func HitTestInventory(slots []InventorySlot, x, y float32) int {
	for i := range slots {
		r := inventorySlotRect(i)
		if x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H {
			return i
		}
	}
	return -1
}

func (r *Renderer) drawInventoryPanel() {
	panel := sdl.FRect{X: 0, Y: SceneHeight, W: LogicalWidth, H: LogicalHeight - SceneHeight}
	sdl.SetRenderDrawColor(r.renderer, 15, 15, 25, 235)
	sdl.RenderFillRect(r.renderer, &panel)
}

func (r *Renderer) drawSceneInventory(scene *engine.Scene) error {
	if scene == nil {
		return nil
	}
	var windowX, windowY float32
	buttons := sdl.GetMouseState(&windowX, &windowY)
	x, y := r.WindowToLogical(windowX, windowY)
	hover := -1
	for i := range scene.InventoryItems {
		rect := inventorySlotRect(i)
		if x >= rect.X && x < rect.X+rect.W && y >= rect.Y && y < rect.Y+rect.H {
			hover = i
			break
		}
	}
	scene.InventoryHoverIndex = hover
	leftDown := uint32(buttons)&1 != 0
	if leftDown && !r.inventoryMouseDown && hover >= 0 {
		scene.InventoryClickIndex = hover
	}
	r.inventoryMouseDown = leftDown

	for i, item := range scene.InventoryItems {
		if item == nil || item.Sprite == nil {
			continue
		}
		frame := 0
		if i < len(scene.InventoryAnims) && scene.InventoryAnims[i] != nil {
			frame = scene.InventoryAnims[i].Frame
		}
		tex, err := r.getTexture(item.Sprite, frame)
		if err != nil {
			return err
		}
		dst := inventorySlotRect(i)
		sdl.RenderTexture(r.renderer, tex, nil, &dst)
		if i == hover {
			sdl.SetRenderDrawColor(r.renderer, 255, 220, 0, 255)
			sdl.RenderRect(r.renderer, &dst)
		}
	}
	if hover >= 0 && hover < len(scene.InventoryItems) && scene.InventoryItems[hover] != nil {
		sdl.SetRenderDrawColor(r.renderer, 255, 255, 255, 255)
		renderDebugTextExtended(r.renderer, inventorySlotPadding, float32(SceneHeight)+4, scene.InventoryItems[hover].Name)
	}
	return nil
}

// DrawInventory renders the inventory overlay: a dimmed panel across the
// reserved bottom UI strip, one icon per collected item in acquisition
// order, and the hovered item's name (see agents/GAMEPLAY.md "Inventory
// UI": "hover -> item name").
func (r *Renderer) DrawInventory(slots []InventorySlot, hoverIndex int) error {
	r.drawInventoryPanel()

	for i, slot := range slots {
		dst := inventorySlotRect(i)

		tex, err := r.getTexture(slot.Item.Sprite, slot.Anim.Frame)
		if err != nil {
			return err
		}

		sdl.RenderTexture(r.renderer, tex, nil, &dst)

		if i == hoverIndex {
			sdl.SetRenderDrawColor(r.renderer, 255, 220, 0, 255)
			sdl.RenderRect(r.renderer, &dst)
		}
	}

	if hoverIndex >= 0 && hoverIndex < len(slots) {
		name := slots[hoverIndex].Item.Name
		sdl.SetRenderDrawColor(r.renderer, 255, 255, 255, 255)
		renderDebugTextExtended(r.renderer, inventorySlotPadding, float32(SceneHeight)+4, name)
	}

	return nil
}
