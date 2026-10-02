package main

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	installShieldSignature = 0x28635349
	isFileSplit           = 1
	isFileObfuscated      = 2
	isFileCompressed      = 4
	isFileInvalid         = 8
	isMaxFileGroups       = 71
)

type installShieldHeader struct {
	version             uint32
	cabDescriptorOffset uint32
	cabDescriptorSize   uint32
	fileTableOffset     uint32
	fileTableSize       uint32
	directoryCount      uint32
	fileCount           uint32
	fileTableOffset2    uint32
	fileGroupOffsets    [isMaxFileGroups]uint32
	fileTable           []uint32
}

type installShieldFile struct {
	index          int
	name           string
	directory      string
	flags          uint16
	expandedSize   uint32
	compressedSize uint32
	dataOffset     uint32
}

type installShieldFileGroup struct {
	name      string
	firstFile int32
	lastFile  int32
}

type installShieldArchive struct {
	r      io.ReadSeeker
	header installShieldHeader
	files  []installShieldFile
	groups []installShieldFileGroup
}

func extractInstallShieldCommon(src sourceFS, cabName, target string) error {
	f, err := src.Open(cabName)
	if err != nil {
		return err
	}
	defer f.Close()

	archive, err := openInstallShield5(f)
	if err != nil {
		return err
	}

	indices := archive.commonFileIndices()

	if len(indices) == 0 {
		var groups []string
		for _, group := range archive.groups {
			groups = append(groups, group.name)
		}

		sort.Strings(groups)

		if len(groups) != 0 {
			return fmt.Errorf("could not identify Common file group; cabinet groups: %s", strings.Join(groups, ", "))
		}

		return fmt.Errorf("could not identify Common files in cabinet")
	}

	commonTarget := filepath.Join(target, "Common")

	if err := os.MkdirAll(commonTarget, 0755); err != nil {
		return err
	}

	extracted := 0

	for _, index := range indices {
		file := archive.files[index]

		if file.flags&isFileInvalid != 0 {
			continue
		}

		relative := commonRelativePath(file)
		if relative == "" {
			relative = file.name
		}

		relative, err := safeRelativePath(relative)
		if err != nil {
			return err
		}

		if relative == "" {
			continue
		}

		dst := filepath.Join(commonTarget, relative)

		relCheck, err := filepath.Rel(commonTarget, dst)
		if err != nil {
			return err
		}

		if relCheck == ".." || strings.HasPrefix(relCheck, ".."+string(filepath.Separator)) {
			return fmt.Errorf("cabinet path escapes Common: %q", relative)
		}

		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}

		if err := archive.extractFile(index, dst); err != nil {
			return fmt.Errorf("%s: %w", file.name, err)
		}

		fmt.Printf("  %s\n", dst)
		extracted++
	}

	if extracted == 0 {
		return fmt.Errorf("Common file set was empty")
	}

	return nil
}

