package indeo5

// Inverse "slant" transforms (Indeo's variant of a Loeffler-style DCT-like
// transform), ported from ivi_dsp.c's IVI_INV_SLANT8/4 macros. band selects
// which of these applies per plane/band (see decodeGOPHeader): band 0 gets
// the full 2D transform, bands 1/2 get a row-only or column-only pass
// (their perpendicular direction was already handled by the wavelet
// decomposition itself), band 3 is pass-through (no frequency transform),
// and chroma's single band gets the 4x4 2D transform.
//
// in is the 64 (or 16) dequantized coefficients in raster order; out/pitch
// address the destination band buffer, and flags[i] gates column/row i
// exactly as FFmpeg's decoder does: an all-zero column can be skipped
// (written as zero) without running the transform on it.

func slantButterfly(s1, s2 int32) (o1, o2 int32) {
	return s1 + s2, s1 - s2
}

// ireflect is IVI_IREFLECT: a 1/2, 5/4 reflection step.
func ireflect(s1, s2 int32) (o1, o2 int32) {
	o1 = ((s1+s2*2+2)>>2 + s1)
	o2 = ((s1*2-s2+2)>>2 - s2)
	return
}

// slantPart4 is IVI_SLANT_PART4: a 1/2, 7/8 reflection step.
func slantPart4(s1, s2 int32) (o1, o2 int32) {
	o1 = s2 + ((s1*4 - s2 + 4) >> 3)
	o2 = s1 + ((-s1 - s2*4 + 4) >> 3)
	return
}

// invSlant8 is IVI_INV_SLANT8: an 8-point inverse slant transform.
func invSlant8(s1, s4, s8, s5, s2, s6, s3, s7 int32) (d1, d2, d3, d4, d5, d6, d7, d8 int32) {
	t4, t5 := slantPart4(s4, s5)

	t1, t5b := slantButterfly(s1, t5)
	t2, t6 := slantButterfly(s2, s6)
	t7, t3 := slantButterfly(s7, s3)
	t4b, t8 := slantButterfly(t4, s8)

	t1b, t2b := slantButterfly(t1, t2)
	t4c, t3b := ireflect(t4b, t3)
	t5c, t6b := slantButterfly(t5b, t6)
	t8b, t7b := ireflect(t8, t7)

	d1, d4 = slantButterfly(t1b, t4c)
	d2, d3 = slantButterfly(t2b, t3b)
	d5, d8 = slantButterfly(t5c, t8b)
	d6, d7 = slantButterfly(t6b, t7b)
	return
}

// invSlant4 is IVI_INV_SLANT4: a 4-point inverse slant transform.
func invSlant4(s1, s4, s2, s3 int32) (d1, d2, d3, d4 int32) {
	t1, t2 := slantButterfly(s1, s2)
	t4, t3 := ireflect(s4, s3)
	d1, d4 = slantButterfly(t1, t4)
	d2, d3 = slantButterfly(t2, t3)
	return
}

// compensate1 and compensate2 are the two COMPENSATE(x) variants used
// across ivi_dsp.c's transform passes: identity for the column pass,
// (x+1)>>1 for the row pass (the extra half-bit of precision the vertical
// pass alone doesn't need).
func compensate1(x int32) int32 { return x }
func compensate2(x int32) int32 { return (x + 1) >> 1 }

// inverseSlant8x8 is ff_ivi_inverse_slant_8x8: full 2D inverse transform.
func inverseSlant8x8(in []int32, out []int16, pitch int, flags [8]bool) {
	var tmp [64]int32

	for i := range 8 {
		if flags[i] {
			d1, d2, d3, d4, d5, d6, d7, d8 := invSlant8(
				in[i], in[8+i], in[16+i], in[24+i], in[32+i], in[40+i], in[48+i], in[56+i])
			tmp[i], tmp[8+i], tmp[16+i], tmp[24+i] = compensate1(d1), compensate1(d2), compensate1(d3), compensate1(d4)
			tmp[32+i], tmp[40+i], tmp[48+i], tmp[56+i] = compensate1(d5), compensate1(d6), compensate1(d7), compensate1(d8)
		} else {
			tmp[i], tmp[8+i], tmp[16+i], tmp[24+i] = 0, 0, 0, 0
			tmp[32+i], tmp[40+i], tmp[48+i], tmp[56+i] = 0, 0, 0, 0
		}
	}

	for row := range 8 {
		src := tmp[row*8 : row*8+8]
		o := row * pitch
		if src[0] == 0 && src[1] == 0 && src[2] == 0 && src[3] == 0 &&
			src[4] == 0 && src[5] == 0 && src[6] == 0 && src[7] == 0 {
			for i := range 8 {
				out[o+i] = 0
			}
			continue
		}
		d1, d2, d3, d4, d5, d6, d7, d8 := invSlant8(src[0], src[1], src[2], src[3], src[4], src[5], src[6], src[7])
		out[o+0] = int16(compensate2(d1))
		out[o+1] = int16(compensate2(d2))
		out[o+2] = int16(compensate2(d3))
		out[o+3] = int16(compensate2(d4))
		out[o+4] = int16(compensate2(d5))
		out[o+5] = int16(compensate2(d6))
		out[o+6] = int16(compensate2(d7))
		out[o+7] = int16(compensate2(d8))
	}
}

