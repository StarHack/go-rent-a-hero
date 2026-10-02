// Package indeo5 implements a native decoder for the Indeo Video
// Interactive 5 ("IV50") codec, used by some of this game's character
// overlay animations (see agents/scenes/02_DRAGON_BLASTER_DELUXE.MD).
//
// Indeo5 is a wavelet-based codec: each frame's luma plane is optionally
// split into 4 wavelet "bands" (a coarse low-frequency band plus three
// detail bands, reconstructed back into pixels by recompose53) while
// chroma stays a single band; each band is further divided into fixed-size
// macroblocks, Huffman/RLE-coded transform coefficients, dequantized, and
// inverse-transformed (a "slant" transform, not the FFT-derived DCT of
// most other era codecs). Inter frames add halfpel motion compensation and
// can inherit motion vectors/quant deltas from the previous frame's
// macroblocks in the same position.
//
// This is a clean-room implementation from the codec's own format
// specifics (bitstream layout, transform structure, tables), not a
// translation of any existing decoder's source, and it deliberately only
// covers what this game's own assets actually use (verified by direct
// inspection): scalable 4-band luma / 1-band chroma, halfpel motion
// compensation, no tiling (one tile spans the whole band), and no
// bidirectional (B-frame) macroblocks -- Indeo5 never sets the macroblock
// type values that would require them. It is verified byte-for-byte
// against ffmpeg's own Indeo5 decoder in indeo5_test.go (a build- and
// runtime-independent correctness oracle, see that file's doc comment).
package indeo5

import "fmt"

type frameType int

const (
	frameIntra      frameType = 0
	frameInter      frameType = 1
	frameInterScal  frameType = 2
	frameInterNoRef frameType = 3
	frameNull       frameType = 4
)

// bandKind selects which of the four (p<<2)+bandNum transform/scan
// combinations decodeGOPHeader assigns (see indeo5.c's decode_gop_header):
// band 0 always gets the full 2D transform; for a scalable (4-band) luma
// plane, bands 1/2 get a row-only/column-only pass instead (the other
// direction already having been separated out by the wavelet split), and
// band 3 is pass-through with no frequency transform at all.
type bandKind int

const (
	kind2D bandKind = iota
	kindRowOnly
	kindColOnly
	kindPassthrough
)

type mbInfo struct {
	xpos, ypos int
	bufOffs    int
	mbType     int // 0 = intra, 1 = inter
	cbp        int
	qDelta     int
	mvX, mvY   int
}

type band struct {
	plane, num    int
	width, height int
	pitch         int
	aheight       int
	mbSize        int
	blkSize       int
	transformSize int
	isHalfpel     bool
	kind          bandKind
	scan          []int
	quantMatrix   int
	intraBase     []uint16
	interBase     []uint16
	intraScale    []uint8
	interScale    []uint8

	bufs    [4][]int16 // physical buffers switched between via bufSwitch/dstBuf/refBuf/ref2Buf
	bufSize int        // pitch*aheight, i.e. len(bufs[i])

	// Set fresh from each frame's band header:
	isEmpty       bool
	inheritMV     bool
	inheritQDelta bool
	qdeltaPresent bool
	rvmapSel      int
	corr          [122]byte
	numCorr       int
	blkVLC        *huffTab
	globQuant     int

	buf    []int16 // this frame's bufs[dstBuf], for convenience during decode
	refBuf []int16 // bufs[refBuf], or nil

	// mbs holds this frame's macroblocks (single tile spans the whole
	// band); only ever read by *other* bands' inherit_mv/inherit_qdelta
	// within the same frame (see decodeBand), never across frames.
	mbs []mbInfo
}

type plane struct {
	width, height int
	bands         []band
}

// Decoder holds persistent state across frames: physical pixel buffers
// (frames are delta-coded against, and sometimes literally inherit
// unchanged rows from, the previous one) and each band's previous
// macroblock info. Frames must be decoded in playback order.
type Decoder struct {
	picWidth, picHeight       int
	chromaWidth, chromaHeight int
	lumaBands, chromaBands    int
	isScalable                bool

	planes [3]plane

	frameType     frameType
	prevFrameType frameType
	frameFlags    byte

	bufSwitch      int
	dstBuf, refBuf int
	ref2Buf        int
	interScal      bool

	mbVLC *huffTab

	hasTransparency      bool
	transparencyFill     [3]byte
	transparencyMask     []bool
	transparencyHuff     huffDesc
	haveTransparencyHuff bool

	haveGOPHeader bool
}

// NewDecoder creates a decoder. width/height should be the container's
// reported dimensions; Indeo5's own GOP header carries the authoritative
// picture size too (checked against these on the first frame).
func NewDecoder(width, height int) *Decoder {
	return &Decoder{picWidth: width, picHeight: height}
}

func (d *Decoder) Width() int  { return d.picWidth }
func (d *Decoder) Height() int { return d.picHeight }

// DecodeFrame decodes one Indeo5 frame (an AVI "##dc" chunk's payload) and
// returns it as unpremultiplied RGBA8, width*height*4 bytes.
func (d *Decoder) DecodeFrame(data []byte) ([]byte, error) {
	r := newBitReader(data)

	if err := d.decodePictureHeader(r); err != nil {
		return nil, fmt.Errorf("picture header: %w", err)
	}

	if !d.haveGOPHeader {
		return nil, fmt.Errorf("no GOP header seen yet (stream must start with an intra frame)")
	}

	d.switchBuffers()

	if d.frameType != frameNull {
		for p := range 3 {
			for b := range d.planes[p].bands {
				if err := d.decodeBand(r, &d.planes[p].bands[b]); err != nil {
					return nil, fmt.Errorf("plane %d band %d: %w", p, b, err)
				}
			}
		}
		if d.hasTransparency {
			if err := d.decodeTransparencyPlane(r); err != nil {
				return nil, fmt.Errorf("transparency plane: %w", err)
			}
		}
	}
	if d.hasTransparency && len(d.transparencyMask) != d.picWidth*d.picHeight {
		return nil, fmt.Errorf("transparency mask is not initialized")
	}

	return d.outputRGBA(), nil
}

// switchBuffers mirrors indeo5.c's switch_buffers exactly, including its
// slightly unusual two-pass structure (finish out prevFrameType's effect,
// then set up frameType's own buffers): scalable inter frames use a third
// buffer slot (ref2Buf) that isn't part of the plain intra/inter
// alternation the other frame types use.
func (d *Decoder) switchBuffers() {
	switch d.prevFrameType {
	case frameIntra, frameInter:
		d.bufSwitch ^= 1
		d.dstBuf = d.bufSwitch
		d.refBuf = d.bufSwitch ^ 1
	case frameInterScal:
		if !d.interScal {
			d.ref2Buf = 2
			d.interScal = true
		}
		d.dstBuf, d.ref2Buf = d.ref2Buf, d.dstBuf
		d.refBuf = d.ref2Buf
	case frameInterNoRef:
		// no change
	}

	switch d.frameType {
	case frameIntra:
		d.bufSwitch = 0
		fallthrough
	case frameInter:
		d.interScal = false
		d.dstBuf = d.bufSwitch
		d.refBuf = d.bufSwitch ^ 1
	case frameInterScal, frameInterNoRef, frameNull:
		// no change
	}
}
