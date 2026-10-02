// Package tcc decodes the game's custom compressed image format.
//
// TCC stores three colour planes and, when present, a fourth alpha plane
// behind a fixed 28-byte header. A16 animation frames reuse exactly the same
// frame record and decoder.
//
// This implementation follows the original decoder in dec(2).c:
//   - FUN_004380b0: normal command/RLE plane decoder
//   - FUN_00438800: alternate packed-nibble RGB decoder
//   - FUN_0043b740: frame/plane dispatch
//
// Important original behaviour preserved here:
//   - decode buffers are zero-initialized and include 16 bytes of padding;
//   - bit 0 of Flags selects the packed-nibble decoder for RGB only;
//   - alpha, when present, always uses the normal decoder;
//   - no RGB matte-key heuristic is applied;
//   - a compressed stream may legitimately leave zero-filled output bytes.
package tcc

import (
	"encoding/binary"
	"fmt"
	"io"
)

const HeaderSize = 28

type Header struct {
	Flags       uint32
	StreamSize0 uint32
	StreamSize1 uint32
	StreamSize2 uint32
	StreamSize3 uint32
	Width       uint32
	Height      uint32
}

const FlagAltCodec uint32 = 1

const (
	MaxDimension = 4096
	MaxPixels    = 16 * 1024 * 1024
	decodePad    = 16
)

type Image struct {
	Width  int
	Height int
	Pixels []byte
}

func ParseHeader(buf []byte) (Header, error) {
	if len(buf) < HeaderSize {
		return Header{}, fmt.Errorf("tcc: header: %w", io.ErrUnexpectedEOF)
	}
	return Header{
		Flags:       binary.LittleEndian.Uint32(buf[0:4]),
		StreamSize0: binary.LittleEndian.Uint32(buf[4:8]),
		StreamSize1: binary.LittleEndian.Uint32(buf[8:12]),
		StreamSize2: binary.LittleEndian.Uint32(buf[12:16]),
		StreamSize3: binary.LittleEndian.Uint32(buf[16:20]),
		Width:       binary.LittleEndian.Uint32(buf[20:24]),
		Height:      binary.LittleEndian.Uint32(buf[24:28]),
	}, nil
}

func FrameSize(h Header) (int, error) {
	if h.Width == 0 || h.Height == 0 {
		return 0, fmt.Errorf("tcc: invalid dimensions %dx%d", h.Width, h.Height)
	}
	if h.Width > MaxDimension || h.Height > MaxDimension {
		return 0, fmt.Errorf("tcc: dimensions too large %dx%d", h.Width, h.Height)
	}
	pixels := uint64(h.Width) * uint64(h.Height)
	if pixels > MaxPixels {
		return 0, fmt.Errorf("tcc: pixel count %d exceeds limit", pixels)
	}
	total := uint64(HeaderSize)
	for _, n := range [...]uint32{h.StreamSize0, h.StreamSize1, h.StreamSize2, h.StreamSize3} {
		total += uint64(n)
	}
	if total > 1<<32 {
		return 0, fmt.Errorf("tcc: frame size overflow")
	}
	return int(total), nil
}

func Decode(data []byte) (*Image, error) {
	h, err := ParseHeader(data)
	if err != nil {
		return nil, err
	}
	size, err := FrameSize(h)
	if err != nil {
		return nil, err
	}
	if len(data) < size {
		return nil, fmt.Errorf("tcc: truncated file: need %d bytes, have %d: %w", size, len(data), io.ErrUnexpectedEOF)
	}
	return DecodeFrame(h, data[HeaderSize:size])
}

func DecodeFrame(h Header, streams []byte) (*Image, error) {
	return DecodeFrameOver(h, streams, nil)
}

func DecodeFrameOver(h Header, streams []byte, base *Image) (*Image, error) {
	pixelCount, err := checkedPixelCount(h)
	if err != nil {
		return nil, err
	}

	if base != nil && (base.Width != int(h.Width) || base.Height != int(h.Height) || len(base.Pixels) != pixelCount*4) {
		return nil, fmt.Errorf("tcc: base image dimensions do not match %dx%d", h.Width, h.Height)
	}

	sizes := [...]uint32{h.StreamSize0, h.StreamSize1, h.StreamSize2, h.StreamSize3}
	expected := uint64(0)
	for _, n := range sizes {
		expected += uint64(n)
	}
	if uint64(len(streams)) != expected {
		return nil, fmt.Errorf("tcc: stream length mismatch: have %d, want %d", len(streams), expected)
	}

	var planes [4][]byte
	for i := 0; i < 4; i++ {
		planes[i] = make([]byte, pixelCount+decodePad)
	}

	if base != nil {
		for i := 0; i < pixelCount; i++ {
			planes[0][i] = base.Pixels[i*4]
			planes[1][i] = base.Pixels[i*4+1]
			planes[2][i] = base.Pixels[i*4+2]
			planes[3][i] = base.Pixels[i*4+3]
		}
	}

	offset := 0
	for i, compressedSize := range sizes {
		if compressedSize == 0 {
			continue
		}

		end := offset + int(compressedSize)
		if end < offset || end > len(streams) {
			return nil, fmt.Errorf("tcc: plane %d stream exceeds frame", i)
		}
		src := streams[offset:end]
		offset = end

		if i < 3 && h.Flags&FlagAltCodec != 0 && base != nil {
			packExpandedNibbles(planes[i], pixelCount)
		}

		if err := decodePlaneInto(src, planes[i]); err != nil {
			return nil, fmt.Errorf("tcc: plane %d: %w", i, err)
		}

		if i < 3 && h.Flags&FlagAltCodec != 0 {
			expandPackedNibblesExact(planes[i], pixelCount)
		}
	}

	rgba := make([]byte, pixelCount*4)
	for i := 0; i < pixelCount; i++ {
		rgba[i*4] = planes[0][i]
		rgba[i*4+1] = planes[1][i]
		rgba[i*4+2] = planes[2][i]
		if h.StreamSize3 != 0 || base != nil {
			rgba[i*4+3] = planes[3][i]
		} else {
			rgba[i*4+3] = 0xff
		}
	}

	return &Image{Width: int(h.Width), Height: int(h.Height), Pixels: rgba}, nil
}

