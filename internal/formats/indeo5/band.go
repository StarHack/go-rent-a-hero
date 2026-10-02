package indeo5

import "fmt"

// decodeBand mirrors indeo5.c's decode_band: prepares this band's current
// and reference pixel buffers, reads the band header, applies any rvmap
// corrections, then decodes its one tile (see decodeGOPHeader's rejection
// of tiled streams) of macroblock info and block data.
func (d *Decoder) decodeBand(r *bitReader, b *band) error {
	b.buf = b.prepareBufSlot(d.lumaBands, d.dstBuf)
	if b.buf == nil {
		return fmt.Errorf("band buffer not allocated")
	}
	b.refBuf = b.prepareBufSlot(d.lumaBands, d.refBuf)
	if b.refBuf == nil {
		return fmt.Errorf("reference buffer not allocated")
	}

	if err := d.decodeBandHeader(r, b); err != nil {
		return err
	}
	if b.isEmpty {
		return fmt.Errorf("empty band encountered")
	}

	rv := rvMapTabs[b.rvmapSel] // local copy: corrections below must not mutate the shared table
	for i := 0; i < b.numCorr; i++ {
		idx1, idx2 := b.corr[i*2], b.corr[i*2+1]
		rv.runTab[idx1], rv.runTab[idx2] = rv.runTab[idx2], rv.runTab[idx1]
		rv.valTab[idx1], rv.valTab[idx2] = rv.valTab[idx2], rv.valTab[idx1]
		if int(idx1) == rv.eobSym || int(idx2) == rv.eobSym {
			rv.eobSym ^= int(idx1) ^ int(idx2)
		}
		if int(idx1) == rv.escSym || int(idx2) == rv.escSym {
			rv.escSym ^= int(idx1) ^ int(idx2)
		}
	}

	// This game's assets never use tiling (decodeGOPHeader already rejects
	// a nonzero tile size), so one tile always spans the whole band.
	tw, th := b.width, b.height
	numMBs := mbsPerTile(tw, th, b.mbSize)

	lumaBand0 := &d.planes[0].bands[0]
	mvScale := (lumaBand0.mbSize >> 3) - (b.mbSize >> 3)

	// inherit_mv/inherit_qdelta reference luma plane 0's band 0 *from this
	// same frame* (see ff_ivi_init_tiles's ref_tile wiring: every band's
	// tile takes planes[0].bands[0]'s tile as its reference, except band 0
	// itself, which never has anything to inherit from). Band 0 always
	// decodes first (see DecodeFrame's plane/band loop order), so its mbs
	// are already populated by the time any other band needs them here.
	// This is spatial (cross-band, same-frame) inheritance, not temporal:
	// nothing here persists across frames except the pixel buffers.
	var refMBs []mbInfo
	if b != lumaBand0 {
		refMBs = lumaBand0.mbs
	}

	// data_size accounting starts here, *before* the is_empty bit -- it
	// covers that bit and the tile_data_size field itself, not just the
	// mb-info/block payload that follows them (see indeo5.c's decode_band:
	// pos is captured before the tile loop reads tile->is_empty at all).
	startBit := r.pos

	isEmptyTile, err := r.bit()
	if err != nil {
		return err
	}

	if isEmptyTile != 0 {
		if err := d.processEmptyTile(b, refMBs, tw, th, mvScale); err != nil {
			return err
		}
	} else {
		dataSize, err := decodeTileDataSize(r)
		if err != nil {
			return err
		}
		if dataSize == 0 {
			return fmt.Errorf("tile data size is zero")
		}

		if err := d.decodeMBInfo(r, b, refMBs, numMBs, tw, th, mvScale); err != nil {
			return err
		}
		if err := d.decodeMBBlocks(r, b, &rv); err != nil {
			return err
		}

		if (r.pos-startBit)>>3 != dataSize {
			return fmt.Errorf("tile data size mismatch: consumed %d bytes, header said %d", (r.pos-startBit)>>3, dataSize)
		}
	}

	r.align()

	return nil
}