// inverseSlant4x4 is ff_ivi_inverse_slant_4x4: full 2D inverse transform.
func inverseSlant4x4(in []int32, out []int16, pitch int, flags [4]bool) {
	var tmp [16]int32

	for i := range 4 {
		if flags[i] {
			d1, d2, d3, d4 := invSlant4(in[i], in[4+i], in[8+i], in[12+i])
			tmp[i], tmp[4+i], tmp[8+i], tmp[12+i] = compensate1(d1), compensate1(d2), compensate1(d3), compensate1(d4)
		} else {
			tmp[i], tmp[4+i], tmp[8+i], tmp[12+i] = 0, 0, 0, 0
		}
	}

	for row := range 4 {
		src := tmp[row*4 : row*4+4]
		o := row * pitch
		if src[0] == 0 && src[1] == 0 && src[2] == 0 && src[3] == 0 {
			out[o], out[o+1], out[o+2], out[o+3] = 0, 0, 0, 0
			continue
		}
		d1, d2, d3, d4 := invSlant4(src[0], src[1], src[2], src[3])
		out[o+0] = int16(compensate2(d1))
		out[o+1] = int16(compensate2(d2))
		out[o+2] = int16(compensate2(d3))
		out[o+3] = int16(compensate2(d4))
	}
}

// rowSlant8 is ff_ivi_row_slant8: row-only inverse transform (used when the
// wavelet decomposition already separated the vertical frequency band).
func rowSlant8(in []int32, out []int16, pitch int) {
	for row := range 8 {
		src := in[row*8 : row*8+8]
		o := row * pitch
		if src[0] == 0 && src[1] == 0 && src[2] == 0 && src[3] == 0 &&
			src[4] == 0 && src[5] == 0 && src[6] == 0 && src[7] == 0 {
			for i := range 8 {
				out[o+i] = 0
			}
			continue
		}
		d1, d2, d3, d4, d5, d6, d7, d8 := invSlant8(src[0], src[1], src[2], src[3], src[4], src[5], src[6], src[7])
		out[o+0] = int16(compensate2(d1))
		out[o+1] = int16(compensate2(d2))
		out[o+2] = int16(compensate2(d3))
		out[o+3] = int16(compensate2(d4))
		out[o+4] = int16(compensate2(d5))
		out[o+5] = int16(compensate2(d6))
		out[o+6] = int16(compensate2(d7))
		out[o+7] = int16(compensate2(d8))
	}
}

// colSlant8 is ff_ivi_col_slant8: column-only inverse transform.
func colSlant8(in []int32, out []int16, pitch int, flags [8]bool) {
	row2, row4, row8 := pitch*2, pitch*4, pitch*8

	for i := range 8 {
		o := i
		if flags[i] {
			d1, d2, d3, d4, d5, d6, d7, d8 := invSlant8(
				in[i], in[8+i], in[16+i], in[24+i], in[32+i], in[40+i], in[48+i], in[56+i])
			out[o] = int16(compensate2(d1))
			out[o+pitch] = int16(compensate2(d2))
			out[o+row2] = int16(compensate2(d3))
			out[o+row2+pitch] = int16(compensate2(d4))
			out[o+row4] = int16(compensate2(d5))
			out[o+row4+pitch] = int16(compensate2(d6))
			out[o+row4+row2] = int16(compensate2(d7))
			out[o+row8-pitch] = int16(compensate2(d8))
		} else {
			out[o] = 0
			out[o+pitch] = 0
			out[o+row2] = 0
			out[o+row2+pitch] = 0
			out[o+row4] = 0
			out[o+row4+pitch] = 0
			out[o+row4+row2] = 0
			out[o+row8-pitch] = 0
		}
	}
}

// putPixels8x8 is ff_ivi_put_pixels_8x8: pass-through (no transform).
func putPixels8x8(in []int32, out []int16, pitch int) {
	for y := range 8 {
		o := y * pitch
		s := y * 8
		for x := range 8 {
			out[o+x] = int16(in[s+x])
		}
	}
}

// DC-only variants: used when a block's coded block pattern says "not
// coded" for intra blocks, where only the (inherited) DC coefficient
// matters and every AC coefficient is implicitly zero.

func dcSlant2D(dc int32, out []int16, pitch, blkSize int) {
	v := int16((dc + 1) >> 1)
	for y := range blkSize {
		o := y * pitch
		for x := range blkSize {
			out[o+x] = v
		}
	}
}

func dcRowSlant(dc int32, out []int16, pitch, blkSize int) {
	v := int16((dc + 1) >> 1)
	for x := range blkSize {
		out[x] = v
	}
	for y := 1; y < blkSize; y++ {
		o := y * pitch
		for x := range blkSize {
			out[o+x] = 0
		}
	}
}

func dcColSlant(dc int32, out []int16, pitch, blkSize int) {
	v := int16((dc + 1) >> 1)
	for y := range blkSize {
		o := y * pitch
		out[o] = v
		for x := 1; x < blkSize; x++ {
			out[o+x] = 0
		}
	}
}

func putDCPixel8x8(dc int32, out []int16, pitch int) {
	out[0] = int16(dc)
	for x := 1; x < 8; x++ {
		out[x] = 0
	}
	for y := 1; y < 8; y++ {
		o := y * pitch
		for x := range 8 {
			out[o+x] = 0
		}
	}
}