func openInstallShield5(r io.ReadSeeker) (*installShieldArchive, error) {
	var common [20]byte

	if _, err := io.ReadFull(r, common[:]); err != nil {
		return nil, err
	}

	if binary.LittleEndian.Uint32(common[0:4]) != installShieldSignature {
		return nil, fmt.Errorf("not an InstallShield cabinet")
	}

	header := installShieldHeader{
		version:             binary.LittleEndian.Uint32(common[4:8]),
		cabDescriptorOffset: binary.LittleEndian.Uint32(common[12:16]),
		cabDescriptorSize:   binary.LittleEndian.Uint32(common[16:20]),
	}

	if header.cabDescriptorOffset == 0 || header.cabDescriptorSize < 0x30 {
		return nil, fmt.Errorf("invalid InstallShield cabinet descriptor")
	}

	desc := make([]byte, header.cabDescriptorSize)

	if _, err := r.Seek(int64(header.cabDescriptorOffset), io.SeekStart); err != nil {
		return nil, err
	}

	if _, err := io.ReadFull(r, desc); err != nil {
		return nil, err
	}

	if len(desc) < 0x3e+isMaxFileGroups*4 {
		return nil, fmt.Errorf("InstallShield descriptor is too short")
	}

	header.fileTableOffset = le32(desc, 0x0c)
	header.fileTableSize = le32(desc, 0x14)
	header.directoryCount = le32(desc, 0x1c)
	header.fileCount = le32(desc, 0x28)
	header.fileTableOffset2 = le32(desc, 0x2c)

	if header.fileCount > 1000000 || header.directoryCount > 1000000 {
		return nil, fmt.Errorf("unreasonable InstallShield table sizes")
	}

	for i := 0; i < isMaxFileGroups; i++ {
		header.fileGroupOffsets[i] = le32(desc, 0x3e+i*4)
	}

	tableCount := uint64(header.directoryCount) + uint64(header.fileCount)
	if tableCount > 1000000 {
		return nil, fmt.Errorf("InstallShield file table is too large")
	}

	tableBase := uint64(header.cabDescriptorOffset) + uint64(header.fileTableOffset)
	tableBytes := make([]byte, int(tableCount)*4)

	if _, err := r.Seek(int64(tableBase), io.SeekStart); err != nil {
		return nil, err
	}

	if _, err := io.ReadFull(r, tableBytes); err != nil {
		return nil, err
	}

	header.fileTable = make([]uint32, tableCount)

	for i := range header.fileTable {
		header.fileTable[i] = binary.LittleEndian.Uint32(tableBytes[i*4 : i*4+4])
	}

	archive := &installShieldArchive{
		r:      r,
		header: header,
	}

	directories := make([]string, header.directoryCount)

	for i := uint32(0); i < header.directoryCount; i++ {
		name, err := archive.stringAt(header.fileTable[i])
		if err != nil {
			return nil, fmt.Errorf("directory %d: %w", i, err)
		}

		directories[i] = normalizeInstallShieldDirectory(name)
	}

	archive.files = make([]installShieldFile, header.fileCount)

	for i := uint32(0); i < header.fileCount; i++ {
		offsetIndex := header.directoryCount + i
		descriptorOffset := uint64(header.fileTable[offsetIndex])

		file, err := archive.readV5FileDescriptor(int(i), descriptorOffset, directories)
		if err != nil {
			return nil, err
		}

		archive.files[i] = file
	}

	groups, err := archive.readV5FileGroups()
	if err != nil {
		return nil, err
	}

	archive.groups = groups

	return archive, nil
}

func le32(data []byte, offset int) uint32 {
	return binary.LittleEndian.Uint32(data[offset : offset+4])
}

func (a *installShieldArchive) tableBase() uint64 {
	return uint64(a.header.cabDescriptorOffset) + uint64(a.header.fileTableOffset)
}

func (a *installShieldArchive) stringAt(offset uint32) (string, error) {
	absolute := a.tableBase() + uint64(offset)

	if _, err := a.r.Seek(int64(absolute), io.SeekStart); err != nil {
		return "", err
	}

	var result []byte
	var one [1]byte

	for len(result) < 65536 {
		if _, err := io.ReadFull(a.r, one[:]); err != nil {
			return "", err
		}

		if one[0] == 0 {
			return string(result), nil
		}

		result = append(result, one[0])
	}

	return "", fmt.Errorf("unterminated cabinet string")
}

func (a *installShieldArchive) descriptorBytes(relative uint64, size int) ([]byte, error) {
	absolute := a.tableBase() + relative

	if _, err := a.r.Seek(int64(absolute), io.SeekStart); err != nil {
		return nil, err
	}

	data := make([]byte, size)

	if _, err := io.ReadFull(a.r, data); err != nil {
		return nil, err
	}

	return data, nil
}

