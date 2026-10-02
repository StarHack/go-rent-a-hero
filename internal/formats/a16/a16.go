// Package a16 parses the game's multi-frame sprite animation format.
//
// An A16 file is a plain concatenation of TCC-style frame records: there is
// no file-level header or frame count. Frame boundaries are derived by
// walking each frame's 28-byte header and stream sizes until EOF.
package a16

import (
	"fmt"
	"io"

	"github.com/wok/rent-a-hero/internal/formats/tcc"
)

// Frame is one indexed frame within an animation: its byte offset into the
// original file data and its parsed header.
type Frame struct {
	Offset int
	Header tcc.Header
}

// Animation is an indexed, lazily-decodable A16 sprite sheet.
type Animation struct {
	Data   []byte
	Frames []Frame
}

// ErrTrailingData is returned when frame records do not exactly cover the
// input buffer.
var ErrTrailingData = fmt.Errorf("a16: trailing data after last frame")

// Parse indexes every frame in data without decoding pixels.
func Parse(data []byte) (*Animation, error) {
	a := &Animation{Data: data}
	offset := 0

	for offset < len(data) {
		if len(data)-offset < tcc.HeaderSize {
			return nil, fmt.Errorf("a16: frame %d header: %w", len(a.Frames), io.ErrUnexpectedEOF)
		}

		h, err := tcc.ParseHeader(data[offset : offset+tcc.HeaderSize])
		if err != nil {
			return nil, fmt.Errorf("a16: frame %d: %w", len(a.Frames), err)
		}

		size, err := tcc.FrameSize(h)
		if err != nil {
			return nil, fmt.Errorf("a16: frame %d: %w", len(a.Frames), err)
		}

		if size < tcc.HeaderSize || offset+size > len(data) {
			return nil, fmt.Errorf("a16: frame %d: invalid frame size %d at offset %d", len(a.Frames), size, offset)
		}

		a.Frames = append(a.Frames, Frame{Offset: offset, Header: h})

		offset += size
	}

	if offset != len(data) {
		return nil, ErrTrailingData
	}

	return a, nil
}

// Count returns the number of indexed frames.
func (a *Animation) Count() int {
	return len(a.Frames)
}

// Decode fully decodes frame n into an RGBA image.
func (a *Animation) Decode(n int) (*tcc.Image, error) {
	if n < 0 || n >= len(a.Frames) {
		return nil, fmt.Errorf("a16: frame index %d out of range [0,%d)", n, len(a.Frames))
	}

	start := n
	for start > 0 && a.Frames[start].Header.Flags&0x10 != 0 {
		start--
	}

	var img *tcc.Image
	for i := start; i <= n; i++ {
		f := a.Frames[i]
		size, err := tcc.FrameSize(f.Header)
		if err != nil {
			return nil, fmt.Errorf("a16: frame %d: %w", i, err)
		}

		streams := a.Data[f.Offset+tcc.HeaderSize : f.Offset+size]
		img, err = tcc.DecodeFrameOver(f.Header, streams, img)
		if err != nil {
			return nil, fmt.Errorf("a16: frame %d: %w", i, err)
		}
	}

	return img, nil
}
