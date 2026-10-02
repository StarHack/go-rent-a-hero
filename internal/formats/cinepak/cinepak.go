// Package cinepak implements a native decoder for the Cinepak ("cvid")
// video codec used by this game's AVI cutscenes, so playback does not
// depend on an external tool (see agents/IMPLEMENTATION.md "AVI strategy").
//
// Cinepak is a vector-quantization codec: each frame is divided into
// horizontal strips, each strip maintains its own two codebooks (up to 256
// entries each) of small YUV vectors, and each 4x4 pixel block within a
// strip is drawn either by replicating one codebook entry's four
// sub-values across the block's four 2x2 quadrants ("V1") or by using four
// separate entries, one per quadrant, for finer detail ("V4"). Strips carry
// their codebook and pixel state forward between frames -- a block can be
// "skipped", meaning it keeps whatever pixels the previous frame left
// there -- so frames must be decoded in order by one persistent Decoder.
package cinepak

import (
	"encoding/binary"
	"fmt"
)

// codebookEntry holds one vector's four luma samples (one per quadrant,
// see quadColors) plus its shared signed chroma pair, exactly as they
// appear on the wire. A grayscale ("8-bit") chunk update sets U=V=0.
type codebookEntry struct {
	y    [4]byte
	u, v int8
}

type stripState struct {
	v1, v4 [256]codebookEntry
}

// Decoder holds the persistent per-strip codebook state and output pixel
// buffer that Cinepak's inter-frame prediction depends on. Frames must be
// fed to DecodeFrame strictly in playback order.
type Decoder struct {
	width, height int
	strips        []stripState
	frame         []byte // RGBA8, width*height*4, mutated in place across frames
}

// NewDecoder creates a decoder for a video of the given dimensions, with an
// initially black frame and empty (zeroed) codebooks -- the first frame of
// a well-formed Cinepak stream always fully populates every strip it uses
// via full codebook updates.
func NewDecoder(width, height int) *Decoder {
	return &Decoder{
		width:  width,
		height: height,
		frame:  make([]byte, width*height*4),
	}
}

// DecodeFrame decodes one compressed frame (a "##dc" chunk's payload, as
// produced by internal/formats/avi) into the decoder's persistent RGBA8
// buffer and returns a copy of it.
func (d *Decoder) DecodeFrame(data []byte) ([]byte, error) {
	if len(data) < 10 {
		return nil, fmt.Errorf("cinepak: frame too short: %d bytes", len(data))
	}

	flags := data[0]
	numStrips := int(binary.BigEndian.Uint16(data[8:10]))
	if numStrips < 0 || numStrips > 64 {
		return nil, fmt.Errorf("cinepak: implausible strip count %d", numStrips)
	}
	for len(d.strips) < numStrips {
		d.strips = append(d.strips, stripState{})
	}

	offset := 10
	y0 := 0

	for i := range numStrips {
		if offset+12 > len(data) {
			return nil, fmt.Errorf("cinepak: truncated strip header at strip %d", i)
		}

		header := data[offset : offset+12]
		stripID := header[0]
		stripLen := int(be24(header[1:4]))
		if stripLen < 12 {
			return nil, fmt.Errorf("cinepak: invalid strip length %d at strip %d", stripLen, i)
		}

		x1 := int(binary.BigEndian.Uint16(header[6:8]))
		x2 := int(binary.BigEndian.Uint16(header[10:12]))
		y1 := int(binary.BigEndian.Uint16(header[4:6]))
		y2 := int(binary.BigEndian.Uint16(header[8:10]))
		if y1 == 0 {
			// Zero means "top edge is wherever the previous strip ended",
			// and y2 is then a height rather than an absolute row.
			y1 = y0
			y2 += y0
		}

		if i > 0 && flags&0x01 == 0 {
			d.strips[i] = d.strips[i-1]
		}

		end := min(offset+stripLen, len(data))

		if err := d.decodeStrip(i, stripID, x1, y1, x2, y2, data[offset+12:end]); err != nil {
			return nil, fmt.Errorf("cinepak: strip %d: %w", i, err)
		}

		y0 = y2
		offset = end
	}

	out := make([]byte, len(d.frame))
	copy(out, d.frame)

	return out, nil
}

