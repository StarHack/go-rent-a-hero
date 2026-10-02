// Package config reads the project's root config.ini: small settings for
// running the game itself (as opposed to internal/formats/ini, which
// parses the original game's own INI-style assets).
package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds config.ini's settings.
type Config struct {
	// InitScene is the scene ID to load at startup (init_scene), e.g. "7B"
	// or "114". Empty if config.ini is absent or doesn't set it. See
	// game.ResolveLaunchPhase: the engine determines on its own which
	// location/launch phase a given scene ID belongs to, so this alone is
	// enough to jump directly into any known scene, cinematic or not,
	// without separately specifying which location it lives under.
	InitScene string

	// StretchVideo is stretch_video: whether AVI cutscenes (including the
	// launch intro) are stretched to fill the whole logical window instead
	// of playing at their native size within the scene viewport.
	StretchVideo bool

	// EnableSound is enable_sound: whether to initialize the real audio
	// backend (internal/audio's Oto engine) at all. Defaults to true
	// (sound on) when config.ini is absent or doesn't set it; set to 0 to
	// force a silent internal/audio.NullEngine regardless of whether an
	// audio device is available, e.g. for running headless without device
	// probing/output.
	EnableSound bool

	// FullScreen is full_screen: whether to open the game window in
	// (borderless desktop) fullscreen instead of a normal resizable window.
	// Defaults to false (windowed) when config.ini is absent or doesn't set
	// it.
	FullScreen bool

	// DebugHotSpots is debug_hot_spots: whether to draw the green hotspot
	// debug rectangles on start (see render.Renderer.SetDebugAreas).
	// Defaults to true when config.ini is absent or doesn't set it, and is
	// overridden by cmd/rah's --debug-areas flag if that is given
	// explicitly.
	DebugHotSpots bool

	// DebugArea is debug_area: whether to draw the current scene and
	// location IDs in yellow at the top-left of the screen
	// ("S: <scene> | L: <location>"). Defaults to false when config.ini is
	// absent or doesn't set it.
	DebugArea bool

	OverrideInventory []string
}

// Load reads and parses a flat "key = value" config file: blank lines and
// ";"/"#" comments are ignored, and (unlike internal/formats/ini) no
// [section] headers are needed. A missing file is not an error -- it just
// yields a zero Config -- since config.ini is optional.
func Load(path string) (Config, error) {
	cfg := Config{EnableSound: true, DebugHotSpots: true} // defaults: sound on, hotspot debug rectangles on

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return Config{}, fmt.Errorf("config: read %s: %w", path, err)
	}

	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		value = strings.TrimSpace(value)

		switch strings.ToLower(strings.TrimSpace(key)) {
		case "init_scene":
			cfg.InitScene = value
		case "stretch_video":
			cfg.StretchVideo = value == "1"
		case "enable_sound":
			cfg.EnableSound = value != "0"
		case "full_screen":
			cfg.FullScreen = value == "1"
		case "debug_hot_spots":
			cfg.DebugHotSpots = value != "0"
		case "debug_area":
			cfg.DebugArea = value == "1"
		case "override_inventory":
			cfg.OverrideInventory = parseList(value)
		}
	}

	return cfg, nil
}

func parseList(value string) []string {
	items := []string{}
	for item := range strings.SplitSeq(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}
