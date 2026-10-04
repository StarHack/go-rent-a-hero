package indeo5

// outputRGBA converts the just-decoded planes to unpremultiplied RGBA8.
// Per ff_ivi_decode_frame, the second chroma plane (index 2 in this
// decoder, matching indeo5.c's own plane order) carries U and the first
// (index 1) carries V -- a naming quirk of the original decoder, not a
// mistake here.
func (d *Decoder) outputRGBA() []byte {
	y := make([]byte, d.picWidth*d.picHeight)
	if d.isScalable {
		d.recompose53(&d.planes[0], y, d.picWidth)
	} else {
		outputPlane(&d.planes[0], y, d.picWidth)
	}

	cw, ch := d.chromaWidth, d.chromaHeight
	u := make([]byte, cw*ch)
	v := make([]byte, cw*ch)
	outputPlane(&d.planes[2], u, cw)
	outputPlane(&d.planes[1], v, cw)

	out := make([]byte, d.picWidth*d.picHeight*4)

	for py := range d.picHeight {
		cy := min(py>>2, ch-1)
		for px := range d.picWidth {
			cx := min(px>>2, cw-1)

			o := (py*d.picWidth + px) * 4
			if d.hasTransparency && !d.transparencyMask[py*d.picWidth+px] {
				out[o+0] = d.transparencyFill[0]
				out[o+1] = d.transparencyFill[1]
				out[o+2] = d.transparencyFill[2]
				out[o+3] = 255
				continue
			}

			Y := int(y[py*d.picWidth+px]) - 16
			U := int(u[cy*cw+cx]) - 128
			V := int(v[cy*cw+cx]) - 128

			out[o+0] = clipUint8((298*Y + 409*V + 128) >> 8)
			out[o+1] = clipUint8((298*Y - 100*U - 208*V + 128) >> 8)
			out[o+2] = clipUint8((298*Y + 516*U + 128) >> 8)
			out[o+3] = 255
		}
	}

	return out
}

// outputPlane is ivi_output_plane: add back the encoder's 128 bias and
// clip to a byte.
func outputPlane(p *plane, dst []byte, dstPitch int) {
	band := &p.bands[0]
	src, pitch := band.buf, band.pitch

	for y := range p.height {
		row := src[y*pitch:]
		out := dst[y*dstPitch:]
		for x := range p.width {
			out[x] = clipUint8(int(row[x]) + 128)
		}
	}
}

func clipUint8(v int) byte {
	switch {
	case v < 0:
		return 0
	case v > 255:
		return 255
	default:
		return byte(v)
	}
}
