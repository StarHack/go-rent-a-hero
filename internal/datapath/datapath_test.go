package datapath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveUsesCWDWhenPresent(t *testing.T) {
	dir := t.TempDir()
	data := filepath.Join(dir, "data", "installation", "Common")
	if err := os.MkdirAll(data, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(data, "DefInvent.eng")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(wd) })

	got, err := Resolve("data/installation/Common/DefInvent.eng")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want, _ := filepath.Abs(file)
	if got != want {
		t.Errorf("Resolve = %q, want %q", got, want)
	}
}

func TestResolveFallsBackToBundleLayout(t *testing.T) {
	root := t.TempDir()
	// dist/rah.app/Contents/MacOS/rah
	macos := filepath.Join(root, "dist", "rah.app", "Contents", "MacOS")
	if err := os.MkdirAll(macos, 0o755); err != nil {
		t.Fatal(err)
	}
	// dist/data/installation/Common/DefInvent.eng (sibling of rah.app)
	dataFile := filepath.Join(root, "dist", "data", "installation", "Common", "DefInvent.eng")
	if err := os.MkdirAll(filepath.Dir(dataFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dataFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// Cwd has no data/ — forces bundle fallback.
	t.Chdir(root)
	t.Cleanup(func() { _ = os.Chdir(wd) })

	executableDirOverride = macos
	t.Cleanup(func() { executableDirOverride = "" })

	got, err := Resolve("data/installation/Common/DefInvent.eng")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want := filepath.Clean(dataFile)
	if got != want {
		t.Errorf("Resolve = %q, want %q", got, want)
	}
}
