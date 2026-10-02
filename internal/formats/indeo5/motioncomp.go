package indeo5

// mcType selects fullpel vs. one of the three halfpel interpolation modes,
// exactly as ivi_mc_NxN's mc_type parameter does: 0 = fullpel, 1 =
// horizontal halfpel, 2 = vertical halfpel, 3 = both.
//
// put writes the motion-compensated block into buf (used for the block's
// only reference); add accumulates it (used when a delta -- coded
// residual -- still needs to be applied on top, i.e. the block was coded
// and ivi_decode_coded_blocks's inverse transform already wrote the
// residual into buf, so motion compensation must add rather than
// overwrite).
func mcPut(size int, buf []int16, bufPitch int, ref []int16, refPitch int, mcType int) {
	mc(size, buf, bufPitch, ref, refPitch, mcType, false)
}

func mcAdd(size int, buf []int16, bufPitch int, ref []int16, refPitch int, mcType int) {
	mc(size, buf, bufPitch, ref, refPitch, mcType, true)
}

func mc(size int, buf []int16, bufPitch int, ref []int16, refPitch int, mcType int, add bool) {
	apply := func(bo, ro int) {
		if add {
			buf[bo] += ref[ro]
		} else {
			buf[bo] = ref[ro]
		}
	}
	applyVal := func(bo int, v int32) {
		if add {
			buf[bo] += int16(v)
		} else {
			buf[bo] = int16(v)
		}
	}

	switch mcType {
	case 0: // fullpel
		for i := range size {
			bo, ro := i*bufPitch, i*refPitch
			for j := range size {
				apply(bo+j, ro+j)
			}
		}
	case 1: // horizontal halfpel
		for i := range size {
			bo, ro := i*bufPitch, i*refPitch
			for j := range size {
				applyVal(bo+j, (int32(ref[ro+j])+int32(ref[ro+j+1]))>>1)
			}
		}
	case 2: // vertical halfpel
		for i := range size {
			bo, ro, wo := i*bufPitch, i*refPitch, (i+1)*refPitch
			for j := range size {
				applyVal(bo+j, (int32(ref[ro+j])+int32(ref[wo+j]))>>1)
			}
		}
	case 3: // vertical and horizontal halfpel
		for i := range size {
			bo, ro, wo := i*bufPitch, i*refPitch, (i+1)*refPitch
			for j := range size {
				applyVal(bo+j, (int32(ref[ro+j])+int32(ref[ro+j+1])+int32(ref[wo+j])+int32(ref[wo+j+1]))>>2)
			}
		}
	}
}