func mbsPerTile(tileWidth, tileHeight, mbSize int) int {
	return ((tileWidth + mbSize - 1) / mbSize) * ((tileHeight + mbSize - 1) / mbSize)
}

func numBlocksPerMB(b *band) int {
	if b.mbSize != b.blkSize {
		return 4
	}
	return 1
}

// decodeBandHeader mirrors indeo5.c's decode_band_hdr.
func (d *Decoder) decodeBandHeader(r *bitReader, b *band) error {
	flags, err := r.bits(8)
	if err != nil {
		return err
	}

	if flags&1 != 0 {
		b.isEmpty = true
		return nil
	}
	b.isEmpty = false

	if d.frameFlags&0x80 != 0 {
		if _, err := r.bits(24); err != nil { // data_size, unused: we track consumed bits directly
			return err
		}
	}

	b.inheritMV = flags&2 != 0
	b.inheritQDelta = flags&8 != 0
	b.qdeltaPresent = flags&4 != 0
	if !b.qdeltaPresent {
		b.inheritQDelta = true
	}

	b.numCorr = 0
	if flags&0x10 != 0 {
		n, err := r.bits(8)
		if err != nil {
			return err
		}
		if n > 61 {
			return fmt.Errorf("too many rvmap corrections: %d", n)
		}
		b.numCorr = int(n)
		for i := 0; i < b.numCorr*2; i++ {
			v, err := r.bits(8)
			if err != nil {
				return err
			}
			b.corr[i] = byte(v)
		}
	}

	b.rvmapSel = 8
	if flags&0x40 != 0 {
		sel, err := r.bits(3)
		if err != nil {
			return err
		}
		b.rvmapSel = int(sel)
	}

	tab, err := decodeHuffTab(r, flags&0x80 != 0, &blkHuffDescs)
	if err != nil {
		return err
	}
	b.blkVLC = tab

	checksumPresent, err := r.bit()
	if err != nil {
		return err
	}
	if checksumPresent != 0 {
		if _, err := r.bits(16); err != nil {
			return err
		}
	}

	gq, err := r.bits(5)
	if err != nil {
		return err
	}
	b.globQuant = int(gq)

	if flags&0x20 != 0 {
		r.align()
		if err := skipHeaderExtension(r); err != nil {
			return err
		}
	}

	r.align()

	return nil
}

// decodeTileDataSize mirrors ivi_dec_tile_data_size.
func decodeTileDataSize(r *bitReader) (int, error) {
	has, err := r.bit()
	if err != nil {
		return 0, err
	}

	size := 0
	if has != 0 {
		v, err := r.bits(8)
		if err != nil {
			return 0, err
		}
		size = int(v)
		if size == 255 {
			v2, err := r.bits(24)
			if err != nil {
				return 0, err
			}
			size = int(v2)
		}
	}

	r.align()

	return size, nil
}

