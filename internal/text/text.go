// Package text implements the localized inventory/speech text database
// described in agents/TEXT_FORMATS.md: DefInvent.<lang> and per-location
// speech text, all backed by the case-insensitive ini package.
package text

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/wok/rent-a-hero/internal/formats/ini"
)

// DB is a generic case-insensitive INI-backed text database.
type DB interface {
	Get(section, key string) (string, bool)
}

// IniDB adapts an *ini.File to the DB interface.
type IniDB struct {
	File *ini.File
}

// Get implements DB.
func (d *IniDB) Get(section, key string) (string, bool) {
	return d.File.Get(section, key)
}

// LoadIni decodes raw bytes (UTF-8 or Windows-1252) and parses them as INI.
func LoadIni(data []byte) *ini.File {
	return ini.Parse(ini.Decode(data))
}

// InventoryItemText holds the localized display name and description(s) for
// one inventory item, per the TEXT%03d/INVENT%03d/INVENT%03dH convention.
type InventoryItemText struct {
	ID          int
	Name        string
	Description string
	Alternate   string
	HasAlt      bool
}

// InventoryText is the parsed [SPEECH] section of a DefInvent.<lang> file.
type InventoryText struct {
	Items map[int]InventoryItemText
}

// speechKeyPattern matches TEXT%03d, INVENT%03d and INVENT%03dH (any digit
// width, not just 3, since the doc only confirms the convention not a fixed
// width).
var speechKeyPattern = regexp.MustCompile(`(?i)^(TEXT|INVENT)([0-9]+)(H)?$`)

// ParseInventoryText parses a DefInvent.<lang> file's [SPEECH] section into
// per-item text records.
func ParseInventoryText(data []byte) (*InventoryText, error) {
	f := LoadIni(data)

	section, ok := f.Section("SPEECH")
	if !ok {
		return nil, fmt.Errorf("text: missing [SPEECH] section")
	}

	items := map[int]InventoryItemText{}

	for _, key := range section.Keys {
		m := speechKeyPattern.FindStringSubmatch(key)
		if m == nil {
			continue
		}

		id, err := strconv.Atoi(m[2])
		if err != nil {
			continue
		}

		item := items[id]
		item.ID = id

		value, _ := section.Get(key)
		prefix := strings.ToUpper(m[1])
		alt := m[3] != ""

		switch {
		case prefix == "TEXT":
			item.Name = value
		case prefix == "INVENT" && alt:
			item.Alternate = value
			item.HasAlt = true
		case prefix == "INVENT":
			item.Description = value
		}

		items[id] = item
	}

	return &InventoryText{Items: items}, nil
}
