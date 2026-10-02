package assets

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestNewIndexAndResolve(t *testing.T) {
	idx, err := NewIndex("../../testdata/fixtures")
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}

	if idx.Len() == 0 {
		t.Fatal("expected at least one indexed file")
	}

	// Original file is "HEBEL.TCC"; resolve with different case and
	// backslash separator to emulate original Windows-style references.
	real, err := idx.Resolve("hebel.tcc")
	if err != nil {
		t.Fatalf("Resolve(hebel.tcc): %v", err)
	}

	if filepath.Base(real) != "HEBEL.TCC" {
		t.Errorf("resolved base = %q, want HEBEL.TCC", filepath.Base(real))
	}

	if !idx.Has("HEBEL.TCC") {
		t.Error("Has(HEBEL.TCC) = false, want true")
	}

	if idx.Has("does-not-exist.foo") {
		t.Error("Has(does-not-exist.foo) = true, want false")
	}
}

func TestResolveNormalizesBackslashes(t *testing.T) {
	idx, err := NewIndex("../../testdata/fixtures")
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}

	if _, err := idx.Resolve(`HEBEL.TCC`); err != nil {
		t.Errorf("Resolve failed: %v", err)
	}
}

func TestDuplicateAcrossRootsRejected(t *testing.T) {
	dir := t.TempDir()

	rootA := filepath.Join(dir, "a")
	rootB := filepath.Join(dir, "b")

	if err := os.MkdirAll(rootA, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(rootB, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(rootA, "Item001.a16"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootB, "item001.a16"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := NewIndex(rootA, rootB)
	if err == nil {
		t.Fatal("expected duplicate-path error")
	}

	var dupErr *DuplicateError
	if !errors.As(err, &dupErr) {
		t.Fatalf("expected *DuplicateError, got %T: %v", err, err)
	}
}

func TestNewIndexMissingRoot(t *testing.T) {
	_, err := NewIndex(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected error for missing root")
	}
}

func TestResolveByStemFindsCaseInsensitiveAnimation(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "RODSETZEN.AVI")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	idx, err := NewIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	real, name, err := idx.ResolveByStem("RodSetzen", ".avi", ".a16")
	if err != nil {
		t.Fatal(err)
	}
	if real != path {
		t.Fatalf("real = %q, want %q", real, path)
	}
	if name != "RODSETZEN.AVI" {
		t.Fatalf("name = %q, want RODSETZEN.AVI", name)
	}
}

func TestResolveByStemPrefersExactRootRelativeMatch(t *testing.T) {
	root := t.TempDir()
	avi := filepath.Join(root, "RODREIN.AVI")
	a16 := filepath.Join(root, "RODREIN.A16")
	if err := os.WriteFile(avi, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(a16, []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	idx, err := NewIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	real, _, err := idx.ResolveByStem("RodRein", ".avi", ".a16")
	if err != nil {
		t.Fatal(err)
	}
	if real != avi {
		t.Fatalf("real = %q, want %q", real, avi)
	}
}
