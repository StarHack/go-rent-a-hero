package szn

import (
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "fixtures", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}

	return data
}

func TestLoad46SZN(t *testing.T) {
	def, err := Load("46.SZN", fixture(t, "46.SZN"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(def.Backgrounds) != 1 {
		t.Errorf("backgrounds = %d, want 1", len(def.Backgrounds))
	}
	if len(def.Characters) != 1 {
		t.Errorf("characters = %d, want 1", len(def.Characters))
	}
	if len(def.Layers) != 1 {
		t.Errorf("layers = %d, want 1", len(def.Layers))
	}
	if len(def.Areas) != 5 {
		t.Errorf("areas = %d, want 5", len(def.Areas))
	}

	layer, ok := def.Layers["KanalTalk"]
	if !ok {
		t.Fatal("missing KanalTalk layer")
	}

	if layer.Filename != "KanalTalk.a16" {
		t.Errorf("Filename = %q, want KanalTalk.a16", layer.Filename)
	}
	if layer.X != 37 {
		t.Errorf("X = %d, want 37", layer.X)
	}
	if layer.Y != 106 {
		t.Errorf("Y = %d, want 106", layer.Y)
	}
	if layer.Z == nil || *layer.Z != 95 {
		t.Errorf("Z = %v, want 95", layer.Z)
	}
	if layer.ZoomVal == nil || *layer.ZoomVal != 100 {
		t.Errorf("ZoomVal = %v, want 100", layer.ZoomVal)
	}
	if layer.FPS == nil || *layer.FPS != 10 {
		t.Errorf("FPS = %v, want 10", layer.FPS)
	}
	if layer.LoadCond == nil || *layer.LoadCond != 1 {
		t.Errorf("LoadCond = %v, want 1", layer.LoadCond)
	}

	area, ok := def.Areas["046_Kanal"]
	if !ok {
		t.Fatal("missing 046_Kanal area")
	}

	if area.X1 != 135 || area.Y1 != 204 || area.X2 != 230 || area.Y2 != 270 {
		t.Errorf("rect = %d,%d,%d,%d want 135,204,230,270", area.X1, area.Y1, area.X2, area.Y2)
	}

	if area.CursorType == nil || *area.CursorType != 2 {
		t.Errorf("CursorType = %v, want 2", area.CursorType)
	}
}

func TestLoad46SZNCharacterAndBackground(t *testing.T) {
	def, err := Load("46.SZN", fixture(t, "46.SZN"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	bg, ok := def.Backgrounds["back46_mk"]
	if !ok {
		t.Fatal("missing back46_mk background")
	}
	if bg.Filename != "back46_mk.bmp" || bg.ZBuf != "back46_depth.zbf" || bg.WBuf != "back46_path.zbf" {
		t.Errorf("background = %+v", bg)
	}
	if bg.FPS != 0 {
		t.Errorf("FPS = %d, want 0", bg.FPS)
	}

	ch, ok := def.Characters["RodrigoSmall"]
	if !ok {
		t.Fatal("missing RodrigoSmall character")
	}

	if ch.X != 342 || ch.Y != 238 {
		t.Errorf("X,Y = %d,%d, want 342,238", ch.X, ch.Y)
	}
	if ch.ZPos != 54 {
		t.Errorf("ZPos = %d, want 54 (leading zero must parse as decimal)", ch.ZPos)
	}
	if ch.ZoomVal != 91 {
		t.Errorf("ZoomVal = %d, want 91", ch.ZoomVal)
	}

	if len(ch.Zones) != 1 {
		t.Fatalf("zones = %d, want 1", len(ch.Zones))
	}

	z := ch.Zones[0]
	if z.Red == nil || *z.Red != -200 {
		t.Errorf("zone Red = %v, want -200", z.Red)
	}
	if z.X == nil || *z.X != 415 {
		t.Errorf("zone X = %v, want 415", z.X)
	}
	if z.Range == nil || *z.Range != 50 {
		t.Errorf("zone Range = %v, want 50", z.Range)
	}

	// Type is absent in 46.SZN's RodrigoSmall section; must be nil, not 0.
	if ch.Type != nil {
		t.Errorf("Type = %v, want nil (absent)", ch.Type)
	}
}

func TestLoad7BSZN(t *testing.T) {
	def, err := Load("7B.SZN", fixture(t, "7B.SZN"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(def.Layers) != 2 {
		t.Errorf("layers = %d, want 2", len(def.Layers))
	}
	if len(def.Areas) != 6 {
		t.Errorf("areas = %d, want 6", len(def.Areas))
	}

	// RodrigoSmall in 7B.SZN explicitly sets Type=0: must be present and 0,
	// distinct from absent.
	ch, ok := def.Characters["RodrigoSmall"]
	if !ok {
		t.Fatal("missing RodrigoSmall")
	}
	if ch.Type == nil {
		t.Fatal("Type = nil, want explicit 0")
	}
	if *ch.Type != 0 {
		t.Errorf("Type = %d, want 0", *ch.Type)
	}

	hebel, ok := def.Layers["Hebel"]
	if !ok {
		t.Fatal("missing Hebel layer")
	}
	if hebel.Filename != "Hebel.TCC" {
		t.Errorf("Filename = %q, want Hebel.TCC", hebel.Filename)
	}
}

func TestMissingReferencedSectionErrors(t *testing.T) {
	text := "[GENERAL]\nBackground1=nope\n"

	_, err := Load("bad.SZN", []byte(text))
	if err == nil {
		t.Fatal("expected error for missing referenced section")
	}
}

func TestUnknownKeysPreservedInExtra(t *testing.T) {
	text := `[GENERAL]
Area1=A1

[A1]
X1=1
X2=2
Y1=3
Y2=4
SomeFutureKey=hello
`

	def, err := Load("t.SZN", []byte(text))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	area := def.Areas["A1"]
	if v, ok := area.Extra["SomeFutureKey"]; !ok || v != "hello" {
		t.Errorf("Extra[SomeFutureKey] = %q,%v, want hello,true", v, ok)
	}
}

func TestAreaNormalized(t *testing.T) {
	a := &AreaDef{X1: 230, X2: 135, Y1: 270, Y2: 204}

	minX, minY, maxX, maxY := a.Normalized()
	if minX != 135 || maxX != 230 || minY != 204 || maxY != 270 {
		t.Errorf("Normalized = %d,%d,%d,%d", minX, minY, maxX, maxY)
	}

	// Raw values must remain untouched.
	if a.X1 != 230 || a.X2 != 135 {
		t.Error("Normalized must not mutate raw X1/X2")
	}
}

func TestGeneralSparseSuffixesSortedNotAssumedContiguous(t *testing.T) {
	text := `[GENERAL]
Area2=Second
Area1=First
Area10=Tenth

[First]
X1=0
X2=1
Y1=0
Y2=1

[Second]
X1=0
X2=1
Y1=0
Y2=1

[Tenth]
X1=0
X2=1
Y1=0
Y2=1
`

	def, err := Load("t.SZN", []byte(text))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := []string{"First", "Second", "Tenth"}
	if len(def.General.Areas) != len(want) {
		t.Fatalf("General.Areas = %v, want %v", def.General.Areas, want)
	}
	for i, name := range want {
		if def.General.Areas[i] != name {
			t.Errorf("General.Areas[%d] = %q, want %q", i, def.General.Areas[i], name)
		}
	}
}