func checkedPixelCount(h Header) (int, error) {
	if h.Width == 0 || h.Height == 0 {
		return 0, fmt.Errorf("tcc: invalid dimensions %dx%d", h.Width, h.Height)
	}
	if h.Width > MaxDimension || h.Height > MaxDimension {
		return 0, fmt.Errorf("tcc: dimensions too large %dx%d", h.Width, h.Height)
	}
	n := uint64(h.Width) * uint64(h.Height)
	if n > MaxPixels {
		return 0, fmt.Errorf("tcc: pixel count %d exceeds limit", n)
	}
	return int(n), nil
}

// decodePlaneExact is a safe translation of FUN_004380b0.
//
// Command forms:
//
//	bit0 set:              [flags][count][value] => repeat
//	bit0 clear, bit1 set:  [flags][count]        => skip zero-filled output
//	both clear:            [flags][count][data]  => literal
//
// The original terminates when the compressed input is exhausted. It does not
// require every output byte to have been explicitly written; untouched bytes
// remain zero because the allocation is zero-initialized.
func decodePlaneExact(src []byte, pixelCount int) ([]byte, error) {
	dst := make([]byte, pixelCount+decodePad)
	if err := decodePlaneInto(src, dst); err != nil {
		return nil, err
	}
	return dst, nil
}

func decodePlaneInto(src, dst []byte) error {
	si := 0
	di := 0

	for si < len(src) {
		op := src[si]

		switch {
		case op&1 != 0:
			if len(src)-si < 3 {
				return fmt.Errorf("repeat at compressed offset %d: %w", si, io.ErrUnexpectedEOF)
			}
			count := int(src[si+1])
			value := src[si+2]
			if di+count > len(dst) {
				return fmt.Errorf("tcc: repeat overflow at di=%d count=%d", di, count)
			}
			for j := 0; j < count; j++ {
				dst[di+j] = value
			}
			di += count
			si += 3

		case op&2 != 0:
			if len(src)-si < 2 {
				return fmt.Errorf("skip at compressed offset %d: %w", si, io.ErrUnexpectedEOF)
			}
			count := int(src[si+1])
			if di+count > len(dst) {
				return fmt.Errorf("tcc: skip overflow at di=%d count=%d", di, count)
			}
			di += count
			si += 2

		default:
			if len(src)-si < 2 {
				return fmt.Errorf("literal header at compressed offset %d: %w", si, io.ErrUnexpectedEOF)
			}
			count := int(src[si+1])
			dataStart := si + 2
			dataEnd := dataStart + count
			if dataEnd > len(src) {
				return fmt.Errorf("literal at compressed offset %d count=%d: %w", si, count, io.ErrUnexpectedEOF)
			}
			if di+count > len(dst) {
				return fmt.Errorf("tcc: literal overflow at di=%d count=%d", di, count)
			}
			copy(dst[di:di+count], src[dataStart:dataEnd])
			di += count
			si = dataEnd
		}
	}

	return nil
}

func packExpandedNibbles(dst []byte, pixelCount int) {
	packedCount := (pixelCount + 1) / 2
	for i := 0; i < packedCount; i++ {
		even := i * 2
		v := dst[even] & 0xf0
		if even+1 < pixelCount {
			v |= dst[even+1] >> 4
		}
		dst[i] = v
	}
}

// expandPackedNibblesExact mirrors FUN_00438800. The original applies this
// only to RGB planes when Flags bit 0 is set. Alpha always stays 8-bit.
func expandPackedNibblesExact(dst []byte, pixelCount int) {
	n := pixelCount
	for {
		out := n - 2
		if out <= 0 {
			return
		}

		packedIndex := out / 2
		v := dst[packedIndex]

		dst[out] = v & 0xf0
		dst[n-1] = v << 4

		n = out
	}
}
