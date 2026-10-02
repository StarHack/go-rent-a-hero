// Package datapath resolves project-relative paths (data/, config.ini, …)
// for both repo development (cwd) and macOS app bundles, where game assets
// sit next to the .app: Contents/MacOS/<exe> → ../../../data/…
package datapath

import (
	"fmt"
	"os"
	"path/filepath"
)

// executableDirOverride is set by tests to simulate a bundled .app layout.
var executableDirOverride string

// Resolve returns an absolute path to relativePath. It tries, in order:
//
//  1. relativePath as given (absolute, or relative to the process cwd)
//  2. filepath.Join(<executable>/../../../, relativePath) — packaged layout
//     with data/ (and config.ini) beside the .app bundle
func Resolve(relativePath string) (string, error) {
	if relativePath == "" {
		return "", fmt.Errorf("datapath: empty path")
	}

	if filepath.IsAbs(relativePath) {
		if _, err := os.Stat(relativePath); err != nil {
			return "", fmt.Errorf("datapath: %s: %w", relativePath, err)
		}
		return filepath.Clean(relativePath), nil
	}

	if abs, err := filepath.Abs(relativePath); err == nil {
		if _, err := os.Stat(abs); err == nil {
			return filepath.Clean(abs), nil
		}
	}

	bundled, err := bundlePath(relativePath)
	if err != nil {
		return "", fmt.Errorf("datapath: %s: not found (cwd or next to .app bundle)", relativePath)
	}
	if _, err := os.Stat(bundled); err != nil {
		return "", fmt.Errorf("datapath: %s: not found at %q or %q", relativePath, relativePath, bundled)
	}
	return bundled, nil
}

func bundlePath(relativePath string) (string, error) {
	exeDir, err := executableDir()
	if err != nil {
		return "", err
	}
	return filepath.Clean(filepath.Join(exeDir, "..", "..", "..", relativePath)), nil
}

func executableDir() (string, error) {
	if executableDirOverride != "" {
		return executableDirOverride, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}
