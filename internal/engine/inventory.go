package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/wok/rent-a-hero/internal/assets"
	"github.com/wok/rent-a-hero/internal/text"
)

// InventoryItem is one unique adventure-game item: localized text plus its
// Item%03d.a16 animation. See agents/IMPLEMENTATION.md "Inventory".
type InventoryItem struct {
	ID                   int
	Name                 string
	Description          string
	AlternateDescription string
	HasAlternate         bool
	Sprite               SpriteSource
}

// LoadInventoryItem resolves "Item%03d.a16" through idx and combines it
// with localized text from itemText (parsed from a DefInvent.<lang> file
// via internal/text.ParseInventoryText).
//
// If itemText has no entry for id, Name falls back to a visible
// placeholder ("[TEXT001]") rather than silently returning empty text, per
// agents/TEXT_FORMATS.md.
func LoadInventoryItem(idx *assets.Index, itemText *text.InventoryText, id int) (*InventoryItem, error) {
	filename := fmt.Sprintf("Item%03d.a16", id)

	sprite, err := LoadSpriteSource(idx, filename)
	if err != nil {
		return nil, fmt.Errorf("inventory item %d: %w", id, err)
	}

	t, ok := itemText.Items[id]
	if !ok {
		t = text.InventoryItemText{Name: fmt.Sprintf("[TEXT%03d]", id)}
	}

	return &InventoryItem{
		ID:                   id,
		Name:                 t.Name,
		Description:          t.Description,
		AlternateDescription: t.Alternate,
		HasAlternate:         t.HasAlt,
		Sprite:               sprite,
	}, nil
}

func ResolveInventoryNames(idx *assets.Index, names []string) ([]int, error) {
	byName := map[string][]int{}
	foundText := false

	for _, key := range idx.Keys() {
		base := strings.ToLower(filepath.Base(key))
		if !strings.HasPrefix(base, "definvent.") {
			continue
		}

		path, err := idx.Resolve(key)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		inv, err := text.ParseInventoryText(data)
		if err != nil {
			continue
		}
		foundText = true

		for id, item := range inv.Items {
			name := normalizeInventoryName(item.Name)
			if name == "" {
				continue
			}
			if !slices.Contains(byName[name], id) {
				byName[name] = append(byName[name], id)
			}
		}
	}

	if !foundText {
		return nil, fmt.Errorf("inventory override: no DefInvent language file found")
	}

	for name := range byName {
		sort.Ints(byName[name])
	}
	byName[normalizeInventoryName("Dragon Blaster Deluxe")] = []int{27}
	byName[normalizeInventoryName("DragonBlaster Deluxe")] = []int{27}

	used := map[string]int{}
	ids := make([]int, 0, len(names))
	seen := map[int]bool{}
	for _, raw := range names {
		name := normalizeInventoryName(raw)
		matches := byName[name]
		if len(matches) == 0 {
			return nil, fmt.Errorf("inventory override: unknown item %q", raw)
		}
		index := used[name]
		if index >= len(matches) {
			index = len(matches) - 1
		}
		id := matches[index]
		used[name]++
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}

	return ids, nil
}

func normalizeInventoryName(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(name), " "))
}

// HasItem reports whether id is in the player's inventory.
func (s *GameState) HasItem(id int) bool {
	return slices.Contains(s.Inventory, id)
}

// AddItem adds id to the inventory (in acquisition order) unless already
// present; items are unique, not stacked, per
// agents/IMPLEMENTATION.md "Inventory".
func (s *GameState) AddItem(id int) {
	if !s.HasItem(id) {
		s.Inventory = append(s.Inventory, id)
	}
}

// RemoveItem removes id from the inventory if present.
func (s *GameState) RemoveItem(id int) {
	for i, v := range s.Inventory {
		if v == id {
			s.Inventory = append(s.Inventory[:i], s.Inventory[i+1:]...)
			return
		}
	}
}
