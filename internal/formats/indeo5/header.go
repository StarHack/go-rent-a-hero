package indeo5

import "fmt"

// decodePictureHeader mirrors indeo5.c's decode_pic_hdr: a fixed 5-bit sync
// code, 3-bit frame type, 8-bit frame number, then (intra frames only) the
// full GOP header, then (any non-null frame) frame-level flags and the
// macroblock Huffman table selection.
func (d *Decoder) decodePictureHeader(r *bitReader) error {
	start, err := r.bits(5)
	if err != nil {
		return err
	}
	if start != 0x1F {
		return fmt.Errorf("invalid picture start code %#x", start)
	}

	ft, err := r.bits(3)
	if err != nil {
		return err
	}
	d.prevFrameType = d.frameType
	d.frameType = frameType(ft)
	if d.frameType > frameNull {
		return fmt.Errorf("invalid frame type %d", ft)
	}

	if _, err := r.bits(8); err != nil { // frame_num, unused
		return err
	}

	if d.frameType == frameIntra {
		if err := d.decodeGOPHeader(r); err != nil {
			return err
		}
		d.haveGOPHeader = true
	}

	if d.frameType == frameInterScal && !d.isScalable {
		return fmt.Errorf("scalable inter frame in non-scalable stream")
	}

	if d.frameType != frameNull {
		flags, err := r.bits(8)
		if err != nil {
			return err
		}
		d.frameFlags = byte(flags)

		if d.frameFlags&1 != 0 {
			if _, err := r.bits(24); err != nil { // pic_hdr_size, unused
				return err
			}
		}
		if d.frameFlags&0x10 != 0 {
			if _, err := r.bits(16); err != nil { // checksum, unused
				return err
			}
		}
		if d.frameFlags&0x20 != 0 {
			if err := skipHeaderExtension(r); err != nil {
				return err
			}
		}

		tab, err := decodeHuffTab(r, d.frameFlags&0x40 != 0, &mbHuffDescs)
		if err != nil {
			return err
		}
		d.mbVLC = tab

		if _, err := r.bits(3); err != nil { // unknown, per FFmpeg's own comment
			return err
		}
	}

	r.align()

	return nil
}

func skipHeaderExtension(r *bitReader) error {
	for {
		length, err := r.bits(8)
		if err != nil {
			return err
		}
		if 8*int(length) > r.bitsLeft() {
			return fmt.Errorf("header extension length overruns bitstream")
		}
		if _, err := r.bits(int(length) * 8); length > 0 && err != nil {
			return err
		}
		if length == 0 {
			return nil
		}
	}
}