func (d *Decoder) decodeStrip(idx int, stripID byte, x1, y1, x2, y2 int, data []byte) error {
	_ = stripID // 0x10 (intra/key) vs 0x11 (inter): decoding is identical either way, since inheritance is driven by the frame-level flag byte, not this ID.

	strip := &d.strips[idx]
	offset := 0

	for offset+4 <= len(data) {
		chunkID := data[offset]
		chunkLen := int(be24(data[offset+1:offset+4])) - 4
		if chunkLen < 0 {
			return fmt.Errorf("invalid chunk length")
		}

		avail := len(data) - offset - 4
		if chunkLen > avail {
			chunkLen = avail
		}
		body := data[offset+4 : offset+4+chunkLen]

		switch chunkID {
		case 0x20, 0x21, 0x24, 0x25:
			updateCodebook(&strip.v4, chunkID, body)
		case 0x22, 0x23, 0x26, 0x27:
			updateCodebook(&strip.v1, chunkID, body)
		case 0x30, 0x31, 0x32:
			return decodeVectors(strip, chunkID, x1, y1, x2, y2, body, d.frame, d.width, d.height)
		}

		offset += 4 + chunkLen
	}

	return fmt.Errorf("strip has no vector chunk")
}

// cursor is Cinepak's shared bitstream cursor: codebook-update and
// block-vector chunks interleave whole-byte reads (codebook indices, entry
// data) with individual bits packed MSB-first into big-endian 32-bit words
// -- both drawn from the very same forward position, not two independent
// streams.
type cursor struct {
	data     []byte
	pos      int
	word     uint32
	bitsLeft uint
}

func (c *cursor) bit() (uint32, bool) {
	if c.bitsLeft == 0 {
		if c.pos+4 > len(c.data) {
			return 0, false
		}
		c.word = binary.BigEndian.Uint32(c.data[c.pos : c.pos+4])
		c.pos += 4
		c.bitsLeft = 32
	}
	c.bitsLeft--
	return (c.word >> c.bitsLeft) & 1, true
}

func (c *cursor) bytes(n int) ([]byte, bool) {
	if c.pos+n > len(c.data) {
		return nil, false
	}
	b := c.data[c.pos : c.pos+n]
	c.pos += n
	return b, true
}

// updateCodebook applies one codebook-update chunk to table. chunkID's bit
// 0x04 selects 4-byte grayscale (Y-only) entries over 6-byte YUV entries;
// bit 0x01 selects a "selective" update, where a run of MSB-first bits (one
// per table index, refilled 32 at a time from the same cursor the entry
// bytes are read from) marks which indices are replaced -- a "full" update
// simply replaces every index in sequence. Either way, running out of data
// before all 256 indices are visited just stops early, per real encoder
// output, which never bothers writing unused low-numbered trailing
// entries.
func updateCodebook(table *[256]codebookEntry, chunkID byte, data []byte) {
	entrySize := 6
	if chunkID&0x04 != 0 {
		entrySize = 4
	}
	selective := chunkID&0x01 != 0

	c := cursor{data: data}

	for i := range table {
		if selective {
			bit, ok := c.bit()
			if !ok {
				return
			}
			if bit == 0 {
				continue
			}
		}

		raw, ok := c.bytes(entrySize)
		if !ok {
			return
		}

		var e codebookEntry
		copy(e.y[:], raw[:4])
		if entrySize == 6 {
			e.u, e.v = int8(raw[4]), int8(raw[5])
		}
		table[i] = e
	}
}

// decodeVectors walks the strip's 4x4 blocks in raster order (left to
// right, top to bottom) and draws each one. chunkID selects how each
// block's coding is determined:
//
//   - 0x30: every block is coded (no skip); a second run of mode bits (from
//     the same shared cursor) says V1 (0) or V4 (1) per block.
//   - 0x31: one bit per block says skip (0) or coded (1); if coded, the
//     next bit (same running bitstream) says V1 (0) or V4 (1) -- i.e. the
//     classic "0 = skip, 10 = V1, 11 = V4" variable-length code.
//   - 0x32: every block is coded as V1; no mode/skip bits at all.
//
// A skipped block is left untouched, so it keeps whatever the previous
// frame drew there.
func decodeVectors(strip *stripState, chunkID byte, x1, y1, x2, y2 int, data, frame []byte, width, height int) error {
	c := cursor{data: data}
	selective := chunkID&0x01 != 0
	allV1 := chunkID&0x02 != 0

	for by := y1; by < y2; by += 4 {
		for bx := x1; bx < x2; bx += 4 {
			coded := true
			if selective {
				bit, ok := c.bit()
				if !ok {
					return fmt.Errorf("truncated skip flags")
				}
				coded = bit != 0
			}
			if !coded {
				continue
			}

			useV1 := allV1
			if !allV1 {
				bit, ok := c.bit()
				if !ok {
					return fmt.Errorf("truncated mode flags")
				}
				useV1 = bit == 0
			}

			if useV1 {
				idx, ok := c.bytes(1)
				if !ok {
					return fmt.Errorf("truncated V1 index")
				}
				writeV1Block(frame, width, height, bx, by, &strip.v1[idx[0]])
			} else {
				idx, ok := c.bytes(4)
				if !ok {
					return fmt.Errorf("truncated V4 indices")
				}
				writeV4Block(frame, width, height, bx, by,
					&strip.v4[idx[0]], &strip.v4[idx[1]], &strip.v4[idx[2]], &strip.v4[idx[3]])
			}
		}
	}

	return nil
}