// decodeMBInfo mirrors indeo5.c's decode_mb_info for a single tile
// spanning the whole band (tx=ty=0, tw/th = the band's own size). refMBs is
// luma band 0's already-decoded macroblocks for this same frame (see
// decodeBand), or nil when b is that very band.
func (d *Decoder) decodeMBInfo(r *bitReader, b *band, refMBs []mbInfo, numMBs, tw, th, mvScale int) error {
	if refMBs == nil && ((b.qdeltaPresent && b.inheritQDelta) || b.inheritMV) {
		return fmt.Errorf("motion vector/quant-delta inheritance requested but no reference macroblocks available")
	}

	b.mbs = make([]mbInfo, numMBs)

	rowOffset := b.mbSize * b.pitch
	offs := 0
	mvX, mvY := 0, 0
	idx := 0

	for y := 0; y < th; y += b.mbSize {
		mbOffset := offs

		for x := 0; x < tw; x += b.mbSize {
			mb := &b.mbs[idx]
			mb.xpos, mb.ypos, mb.bufOffs = x, y, mbOffset

			var refMB *mbInfo
			if idx < len(refMBs) {
				refMB = &refMBs[idx]
			}

			empty, err := r.bit()
			if err != nil {
				return err
			}

			if empty != 0 {
				if d.frameType == frameIntra {
					return fmt.Errorf("empty macroblock in an intra picture")
				}
				mb.mbType = 1
				mb.cbp = 0
				mb.qDelta = 0

				if b.plane == 0 && b.num == 0 && d.frameFlags&8 != 0 {
					sym, err := decodeSymbol(r, &d.mbVLC.desc)
					if err != nil {
						return err
					}
					mb.qDelta = toSigned(sym)
				}

				mb.mvX, mb.mvY = 0, 0
				if b.inheritMV && refMB != nil {
					mb.mvX, mb.mvY = inheritMV(refMB, mvScale)
				}
			} else {
				switch {
				case b.inheritMV && refMB != nil:
					mb.mbType = refMB.mbType
				case d.frameType == frameIntra:
					mb.mbType = 0
				default:
					bit, err := r.bit()
					if err != nil {
						return err
					}
					mb.mbType = int(bit)
				}

				blksPerMB := 1
				if b.mbSize != b.blkSize {
					blksPerMB = 4
				}
				cbp, err := r.bits(blksPerMB)
				if err != nil {
					return err
				}
				mb.cbp = int(cbp)

				mb.qDelta = 0
				if b.qdeltaPresent {
					switch {
					case b.inheritQDelta:
						if refMB != nil {
							mb.qDelta = refMB.qDelta
						}
					case mb.cbp != 0 || (b.plane == 0 && b.num == 0 && d.frameFlags&8 != 0):
						sym, err := decodeSymbol(r, &d.mbVLC.desc)
						if err != nil {
							return err
						}
						mb.qDelta = toSigned(sym)
					}
				}

				if mb.mbType == 0 {
					mb.mvX, mb.mvY = 0, 0
				} else if b.inheritMV && refMB != nil {
					mb.mvX, mb.mvY = inheritMV(refMB, mvScale)
				} else {
					dSym, err := decodeSymbol(r, &d.mbVLC.desc)
					if err != nil {
						return err
					}
					mvY += toSigned(dSym)
					dSym2, err := decodeSymbol(r, &d.mbVLC.desc)
					if err != nil {
						return err
					}
					mvX += toSigned(dSym2)
					mb.mvX, mb.mvY = mvX, mvY
				}
			}

			idx++
			mbOffset += b.mbSize
		}

		offs += rowOffset
	}

	r.align()

	return nil
}

func inheritMV(ref *mbInfo, mvScale int) (int, int) {
	if mvScale != 0 {
		return scaleMV(ref.mvX, mvScale), scaleMV(ref.mvY, mvScale)
	}
	return ref.mvX, ref.mvY
}

// processEmptyTile mirrors ivi_process_empty_tile.
func (d *Decoder) processEmptyTile(b *band, refMBs []mbInfo, tw, th, mvScale int) error {
	numMBs := mbsPerTile(tw, th, b.mbSize)
	b.mbs = make([]mbInfo, numMBs)

	clearFirst := !b.qdeltaPresent && b.plane == 0 && b.num == 0
	rowOffset := b.mbSize * b.pitch
	offs := 0
	needMC := false
	idx := 0

	for y := 0; y < th; y += b.mbSize {
		mbOffset := offs

		for x := 0; x < tw; x += b.mbSize {
			mb := &b.mbs[idx]
			mb.xpos, mb.ypos, mb.bufOffs = x, y, mbOffset
			mb.mbType = 1
			mb.cbp = 0

			if clearFirst {
				mb.qDelta = b.globQuant
				mb.mvX, mb.mvY = 0, 0
			}

			if idx < len(refMBs) {
				ref := &refMBs[idx]
				if b.inheritQDelta {
					mb.qDelta = ref.qDelta
				}
				if b.inheritMV {
					mb.mvX, mb.mvY = inheritMV(ref, mvScale)
					if mb.mvX != 0 || mb.mvY != 0 {
						needMC = true
					}
				}
			}

			idx++
			mbOffset += b.mbSize
		}

		offs += rowOffset
	}

	if b.inheritMV && needMC {
		for i := range b.mbs {
			mb := &b.mbs[i]
			mvX, mvY := mb.mvX, mb.mvY
			mcType := 0
			if b.isHalfpel {
				mcType = ((mvY & 1) << 1) | (mvX & 1)
				mvX >>= 1
				mvY >>= 1
			}
			for blk := range numBlocksPerMB(b) {
				offs := mb.bufOffs + b.blkSize*((blk&1)+btoi(blk&2 != 0)*b.pitch)
				if err := d.motionCompensate(b, offs, mvX, mvY, mcType, false); err != nil {
					return err
				}
			}
		}
	} else {
		for row := range th {
			o := row * b.pitch
			copy(b.buf[o:o+tw], b.refBuf[o:o+tw])
		}
	}

	return nil
}