func (a *installShieldArchive) readV5FileDescriptor(index int, offset uint64, directories []string) (installShieldFile, error) {
	data, err := a.descriptorBytes(offset, 0x3a)
	if err != nil {
		return installShieldFile{}, fmt.Errorf("file descriptor %d: %w", index, err)
	}

	nameOffset := binary.LittleEndian.Uint32(data[0:4])
	directoryIndex := binary.LittleEndian.Uint16(data[4:6])
	flags := binary.LittleEndian.Uint16(data[8:10])
	expandedSize := binary.LittleEndian.Uint32(data[10:14])
	compressedSize := binary.LittleEndian.Uint32(data[14:18])
	dataOffset := binary.LittleEndian.Uint32(data[38:42])

	if int(directoryIndex) >= len(directories) {
		return installShieldFile{}, fmt.Errorf("file %d has invalid directory index %d", index, directoryIndex)
	}

	name, err := a.stringAt(nameOffset)
	if err != nil {
		return installShieldFile{}, fmt.Errorf("file %d name: %w", index, err)
	}

	return installShieldFile{
		index:          index,
		name:           name,
		directory:      directories[directoryIndex],
		flags:          flags,
		expandedSize:   expandedSize,
		compressedSize: compressedSize,
		dataOffset:     dataOffset,
	}, nil
}

func (a *installShieldArchive) readV5FileGroups() ([]installShieldFileGroup, error) {
	var result []installShieldFileGroup
	visited := make(map[uint32]bool)

	for _, first := range a.header.fileGroupOffsets {
		next := first

		for next != 0 {
			if visited[next] {
				return nil, fmt.Errorf("cycle in InstallShield file group list")
			}
			visited[next] = true

			listData, err := a.readDescriptorRelative(next, 12)
			if err != nil {
				return nil, err
			}

			descriptorOffset := binary.LittleEndian.Uint32(listData[4:8])
			nextOffset := binary.LittleEndian.Uint32(listData[8:12])

			groupData, err := a.readDescriptorRelative(descriptorOffset, 0x54)
			if err != nil {
				return nil, err
			}

			nameOffset := binary.LittleEndian.Uint32(groupData[0:4])
			firstFile := int32(binary.LittleEndian.Uint32(groupData[0x4c:0x50]))
			lastFile := int32(binary.LittleEndian.Uint32(groupData[0x50:0x54]))

			name, err := a.stringAt(nameOffset)
			if err != nil {
				return nil, err
			}

			result = append(result, installShieldFileGroup{
				name:      name,
				firstFile: firstFile,
				lastFile:  lastFile,
			})

			next = nextOffset
		}
	}

	return result, nil
}

func (a *installShieldArchive) readDescriptorRelative(relative uint32, size int) ([]byte, error) {
	absolute := uint64(a.header.cabDescriptorOffset) + uint64(relative)

	if _, err := a.r.Seek(int64(absolute), io.SeekStart); err != nil {
		return nil, err
	}

	data := make([]byte, size)

	if _, err := io.ReadFull(a.r, data); err != nil {
		return nil, err
	}

	return data, nil
}

func normalizeInstallShieldDirectory(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.Trim(name, "/")
	return name
}

func normalizeGroupName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.Join(strings.Fields(name), " ")
	return name
}

func isCommonGroup(name string) bool {
	n := normalizeGroupName(name)

	if n == "common" {
		return true
	}

	if strings.HasPrefix(n, "common ") {
		return true
	}

	if strings.HasSuffix(n, " common") {
		return true
	}

	return false
}

func directoryContainsCommon(name string) bool {
	name = strings.ReplaceAll(name, "\\", "/")

	for _, part := range strings.Split(name, "/") {
		if strings.EqualFold(strings.TrimSpace(part), "Common") {
			return true
		}
	}

	return false
}

