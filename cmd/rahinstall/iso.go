package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

const isoSectorSize = 2048

type isoNode struct {
	name     string
	isDir    bool
	extent   uint32
	size     uint32
	children map[string]*isoNode
}

type isoSource struct {
	file       io.Closer
	reader     io.ReaderAt
	size       int64
	root       *isoNode
	decodeName func([]byte) (string, error)
}

type sectionReadSeekCloser struct {
	*io.SectionReader
}

func (s *sectionReadSeekCloser) Close() error {
	return nil
}

func openISOSource(name string) (sourceFS, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	src, err := newISOSource(f, f, info.Size())
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("%s is not a supported ISO9660 image: %w", name, err)
	}

	return src, nil
}

func newISOSource(file io.Closer, reader io.ReaderAt, size int64) (*isoSource, error) {
	src := &isoSource{
		file:   file,
		reader: reader,
		size:   size,
	}

	if err := src.load(); err != nil {
		return nil, err
	}

	return src, nil
}

func (s *isoSource) load() error {
	var descriptor [isoSectorSize]byte
	var primary []byte
	var joliet []byte
	jolietLevel := 0

	for sector := int64(16); sector < 64; sector++ {
		if _, err := s.reader.ReadAt(descriptor[:], sector*isoSectorSize); err != nil {
			return err
		}

		if string(descriptor[1:6]) != "CD001" {
			continue
		}

		switch descriptor[0] {
		case 1:
			if primary == nil {
				primary = append([]byte(nil), descriptor[:]...)
			}
		case 2:
			level := jolietDescriptorLevel(descriptor[:])
			if level > jolietLevel {
				joliet = append([]byte(nil), descriptor[:]...)
				jolietLevel = level
			}
		case 255:
			sector = 64
		}
	}

	selected := primary
	s.decodeName = decodeISOIdentifier
	if joliet != nil {
		selected = joliet
		s.decodeName = decodeJolietIdentifier
	}
	if selected == nil {
		return fmt.Errorf("primary volume descriptor not found")
	}

	rootRecord := selected[156:]
	if len(rootRecord) < 34 || rootRecord[0] < 34 {
		return fmt.Errorf("invalid root directory record")
	}

	root, err := parseISORecordWithDecoder(rootRecord[:rootRecord[0]], s.decodeName)
	if err != nil {
		return err
	}

	root.name = ""
	root.children = make(map[string]*isoNode)
	s.root = root

	seen := make(map[uint64]bool)
	return s.loadDirectory(root, seen)
}

func jolietDescriptorLevel(descriptor []byte) int {
	if len(descriptor) < 91 || descriptor[0] != 2 || string(descriptor[1:6]) != "CD001" {
		return 0
	}

	switch string(descriptor[88:91]) {
	case "%/@":
		return 1
	case "%/C":
		return 2
	case "%/E":
		return 3
	default:
		return 0
	}
}

func parseISORecord(record []byte) (*isoNode, error) {
	return parseISORecordWithDecoder(record, decodeISOIdentifier)
}

func parseISORecordWithDecoder(record []byte, decodeName func([]byte) (string, error)) (*isoNode, error) {
	if len(record) < 34 {
		return nil, fmt.Errorf("short directory record")
	}

	nameLen := int(record[32])
	if 33+nameLen > len(record) {
		return nil, fmt.Errorf("invalid directory record name")
	}

	nameBytes := record[33 : 33+nameLen]
	name := ""
	if nameLen == 1 && (nameBytes[0] == 0 || nameBytes[0] == 1) {
		name = string(nameBytes)
	} else {
		var err error
		name, err = decodeName(nameBytes)
		if err != nil {
			return nil, err
		}
	}

	return &isoNode{
		name:   name,
		isDir:  record[25]&2 != 0,
		extent: binary.LittleEndian.Uint32(record[2:6]),
		size:   binary.LittleEndian.Uint32(record[10:14]),
	}, nil
}

func decodeISOIdentifier(name []byte) (string, error) {
	if pos := bytes.IndexByte(name, ';'); pos >= 0 {
		name = name[:pos]
	}
	name = bytes.TrimSuffix(name, []byte("."))
	return decodeISOName(name), nil
}

