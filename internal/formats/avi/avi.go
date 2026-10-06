// Package avi implements a minimal RIFF/AVI container reader sufficient to
// pull the compressed video stream out of this game's cutscenes: stream
// dimensions, frame rate and codec FourCC (all Cinepak/"cvid" in this
// game), plus every compressed video frame's raw chunk bytes in playback
// order. It does not decode the video itself (see internal/formats/cinepak)
// or handle audio-in-AVI (none of this game's cutscenes have any; speech
// and SFX ship as separate .WAV files).
package avi

import (
	"encoding/binary"
	"fmt"
)

// VideoStream is the video stream extracted from one AVI file.
type VideoStream struct {
	Width, Height int
	BitCount      int
	Compression   string // FourCC, e.g. "cvid"
	FPS           float64

	// Frames holds each frame's raw compressed chunk payload (e.g. a
	// Cinepak bitstream frame), in playback order.
	Frames [][]byte

	streamIndex int // set once the video stream's strl position is known
}

// Parse extracts the video stream from data, an entire AVI file's bytes.
func Parse(data []byte) (*VideoStream, error) {
	if len(data) < 12 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "AVI " {
		return nil, fmt.Errorf("avi: missing RIFF/\"AVI \" magic")
	}

	vs := &VideoStream{streamIndex: -1}
	offset := 0
	segment := 0

	for offset+12 <= len(data) {
		if string(data[offset:offset+4]) != "RIFF" {
			break
		}

		riffSize := uint64(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		if riffSize < 4 {
			return nil, fmt.Errorf("avi: RIFF segment %d is too small: %d bytes", segment, riffSize)
		}

		bodyStart := uint64(offset) + 8
		bodyEnd := bodyStart + riffSize
		if bodyEnd > uint64(len(data)) {
			return nil, fmt.Errorf("avi: RIFF segment %d declares size %d beyond available %d bytes", segment, riffSize, len(data)-int(bodyStart))
		}

		formType := string(data[offset+8 : offset+12])
		if segment == 0 && formType != "AVI " {
			return nil, fmt.Errorf("avi: first RIFF segment has form %q, want \"AVI \"", formType)
		}
		if segment > 0 && formType != "AVIX" && formType != "AVI " {
			break
		}

		contentStart := offset + 12
		contentEnd := int(bodyEnd)
		err := walk(data[contentStart:contentEnd], func(id string, body []byte) error {
			listType, rest, ok := asList(id, body)
			if !ok {
				return nil
			}

			switch listType {
			case "hdrl":
				return parseHeaderList(rest, vs)
			case "movi":
				if vs.streamIndex < 0 {
					return fmt.Errorf("avi: \"movi\" list appears before stream headers")
				}
				return collectFrames(rest, vs.streamIndex, &vs.Frames)
			}

			return nil
		})
		if err != nil {
			return nil, err
		}

		next := bodyEnd + riffSize%2
		if next > uint64(len(data)) {
			break
		}
		offset = int(next)
		segment++
	}

	if vs.streamIndex < 0 {
		return nil, fmt.Errorf("avi: no video (\"vids\") stream found")
	}
	if len(vs.Frames) == 0 {
		return nil, fmt.Errorf("avi: video stream has no frames")
	}

	return vs, nil
}

// asList reports whether id/body is a "LIST" chunk, returning its list-type
// FourCC and the bytes following it.
func asList(id string, body []byte) (listType string, rest []byte, ok bool) {
	if id != "LIST" || len(body) < 4 {
		return "", nil, false
	}
	return string(body[0:4]), body[4:], true
}

// parseHeaderList walks the top-level "hdrl" LIST's children, numbering
// each "LIST strl" it finds in file order (AVI stream indices, used later
// to match "movi" chunk IDs like "00dc", are positional).
func parseHeaderList(data []byte, vs *VideoStream) error {
	streamIndex := 0

	return walk(data, func(id string, body []byte) error {
		listType, rest, ok := asList(id, body)
		if !ok || listType != "strl" {
			return nil
		}

		idx := streamIndex
		streamIndex++

		return parseStreamList(rest, idx, vs)
	})
}

// parseStreamList reads one "strl" LIST's "strh" (stream header) and, if
// it declares fccType "vids" and no video stream has been found yet, its
// "strf" (BITMAPINFOHEADER).
func parseStreamList(data []byte, idx int, vs *VideoStream) error {
	isVideo := false

	return walk(data, func(id string, body []byte) error {
		switch id {
		case "strh":
			if len(body) < 28 {
				return fmt.Errorf("avi: strh chunk too small: %d bytes", len(body))
			}
			if string(body[0:4]) != "vids" || vs.streamIndex >= 0 {
				return nil
			}

			scale := binary.LittleEndian.Uint32(body[20:24])
			rate := binary.LittleEndian.Uint32(body[24:28])
			if scale == 0 {
				return fmt.Errorf("avi: strh dwScale is 0")
			}

			isVideo = true
			vs.streamIndex = idx
			vs.FPS = float64(rate) / float64(scale)

		case "strf":
			if !isVideo {
				return nil
			}
			if len(body) < 20 {
				return fmt.Errorf("avi: strf (BITMAPINFOHEADER) chunk too small: %d bytes", len(body))
			}

			vs.Width = int(int32(binary.LittleEndian.Uint32(body[4:8])))
			vs.Height = abs32(int32(binary.LittleEndian.Uint32(body[8:12])))
			vs.BitCount = int(binary.LittleEndian.Uint16(body[14:16]))
			vs.Compression = string(body[16:20])
		}

		return nil
	})
}

// collectFrames walks one "movi" LIST (or nested "rec " LIST, used for
// interleaved multi-stream AVIs), appending every chunk belonging to
// streamIndex's compressed ("dc") or uncompressed ("db") video data to
// *frames, in order.
func collectFrames(data []byte, streamIndex int, frames *[][]byte) error {
	wantID := fmt.Sprintf("%02d", streamIndex)

	return walk(data, func(id string, body []byte) error {
		if listType, rest, ok := asList(id, body); ok {
			if listType == "rec " {
				return collectFrames(rest, streamIndex, frames)
			}
			return nil
		}

		if id[0:2] == wantID && (id[2:4] == "dc" || id[2:4] == "db") {
			*frames = append(*frames, body)
		}

		return nil
	})
}

// walk invokes fn once per RIFF chunk in data with that chunk's 4-byte ID
// and body (the bytes after its 8-byte id+size header, excluding any
// trailing pad byte). It does not recurse into "LIST" chunks; callers that
// care about a particular list's contents call walk again on body[4:]
// themselves (see asList).
func walk(data []byte, fn func(id string, body []byte) error) error {
	offset := 0

	for offset+8 <= len(data) {
		id := string(data[offset : offset+4])
		size := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		body := offset + 8

		if size < 0 || body+size > len(data) {
			return fmt.Errorf("avi: chunk %q at offset %d declares size %d beyond available %d bytes", id, offset, size, len(data)-body)
		}

		if err := fn(id, data[body:body+size]); err != nil {
			return err
		}

		advance := size + size%2
		offset = body + advance
	}

	return nil
}

func abs32(v int32) int {
	if v < 0 {
		return int(-v)
	}
	return int(v)
}
