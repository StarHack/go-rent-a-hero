// Package ini parses the Windows profile/INI-style text files used
// throughout the original game: SZN scene definitions, DefInvent.<lang>
// inventory text, and per-location .GER/.ENG speech text.
//
// Section names and keys are case-insensitive, matching the original
// Windows profile API (GetPrivateProfileString) semantics. Values preserve
// original spelling/case.
package ini

import (
	"bufio"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// Section is an ordered set of key/value pairs from one [Section] block.
// Keys are stored in original case; lookups are case-insensitive via
// File.Get.
type Section struct {
	Name string
	Keys []string          // original-case keys, in file order
	Vals map[string]string // lower(key) -> value
}

// File is a parsed INI document.
type File struct {
	sections     []*Section
	sectionIndex map[string]*Section // lower(name) -> section
}

// Decode converts raw bytes to a UTF-8 string, trying UTF-8 first and
// falling back to Windows-1252 (common for German-language assets of this
// era).
func Decode(data []byte) string {
	if utf8.Valid(data) {
		return string(data)
	}

	decoded, err := charmap.Windows1252.NewDecoder().Bytes(data)
	if err != nil {
		// Fall back to a lossy direct cast rather than dropping the file.
		return string(data)
	}

	return string(decoded)
}

// Parse parses INI-formatted text (already decoded to UTF-8).
func Parse(text string) *File {
	f := &File{sectionIndex: map[string]*Section{}}

	var current *Section

	scanner := bufio.NewScanner(strings.NewReader(text))
	// SZN/help text lines are short, but raise the limit generously in case
	// of long localized strings.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			name := strings.TrimSpace(line[1 : len(line)-1])
			current = f.getOrCreateSection(name)
			continue
		}

		if current == nil {
			continue
		}

		rawKey, rawValue, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key := strings.TrimSpace(rawKey)
		value := unquote(strings.TrimSpace(rawValue))

		lowerKey := strings.ToLower(key)
		if _, exists := current.Vals[lowerKey]; !exists {
			current.Keys = append(current.Keys, key)
		}
		current.Vals[lowerKey] = value
	}

	return f
}

// unquote removes exactly one matching pair of surrounding double quotes.
// It does not touch internal quotes.
func unquote(v string) string {
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		return v[1 : len(v)-1]
	}

	return v
}

func (f *File) getOrCreateSection(name string) *Section {
	key := strings.ToLower(name)

	if s, ok := f.sectionIndex[key]; ok {
		return s
	}

	s := &Section{Name: name, Vals: map[string]string{}}
	f.sections = append(f.sections, s)
	f.sectionIndex[key] = s

	return s
}

// Section returns the named section (case-insensitive) if present.
func (f *File) Section(name string) (*Section, bool) {
	s, ok := f.sectionIndex[strings.ToLower(name)]
	return s, ok
}

// Sections returns all sections in file order.
func (f *File) Sections() []*Section {
	return f.sections
}

// Get looks up a key (case-insensitive) within a section (case-insensitive).
func (f *File) Get(section, key string) (string, bool) {
	s, ok := f.Section(section)
	if !ok {
		return "", false
	}

	v, ok := s.Vals[strings.ToLower(key)]
	return v, ok
}

// Get looks up a key (case-insensitive) within this section.
func (s *Section) Get(key string) (string, bool) {
	v, ok := s.Vals[strings.ToLower(key)]
	return v, ok
}
