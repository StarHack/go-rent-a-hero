// Package assets implements a case-insensitive asset index over the
// original game's data directories, standing in for the original
// case-insensitive Windows filesystem.
package assets

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Index maps case-insensitive relative asset paths to their real,
// original-case path on disk.
type Index struct {
	roots   []string
	byLower map[string]string // normalized lowercase rel path -> real absolute path
}

// DuplicateError is returned by NewIndex when two different real paths
// normalize to the same case-insensitive key.
type DuplicateError struct {
	Key      string
	Existing string
	New      string
}

func (e *DuplicateError) Error() string {
	return fmt.Sprintf("assets: duplicate case-insensitive path %q: %q and %q both map to it", e.Key, e.Existing, e.New)
}

// normalize converts a path to the index's lookup key: forward slashes,
// lowercase, no leading "./".
func normalize(p string) string {
	p = filepath.ToSlash(p)
	p = strings.TrimPrefix(p, "./")
	return strings.ToLower(p)
}

// NewIndex walks each root directory once and builds a case-insensitive
// index of every regular file found, keyed by its path relative to that
// root. Roots are walked in the given order; a later root may add new files
// but must not collide (by normalized key) with an earlier one -- this
// mirrors the original single Windows-filesystem-of-truth model and
// surfaces genuine asset conflicts instead of silently shadowing them.
func NewIndex(roots ...string) (*Index, error) {
	idx := &Index{
		roots:   append([]string(nil), roots...),
		byLower: map[string]string{},
	}

	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			rel, err := filepath.Rel(root, path)
			if err != nil {
				return fmt.Errorf("assets: relativize %q under %q: %w", path, root, err)
			}

			key := normalize(rel)

			if existing, ok := idx.byLower[key]; ok {
				return &DuplicateError{Key: key, Existing: existing, New: path}
			}

			idx.byLower[key] = path

			return nil
		})
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("assets: root %q does not exist: %w", root, err)
			}
			return nil, err
		}
	}

	return idx, nil
}

// Resolve looks up a game-referenced path (which may use '\' separators and
// arbitrary case) and returns the real on-disk path.
func (idx *Index) Resolve(gamePath string) (string, error) {
	key := normalize(strings.ReplaceAll(gamePath, `\`, "/"))

	if real, ok := idx.byLower[key]; ok {
		return real, nil
	}

	return "", fmt.Errorf("assets: %q not found in index", gamePath)
}

// Has reports whether gamePath resolves to a known asset.

func (idx *Index) ResolveBaseName(gamePath string) (string, error) {
	if real, err := idx.Resolve(gamePath); err == nil {
		return real, nil
	}

	want := strings.ToLower(filepath.Base(strings.ReplaceAll(gamePath, `\`, "/")))
	var match string
	for key, real := range idx.byLower {
		if strings.ToLower(filepath.Base(key)) != want {
			continue
		}
		if match != "" && match != real {
			return "", fmt.Errorf("assets: basename %q is ambiguous", gamePath)
		}
		match = real
	}

	if match == "" {
		return "", fmt.Errorf("assets: %q not found in index", gamePath)
	}

	return match, nil
}

func (idx *Index) ResolveByStem(stem string, extensions ...string) (string, string, error) {
	stem = strings.TrimSpace(stem)
	if stem == "" {
		return "", "", fmt.Errorf("assets: empty asset stem")
	}

	if filepath.Ext(stem) != "" {
		real, err := idx.ResolveBaseName(stem)
		if err != nil {
			return "", "", err
		}
		return real, filepath.Base(real), nil
	}

	if len(extensions) == 0 {
		extensions = []string{".avi", ".a16", ".tcc", ".bmp"}
	}

	for _, ext := range extensions {
		if ext == "" {
			continue
		}
		if ext[0] != '.' {
			ext = "." + ext
		}
		name := stem + ext
		if real, err := idx.Resolve(name); err == nil {
			return real, filepath.Base(real), nil
		}
	}

	for _, ext := range extensions {
		if ext == "" {
			continue
		}
		if ext[0] != '.' {
			ext = "." + ext
		}
		name := stem + ext
		if real, err := idx.ResolveBaseName(name); err == nil {
			return real, filepath.Base(real), nil
		}
	}

	return "", "", fmt.Errorf("assets: no asset found for stem %q", stem)
}

func (idx *Index) Has(gamePath string) bool {
	_, err := idx.Resolve(gamePath)
	return err == nil
}

// Len returns the number of indexed files.
func (idx *Index) Len() int {
	return len(idx.byLower)
}

// Keys returns all normalized index keys, sorted, for diagnostics.
func (idx *Index) Keys() []string {
	keys := make([]string, 0, len(idx.byLower))
	for k := range idx.byLower {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
