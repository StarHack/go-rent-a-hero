package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type sourceEntry struct {
	Name  string
	IsDir bool
	Size  int64
}

type sourceFS interface {
	Open(string) (io.ReadSeekCloser, error)
	ReadDir(string) ([]sourceEntry, error)
	Close() error
}

type dirSource struct {
	root string
}

type nopReadSeekCloser struct {
	io.ReadSeeker
}

func (n *nopReadSeekCloser) Close() error {
	return nil
}

func openSource(name string) (sourceFS, error) {
	info, err := os.Stat(name)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		return &dirSource{root: name}, nil
	}

	if strings.EqualFold(filepath.Ext(name), ".zip") {
		return openZIPSource(name)
	}

	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	var header [16]byte
	n, readErr := f.ReadAt(header[:], 0)
	f.Close()

	if readErr != nil && readErr != io.EOF {
		return nil, readErr
	}

	if n >= 12 && isRawCDSync(header[:12]) {
		return openBINSource(name)
	}

	ext := strings.ToLower(filepath.Ext(name))
	if ext == ".bin" || ext == ".img" {
		return openBINSource(name)
	}
	if ext == ".cue" {
		return openCUESource(name)
	}

	return openISOSource(name)
}

func (d *dirSource) native(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.TrimPrefix(name, "/")

	if name == "" {
		return d.root
	}

	current := d.root

	for _, component := range strings.Split(name, "/") {
		if component == "" {
			continue
		}

		entries, err := os.ReadDir(current)
		if err != nil {
			return filepath.Join(current, component)
		}

		found := ""
		for _, entry := range entries {
			if strings.EqualFold(entry.Name(), component) {
				found = entry.Name()
				break
			}
		}

		if found == "" {
			found = component
		}

		current = filepath.Join(current, found)
	}

	return current
}

func (d *dirSource) Open(name string) (io.ReadSeekCloser, error) {
	return os.Open(d.native(name))
}

func (d *dirSource) ReadDir(name string) ([]sourceEntry, error) {
	entries, err := os.ReadDir(d.native(name))
	if err != nil {
		return nil, err
	}

	result := make([]sourceEntry, 0, len(entries))

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}

		result = append(result, sourceEntry{
			Name:  entry.Name(),
			IsDir: entry.IsDir(),
			Size:  info.Size(),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})

	return result, nil
}

func (d *dirSource) Close() error {
	return nil
}

func copyTree(src sourceFS, sourceName, target string) error {
	entries, err := src.ReadDir(sourceName)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcName := joinSourcePath(sourceName, entry.Name)
		dstName := filepath.Join(target, entry.Name)

		if entry.IsDir {
			if err := os.MkdirAll(dstName, 0755); err != nil {
				return err
			}

			if err := copyTree(src, srcName, dstName); err != nil {
				return err
			}

			continue
		}

		if err := copySourceFile(src, srcName, dstName); err != nil {
			return err
		}
	}

	return nil
}

func copySourceFile(src sourceFS, sourceName, targetName string) error {
	in, err := src.Open(sourceName)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(targetName), 0755); err != nil {
		return err
	}

	tmp := targetName + ".rahinstall.tmp"

	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	ok := false

	defer func() {
		out.Close()
		if !ok {
			os.Remove(tmp)
		}
	}()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	if err := out.Close(); err != nil {
		return err
	}

	if err := replaceFile(tmp, targetName); err != nil {
		return err
	}

	ok = true

	fmt.Printf("  %s\n", targetName)
	return nil
}

func replaceFile(source, target string) error {
	if err := os.Rename(source, target); err == nil {
		return nil
	}

	if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return os.Rename(source, target)
}

func joinSourcePath(parts ...string) string {
	result := ""

	for _, part := range parts {
		part = strings.ReplaceAll(part, "\\", "/")
		part = strings.Trim(part, "/")

		if part == "" {
			continue
		}

		if result != "" {
			result += "/"
		}

		result += part
	}

	return result
}

func safeRelativePath(name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.TrimSpace(name)

	parts := make([]string, 0)

	for _, part := range strings.Split(name, "/") {
		part = strings.TrimSpace(part)

		if part == "" || part == "." {
			continue
		}

		if part == ".." {
			return "", fmt.Errorf("unsafe path %q", name)
		}

		if strings.ContainsRune(part, ':') {
			return "", fmt.Errorf("unsafe path %q", name)
		}

		parts = append(parts, part)
	}

	if len(parts) == 0 {
		return "", nil
	}

	return filepath.Join(parts...), nil
}