func btoi(v bool) int {
	if v {
		return 1
	}
	return 0
}

// decodeMBBlocks mirrors ivi_decode_blocks: walks every macroblock in
// b.mbs and decodes (or synthesizes, for uncoded blocks) its 1 or 4
// transform blocks.
func (d *Decoder) decodeMBBlocks(r *bitReader, b *band, rv *rvMap) error {
	prevDC := 0
	numBlocks := numBlocksPerMB(b)

	for i := range b.mbs {
		mb := &b.mbs[i]
		isIntra := mb.mbType == 0
		cbp := mb.cbp
		bufOffs := mb.bufOffs

		quant := clampInt(b.globQuant+mb.qDelta, 0, 23)
		scaleTab := b.interScale
		if isIntra {
			scaleTab = b.intraScale
		}
		if scaleTab != nil {
			quant = int(scaleTab[quant])
		}

		var mvX, mvY, mcType int
		if !isIntra {
			mvX, mvY = mb.mvX, mb.mvY
			if b.isHalfpel {
				mcType = ((mvY & 1) << 1) | (mvX & 1)
				mvX >>= 1
				mvY >>= 1
			}
		}

		for blk := range numBlocks {
			switch {
			case blk&1 != 0:
				bufOffs += b.blkSize
			case blk == 2:
				bufOffs -= b.blkSize
				bufOffs += b.blkSize * b.pitch
			}

			if !d.blockFits(b, bufOffs) {
				return fmt.Errorf("block at offset %d falls outside the band buffer", bufOffs)
			}

			if cbp&1 != 0 {
				var err error
				prevDC, err = d.decodeCodedBlock(r, b, rv, mvX, mvY, prevDC, isIntra, mcType, quant, bufOffs)
				if err != nil {
					return err
				}
			} else if isIntra {
				d.applyDC(b, int32(prevDC), bufOffs)
			} else {
				if err := d.motionCompensate(b, bufOffs, mvX, mvY, mcType, false); err != nil {
					return err
				}
			}

			cbp >>= 1
		}
	}

	r.align()

	return nil
}

func (d *Decoder) blockFits(b *band, offs int) bool {
	size := b.transformSize
	minSize := b.pitch*(size-1) + size
	return offs >= 0 && offs+minSize <= len(b.buf)
}

func (d *Decoder) applyDC(b *band, dc int32, offs int) {
	out := b.buf[offs:]
	switch b.kind {
	case kind2D:
		dcSlant2D(dc, out, b.pitch, b.blkSize)
	case kindRowOnly:
		dcRowSlant(dc, out, b.pitch, b.blkSize)
	case kindColOnly:
		dcColSlant(dc, out, b.pitch, b.blkSize)
	case kindPassthrough:
		putDCPixel8x8(dc, out, b.pitch)
	}
}

