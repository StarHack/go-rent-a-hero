package indeo5

import "fmt"

// bitReader reads Indeo 5's bitstream: bits are consumed least-significant-
// bit first within each byte, byte 0 of the buffer first (verified against
// real game assets: reading the fixed 5-bit picture start code first gives
// exactly 0x1F, and the following GOP header's picture width/height decode
// to exactly the AVI container's own reported dimensions).
type bitReader struct {
	data []byte
	pos  int // absolute bit position
}

func newBitReader(data []byte) *bitReader {
	return &bitReader{data: data}
}

func (r *bitReader) bitsLeft() int {
	return len(r.data)*8 - r.pos
}

// bits reads n (1..32) bits and returns them as an unsigned value, LSB of
// the stream mapping to the lowest bit of successive reads (i.e. the first
// bit consumed is bit 0 of the very first read, not bit n-1).
func (r *bitReader) bits(n int) (uint32, error) {
	if n < 0 || n > 32 {
		return 0, fmt.Errorf("indeo5: invalid bit read width %d", n)
	}
	if r.bitsLeft() < n {
		return 0, fmt.Errorf("indeo5: unexpected end of bitstream")
	}

	byteIdx := r.pos >> 3
	bitOff := uint(r.pos & 7)

	var v uint64
	for i := 0; i < 5 && byteIdx+i < len(r.data); i++ {
		v |= uint64(r.data[byteIdx+i]) << (8 * uint(i))
	}
	v >>= bitOff

	r.pos += n
	if n == 32 {
		return uint32(v), nil
	}
	return uint32(v) & (1<<uint(n) - 1), nil
}

func (r *bitReader) bit() (uint32, error) {
	return r.bits(1)
}

// align advances to the next byte boundary.
func (r *bitReader) align() {
	if rem := r.pos % 8; rem != 0 {
		r.pos += 8 - rem
	}
}

func (r *bitReader) bytePos() int {
	return r.pos >> 3
}
