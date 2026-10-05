package main

import (
	"bytes"
	"encoding/binary"
	"testing"
	"unicode/utf8"
)

func TestDecodeISONameCP850Umlaut(t *testing.T) {
	// 0x8E is Ä in CP850, the byte this disc uses in 038_MÄD_02.WAV.
	got := decodeISOName([]byte("038_M\x8eD_02.WAV"))
	if got != "038_MÄD_02.WAV" {
		t.Fatalf("decodeISOName = %q, want %q", got, "038_MÄD_02.WAV")
	}
	if !utf8.ValidString(got) {
		t.Fatal("decoded name is not valid UTF-8")
	}
}

func TestDecodeISONameLeavesASCIIAlone(t *testing.T) {
	got := decodeISOName([]byte("038_MAED_02.WAV"))
	if got != "038_MAED_02.WAV" {
		t.Fatalf("decodeISOName = %q, want ASCII name unchanged", got)
	}
}

func TestParseISORecordDecodesUmlautName(t *testing.T) {
	identifier := []byte("038_M\x8eD_02.WAV;1")
	record := make([]byte, 33+len(identifier))
	record[0] = byte(len(record))
	record[25] = 0
	record[32] = byte(len(identifier))
	copy(record[33:], identifier)

	node, err := parseISORecord(record)
	if err != nil {
		t.Fatal(err)
	}
	if node.name != "038_MÄD_02.WAV" {
		t.Fatalf("name = %q, want %q", node.name, "038_MÄD_02.WAV")
	}
}

func TestDecodeJolietIdentifier(t *testing.T) {
	input := encodeJolietTestName("S114_IntroPart1.avi;1")
	got, err := decodeJolietIdentifier(input)
	if err != nil {
		t.Fatal(err)
	}
	if got != "S114_IntroPart1.avi" {
		t.Fatalf("decodeJolietIdentifier = %q, want %q", got, "S114_IntroPart1.avi")
	}
}

func TestISOSourcePrefersJolietNames(t *testing.T) {
	image := make([]byte, 24*isoSectorSize)

	primary := image[16*isoSectorSize : 17*isoSectorSize]
	primary[0] = 1
	copy(primary[1:6], "CD001")
	primary[6] = 1
	copy(primary[156:], makeISOTestRecord([]byte{0}, true, 20, isoSectorSize))

	joliet := image[17*isoSectorSize : 18*isoSectorSize]
	joliet[0] = 2
	copy(joliet[1:6], "CD001")
	joliet[6] = 1
	copy(joliet[88:91], "%/E")
	copy(joliet[156:], makeISOTestRecord([]byte{0}, true, 21, isoSectorSize))

	terminator := image[18*isoSectorSize : 19*isoSectorSize]
	terminator[0] = 255
	copy(terminator[1:6], "CD001")
	terminator[6] = 1

	primaryDir := image[20*isoSectorSize : 21*isoSectorSize]
	primaryRecord := makeISOTestRecord([]byte("S114_I~1.AVI;1"), false, 22, 1)
	copy(primaryDir, primaryRecord)

	jolietDir := image[21*isoSectorSize : 22*isoSectorSize]
	jolietRecord := makeISOTestRecord(encodeJolietTestName("S114_IntroPart1.avi;1"), false, 22, 1)
	copy(jolietDir, jolietRecord)

	src, err := newISOSource(nil, bytes.NewReader(image), int64(len(image)))
	if err != nil {
		t.Fatal(err)
	}

	entries, err := src.ReadDir("")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("ReadDir returned %d entries, want 1", len(entries))
	}
	if entries[0].Name != "S114_IntroPart1.avi" {
		t.Fatalf("entry name = %q, want %q", entries[0].Name, "S114_IntroPart1.avi")
	}
}

func encodeJolietTestName(s string) []byte {
	result := make([]byte, 0, len(s)*2)
	for _, r := range s {
		result = append(result, byte(r>>8), byte(r))
	}
	return result
}

func makeISOTestRecord(name []byte, isDir bool, extent, size uint32) []byte {
	length := 33 + len(name)
	if length%2 != 0 {
		length++
	}
	record := make([]byte, length)
	record[0] = byte(length)
	binary.LittleEndian.PutUint32(record[2:6], extent)
	binary.LittleEndian.PutUint32(record[10:14], size)
	if isDir {
		record[25] = 2
	}
	record[32] = byte(len(name))
	copy(record[33:], name)
	return record
}