// decodeCodedBlock mirrors ivi_decode_coded_blocks: reads this block's
// Huffman/RLE-coded coefficients, dequantizes and inverse-transforms them,
// and (for inter blocks) adds the motion-compensated reference on top. It
// returns the updated DC predictor (only actually changed for 2D-transform
// bands, matching the original's is_2d_trans gate).
func (d *Decoder) decodeCodedBlock(r *bitReader, b *band, rv *rvMap, mvX, mvY, prevDC int, isIntra bool, mcType, quant, offs int) (int, error) {
	baseTab := b.interBase
	if isIntra {
		baseTab = b.intraBase
	}

	numCoeffs := b.blkSize * b.blkSize
	colMask := b.blkSize - 1

	var trvec [64]int32
	var colFlags [8]bool

	scanPos := -1
	sym := 0

	for scanPos <= numCoeffs {
		s, err := decodeSymbol(r, &b.blkVLC.desc)
		if err != nil {
			return prevDC, err
		}
		sym = s
		if sym == rv.eobSym {
			break
		}

		var run, val int
		if sym == rv.escSym {
			runSym, err := decodeSymbol(r, &b.blkVLC.desc)
			if err != nil {
				return prevDC, err
			}
			run = runSym + 1
			lo, err := decodeSymbol(r, &b.blkVLC.desc)
			if err != nil {
				return prevDC, err
			}
			hi, err := decodeSymbol(r, &b.blkVLC.desc)
			if err != nil {
				return prevDC, err
			}
			val = toSigned((hi << 6) | lo)
		} else {
			if sym < 0 || sym >= 256 {
				return prevDC, fmt.Errorf("invalid rvmap symbol %d", sym)
			}
			run = int(rv.runTab[sym])
			val = int(rv.valTab[sym])
		}

		scanPos += run
		if scanPos >= numCoeffs || scanPos < 0 {
			break
		}
		pos := b.scan[scanPos]

		q := (int(baseTab[pos]) * quant) >> 9
		if q > 1 {
			sign := 1
			if val < 0 {
				sign = -1
			}
			val = val*q + sign*(((q^1)-1)>>1)
		}
		trvec[pos] = int32(val)
		if val != 0 {
			colFlags[pos&colMask] = true
		}
	}

	if scanPos < 0 || (scanPos >= numCoeffs && sym != rv.eobSym) {
		return prevDC, fmt.Errorf("corrupt block data")
	}

	is2D := b.kind == kind2D
	if isIntra && is2D {
		prevDC += int(trvec[0])
		trvec[0] = int32(prevDC)
		if prevDC != 0 {
			colFlags[0] = true
		}
	}

	out := b.buf[offs:]
	switch b.kind {
	case kind2D:
		if b.blkSize == 8 {
			inverseSlant8x8(trvec[:], out, b.pitch, [8]bool(colFlags[:8]))
		} else {
			inverseSlant4x4(trvec[:], out, b.pitch, [4]bool(colFlags[:4]))
		}
	case kindRowOnly:
		rowSlant8(trvec[:], out, b.pitch)
	case kindColOnly:
		colSlant8(trvec[:], out, b.pitch, [8]bool(colFlags[:8]))
	case kindPassthrough:
		putPixels8x8(trvec[:], out, b.pitch)
	}

	if !isIntra {
		if err := d.motionCompensate(b, offs, mvX, mvY, mcType, true); err != nil {
			return prevDC, err
		}
	}

	return prevDC, nil
}

// motionCompensate mirrors ivi_mc for the (for this codec, always single)
// forward reference: delta selects whether the reference is added to an
// already-written residual (a coded block) or written outright (skipped/
// uncoded blocks and empty tiles).
func (d *Decoder) motionCompensate(b *band, offs, mvX, mvY, mcType int, delta bool) error {
	refOffs := offs + mvY*b.pitch + mvX
	if refOffs < 0 || !d.blockFits(b, refOffs) {
		return fmt.Errorf("motion vector references outside the reference buffer")
	}

	if delta {
		mcAdd(b.blkSize, b.buf[offs:], b.pitch, b.refBuf[refOffs:], b.pitch, mcType)
	} else {
		mcPut(b.blkSize, b.buf[offs:], b.pitch, b.refBuf[refOffs:], b.pitch, mcType)
	}

	return nil
}

func toSigned(v int) int {
	return -((v >> 1) ^ -(v & 1))
}

func scaleMV(mv, mvScale int) int {
	extra := 0
	if mv > 0 {
		extra = 1
	}
	return (mv + extra + (mvScale - 1)) >> mvScale
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
