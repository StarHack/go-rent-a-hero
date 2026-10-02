package recorder

import (
	"bytes"
	"encoding/binary"
)

// box wraps payload in an ISO-BMFF box: a 4-byte big-endian size (including
// the 8-byte header) followed by the 4-byte ASCII type and the payload
// itself.
func box(kind string, payload []byte) []byte {
	buf := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(buf[0:4], uint32(8+len(payload)))
	copy(buf[4:8], kind)
	copy(buf[8:], payload)
	return buf
}

// u32/u16 append a big-endian value to buf, for the many fixed-width
// integer fields ISO-BMFF boxes are built from.
func u32(buf *bytes.Buffer, v uint32) { binary.Write(buf, binary.BigEndian, v) } //nolint:errcheck // bytes.Buffer.Write never fails
func u16(buf *bytes.Buffer, v uint16) { binary.Write(buf, binary.BigEndian, v) } //nolint:errcheck

// identityMatrix9 is the standard 3x3 unity transform (9 32-bit 16.16
// fixed-point values) every mvhd/tkhd carries.
func identityMatrix9(buf *bytes.Buffer) {
	u32(buf, 0x00010000)
	u32(buf, 0)
	u32(buf, 0)
	u32(buf, 0)
	u32(buf, 0x00010000)
	u32(buf, 0)
	u32(buf, 0)
	u32(buf, 0)
	u32(buf, 0x40000000)
}

// ftypBox declares this file as a QuickTime-family container (isom-
// compatible, but the video sample entry below, 'jpeg', is a QuickTime
// codec type rather than a strictly ISO/MPEG-4 one -- mainstream players
// handle this without issue).
func ftypBox() []byte {
	var buf bytes.Buffer
	buf.WriteString("qt  ") // major_brand
	u32(&buf, 0x00000200)   // minor_version
	buf.WriteString("qt  ") // compatible_brands[0]
	return box("ftyp", buf.Bytes())
}

// buildMoov builds the full movie box for a single video track of
// r.width x r.height JPEG samples at r.fps frames/second, with sample data
// starting at absolute file offset dataStart.
func (r *Recorder) buildMoov(dataStart int) []byte {
	frameCount := uint32(len(r.sizes))
	timescale := uint32(r.fps)

	mvhd := r.buildMvhd(timescale, frameCount)
	trak := box("trak", concat(
		r.buildTkhd(frameCount),
		box("mdia", concat(
			r.buildMdhd(timescale, frameCount),
			buildHdlr(),
			box("minf", concat(
				buildVmhd(),
				box("dinf", box("dref", buildDref())),
				box("stbl", concat(
					r.buildStsd(),
					buildStts(frameCount),
					buildStsc(frameCount),
					r.buildStsz(),
					r.buildStco(dataStart),
				)),
			)),
		)),
	))

	return box("moov", concat(mvhd, trak))
}

func concat(parts ...[]byte) []byte {
	var buf bytes.Buffer
	for _, p := range parts {
		buf.Write(p)
	}
	return buf.Bytes()
}

func (r *Recorder) buildMvhd(timescale, frameCount uint32) []byte {
	var buf bytes.Buffer
	u32(&buf, 0) // version(0) + flags(0)
	u32(&buf, 0) // creation_time
	u32(&buf, 0) // modification_time
	u32(&buf, timescale)
	u32(&buf, frameCount) // duration, in movie timescale units
	u32(&buf, 0x00010000) // rate: 1.0
	u16(&buf, 0x0100)     // volume: 1.0
	u16(&buf, 0)          // reserved
	u32(&buf, 0)          // reserved
	u32(&buf, 0)          // reserved
	identityMatrix9(&buf)
	buf.Write(make([]byte, 24)) // pre_defined
	u32(&buf, 2)                // next_track_ID
	return box("mvhd", buf.Bytes())
}

func (r *Recorder) buildTkhd(frameCount uint32) []byte {
	var buf bytes.Buffer
	u32(&buf, 0x00000007) // version(0) + flags: enabled|in movie|in preview
	u32(&buf, 0)          // creation_time
	u32(&buf, 0)          // modification_time
	u32(&buf, 1)          // track_ID
	u32(&buf, 0)          // reserved
	u32(&buf, frameCount) // duration, in movie timescale units
	u32(&buf, 0)          // reserved
	u32(&buf, 0)          // reserved
	u16(&buf, 0)          // layer
	u16(&buf, 0)          // alternate_group
	u16(&buf, 0)          // volume: 0 for a video track
	u16(&buf, 0)          // reserved
	identityMatrix9(&buf)
	u32(&buf, uint32(r.width)<<16)  // width, 16.16 fixed point
	u32(&buf, uint32(r.height)<<16) // height, 16.16 fixed point
	return box("tkhd", buf.Bytes())
}