func (a *installShieldArchive) commonFileIndices() []int {
	selected := make(map[int]bool)

	foundGroup := false

	for _, group := range a.groups {
		if !isCommonGroup(group.name) {
			continue
		}

		foundGroup = true

		first := int(group.firstFile)
		last := int(group.lastFile)

		if first < 0 || last < first {
			continue
		}

		if first >= len(a.files) {
			continue
		}

		if last >= len(a.files) {
			last = len(a.files) - 1
		}

		for i := first; i <= last; i++ {
			selected[i] = true
		}
	}

	if !foundGroup {
		for i, file := range a.files {
			if directoryContainsCommon(file.directory) {
				selected[i] = true
			}
		}
	}

	result := make([]int, 0, len(selected))

	for i := range selected {
		result = append(result, i)
	}

	sort.Ints(result)

	return result
}

func commonRelativePath(file installShieldFile) string {
	dir := strings.ReplaceAll(file.directory, "\\", "/")
	parts := strings.Split(dir, "/")

	common := -1

	for i, part := range parts {
		if strings.EqualFold(strings.TrimSpace(part), "Common") {
			common = i
			break
		}
	}

	var output []string

	if common >= 0 {
		output = append(output, parts[common+1:]...)
	} else {
		for _, part := range parts {
			part = strings.TrimSpace(part)

			if part == "" {
				continue
			}

			if strings.HasPrefix(part, "<") && strings.HasSuffix(part, ">") {
				continue
			}

			output = append(output, part)
		}
	}

	output = append(output, file.name)

	return strings.Join(output, "/")
}

func (a *installShieldArchive) extractFile(index int, target string) error {
	if index < 0 || index >= len(a.files) {
		return fmt.Errorf("invalid file index %d", index)
	}

	file := a.files[index]

	if file.flags&isFileSplit != 0 {
		return fmt.Errorf("split InstallShield volumes are not supported")
	}

	if file.dataOffset == 0 {
		return fmt.Errorf("file has no cabinet data offset")
	}

	inputSize := file.expandedSize
	if file.flags&isFileCompressed != 0 {
		inputSize = file.compressedSize
	}

	data := make([]byte, inputSize)

	if _, err := a.r.Seek(int64(file.dataOffset), io.SeekStart); err != nil {
		return err
	}

	if _, err := io.ReadFull(a.r, data); err != nil {
		return err
	}

	if file.flags&isFileObfuscated != 0 {
		deobfuscateInstallShield(data)
	}

	tmp := target + ".rahinstall.tmp"

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

	var written uint64

	if file.flags&isFileCompressed != 0 {
		n, err := writeInstallShieldCompressed(out, data)
		if err != nil {
			return err
		}
		written = n
	} else {
		n, err := out.Write(data)
		if err != nil {
			return err
		}
		written = uint64(n)
	}

	if written != uint64(file.expandedSize) {
		return fmt.Errorf("expanded size mismatch: expected %d, got %d", file.expandedSize, written)
	}

	if err := out.Close(); err != nil {
		return err
	}

	if err := replaceFile(tmp, target); err != nil {
		return err
	}

	ok = true
	return nil
}

func writeInstallShieldCompressed(out io.Writer, data []byte) (uint64, error) {
	position := 0
	var total uint64

	for position < len(data) {
		if len(data)-position < 2 {
			return total, fmt.Errorf("truncated compressed chunk header")
		}

		chunkSize := int(binary.LittleEndian.Uint16(data[position : position+2]))
		position += 2

		if chunkSize == 0 {
			return total, fmt.Errorf("zero-sized compressed chunk")
		}

		if chunkSize > len(data)-position {
			return total, fmt.Errorf("compressed chunk extends past file data")
		}

		chunk := data[position : position+chunkSize]
		position += chunkSize

		reader := flate.NewReader(bytes.NewReader(chunk))

		n, err := io.Copy(out, reader)
		closeErr := reader.Close()

		total += uint64(n)

		if err != nil {
			return total, fmt.Errorf("deflate chunk: %w", err)
		}

		if closeErr != nil {
			return total, closeErr
		}
	}

	return total, nil
}

func deobfuscateInstallShield(data []byte) {
	for i := range data {
		value := data[i] ^ 0xd5
		value = value>>2 | value<<6
		data[i] = value - byte(i%0x47)
	}
}
