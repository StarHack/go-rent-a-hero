package indeo5

import "fmt"

// huffDesc describes one Indeo Huffman codebook: num_rows "rows", row i
// holding 2^xbits[i] codes. A code in row i is, from the first bit
// consumed: i one-bits, then (unless i is the last row) a terminating
// zero-bit, then xbits[i] more bits giving the code's position within its
// row (see ivi_create_huff_from_desc in FFmpeg's ivi.c). The overall
// symbol value is the running count of codes in all earlier rows plus that
// row-local position -- i.e. codes are numbered 0.. in the exact order
// ivi_create_huff_from_desc assigns them.
type huffDesc struct {
	numRows int
	xbits   [16]int
}

// predefined macroblock and block Huffman descriptors (ivi_mb_huff_desc /
// ivi_blk_huff_desc in FFmpeg's ivi.c). Index 7 is never selected by these
// tables themselves (tab_sel==7 means "custom", decoded from the
// bitstream instead), but the slice still has 8 entries to keep indices
// aligned with the bitstream's 3-bit tab_sel field.
var mbHuffDescs = [8]huffDesc{
	{8, [16]int{0, 4, 5, 4, 4, 4, 6, 6}},
	{12, [16]int{0, 2, 2, 3, 3, 3, 3, 5, 3, 2, 2, 2}},
	{12, [16]int{0, 2, 3, 4, 3, 3, 3, 3, 4, 3, 2, 2}},
	{12, [16]int{0, 3, 4, 4, 3, 3, 3, 3, 3, 2, 2, 2}},
	{13, [16]int{0, 4, 4, 3, 3, 3, 3, 2, 3, 3, 2, 1, 1}},
	{9, [16]int{0, 4, 4, 4, 4, 3, 3, 3, 2}},
	{10, [16]int{0, 4, 4, 4, 4, 3, 3, 2, 2, 2}},
	{12, [16]int{0, 4, 4, 4, 3, 3, 2, 3, 2, 2, 2, 2}},
}

var blkHuffDescs = [8]huffDesc{
	{10, [16]int{1, 2, 3, 4, 4, 7, 5, 5, 4, 1}},
	{11, [16]int{2, 3, 4, 4, 4, 7, 5, 4, 3, 3, 2}},
	{12, [16]int{2, 4, 5, 5, 5, 5, 6, 4, 4, 3, 1, 1}},
	{13, [16]int{3, 3, 4, 4, 5, 6, 6, 4, 4, 3, 2, 1, 1}},
	{11, [16]int{3, 4, 4, 5, 5, 5, 6, 5, 4, 2, 2}},
	{13, [16]int{3, 4, 5, 5, 5, 5, 6, 4, 3, 3, 2, 1, 1}},
	{13, [16]int{3, 4, 5, 5, 5, 6, 5, 4, 3, 3, 2, 1, 1}},
	{9, [16]int{3, 4, 4, 5, 5, 5, 6, 5, 5}},
}

// decodeSymbol reads one Huffman-coded symbol per desc from r: a unary
// row-selector (see huffDesc's doc comment) followed by that row's
// fixed-width position field.
func decodeSymbol(r *bitReader, desc *huffDesc) (int, error) {
	row := 0
	for row < desc.numRows-1 {
		bit, err := r.bit()
		if err != nil {
			return 0, err
		}
		if bit == 0 {
			break
		}
		row++
	}

	pos := 0
	for i := 0; i < row; i++ {
		pos += 1 << uint(desc.xbits[i])
	}

	col := 0
	for i := 0; i < desc.xbits[row]; i++ {
		bit, err := r.bit()
		if err != nil {
			return 0, err
		}
		col = (col << 1) | int(bit)
	}

	return pos + col, nil
}

// decodeHuffDesc reads a huffman codebook descriptor from the bitstream in
// the "custom table" format used by both macroblock and block huffman
// signalling (ff_ivi_dec_huff_desc's tab_sel==7 branch).
func decodeCustomHuffDesc(r *bitReader) (huffDesc, error) {
	var desc huffDesc

	numRows, err := r.bits(4)
	if err != nil {
		return desc, err
	}
	if numRows == 0 {
		return desc, fmt.Errorf("indeo5: empty custom huffman table")
	}
	desc.numRows = int(numRows)

	for i := range desc.numRows {
		xb, err := r.bits(4)
		if err != nil {
			return desc, err
		}
		desc.xbits[i] = int(xb)
	}

	return desc, nil
}

// huffTab is a fully-resolved Huffman table selection: either one of the
// eight predefined descriptors, or a custom one just decoded from the
// bitstream.
type huffTab struct {
	desc huffDesc
}

// decodeHuffTab mirrors ff_ivi_dec_huff_desc: if descCoded is false the
// default (predefined table 7) applies; otherwise a 3-bit selector picks
// one of the eight predefined tables, or -- selector 7 -- a custom table
// follows inline.
func decodeHuffTab(r *bitReader, descCoded bool, predefined *[8]huffDesc) (*huffTab, error) {
	if !descCoded {
		return &huffTab{desc: predefined[7]}, nil
	}

	sel, err := r.bits(3)
	if err != nil {
		return nil, err
	}
	if sel == 7 {
		desc, err := decodeCustomHuffDesc(r)
		if err != nil {
			return nil, err
		}
		return &huffTab{desc: desc}, nil
	}

	return &huffTab{desc: predefined[sel]}, nil
}
