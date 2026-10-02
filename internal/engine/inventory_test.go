package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wok/rent-a-hero/internal/assets"
	"github.com/wok/rent-a-hero/internal/text"
)

func commonIndex(t *testing.T) *assets.Index {
	t.Helper()

	idx, err := assets.NewIndex(filepath.Join(repoRoot(t), "data", "installation", "Common"))
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}

	return idx
}

func loadDefInventEnglish(t *testing.T) *text.InventoryText {
	t.Helper()

	path := filepath.Join(repoRoot(t), "data", "installation", "Common", "DefInvent.eng")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read DefInvent.eng: %v", err)
	}

	inv, err := text.ParseInventoryText(data)
	if err != nil {
		t.Fatalf("ParseInventoryText: %v", err)
	}

	return inv
}

func TestLoadInventoryItem1DisplaysLocalizedTextAndAnimation(t *testing.T) {
	idx := commonIndex(t)
	itemText := loadDefInventEnglish(t)

	item, err := LoadInventoryItem(idx, itemText, 1)
	if err != nil {
		t.Fatalf("LoadInventoryItem: %v", err)
	}

	if item.Name != "The stranger's cloak" {
		t.Errorf("Name = %q, want %q", item.Name, "The stranger's cloak")
	}

	if item.Description == "" {
		t.Error("Description is empty")
	}

	if item.Sprite == nil {
		t.Fatal("Sprite is nil")
	}

	// Item001.a16 is confirmed (Milestone 1) to have 20 60x60 frames; this
	// proves the "and animation" half of the acceptance criterion.
	if item.Sprite.Frames() != 20 {
		t.Errorf("Sprite.Frames() = %d, want 20", item.Sprite.Frames())
	}
	if item.Sprite.Width() != 60 || item.Sprite.Height() != 60 {
		t.Errorf("Sprite dims = %dx%d, want 60x60", item.Sprite.Width(), item.Sprite.Height())
	}

	if _, err := item.Sprite.RGBA(0); err != nil {
		t.Errorf("decode frame 0: %v", err)
	}
}

func TestLoadInventoryItemGermanHasAlternate(t *testing.T) {
	idx := commonIndex(t)

	data, err := os.ReadFile(filepath.Join(repoRoot(t), "data", "installation", "Common", "DefInvent.ger"))
	if err != nil {
		t.Fatalf("read DefInvent.ger: %v", err)
	}

	itemText, err := text.ParseInventoryText(data)
	if err != nil {
		t.Fatalf("ParseInventoryText: %v", err)
	}

	item, err := LoadInventoryItem(idx, itemText, 1)
	if err != nil {
		t.Fatalf("LoadInventoryItem: %v", err)
	}

	if !item.HasAlternate || item.AlternateDescription == "" {
		t.Error("expected item 1 to have a populated alternate description in German")
	}
}

func TestLoadInventoryItemMissingTextFallsBackToPlaceholder(t *testing.T) {
	idx := commonIndex(t)
	itemText := loadDefInventEnglish(t)

	// Item027.a16 exists as a sprite, but DefInvent.eng's TEXT keys stop at
	// 026: a real example of an item with art but no confirmed English
	// name, exercising the documented placeholder fallback.
	item, err := LoadInventoryItem(idx, itemText, 27)
	if err != nil {
		t.Fatalf("LoadInventoryItem: %v", err)
	}

	if item.Name != "[TEXT027]" {
		t.Errorf("Name = %q, want visible placeholder [TEXT027]", item.Name)
	}
}

func TestLoadInventoryItemMissingSpriteFails(t *testing.T) {
	idx := commonIndex(t)
	itemText := loadDefInventEnglish(t)

	if _, err := LoadInventoryItem(idx, itemText, 9999); err == nil {
		t.Fatal("expected error for a nonexistent Item9999.a16")
	}
}

func TestGameStateInventoryIsUniqueAndOrdered(t *testing.T) {
	s := &GameState{}

	s.AddItem(3)
	s.AddItem(1)
	s.AddItem(3) // duplicate, should be a no-op

	if len(s.Inventory) != 2 {
		t.Fatalf("Inventory = %v, want 2 unique items", s.Inventory)
	}
	if s.Inventory[0] != 3 || s.Inventory[1] != 1 {
		t.Errorf("Inventory = %v, want [3 1] (acquisition order)", s.Inventory)
	}

	if !s.HasItem(1) || !s.HasItem(3) {
		t.Error("HasItem should report true for both added items")
	}
	if s.HasItem(2) {
		t.Error("HasItem should report false for an item never added")
	}

	s.RemoveItem(3)
	if s.HasItem(3) {
		t.Error("expected item 3 to be removed")
	}
	if len(s.Inventory) != 1 || s.Inventory[0] != 1 {
		t.Errorf("Inventory = %v, want [1]", s.Inventory)
	}
}
