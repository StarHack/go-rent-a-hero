// Package game orchestrates gameplay on top of internal/engine: scene
// transitions, per-location puzzle controllers, and the Context API that
// controllers use to script actions. See agents/GAMEPLAY.md.
package game

import "github.com/wok/rent-a-hero/internal/engine"

// Controller implements one original location's puzzle logic. See
// agents/IMPLEMENTATION.md "Location gameplay controller" and
// agents/GAMEPLAY.md "Controller API".
//
// Any method may return nil to mean "no task for this event."
type Controller interface {
	// Enter runs when scene becomes current, having arrived from from
	// (empty on the very first scene load). Controllers use it to
	// reconstruct scene visuals from persistent State, per
	// agents/GAMEPLAY.md "Scene entry state reconstruction".
	Enter(ctx *Context, scene string, from string) engine.Task

	// Exit runs when leaving scene for to. Exit hooks must complete
	// synchronously (within a single Update call) -- use Enter's returned
	// task for any animated transition-in sequence instead.
	Exit(ctx *Context, scene string, to string) engine.Task

	// Click handles a hotspot click by canonical area ID.
	Click(ctx *Context, area string) engine.Task

	// UseItem handles using inventory item id on hotspot area.
	UseItem(ctx *Context, item int, area string) engine.Task

	// SelectItem handles the player directly activating inventory item id
	// (clicking it in the inventory overlay, not using it on a hotspot).
	// Most locations have no such items and return nil, in which case
	// Session.SelectItem falls back to arming id for a "use item on
	// hotspot" click instead (the common case, e.g. agents/GAMEPLAY.md
	// "Item use"). A location with a specially-enabled item that is
	// activated directly -- e.g. Location 35's Dragon Blaster Deluxe, per
	// agents/scenes/02_DRAGON_BLASTER_DELUXE.MD's "event type 3" -- returns
	// a task instead, which runs immediately in place of that fallback.
	SelectItem(ctx *Context, item int) engine.Task

	// LoadConditionMask returns the scene-condition bitmask to instantiate
	// scene with (see engine.InstantiateScene's loadMask and
	// agents/scenes/46.MD section 16, "Critical implementation note --
	// LoadCond=1"): a layer whose SZN LoadCond is nonzero loads exactly
	// when (LoadCond & mask) == LoadCond, mirroring the original engine's
	// own generic scene loader receiving a numeric condition mask from each
	// location's constructor -- LoadCond is never resolved by a per-layer
	// "controller ID" the way earlier reconstructions assumed. Most
	// locations have no such layers and return 0. Called before the new
	// scene is instantiated, so ctx's session still has the *previous*
	// scene attached -- implementations must only use state-level Context
	// methods (GetFlag and friends), never anything that touches the
	// current scene/layers/areas.
	LoadConditionMask(ctx *Context, scene string) int
}
