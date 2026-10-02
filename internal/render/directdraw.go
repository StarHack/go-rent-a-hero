package render

// The original renderer composes scene AVI layers through DirectDraw
// surfaces and DDBLT_KEYSRC.  Its normal low-colour/full-screen path uses a
// 16-bit display surface; keyed comparison therefore happens on the native
// RGB555 value after video conversion, not on our decoder's 24-bit RGB
// samples.  These helpers reproduce that surface-format step before SDL
// presents the already-composited result.

func directDrawRGB555(r, g, b byte) uint16 {
	return uint16(r>>3)<<10 | uint16(g>>3)<<5 | uint16(b>>3)
}

func directDrawExpandRGB555(v uint16) (byte, byte, byte) {
	// Expand the native DirectDraw value back to 8-bit channels for SDL.  Bit
	// replication is the usual lossless expansion of the stored 5/6-bit
	// components and avoids introducing filtering between source pixels.
	r5 := byte((v >> 10) & 0x1f)
	g5 := byte((v >> 5) & 0x1f)
	b5 := byte(v & 0x1f)
	return (r5 << 3) | (r5 >> 2), (g5 << 3) | (g5 >> 2), (b5 << 3) | (b5 >> 2)
}

func directDrawKeyedPixels(src []byte, keyR, keyG, keyB byte) []byte {
	out := make([]byte, len(src))
	copy(out, src)
	key := directDrawRGB555(keyR, keyG, keyB)

	for i := 0; i+3 < len(out); i += 4 {
		if out[i+3] == 0 {
			continue
		}

		pixel := directDrawRGB555(out[i], out[i+1], out[i+2])
		if pixel == key {
			out[i+3] = 0
			continue
		}

		// DrawDib/DirectDraw has already converted the source into the native
		// surface format before Blt evaluates DDBLT_KEYSRC.  Preserve that
		// same 16-bit colour result for keyed AVI layers instead of keying in
		// 32-bit and then displaying the original unquantized sample.
		out[i], out[i+1], out[i+2] = directDrawExpandRGB555(pixel)
		out[i+3] = 0xff
	}
	return out
}

func directDrawKeyMatch(r, g, b, keyR, keyG, keyB byte) bool {
	return directDrawRGB555(r, g, b) == directDrawRGB555(keyR, keyG, keyB)
}

func directDrawScaleExtent(size, zoomPercent int) int {
	if zoomPercent == 0 {
		zoomPercent = 100
	}
	// The original builds integer RECTs before Blt/StretchBlt.  Integer
	// division intentionally truncates just like the recovered __ftol-based
	// rectangle construction, avoiding sub-pixel texture rectangles.
	return size * zoomPercent / 100
}
