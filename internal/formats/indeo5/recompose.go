package indeo5

// recompose53 reconstructs a scalable (4-band) plane's pixels from its
// wavelet bands, a direct port of ff_ivi_recompose53's biorthogonal 5/3
// synthesis filter. It processes the image in 2x2 pixel groups, tracking a
// sliding window of already-read neighboring band samples exactly as the
// original does (re-deriving those from scratch every pixel would also
// need the same neighbor logic, just recomputed instead of carried
// forward -- carrying it forward is only an optimization, not a behavior
// difference, so this keeps that structure for ease of comparing against
// the original rather than for its own sake).
//
// Band 0 (LL) gets a low-pass filter both directions; bands 1 (HL) and 2
// (LH) get a high-pass filter in one direction and low-pass in the other;
// band 3 (HH) gets a high-pass filter both directions -- the classic
// four-way split of a 2D wavelet transform, undone here by combining all
// four bands' contributions per output pixel.
func (d *Decoder) recompose53(p *plane, dst []byte, dstPitch int) {
	pitchFull := p.bands[0].pitch

	b0 := p.bands[0].buf
	b1 := p.bands[1].buf
	b2 := p.bands[2].buf
	b3 := p.bands[3].buf

	b0Base, b1Base, b2Base, b3Base := 0, 0, 0, 0
	backPitch := 0
	dstBase := 0

	for y := 0; y < p.height; y += 2 {
		pitch := pitchFull
		if y+2 >= p.height {
			pitch = 0
		}

		b01 := int32(b0[b0Base+0])
		b02 := int32(b0[b0Base+pitch])

		b11 := int32(b1[b1Base+backPitch])
		b12 := int32(b1[b1Base+0])
		b13 := b11 - b12*6 + int32(b1[b1Base+pitch])

		b22 := int32(b2[b2Base+0])
		b23 := b22
		b25 := int32(b2[b2Base+pitch])
		b26 := b25

		b32 := int32(b3[b3Base+backPitch])
		b33 := b32
		b35 := int32(b3[b3Base+0])
		b36 := b35
		b38 := b32 - b35*6 + int32(b3[b3Base+pitch])
		b39 := b38

		indx := 0
		for x := 0; x < p.width; x += 2 {
			if x+2 >= p.width {
				b0Base--
				b1Base--
				b2Base--
				b3Base--
			}

			b21, b24 := b22, b25
			b22, b25 = b23, b26

			b31, b34, b37 := b32, b35, b38
			b32, b35, b38 = b33, b36, b39

			var p0, p1, p2, p3 int32

			// LL band: low-pass both directions.
			tmp0, tmp2 := b01, b02
			b01 = int32(b0[b0Base+indx+1])
			b02 = int32(b0[b0Base+pitch+indx+1])
			tmp1 := tmp0 + b01

			p0 = tmp0 * 16
			p1 = tmp1 * 8
			p2 = (tmp0 + tmp2) * 8
			p3 = (tmp1 + tmp2 + b02) * 4

			// HL band: high-pass vertically, low-pass horizontally.
			tmp0, tmp1 = b12, b11
			b12 = int32(b1[b1Base+indx+1])
			b11 = int32(b1[b1Base+backPitch+indx+1])
			tmp2 = tmp1 - tmp0*6 + b13
			b13 = b11 - b12*6 + int32(b1[b1Base+pitch+indx+1])

			p0 += (tmp0 + tmp1) * 8
			p1 += (tmp0 + tmp1 + b11 + b12) * 4
			p2 += tmp2 * 4
			p3 += (tmp2 + b13) * 2

			// LH band: low-pass vertically, high-pass horizontally.
			b23 = int32(b2[b2Base+indx+1])
			b26 = int32(b2[b2Base+pitch+indx+1])
			tmp0 = b21 + b22
			tmp1 = b21 - b22*6 + b23

			p0 += tmp0 * 8
			p1 += tmp1 * 4
			p2 += (tmp0 + b24 + b25) * 4
			p3 += (tmp1 + b24 - b25*6 + b26) * 2

			// HH band: high-pass both directions.
			b36 = int32(b3[b3Base+indx+1])
			b33 = int32(b3[b3Base+backPitch+indx+1])
			tmp0 = b31 + b34
			tmp1 = b32 + b35
			tmp2 = b33 + b36
			b39 = b33 - b36*6 + int32(b3[b3Base+pitch+indx+1])

			p0 += (tmp0 + tmp1) * 4
			p1 += (tmp0 - tmp1*6 + tmp2) * 2
			p2 += (b37 + b38) * 2
			p3 += b37 - b38*6 + b39

			dst[dstBase+x] = clipUint8(int(p0>>6) + 128)
			dst[dstBase+x+1] = clipUint8(int(p1>>6) + 128)
			dst[dstBase+dstPitch+x] = clipUint8(int(p2>>6) + 128)
			dst[dstBase+dstPitch+x+1] = clipUint8(int(p3>>6) + 128)

			indx++
		}

		dstBase += dstPitch * 2
		backPitch = -pitch
		b0Base += pitch + 1
		b1Base += pitch + 1
		b2Base += pitch + 1
		b3Base += pitch + 1
	}
}
