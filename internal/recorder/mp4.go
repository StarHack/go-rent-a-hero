// Package recorder implements a minimal, dependency-free MP4 (ISO base
// media file format) writer for screen recording: each captured frame is
// JPEG-encoded via the standard library ("image/jpeg", no external
// dependency) and stored as one QuickTime "Photo JPEG" ('jpeg') sample,
// muxed into a single video-only track.
//
// This is deliberately not a general-purpose MP4 library: it writes just
// enough box structure (ftyp/moov/mdat with one video trak, one sample per
// chunk) to produce a file mainstream players (QuickTime, VLC, ffplay) can
// open, matching this project's standing rule of no runtime dependency on
// ffmpeg or any other external media tool for either decoding or encoding.
package recorder

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"os"
)

// Recorder captures a sequence of RGBA frames and muxes them into path as
// an MP4 file on Close. Frames are JPEG-encoded and appended to a scratch
// file as they arrive (not held in memory), so recording length is bounded
// only by disk space.
type Recorder struct {
	path string
	fps  int

	scratch *os.File
	sizes   []uint32 // one JPEG-encoded byte length per frame, in order

	width, height int // fixed by the first frame; later mismatches are dropped
}

// New creates a Recorder that will write path (its container format is MP4
// regardless of the extension given) once Close is called, at fps frames
// per second. fps must be positive.
func New(path string, fps int) (*Recorder, error) {
	if fps <= 0 {
		return nil, fmt.Errorf("recorder: fps must be positive, got %d", fps)
	}

	scratch, err := os.CreateTemp("", "rah-recording-*.jpegs")
	if err != nil {
		return nil, fmt.Errorf("recorder: create scratch file: %w", err)
	}

	return &Recorder{path: path, fps: fps, scratch: scratch}, nil
}

// AddFrame JPEG-encodes img and appends it as the next sample. The first
// call fixes the recording's dimensions; a later frame with different
// dimensions is silently dropped (logged by the caller if desired) rather
// than corrupting the fixed-size video sample description.
func (r *Recorder) AddFrame(img *image.RGBA) error {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	if r.width == 0 && r.height == 0 {
		r.width, r.height = w, h
	} else if w != r.width || h != r.height {
		return nil
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return fmt.Errorf("recorder: encode frame %d: %w", len(r.sizes), err)
	}

	if _, err := r.scratch.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("recorder: write frame %d: %w", len(r.sizes), err)
	}

	r.sizes = append(r.sizes, uint32(buf.Len()))

	return nil
}

// Close finalizes the MP4 file at the configured path and removes the
// scratch file. It is safe to call even if no frames were ever added (the
// scratch file is still cleaned up); in that case no output file is
// written, since a video track with zero samples/zero dimensions has
// nothing meaningful to describe.
func (r *Recorder) Close() error {
	defer func() {
		name := r.scratch.Name()
		r.scratch.Close()
		os.Remove(name)
	}()

	if len(r.sizes) == 0 || r.width == 0 || r.height == 0 {
		return nil
	}

	out, err := os.Create(r.path)
	if err != nil {
		return fmt.Errorf("recorder: create %s: %w", r.path, err)
	}
	defer out.Close()

	ftyp := ftypBox()

	// stco's chunk offsets are absolute file offsets, so they depend on the
	// moov box's own serialized length -- which does not depend on the
	// *values* inside stco, only its (fixed) entry count. Build moov once
	// with a placeholder data start to measure it, then again with the
	// real one; both have identical length.
	placeholderMoov := r.buildMoov(0)
	dataStart := len(ftyp) + len(placeholderMoov) + 8 // +8 for the mdat box header
	moov := r.buildMoov(dataStart)

	if len(moov) != len(placeholderMoov) {
		return fmt.Errorf("recorder: internal error: moov size changed between passes (%d vs %d)", len(placeholderMoov), len(moov))
	}

	var totalData int64
	for _, sz := range r.sizes {
		totalData += int64(sz)
	}

	mdatHeader := make([]byte, 8)
	binary.BigEndian.PutUint32(mdatHeader[0:4], uint32(8+totalData))
	copy(mdatHeader[4:8], "mdat")

	if _, err := out.Write(ftyp); err != nil {
		return fmt.Errorf("recorder: write ftyp: %w", err)
	}
	if _, err := out.Write(moov); err != nil {
		return fmt.Errorf("recorder: write moov: %w", err)
	}
	if _, err := out.Write(mdatHeader); err != nil {
		return fmt.Errorf("recorder: write mdat header: %w", err)
	}

	if _, err := r.scratch.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("recorder: seek scratch file: %w", err)
	}
	if _, err := io.Copy(out, r.scratch); err != nil {
		return fmt.Errorf("recorder: copy frame data: %w", err)
	}

	return out.Close()
}
