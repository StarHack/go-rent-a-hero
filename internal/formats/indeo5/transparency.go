package indeo5

import "fmt"

func (d *Decoder) decodeTransparencyPlane(r *bitReader) error {
	start := r.bytePos()
	sizeValue, err := r.bits(24)
	if err != nil {
		return err
	}
	size := int(sizeValue)
	if size < 12 || start+size > len(r.data) {
		return fmt.Errorf("invalid transparency payload size %d", size)
	}
	limit := (start + size) * 8

	if _, err := r.bits(4); err != nil {
		return err
	}
	for range 4 {
		if _, err := r.bits(16); err != nil {
			return err
		}
	}

	hasDesc, err := r.bit()
	if err != nil {
		return err
	}
	if hasDesc != 0 {
		desc, err := decodeCustomHuffDesc(r)
		if err != nil {
			return err
		}
		d.transparencyHuff = desc
		d.haveTransparencyHuff = true
	}
	r.align()

	useFillTransparency, err := r.bit()
	if err != nil {
		return err
	}
	if useFillTransparency != 0 {
		runIsOpaque, err := r.bit()
		if err != nil {
			return err
		}
		if r.pos > limit {
			return fmt.Errorf("transparency fill overruns payload")
		}
		mask := make([]bool, d.picWidth*d.picHeight)
		if runIsOpaque != 0 {
			for i := range mask {
				mask[i] = true
			}
		}
		d.transparencyMask = mask
		r.pos = limit
		return nil
	}

	hasDataSize, err := r.bit()
	if err != nil {
		return err
	}
	if hasDataSize != 0 {
		dataSize, err := r.bits(8)
		if err != nil {
			return err
		}
		if dataSize == 0xff {
			if _, err := r.bits(24); err != nil {
				return err
			}
		}
	}
	r.align()

	if !d.haveTransparencyHuff {
		return fmt.Errorf("transparency RLE has no Huffman descriptor")
	}

	stateBit, err := r.bit()
	if err != nil {
		return err
	}
	state := stateBit != 0
	nextState := !state
	alignedWidth := (d.picWidth + 31) &^ 31
	count := alignedWidth * d.picHeight
	raw := make([]bool, count)
	pos := 0

	for pos < count {
		if r.pos >= limit {
			return fmt.Errorf("transparency RLE overruns payload")
		}
		run, err := decodeSymbol(r, &d.transparencyHuff)
		if err != nil {
			return err
		}
		if run == 0 {
			run = 255
			nextState = state
		}
		if run > count-pos {
			return fmt.Errorf("transparency run overruns plane")
		}
		if state {
			for i := pos; i < pos+run; i++ {
				raw[i] = true
			}
		}
		pos += run
		state = nextState
		nextState = !state
	}

	r.align()
	if r.bytePos() != start+size {
		return fmt.Errorf("transparency payload consumed %d bytes, expected %d", r.bytePos()-start, size)
	}

	mask := make([]bool, d.picWidth*d.picHeight)
	for x := range d.picWidth {
		v := false
		for y := range d.picHeight {
			v = v != raw[y*alignedWidth+x]
			mask[y*d.picWidth+x] = v
		}
	}
	d.transparencyMask = mask
	return nil
}
