// Package zbf loads the game's scene scalar-raster container.
//
// The same container format is used for both depth buffers and walk/path
// buffers; the semantic role comes entirely from which SZN key
// (zBuf vs wBuf) references the file, not from anything in the file itself.
package zbf

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// MinHeaderSize is the smallest observed ZBF header (the "minimal" variant).
const MinHeaderSize = 20

// ExtendedHeaderSize is the larger observed ZBF header, used by the
// majority of the full game's scenes (a full asset scan found it on 100 of
// 119 shipped .ZBF files, far more than the two minimal-header regression
// fixtures suggested). It carries the same five leading fields as the
// minimal header, plus explicit Width/Height and further fields that are
// not yet understood; those are preserved verbatim in Header.ExtraRaw.
const ExtendedHeaderSize = 48

// SceneWidth and SceneHeight are the confirmed dimensions used by every
// minimal-header ZBF payload (640*360 bytes after the header). Extended
// headers instead declare their own width/height.
const (
	SceneWidth  = 640
	SceneHeight = 360
)

// Header is the ZBF header, common fields first. The exact semantic names
// of Param1/Param2 are not proven; do not rename them to guessed concepts.
type Header struct {
	HeaderSize uint32
	Param1     float32
	MinValue   uint32
	Param2     float32
	MaxValue   uint32

	// Extended is true when HeaderSize == ExtendedHeaderSize. Width/Height
	// and ExtraRaw are only meaningful when Extended is true.
	Extended bool
	Width    uint32
	Height   uint32
	ExtraRaw []byte // raw bytes from offset 28 to HeaderSize, meaning unknown
}

// Raster is a decoded scalar raster: one unsigned byte per scene pixel.
type Raster struct {
	Header Header
	Width  int
	Height int
	Pixels []byte
}

// ErrUnsupportedHeader is returned when HeaderSize is neither MinHeaderSize
// nor ExtendedHeaderSize.
var ErrUnsupportedHeader = fmt.Errorf("zbf: unsupported header size")

// ErrUnexpectedDimensions is returned when the payload does not exactly
// match the expected width*height for this header variant.
var ErrUnexpectedDimensions = fmt.Errorf("zbf: payload size does not match declared dimensions")

func readHeader(buf []byte) Header {
	h := Header{
		HeaderSize: binary.LittleEndian.Uint32(buf[0:4]),
		Param1:     math.Float32frombits(binary.LittleEndian.Uint32(buf[4:8])),
		MinValue:   binary.LittleEndian.Uint32(buf[8:12]),
		Param2:     math.Float32frombits(binary.LittleEndian.Uint32(buf[12:16])),
		MaxValue:   binary.LittleEndian.Uint32(buf[16:20]),
	}

	if len(buf) >= ExtendedHeaderSize {
		h.Extended = true
		h.Width = binary.LittleEndian.Uint32(buf[20:24])
		h.Height = binary.LittleEndian.Uint32(buf[24:28])
		h.ExtraRaw = append([]byte(nil), buf[28:ExtendedHeaderSize]...)
	}

	return h
}

// Parse decodes a full ZBF file: either the minimal 20-byte header (fixed
// 640x360 payload) or the extended 48-byte header (payload sized by its own
// declared Width/Height).
func Parse(data []byte) (*Raster, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("zbf: %w", io.ErrUnexpectedEOF)
	}

	headerSize := binary.LittleEndian.Uint32(data[0:4])

	if headerSize != MinHeaderSize && headerSize != ExtendedHeaderSize {
		return nil, fmt.Errorf("%w: got %d", ErrUnsupportedHeader, headerSize)
	}

	if uint32(len(data)) < headerSize {
		return nil, fmt.Errorf("zbf: %w", io.ErrUnexpectedEOF)
	}

	h := readHeader(data[:headerSize])

	width, height := SceneWidth, SceneHeight
	if h.Extended {
		width, height = int(h.Width), int(h.Height)
	}

	payload := data[headerSize:]

	if width <= 0 || height <= 0 || len(payload) != width*height {
		return nil, fmt.Errorf("%w: got %d bytes, want %dx%d", ErrUnexpectedDimensions, len(payload), width, height)
	}

	pixels := append([]byte(nil), payload...)

	return &Raster{
		Header: h,
		Width:  width,
		Height: height,
		Pixels: pixels,
	}, nil
}

// At returns the raw scalar byte at (x, y). It panics on out-of-range
// coordinates, matching normal Go slice-indexing behavior.
func (r *Raster) At(x, y int) byte {
	return r.Pixels[y*r.Width+x]
}

// Value is an alias for At, matching the naming used for path semantics in
// agents/ZBF.md.
func (r *Raster) Value(x, y int) byte {
	return r.At(x, y)
}

// Walkable reports whether (x, y) is part of a walkable corridor when this
// raster is a path (wBuf) buffer: value != 0.
func (r *Raster) Walkable(x, y int) bool {
	return r.At(x, y) != 0
}

// InBounds reports whether (x, y) is within the raster.
func (r *Raster) InBounds(x, y int) bool {
	return x >= 0 && y >= 0 && x < r.Width && y < r.Height
}
