package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadParsesInitScene(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.ini")
	writeFile(t, path, "; a comment\ninit_scene = 7B\n")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.InitScene != "7B" {
		t.Errorf("InitScene = %q, want %q", cfg.InitScene, "7B")
	}
}

func TestLoadIsCaseInsensitiveAndTrims(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.ini")
	writeFile(t, path, "  INIT_SCENE   =   46  \n")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.InitScene != "46" {
		t.Errorf("InitScene = %q, want %q", cfg.InitScene, "46")
	}
}

func TestLoadParsesStretchVideo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.ini")
	writeFile(t, path, "init_scene = 7B\nstretch_video = 1\n")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.StretchVideo {
		t.Error("StretchVideo = false, want true")
	}
}

func TestLoadStretchVideoDefaultsFalse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.ini")
	writeFile(t, path, "init_scene = 7B\n")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.StretchVideo {
		t.Error("StretchVideo = true, want false when absent")
	}
}

func TestLoadEnableSoundDefaultsTrue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.ini")
	writeFile(t, path, "init_scene = 7B\n")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.EnableSound {
		t.Error("EnableSound = false, want true when absent")
	}
}

func TestLoadEnableSoundZeroDisables(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.ini")
	writeFile(t, path, "init_scene = 7B\nenable_sound = 0\n")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.EnableSound {
		t.Error("EnableSound = true, want false when enable_sound = 0")
	}
}

func TestLoadMissingFileDefaultsEnableSoundTrue(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.ini"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.EnableSound {
		t.Error("EnableSound = false, want true when config.ini is absent")
	}
}

func TestLoadFullScreenDefaultsFalse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.ini")
	writeFile(t, path, "init_scene = 7B\n")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.FullScreen {
		t.Error("FullScreen = true, want false when absent")
	}
}

func TestLoadFullScreenOneEnables(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.ini")
	writeFile(t, path, "init_scene = 7B\nfull_screen = 1\n")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.FullScreen {
		t.Error("FullScreen = false, want true when full_screen = 1")
	}
}

func TestLoadDebugHotSpotsDefaultsTrue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.ini")
	writeFile(t, path, "init_scene = 7B\n")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.DebugHotSpots {
		t.Error("DebugHotSpots = false, want true when absent")
	}
}

func TestLoadDebugHotSpotsZeroDisables(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.ini")
	writeFile(t, path, "init_scene = 7B\ndebug_hot_spots = 0\n")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DebugHotSpots {
		t.Error("DebugHotSpots = true, want false when debug_hot_spots = 0")
	}
}

func TestLoadMissingFileDefaultsDebugHotSpotsTrue(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.ini"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.DebugHotSpots {
		t.Error("DebugHotSpots = false, want true when config.ini is absent")
	}
}

func TestLoadDebugAreaDefaultsFalse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.ini")
	writeFile(t, path, "init_scene = 7B\n")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DebugArea {
		t.Error("DebugArea = true, want false when absent")
	}
}

func TestLoadDebugAreaOneEnables(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.ini")
	writeFile(t, path, "init_scene = 7B\ndebug_area = 1\n")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.DebugArea {
		t.Error("DebugArea = false, want true when debug_area = 1")
	}
}

func TestLoadMissingFileDefaultsDebugAreaFalse(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.ini"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DebugArea {
		t.Error("DebugArea = true, want false when config.ini is absent")
	}
}

func TestLoadMissingFileIsNotAnError(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.ini"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.InitScene != "" {
		t.Errorf("InitScene = %q, want empty", cfg.InitScene)
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
