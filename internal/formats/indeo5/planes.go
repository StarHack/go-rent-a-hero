package indeo5

// initPlanes mirrors ff_ivi_init_planes: sizes each plane's bands (their
// pixel-buffer pitch and aligned height) from the picture dimensions and
// band counts just read from the GOP header. Band width/height only
// depend on these, not on anything else in the bitstream, so this always
// runs before the rest of the GOP header's per-band fields are read.
func (d *Decoder) initPlanes() {
	d.planes[0].width, d.planes[0].height = d.picWidth, d.picHeight
	d.planes[1].width, d.planes[1].height = d.chromaWidth, d.chromaHeight
	d.planes[2].width, d.planes[2].height = d.chromaWidth, d.chromaHeight

	numBands := [3]int{d.lumaBands, d.chromaBands, d.chromaBands}

	for p := range 3 {
		n := numBands[p]
		d.planes[p].bands = make([]band, n)

		bWidth, bHeight := d.planes[p].width, d.planes[p].height
		if n != 1 {
			bWidth = (bWidth + 1) >> 1
			bHeight = (bHeight + 1) >> 1
		}

		alignFac := 16
		if p != 0 {
			alignFac = 8
		}
		widthAligned := alignUp(bWidth, alignFac)
		heightAligned := alignUp(bHeight, alignFac)

		for i := range n {
			bd := &d.planes[p].bands[i]
			bd.plane, bd.num = p, i
			bd.width, bd.height = bWidth, bHeight
			bd.pitch = widthAligned
			bd.aheight = heightAligned
			bd.bufSize = widthAligned * heightAligned
		}
	}
}

func alignUp(v, align int) int {
	return (v + align - 1) / align * align
}

// prepareBufSlot lazily allocates (ff_ivi_init_planes/prepare_buf) and
// returns bufs[idx], or nil if this stream never needs that slot at all --
// slot 2 (the extra buffer FRAMETYPE_INTER_SCAL needs) is only relevant
// when the stream is actually scalable.
func (b *band) prepareBufSlot(lumaBands, idx int) []int16 {
	if lumaBands <= 1 && idx == 2 {
		return nil
	}
	if b.bufs[idx] == nil {
		b.bufs[idx] = make([]int16, b.bufSize)
	}
	return b.bufs[idx]
}
