// Package wav implements a minimal RIFF/WAVE reader sufficient for the
// original game's speech and sound-effect assets: PCM format tag 1,
// mono/stereo, 8-bit unsigned or 16-bit signed little-endian samples at an
// arbitrary source sample rate.
package wav

import (
	"encoding/binary"
	"fmt"
)

// FormatPCM is the only WAVE format tag this reader supports.
const FormatPCM = 1

// Sound is a decoded PCM WAV file. Samples holds raw interleaved sample
// bytes exactly as stored (BitsPerSample/8 bytes per sample per channel);
// callers convert to their mixer format.
type Sound struct {
	Channels      int
	SampleRate    int
	BitsPerSample int
	Samples       []byte
}

// Frames returns the number of sample frames (one frame = one sample per
// channel).
func (s *Sound) Frames() int {
	bytesPerFrame := s.Channels * (s.BitsPerSample / 8)
	if bytesPerFrame == 0 {
		return 0
	}

	return len(s.Samples) / bytesPerFrame
}

// Duration returns the playback duration in seconds.
func (s *Sound) Duration() float64 {
	if s.SampleRate == 0 {
		return 0
	}

	return float64(s.Frames()) / float64(s.SampleRate)
}

// Parse decodes a RIFF/WAVE file.
func Parse(data []byte) (*Sound, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("wav: file too short for RIFF header")
	}

	if string(data[0:4]) != "RIFF" {
		return nil, fmt.Errorf("wav: missing RIFF magic, got %q", data[0:4])
	}

	if string(data[8:12]) != "WAVE" {
		return nil, fmt.Errorf("wav: missing WAVE magic, got %q", data[8:12])
	}

	var (
		haveFmt       bool
		channels      uint16
		sampleRate    uint32
		bitsPerSample uint16
		formatTag     uint16
		samples       []byte
		haveData      bool
	)

	offset := 12

	for offset+8 <= len(data) {
		id := string(data[offset : offset+4])
		size := binary.LittleEndian.Uint32(data[offset+4 : offset+8])
		body := offset + 8

		if uint64(body)+uint64(size) > uint64(len(data)) {
			if offset > 12 {
				alt := offset - 1
				altSize := binary.LittleEndian.Uint32(data[alt+4 : alt+8])
				altBody := alt + 8
				if uint64(altBody)+uint64(altSize) <= uint64(len(data)) {
					offset = alt
					continue
				}
			}
			if haveFmt && haveData {
				break
			}
			return nil, fmt.Errorf("wav: chunk %q at offset %d declares size %d beyond EOF", id, offset, size)
		}

		switch id {
		case "fmt ":
			if size < 16 {
				return nil, fmt.Errorf("wav: fmt chunk too small: %d bytes", size)
			}

			formatTag = binary.LittleEndian.Uint16(data[body : body+2])
			channels = binary.LittleEndian.Uint16(data[body+2 : body+4])
			sampleRate = binary.LittleEndian.Uint32(data[body+4 : body+8])
			bitsPerSample = binary.LittleEndian.Uint16(data[body+14 : body+16])
			haveFmt = true

		case "data":
			samples = data[body : body+int(size)]
			haveData = true
		}

		// Chunks are word-aligned; skip a pad byte if size is odd.
		advance := int(size)
		if size%2 != 0 {
			advance++
		}

		offset = body + advance
	}

	if !haveFmt {
		return nil, fmt.Errorf("wav: missing fmt chunk")
	}

	if !haveData {
		return nil, fmt.Errorf("wav: missing data chunk")
	}

	if formatTag != FormatPCM {
		return nil, fmt.Errorf("wav: unsupported format tag %d, only PCM (1) is supported", formatTag)
	}

	if bitsPerSample != 8 && bitsPerSample != 16 {
		return nil, fmt.Errorf("wav: unsupported bits per sample %d", bitsPerSample)
	}

	if channels != 1 && channels != 2 {
		return nil, fmt.Errorf("wav: unsupported channel count %d", channels)
	}

	return &Sound{
		Channels:      int(channels),
		SampleRate:    int(sampleRate),
		BitsPerSample: int(bitsPerSample),
		Samples:       append([]byte(nil), samples...),
	}, nil
}

// ToInt16Mono converts the sound to a mono slice of signed 16-bit samples,
// downmixing stereo by averaging channels and rescaling 8-bit unsigned PCM
// to the signed 16-bit range.
func (s *Sound) ToInt16Mono() []int16 {
	frames := s.Frames()
	out := make([]int16, frames)

	switch s.BitsPerSample {
	case 8:
		for i := range frames {
			var sum int
			for c := range s.Channels {
				u := s.Samples[i*s.Channels+c]
				sum += (int(u) - 128) * 256
			}
			out[i] = int16(sum / s.Channels)
		}

	case 16:
		for i := range frames {
			var sum int
			for c := range s.Channels {
				off := (i*s.Channels + c) * 2
				v := int16(binary.LittleEndian.Uint16(s.Samples[off : off+2]))
				sum += int(v)
			}
			out[i] = int16(sum / s.Channels)
		}
	}

	return out
}