// decodeGOPHeader mirrors indeo5.c's decode_gop_header, scoped to this
// game's confirmed configuration: no password protection, no YV12 flag,
// and (as it happens) always tile_size == 0 -- but general tile sizes and
// both the scalable (4 luma bands) and non-scalable (1 band) layouts are
// still handled, since nothing about the format guarantees every asset
// uses the same GOP settings.
func (d *Decoder) decodeGOPHeader(r *bitReader) error {
	gopFlags, err := r.bits(8)
	if err != nil {
		return err
	}

	if gopFlags&1 != 0 {
		if _, err := r.bits(16); err != nil { // gop_hdr_size, unused
			return err
		}
	}

	if gopFlags&0x20 != 0 {
		return fmt.Errorf("password-protected clip is not supported")
	}

	tileSize := 0
	if gopFlags&0x40 != 0 {
		bits2, err := r.bits(2)
		if err != nil {
			return err
		}
		tileSize = 64 << bits2
		if tileSize > 256 {
			return fmt.Errorf("invalid tile size %d", tileSize)
		}
	}
	if tileSize != 0 {
		// Tiling ("local decoding": a band split into independently coded
		// tiles smaller than the whole band) is a real bitstream feature
		// this decoder doesn't implement -- this game's own assets never
		// use it (every band is always exactly one tile), and supporting
		// arbitrary tile grids would mean threading tile-local motion
		// vector/quant-delta inheritance across tile boundaries for no
		// benefit here.
		return fmt.Errorf("tiled bands (local decoding) are not supported")
	}

	lumaBandsBits, err := r.bits(2)
	if err != nil {
		return err
	}
	lumaBands := int(lumaBandsBits)*3 + 1

	chromaBandBit, err := r.bit()
	if err != nil {
		return err
	}
	chromaBands := int(chromaBandBit)*3 + 1

	isScalable := lumaBands != 1 || chromaBands != 1
	if isScalable && (lumaBands != 4 || chromaBands != 1) {
		return fmt.Errorf("unsupported scalability subdivision: luma bands %d, chroma bands %d", lumaBands, chromaBands)
	}

	picSizeIdx, err := r.bits(4)
	if err != nil {
		return err
	}

	var picWidth, picHeight int
	if picSizeIdx == 15 {
		h, err := r.bits(13)
		if err != nil {
			return err
		}
		w, err := r.bits(13)
		if err != nil {
			return err
		}
		picHeight, picWidth = int(h), int(w)
	} else {
		picHeight = picSizesDiv4[picSizeIdx*2+1] << 2
		picWidth = picSizesDiv4[picSizeIdx*2] << 2
	}

	if gopFlags&2 != 0 {
		return fmt.Errorf("YV12 picture format is not supported")
	}

	chromaHeight := (picHeight + 3) >> 2
	chromaWidth := (picWidth + 3) >> 2

	d.picWidth, d.picHeight = picWidth, picHeight
	d.chromaWidth, d.chromaHeight = chromaWidth, chromaHeight
	d.lumaBands, d.chromaBands = lumaBands, chromaBands
	d.isScalable = isScalable

	d.initPlanes()

	for p := range 2 { // 0 = luma, 1 = chroma (plane 2 mirrors plane 1 below)
		n := lumaBands
		if p == 1 {
			n = chromaBands
		}
		for i := 0; i < n; i++ {
			b := &d.planes[p].bands[i]

			halfpel, err := r.bit()
			if err != nil {
				return err
			}
			b.isHalfpel = halfpel != 0

			mbSizeBit, err := r.bit()
			if err != nil {
				return err
			}
			blkSizeBit, err := r.bit()
			if err != nil {
				return err
			}
			blkSize := 8 >> blkSizeBit
			mbSize := blkSize << (1 - mbSizeBit)

			if p == 0 && blkSize == 4 {
				return fmt.Errorf("4x4 luma blocks are not supported")
			}
			b.mbSize, b.blkSize = mbSize, blkSize

			extTransform, err := r.bit()
			if err != nil {
				return err
			}
			if extTransform != 0 {
				return fmt.Errorf("extended transform info is not supported")
			}

			switch p<<2 + i {
			case 0:
				b.kind, b.scan, b.transformSize = kind2D, zigzagDirect8x8[:], 8
			case 1:
				b.kind, b.scan, b.transformSize = kindRowOnly, verticalScan8x8[:], 8
			case 2:
				b.kind, b.scan, b.transformSize = kindColOnly, horizontalScan8x8[:], 8
			case 3:
				b.kind, b.scan, b.transformSize = kindPassthrough, horizontalScan8x8[:], 8
			case 4:
				b.kind, b.scan, b.transformSize = kind2D, directScan4x4[:], 4
			}

			if b.transformSize != b.blkSize {
				return fmt.Errorf("transform/block size mismatch (%d != %d)", b.transformSize, b.blkSize)
			}

			if p == 0 {
				b.quantMatrix = 0
				if lumaBands > 1 {
					b.quantMatrix = i + 1
				}
			} else {
				b.quantMatrix = 5
			}

			if b.blkSize == 8 {
				if b.quantMatrix >= 5 {
					return fmt.Errorf("quant matrix index %d too large", b.quantMatrix)
				}
				b.intraBase = baseQuant8x8Intra[b.quantMatrix][:]
				b.interBase = baseQuant8x8Inter[b.quantMatrix][:]
				b.intraScale = scaleQuant8x8Intra[b.quantMatrix][:]
				b.interScale = scaleQuant8x8Inter[b.quantMatrix][:]
			} else {
				b.intraBase = baseQuant4x4Intra[:]
				b.interBase = baseQuant4x4Inter[:]
				b.intraScale = scaleQuant4x4Intra[:]
				b.interScale = scaleQuant4x4Inter[:]
			}

			endMarker, err := r.bits(2)
			if err != nil {
				return err
			}
			if endMarker != 0 {
				return fmt.Errorf("band end marker missing")
			}
		}
	}

	// Copy chroma band parameters into the 2nd chroma plane (both chroma
	// planes always share layout; only their coded coefficients differ).
	for i := range chromaBands {
		b1 := &d.planes[1].bands[i]
		b2 := &d.planes[2].bands[i]
		*b2 = *b1
		b2.bufs = [4][]int16{} // plane 2 needs its own pixel storage, not plane 1's
	}

	d.hasTransparency = gopFlags&8 != 0
	d.transparencyFill = [3]byte{}
	if d.hasTransparency {
		align3, err := r.bits(3)
		if err != nil {
			return err
		}
		if align3 != 0 {
			return fmt.Errorf("alignment bits are not zero")
		}
		hasFill, err := r.bit()
		if err != nil {
			return err
		}
		if hasFill != 0 {
			fill, err := r.bits(24)
			if err != nil {
				return err
			}
			d.transparencyFill = [3]byte{byte(fill), byte(fill >> 8), byte(fill >> 16)}
		}
	} else {
		d.transparencyMask = nil
	}

	r.align()

	if _, err := r.bits(23); err != nil { // unknown, per FFmpeg's own comment
		return err
	}

	hasExt, err := r.bit()
	if err != nil {
		return err
	}
	if hasExt != 0 {
		for {
			v, err := r.bits(16)
			if err != nil {
				return err
			}
			if v&0x8000 == 0 {
				break
			}
		}
	}

	r.align()

	return nil
}