func (r *Recorder) buildMdhd(timescale, frameCount uint32) []byte {
	var buf bytes.Buffer
	u32(&buf, 0)          // version(0) + flags(0)
	u32(&buf, 0)          // creation_time
	u32(&buf, 0)          // modification_time
	u32(&buf, timescale)  // timescale, in this track's own units
	u32(&buf, frameCount) // duration
	u16(&buf, 0x55c4)     // language: packed ISO-639-2/T "und" (undetermined)
	u16(&buf, 0)          // pre_defined
	return box("mdhd", buf.Bytes())
}

func buildHdlr() []byte {
	var buf bytes.Buffer
	u32(&buf, 0) // version(0) + flags(0)
	u32(&buf, 0) // pre_defined
	buf.WriteString("vide")
	buf.Write(make([]byte, 12)) // reserved
	buf.WriteString("VideoHandler\x00")
	return box("hdlr", buf.Bytes())
}

func buildVmhd() []byte {
	var buf bytes.Buffer
	u32(&buf, 0x00000001) // version(0) + flags: no-lean-ahead
	u16(&buf, 0)          // graphicsmode
	u16(&buf, 0)          // opcolor r
	u16(&buf, 0)          // opcolor g
	u16(&buf, 0)          // opcolor b
	return box("vmhd", buf.Bytes())
}

func buildDref() []byte {
	var buf bytes.Buffer
	u32(&buf, 0) // version(0) + flags(0)
	u32(&buf, 1) // entry_count

	var url bytes.Buffer
	u32(&url, 0x00000001) // version(0) + flags: media data is in this same file
	buf.Write(box("url ", url.Bytes()))

	return buf.Bytes()
}

// buildStsd builds the sample description box: one video sample entry
// describing QuickTime Photo JPEG frames at r.width x r.height.
func (r *Recorder) buildStsd() []byte {
	var entry bytes.Buffer
	buf6 := make([]byte, 6)
	entry.Write(buf6) // reserved
	u16(&entry, 1)    // data_reference_index
	u16(&entry, 0)    // pre_defined (version)
	u16(&entry, 0)    // reserved
	u32(&entry, 0)    // pre_defined
	u32(&entry, 0)    // pre_defined
	u32(&entry, 0)    // pre_defined
	u16(&entry, uint16(r.width))
	u16(&entry, uint16(r.height))
	u32(&entry, 0x00480000)       // horizresolution: 72 dpi
	u32(&entry, 0x00480000)       // vertresolution: 72 dpi
	u32(&entry, 0)                // reserved
	u16(&entry, 1)                // frame_count
	entry.Write(make([]byte, 32)) // compressorname (empty Pascal string)
	u16(&entry, 0x0018)           // depth: 24-bit RGB
	u16(&entry, 0xffff)           // pre_defined: -1

	jpegEntry := box("jpeg", entry.Bytes())

	var buf bytes.Buffer
	u32(&buf, 0) // version(0) + flags(0)
	u32(&buf, 1) // entry_count
	buf.Write(jpegEntry)

	return box("stsd", buf.Bytes())
}

// buildStts builds the (uniform) time-to-sample box: every sample lasts
// exactly one tick of the track's timescale, i.e. 1/fps seconds.
func buildStts(frameCount uint32) []byte {
	var buf bytes.Buffer
	u32(&buf, 0) // version(0) + flags(0)
	if frameCount == 0 {
		u32(&buf, 0) // entry_count
		return box("stts", buf.Bytes())
	}
	u32(&buf, 1) // entry_count
	u32(&buf, frameCount)
	u32(&buf, 1) // sample_delta
	return box("stts", buf.Bytes())
}

// buildStsc builds the sample-to-chunk box: one sample per chunk
// throughout, the simplest (if not most compact) mapping.
func buildStsc(frameCount uint32) []byte {
	var buf bytes.Buffer
	u32(&buf, 0) // version(0) + flags(0)
	if frameCount == 0 {
		u32(&buf, 0)
		return box("stsc", buf.Bytes())
	}
	u32(&buf, 1) // entry_count
	u32(&buf, 1) // first_chunk
	u32(&buf, 1) // samples_per_chunk
	u32(&buf, 1) // sample_description_index
	return box("stsc", buf.Bytes())
}

// buildStsz builds the sample size box from each frame's actual encoded
// JPEG byte length.
func (r *Recorder) buildStsz() []byte {
	var buf bytes.Buffer
	u32(&buf, 0) // version(0) + flags(0)
	u32(&buf, 0) // sample_size: 0 means "sizes vary, see the table below"
	u32(&buf, uint32(len(r.sizes)))
	for _, sz := range r.sizes {
		u32(&buf, sz)
	}
	return box("stsz", buf.Bytes())
}

// buildStco builds the chunk offset box: since every sample is its own
// chunk (see buildStsc), this is one absolute file offset per frame,
// starting at dataStart and advancing by each frame's encoded size.
func (r *Recorder) buildStco(dataStart int) []byte {
	var buf bytes.Buffer
	u32(&buf, 0) // version(0) + flags(0)
	u32(&buf, uint32(len(r.sizes)))

	offset := uint32(dataStart)
	for _, sz := range r.sizes {
		u32(&buf, offset)
		offset += sz
	}

	return box("stco", buf.Bytes())
}
