package audio

// resampleLinear converts mono PCM from srcRate to dstRate via linear
// interpolation. Sufficient quality for short speech/SFX clips; not a
// high-quality resampler.
func resampleLinear(input []int16, srcRate, dstRate int) []int16 {
	if srcRate == dstRate || len(input) == 0 {
		return input
	}

	ratio := float64(srcRate) / float64(dstRate)
	outLen := int(float64(len(input)) * float64(dstRate) / float64(srcRate))
	out := make([]int16, outLen)

	for i := range out {
		srcPos := float64(i) * ratio
		i0 := int(srcPos)
		frac := srcPos - float64(i0)

		s0 := sampleAt(input, i0)
		s1 := sampleAt(input, i0+1)

		out[i] = int16(float64(s0) + frac*(float64(s1)-float64(s0)))
	}

	return out
}

func sampleAt(input []int16, i int) int16 {
	if i < 0 {
		return input[0]
	}
	if i >= len(input) {
		return input[len(input)-1]
	}
	return input[i]
}

// toStereoBytes duplicates mono samples into interleaved little-endian
// stereo PCM16 bytes.
func toStereoBytes(mono []int16) []byte {
	buf := make([]byte, len(mono)*4)

	for i, s := range mono {
		b := uint16(s)
		lo, hi := byte(b), byte(b>>8)

		buf[i*4+0] = lo
		buf[i*4+1] = hi
		buf[i*4+2] = lo
		buf[i*4+3] = hi
	}

	return buf
}
