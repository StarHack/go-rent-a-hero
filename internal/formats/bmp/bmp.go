// Package bmp implements a minimal Windows BMP decoder sufficient for the
// original game's scene background images: uncompressed 24-bit and 8-bit
// paletted BITMAPINFOHEADER files. All 93 background BMPs shipped with the
// game are uncompressed 24bpp 640x360.
//
// Alpha is not meaningful in BMP; decoded images are always fully opaque.
package bmp

import (
	"encoding/binary"
	"fmt"
)

// Image is a decoded, top-down RGBA8 image (unpremultiplied, alpha always
// 255).
type Image struct {
	Width  int
	Height int
	Pixels []byte // len == Width*Height*4, order R,G,B,A
}

const (
	fileHeaderSize = 14
	infoHeaderSize = 40
)

// Decode parses a BMP file into an RGBA8 image.
func Decode(data []byte) (*Image, error) {
	if len(data) < fileHeaderSize+infoHeaderSize {
		return nil, fmt.Errorf("bmp: file too short (%d bytes)", len(data))
	}

	if data[0] != 'B' || data[1] != 'M' {
		return nil, fmt.Errorf("bmp: missing BM magic, got %q", data[0:2])
	}

	dataOffset := binary.LittleEndian.Uint32(data[10:14])
	headerSize := binary.LittleEndian.Uint32(data[14:18])

	if headerSize != infoHeaderSize {
		return nil, fmt.Errorf("bmp: unsupported DIB header size %d, only BITMAPINFOHEADER (40) is supported", headerSize)
	}

	width := int(int32(binary.LittleEndian.Uint32(data[18:22])))
	rawHeight := int32(binary.LittleEndian.Uint32(data[22:26]))
	planes := binary.LittleEndian.Uint16(data[26:28])
	bpp := binary.LittleEndian.Uint16(data[28:30])
	compression := binary.LittleEndian.Uint32(data[30:34])

	if planes != 1 {
		return nil, fmt.Errorf("bmp: unsupported plane count %d", planes)
	}

	if compression != 0 {
		return nil, fmt.Errorf("bmp: unsupported compression %d, only BI_RGB (0) is supported", compression)
	}

	topDown := rawHeight < 0
	height := int(rawHeight)
	if topDown {
		height = -height
	}

	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("bmp: invalid dimensions %dx%d", width, height)
	}

	if uint64(width)*uint64(height) > 64*1024*1024 {
		return nil, fmt.Errorf("bmp: dimensions too large %dx%d", width, height)
	}

	switch bpp {
	case 24:
		return decode24(data, int(dataOffset), width, height, topDown)
	case 8:
		return decode8(data, int(dataOffset), width, height, topDown)
	default:
		return nil, fmt.Errorf("bmp: unsupported bit depth %d", bpp)
	}
}

// rowStride returns the padded row size in bytes for the given bit depth,
// per the BMP spec (rows are padded to a 4-byte boundary).
func rowStride(width, bpp int) int {
	bytesPerRow := (width*bpp + 7) / 8
	return (bytesPerRow + 3) &^ 3
}

func destRow(y, height int, topDown bool) int {
	if topDown {
		return y
	}
	return height - 1 - y
}

func decode24(data []byte, dataOffset, width, height int, topDown bool) (*Image, error) {
	stride := rowStride(width, 24)
	need := dataOffset + stride*height

	if len(data) < need {
		return nil, fmt.Errorf("bmp: truncated pixel data: need %d bytes, have %d", need, len(data))
	}

	img := &Image{Width: width, Height: height, Pixels: make([]byte, width*height*4)}

	for y := range height {
		srcRow := data[dataOffset+y*stride:]
		dstY := destRow(y, height, topDown)
		dstRow := img.Pixels[dstY*width*4:]

		for x := range width {
			b := srcRow[x*3+0]
			g := srcRow[x*3+1]
			r := srcRow[x*3+2]

			dstRow[x*4+0] = r
			dstRow[x*4+1] = g
			dstRow[x*4+2] = b
			dstRow[x*4+3] = 255
		}
	}

	return img, nil
}

func decode8(data []byte, dataOffset, width, height int, topDown bool) (*Image, error) {
	// Palette immediately follows the 40-byte info header, 4 bytes per
	// entry (B,G,R,reserved), up to 256 entries.
	paletteOffset := fileHeaderSize + infoHeaderSize
	paletteEntries := (dataOffset - paletteOffset) / 4

	if paletteEntries <= 0 || paletteOffset+paletteEntries*4 > len(data) {
		return nil, fmt.Errorf("bmp: invalid or missing palette")
	}

	palette := make([][4]byte, paletteEntries)
	for i := range palette {
		off := paletteOffset + i*4
		palette[i] = [4]byte{data[off+2], data[off+1], data[off], 255} // BGR -> RGB, force opaque
	}

	stride := rowStride(width, 8)
	need := dataOffset + stride*height

	if len(data) < need {
		return nil, fmt.Errorf("bmp: truncated pixel data: need %d bytes, have %d", need, len(data))
	}

	img := &Image{Width: width, Height: height, Pixels: make([]byte, width*height*4)}

	for y := range height {
		srcRow := data[dataOffset+y*stride:]
		dstY := destRow(y, height, topDown)
		dstRow := img.Pixels[dstY*width*4:]

		for x := range width {
			idx := srcRow[x]
			if int(idx) >= len(palette) {
				return nil, fmt.Errorf("bmp: palette index %d out of range (%d entries)", idx, len(palette))
			}

			c := palette[idx]
			dstRow[x*4+0] = c[0]
			dstRow[x*4+1] = c[1]
			dstRow[x*4+2] = c[2]
			dstRow[x*4+3] = c[3]
		}
	}

	return img, nil
}
