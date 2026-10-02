// Package video plays this game's original AVI cutscenes and animated
// overlays directly, decoding them natively -- Cinepak ("cvid") via
// internal/formats/cinepak, Indeo5 ("IV50") via internal/formats/indeo5 --
// with no external tool or conversion step, per
// agents/IMPLEMENTATION.md "AVI strategy".
package video

import (
	"fmt"
	"os"

	"github.com/wok/rent-a-hero/internal/formats/avi"
	"github.com/wok/rent-a-hero/internal/formats/cinepak"
	"github.com/wok/rent-a-hero/internal/formats/indeo5"
)

// frameDecoder is the common shape of this package's supported codec
// decoders (internal/formats/cinepak.Decoder and
// internal/formats/indeo5.Decoder): stateful, since both formats are
// delta-coded and must be fed frames strictly in order.
type frameDecoder interface {
	DecodeFrame(data []byte) ([]byte, error)
}

// Source decodes one AVI cutscene/overlay on demand, implementing
// engine.SpriteSource. Both supported codecs are delta-coded -- a skipped
// or inherited block reuses the previous frame's data -- so frames can
// only be produced by decoding forward from a persistent decoder. Source
// keeps only the single most recently decoded frame; playback in this
// engine is always sequential (see engine.PlayLayerOnce), so that is all a
// normal cutscene ever needs, but an out-of-order request is still handled
// correctly by restarting the decoder from frame 0.
type Source struct {
	stream *avi.VideoStream
	dec    frameDecoder

	decoded     int // index of the currently cached decode (raw stream index), or -1
	decodedRGBA []byte
}

func (s *Source) ColorKey() (r, g, b byte, ok bool) {
	return 0, 0, 0, true
}

// Load reads and parses path (an original .AVI file) into a Source ready
// for on-demand frame decoding. The whole (compressed) file is read into
// memory upfront -- a few tens of MB at most for this game's cutscenes --
// but decoded frames are produced lazily, one at a time.
func Load(path string) (*Source, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("video: read %s: %w", path, err)
	}

	stream, err := avi.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("video: parse %s: %w", path, err)
	}

	dec, err := newDecoder(stream)
	if err != nil {
		return nil, fmt.Errorf("video: %s: %w", path, err)
	}

	return &Source{stream: stream, dec: dec, decoded: -1}, nil
}

func newDecoder(stream *avi.VideoStream) (frameDecoder, error) {
	switch stream.Compression {
	case "cvid":
		return cinepak.NewDecoder(stream.Width, stream.Height), nil
	case "IV50":
		return indeo5.NewDecoder(stream.Width, stream.Height), nil
	default:
		return nil, fmt.Errorf("unsupported codec %q (only Cinepak (\"cvid\") and Indeo5 (\"IV50\") are implemented)", stream.Compression)
	}
}

func (s *Source) Width() int  { return s.stream.Width }
func (s *Source) Height() int { return s.stream.Height }

// Frames reports the AVI stream's authored frame count. Matte detection does
// not renumber the stream; controller frame/range indices remain raw.
func (s *Source) Frames() int { return len(s.stream.Frames) }

func (s *Source) FPS() float64 { return s.stream.FPS }

// RGBA decodes (and caches) raw AVI frame n as unpremultiplied RGBA8.
// Matte-enabled overlays keep the same frame numbering and only alter alpha.
func (s *Source) RGBA(n int) ([]byte, error) {
	if n < 0 || n >= s.Frames() {
		return nil, fmt.Errorf("video: frame %d out of range [0,%d)", n, s.Frames())
	}

	real := n

	if real == s.decoded {
		return s.decodedRGBA, nil
	}

	if real < s.decoded {
		dec, err := newDecoder(s.stream)
		if err != nil {
			return nil, err // unreachable: Load already validated the codec
		}
		s.dec = dec
		s.decoded = -1
	}

	var rgba []byte
	for i := s.decoded + 1; i <= real; i++ {
		frame, err := s.dec.DecodeFrame(s.stream.Frames[i])
		if err != nil {
			return nil, fmt.Errorf("video: decode frame %d: %w", i, err)
		}
		rgba = frame
	}

	s.decoded = real
	s.decodedRGBA = rgba

	return rgba, nil
}
