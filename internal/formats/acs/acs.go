// Package acs loads the speech mouth/viseme track format that accompanies
// spoken WAV files (e.g. 046_ROD_01.WAV / 046_ROD_01.ACS).
//
// A track is a time-sampled sequence of uint16 mouth-state indices, sampled
// at a fixed rate (20 Hz when no extended header is present).
package acs

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// DefaultRate is used when no extended header is present.
const DefaultRate = 20.0

// extendedMarker identifies the extended-header form: the first uint16 is
// 0xFFFF.
const extendedMarker = 0xFFFF

// Track is a parsed ACS mouth/viseme sample sequence.
type Track struct {
	Rate    float64
	Samples []uint16
}

// ErrOddPayload is returned when the sample payload is not a whole number of
// uint16 words.
var ErrOddPayload = fmt.Errorf("acs: payload length is odd")

// ErrInvalidOffset is returned when an extended header's payload offset is
// out of range.
var ErrInvalidOffset = fmt.Errorf("acs: invalid payload offset")

// ErrInvalidRate is returned when an extended header specifies a zero rate.
var ErrInvalidRate = fmt.Errorf("acs: invalid rate")

// Parse decodes an ACS track, handling both the headerless form (all bytes
// are samples, implicit 20 Hz) and the extended form (first uint16 ==
// 0xFFFF, followed by a payload offset and rate).
func Parse(data []byte) (*Track, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("acs: %w", io.ErrUnexpectedEOF)
	}

	offset := 0
	rate := DefaultRate

	if binary.LittleEndian.Uint16(data[:2]) == extendedMarker {
		if len(data) < 6 {
			return nil, fmt.Errorf("acs: extended header: %w", io.ErrUnexpectedEOF)
		}

		offset = int(binary.LittleEndian.Uint16(data[2:4]))
		rawRate := binary.LittleEndian.Uint16(data[4:6])

		if offset < 6 || offset > len(data) {
			return nil, fmt.Errorf("%w: %d", ErrInvalidOffset, offset)
		}

		if rawRate == 0 {
			return nil, ErrInvalidRate
		}

		rate = float64(rawRate)
	}

	payload := data[offset:]

	if len(payload)%2 != 0 {
		return nil, ErrOddPayload
	}

	samples := make([]uint16, len(payload)/2)

	for i := range samples {
		samples[i] = binary.LittleEndian.Uint16(payload[i*2 : i*2+2])
	}

	return &Track{Rate: rate, Samples: samples}, nil
}

// Duration returns the nominal playback duration implied by sample count and
// rate.
func (t *Track) Duration() float64 {
	return float64(len(t.Samples)) / t.Rate
}

// MouthStateAt returns the mouth/viseme state at playback time t (seconds).
// If t exceeds the track duration, it returns 0 (neutral).
func (t *Track) MouthStateAt(seconds float64) uint16 {
	if len(t.Samples) == 0 {
		return 0
	}

	index := max(int(math.Floor(seconds*t.Rate)), 0)

	if index >= len(t.Samples) {
		return 0
	}

	return t.Samples[index]
}
