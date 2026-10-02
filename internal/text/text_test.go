package text

import (
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "fixtures", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}

	return data
}

func TestParseInventoryTextEnglish(t *testing.T) {
	inv, err := ParseInventoryText(fixture(t, "DefInvent.eng"))
	if err != nil {
		t.Fatalf("ParseInventoryText: %v", err)
	}

	item, ok := inv.Items[1]
	if !ok {
		t.Fatal("missing item 1")
	}

	if item.Name != "The stranger's cloak" {
		t.Errorf("item 1 name = %q, want %q", item.Name, "The stranger's cloak")
	}

	if item.Description == "" {
		t.Error("item 1 description is empty")
	}
}

func TestParseInventoryTextGermanHasAlternate(t *testing.T) {
	inv, err := ParseInventoryText(fixture(t, "DefInvent.ger"))
	if err != nil {
		t.Fatalf("ParseInventoryText: %v", err)
	}

	item, ok := inv.Items[1]
	if !ok {
		t.Fatal("missing item 1")
	}

	if !item.HasAlt {
		t.Error("expected item 1 to have an alternate (H) description in the German file")
	}

	if item.Alternate == "" {
		t.Error("item 1 alternate description is empty")
	}
}