func decodeJolietIdentifier(name []byte) (string, error) {
	if len(name)%2 != 0 {
		return "", fmt.Errorf("invalid Joliet identifier length %d", len(name))
	}

	runes := make([]rune, 0, len(name)/2)
	for i := 0; i < len(name); i += 2 {
		r := rune(binary.BigEndian.Uint16(name[i : i+2]))
		if r == 0 {
			continue
		}
		runes = append(runes, r)
	}

	decoded := string(runes)
	if pos := strings.IndexByte(decoded, ';'); pos >= 0 {
		decoded = decoded[:pos]
	}
	decoded = strings.TrimSuffix(decoded, ".")
	return decoded, nil
}

// decodeISOName turns an ISO9660 file identifier into a UTF-8 path
// component. Identifiers on this game's disc are ASCII except for German
// umlauts stored in the DOS OEM code page (CP850): Ä is the single byte
// 0x8E, as in 038_MÄD_02.WAV. Passing that byte through as a Go string
// makes os.OpenFile fail on macOS with "illegal byte sequence".
func decodeISOName(name []byte) string {
	if utf8.Valid(name) {
		return string(name)
	}

	decoded, err := charmap.CodePage850.NewDecoder().Bytes(name)
	if err != nil || !utf8.Valid(decoded) {
		return strings.ToValidUTF8(string(name), "_")
	}

	return string(decoded)
}

func (s *isoSource) loadDirectory(dir *isoNode, seen map[uint64]bool) error {
	key := uint64(dir.extent)<<32 | uint64(dir.size)
	if seen[key] {
		return nil
	}
	seen[key] = true

	if dir.size == 0 {
		return nil
	}

	data := make([]byte, int(dir.size))

	offset := int64(dir.extent) * isoSectorSize
	if _, err := s.reader.ReadAt(data, offset); err != nil && err != io.EOF {
		return err
	}

	if dir.children == nil {
		dir.children = make(map[string]*isoNode)
	}

	for pos := 0; pos < len(data); {
		length := int(data[pos])

		if length == 0 {
			pos = ((pos / isoSectorSize) + 1) * isoSectorSize
			continue
		}

		if pos+length > len(data) {
			return fmt.Errorf("directory record extends past directory data")
		}

		node, err := parseISORecordWithDecoder(data[pos:pos+length], s.decodeName)
		if err != nil {
			return err
		}

		pos += length

		if len(node.name) == 1 && (node.name[0] == 0 || node.name[0] == 1) {
			continue
		}

		dir.children[strings.ToLower(node.name)] = node

		if node.isDir {
			node.children = make(map[string]*isoNode)

			if err := s.loadDirectory(node, seen); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *isoSource) lookup(name string) (*isoNode, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.Trim(name, "/")

	node := s.root

	if name == "" {
		return node, nil
	}

	for _, part := range strings.Split(name, "/") {
		if part == "" {
			continue
		}

		if !node.isDir {
			return nil, os.ErrNotExist
		}

		child := node.children[strings.ToLower(part)]
		if child == nil {
			return nil, os.ErrNotExist
		}

		node = child
	}

	return node, nil
}

func (s *isoSource) Open(name string) (io.ReadSeekCloser, error) {
	node, err := s.lookup(name)
	if err != nil {
		return nil, err
	}

	if node.isDir {
		return nil, fmt.Errorf("%s is a directory", name)
	}

	offset := int64(node.extent) * isoSectorSize

	return &sectionReadSeekCloser{
		SectionReader: io.NewSectionReader(s.reader, offset, int64(node.size)),
	}, nil
}

func (s *isoSource) ReadDir(name string) ([]sourceEntry, error) {
	node, err := s.lookup(name)
	if err != nil {
		return nil, err
	}

	if !node.isDir {
		return nil, fmt.Errorf("%s is not a directory", name)
	}

	result := make([]sourceEntry, 0, len(node.children))

	for _, child := range node.children {
		result = append(result, sourceEntry{
			Name:  child.name,
			IsDir: child.isDir,
			Size:  int64(child.size),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})

	return result, nil
}

func (s *isoSource) Close() error {
	if s.file != nil {
		return s.file.Close()
	}

	return nil
}
