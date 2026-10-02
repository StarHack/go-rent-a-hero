package indeo5

import "testing"

type transparencyBitWriter struct {
	data []byte
	pos  int
}

func (w *transparencyBitWriter) bits(v uint32, n int) {
	for i := 0; i < n; i++ {
		if w.pos>>3 >= len(w.data) {
			w.data = append(w.data, 0)
		}
		if v&(1<<uint(i)) != 0 {
			w.data[w.pos>>3] |= 1 << uint(w.pos&7)
		}
		w.pos++
	}
}

func (w *transparencyBitWriter) msb(v, n int) {
	for i := n - 1; i >= 0; i-- {
		w.bits(uint32(v>>uint(i)&1), 1)
	}
}

func (w *transparencyBitWriter) align() {
	for w.pos&7 != 0 {
		w.bits(0, 1)
	}
}

func transparencyPayload(withDesc bool) []byte {
	w := &transparencyBitWriter{}
	w.bits(0, 24)
	w.bits(0, 4)
	for range 4 {
		w.bits(0, 16)
	}
	if withDesc {
		w.bits(1, 1)
		w.bits(1, 4)
		w.bits(5, 4)
	} else {
		w.bits(0, 1)
	}
	w.align()
	w.bits(0, 1)
	w.bits(0, 1)
	w.align()
	w.bits(0, 1)
	for _, run := range []int{8, 8, 16, 8, 8, 16} {
		w.msb(run, 5)
	}
	w.align()
	size := len(w.data)
	w.data[0] = byte(size)
	w.data[1] = byte(size >> 8)
	w.data[2] = byte(size >> 16)
	return w.data
}

func TestDecodeTransparencyPlane(t *testing.T) {
	d := &Decoder{picWidth: 32, picHeight: 2}
	r := newBitReader(transparencyPayload(true))
	if err := d.decodeTransparencyPlane(r); err != nil {
		t.Fatalf("decodeTransparencyPlane: %v", err)
	}
	if !d.haveTransparencyHuff {
		t.Fatal("transparency Huffman descriptor was not retained")
	}
	for x := 0; x < 32; x++ {
		want0 := x >= 8 && x < 16
		if got := d.transparencyMask[x]; got != want0 {
			t.Fatalf("row 0 x=%d = %v, want %v", x, got, want0)
		}
		if got := d.transparencyMask[32+x]; !got {
			t.Fatalf("row 1 x=%d = false, want true", x)
		}
	}
}

func TestTransparencyHuffmanDescriptorPersists(t *testing.T) {
	d := &Decoder{picWidth: 32, picHeight: 2}
	if err := d.decodeTransparencyPlane(newBitReader(transparencyPayload(true))); err != nil {
		t.Fatalf("first transparency plane: %v", err)
	}
	if err := d.decodeTransparencyPlane(newBitReader(transparencyPayload(false))); err != nil {
		t.Fatalf("second transparency plane: %v", err)
	}
}

func solidTransparencyPayload(opaque bool) []byte {
	w := &transparencyBitWriter{}
	w.bits(0, 24)
	w.bits(0, 4)
	for range 4 {
		w.bits(0, 16)
	}
	w.bits(0, 1)
	w.align()
	w.bits(1, 1)
	if opaque {
		w.bits(1, 1)
	} else {
		w.bits(0, 1)
	}
	w.align()
	size := len(w.data)
	w.data[0] = byte(size)
	w.data[1] = byte(size >> 8)
	w.data[2] = byte(size >> 16)
	return w.data
}

func TestDecodeSolidTransparencyPlane(t *testing.T) {
	for _, opaque := range []bool{false, true} {
		d := &Decoder{picWidth: 7, picHeight: 5}
		payload := solidTransparencyPayload(opaque)
		r := newBitReader(payload)
		if err := d.decodeTransparencyPlane(r); err != nil {
			t.Fatalf("opaque=%v: decodeTransparencyPlane: %v", opaque, err)
		}
		if r.pos != len(payload)*8 {
			t.Fatalf("opaque=%v: bit position %d, want %d", opaque, r.pos, len(payload)*8)
		}
		if len(d.transparencyMask) != d.picWidth*d.picHeight {
			t.Fatalf("opaque=%v: mask size %d, want %d", opaque, len(d.transparencyMask), d.picWidth*d.picHeight)
		}
		for i, got := range d.transparencyMask {
			if got != opaque {
				t.Fatalf("opaque=%v: mask[%d] = %v", opaque, i, got)
			}
		}
	}
}
