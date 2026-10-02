package ini

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

func TestParseBasic(t *testing.T) {
	text := "[General]\nBackground1=back46_mk\n; comment\nLayer1=KanalTalk\n"

	f := Parse(text)

	v, ok := f.Get("general", "background1")
	if !ok || v != "back46_mk" {
		t.Errorf("Get(general,background1) = %q,%v, want back46_mk,true", v, ok)
	}

	v, ok = f.Get("GENERAL", "LAYER1")
	if !ok || v != "KanalTalk" {
		t.Errorf("Get(GENERAL,LAYER1) = %q,%v, want KanalTalk,true", v, ok)
	}
}

func TestUnquote(t *testing.T) {
	f := Parse(`[S]
A="hello world"
B=no quotes
C="has \"internal\" text"
`)

	if v, _ := f.Get("S", "A"); v != "hello world" {
		t.Errorf("A = %q, want %q", v, "hello world")
	}

	if v, _ := f.Get("S", "B"); v != "no quotes" {
		t.Errorf("B = %q, want %q", v, "no quotes")
	}

	// Only the outer quote pair is stripped; internal backslash-quotes are
	// left untouched since the format has no documented escaping.
	if v, _ := f.Get("S", "C"); v != `has \"internal\" text` {
		t.Errorf(`C = %q, want %q`, v, `has \"internal\" text`)
	}
}

func TestDecodeUTF8(t *testing.T) {
	s := Decode([]byte("hello"))
	if s != "hello" {
		t.Errorf("Decode = %q, want hello", s)
	}
}

func TestDecodeWindows1252Fallback(t *testing.T) {
	data := fixture(t, "DefInvent.ger")

	decoded := Decode(data)

	// The German file contains ö/ü/ß via Windows-1252; confirm decoding
	// produced valid UTF-8 containing at least one such character.
	if !containsAny(decoded, "öüßÖÜ") {
		t.Errorf("expected decoded text to contain German umlauts")
	}
}

func containsAny(s, chars string) bool {
	for _, r := range chars {
		for _, sr := range s {
			if sr == r {
				return true
			}
		}
	}
	return false
}

func TestParseDefInventGerHasAltVariant(t *testing.T) {
	data := fixture(t, "DefInvent.ger")

	f := Parse(Decode(data))

	if _, ok := f.Get("SPEECH", "INVENT001"); !ok {
		t.Error("missing INVENT001")
	}

	if _, ok := f.Get("SPEECH", "INVENT001H"); !ok {
		t.Error("missing INVENT001H (alternate/revised variant)")
	}
}

func TestSectionCaseInsensitiveDedup(t *testing.T) {
	f := Parse("[Foo]\nA=1\n[FOO]\nB=2\n")

	if len(f.Sections()) != 1 {
		t.Fatalf("expected sections to merge case-insensitively, got %d sections", len(f.Sections()))
	}

	if v, _ := f.Get("foo", "a"); v != "1" {
		t.Errorf("A = %q, want 1", v)
	}

	if v, _ := f.Get("foo", "b"); v != "2" {
		t.Errorf("B = %q, want 2", v)
	}
}
