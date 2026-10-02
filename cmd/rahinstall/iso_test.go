package main

import (
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