type rgb struct{ r, g, b byte }

// quadColors converts one codebook entry's four luma samples (indexed
// top-left, top-right, bottom-left, bottom-right) plus its shared chroma
// into four independent RGB colors, applying the same fixed YUV
// coefficients Cinepak has always used: R = Y+2V, G = Y-U/2-V, B = Y+2U.
func quadColors(e *codebookEntry) [4]rgb {
	u, v := int(e.u), int(e.v)
	uvr := v << 1
	uvg := -((u + 1) >> 1) - v
	uvb := u << 1

	var out [4]rgb
	for i, y := range e.y {
		yy := int(y)
		out[i] = rgb{
			r: clip8(yy + uvr),
			g: clip8(yy + uvg),
			b: clip8(yy + uvb),
		}
	}
	return out
}

func clip8(v int) byte {
	switch {
	case v < 0:
		return 0
	case v > 255:
		return 255
	default:
		return byte(v)
	}
}

// writeV1Block draws a V1-coded 4x4 block: entry's four quadrant colors
// are each replicated across the corresponding 2x2 quadrant of the output
// block, giving Cinepak's characteristic blocky look for flat areas.
func writeV1Block(frame []byte, width, height, bx, by int, entry *codebookEntry) {
	c := quadColors(entry)
	fill2x2(frame, width, height, bx+0, by+0, c[0]) // top-left
	fill2x2(frame, width, height, bx+2, by+0, c[1]) // top-right
	fill2x2(frame, width, height, bx+0, by+2, c[2]) // bottom-left
	fill2x2(frame, width, height, bx+2, by+2, c[3]) // bottom-right
}

// writeV4Block draws a V4-coded 4x4 block: each of the four entries (one
// per output quadrant) contributes its own four quadrant colors directly
// to that 2x2 area's individual pixels, giving finer detail than V1.
func writeV4Block(frame []byte, width, height, bx, by int, tl, tr, bl, br *codebookEntry) {
	writeQuad(frame, width, height, bx+0, by+0, tl)
	writeQuad(frame, width, height, bx+2, by+0, tr)
	writeQuad(frame, width, height, bx+0, by+2, bl)
	writeQuad(frame, width, height, bx+2, by+2, br)
}

func writeQuad(frame []byte, width, height, x, y int, entry *codebookEntry) {
	c := quadColors(entry)
	setPixel(frame, width, height, x+0, y+0, c[0])
	setPixel(frame, width, height, x+1, y+0, c[1])
	setPixel(frame, width, height, x+0, y+1, c[2])
	setPixel(frame, width, height, x+1, y+1, c[3])
}

func fill2x2(frame []byte, width, height, x, y int, c rgb) {
	setPixel(frame, width, height, x+0, y+0, c)
	setPixel(frame, width, height, x+1, y+0, c)
	setPixel(frame, width, height, x+0, y+1, c)
	setPixel(frame, width, height, x+1, y+1, c)
}

func setPixel(frame []byte, width, height, x, y int, c rgb) {
	if x < 0 || y < 0 || x >= width || y >= height {
		return
	}
	i := (y*width + x) * 4
	frame[i+0] = c.r
	frame[i+1] = c.g
	frame[i+2] = c.b
	frame[i+3] = 255
}

func be24(b []byte) int {
	return int(b[0])<<16 | int(b[1])<<8 | int(b[2])
}
