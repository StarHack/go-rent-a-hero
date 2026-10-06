package avi

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	return abs
}

func readFixture(t *testing.T, rel string) []byte {
	t.Helper()
	path := filepath.Join(repoRoot(t), rel)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture %s not available: %v", path, err)
	}
	return data
}

func TestParseLiftVideo(t *testing.T) {
	data := readFixture(t, "data/cd/GAME/LOC01/LIFTMITTERAUF.AVI")

	vs, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if vs.Width != 640 || vs.Height != 360 {
		t.Errorf("dims = %dx%d, want 640x360", vs.Width, vs.Height)
	}
	if vs.Compression != "cvid" {
		t.Errorf("Compression = %q, want \"cvid\"", vs.Compression)
	}
	if vs.BitCount != 24 {
		t.Errorf("BitCount = %d, want 24", vs.BitCount)
	}
	if math.Abs(vs.FPS-10) > 0.01 {
		t.Errorf("FPS = %v, want 10", vs.FPS)
	}
	if len(vs.Frames) != 30 {
		t.Fatalf("len(Frames) = %d, want 30", len(vs.Frames))
	}

	// Every frame chunk should at least contain a 10-byte Cinepak frame
	// header, and its own embedded width/height (big-endian, bytes 4-7)
	// should agree with the container's.
	for i, f := range vs.Frames {
		if len(f) < 10 {
			t.Fatalf("frame %d: only %d bytes", i, len(f))
		}
		w := int(f[4])<<8 | int(f[5])
		h := int(f[6])<<8 | int(f[7])
		if w != vs.Width || h != vs.Height {
			t.Errorf("frame %d: embedded dims %dx%d, want %dx%d", i, w, h, vs.Width, vs.Height)
		}
	}
}

func TestParseRejectsNonRIFF(t *testing.T) {
	if _, err := Parse([]byte("not an avi file at all")); err == nil {
		t.Fatal("expected an error for non-RIFF data")
	}
}

func TestParseCollectsFramesAcrossAVIXSegments(t *testing.T) {
	strh := make([]byte, 28)
	copy(strh[0:4], "vids")
	binary.LittleEndian.PutUint32(strh[20:24], 1)
	binary.LittleEndian.PutUint32(strh[24:28], 15)

	strf := make([]byte, 40)
	binary.LittleEndian.PutUint32(strf[0:4], 40)
	binary.LittleEndian.PutUint32(strf[4:8], 320)
	binary.LittleEndian.PutUint32(strf[8:12], 200)
	binary.LittleEndian.PutUint16(strf[14:16], 24)
	copy(strf[16:20], "cvid")

	strl := riffList("strl", riffChunk("strh", strh), riffChunk("strf", strf))
	hdrl := riffList("hdrl", strl)
	firstFrame := []byte{1, 2, 3}
	secondFrame := []byte{4, 5, 6, 7}
	avi := riffFile("AVI ", hdrl, riffList("movi", riffChunk("00dc", firstFrame)))
	avix := riffFile("AVIX", riffList("movi", riffChunk("00dc", secondFrame)))
	data := append(avi, avix...)

	vs, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(vs.Frames) != 2 {
		t.Fatalf("len(Frames) = %d, want 2", len(vs.Frames))
	}
	if !bytes.Equal(vs.Frames[0], firstFrame) {
		t.Fatalf("frame 0 = %v, want %v", vs.Frames[0], firstFrame)
	}
	if !bytes.Equal(vs.Frames[1], secondFrame) {
		t.Fatalf("frame 1 = %v, want %v", vs.Frames[1], secondFrame)
	}
}

func TestParseIgnoresTrailingBytesAfterRIFFSegments(t *testing.T) {
	strh := make([]byte, 28)
	copy(strh[0:4], "vids")
	binary.LittleEndian.PutUint32(strh[20:24], 1)
	binary.LittleEndian.PutUint32(strh[24:28], 10)

	strf := make([]byte, 40)
	binary.LittleEndian.PutUint32(strf[0:4], 40)
	binary.LittleEndian.PutUint32(strf[4:8], 16)
	binary.LittleEndian.PutUint32(strf[8:12], 16)
	binary.LittleEndian.PutUint16(strf[14:16], 24)
	copy(strf[16:20], "cvid")

	data := riffFile("AVI ", riffList("hdrl", riffList("strl", riffChunk("strh", strh), riffChunk("strf", strf))), riffList("movi", riffChunk("00dc", []byte{9})))
	data = append(data, []byte("trailing-data")...)

	vs, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(vs.Frames) != 1 {
		t.Fatalf("len(Frames) = %d, want 1", len(vs.Frames))
	}
}

func riffChunk(id string, body []byte) []byte {
	var buf bytes.Buffer
	buf.WriteString(id)
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(body)))
	buf.Write(body)
	if len(body)%2 != 0 {
		buf.WriteByte(0)
	}
	return buf.Bytes()
}

func riffList(listType string, chunks ...[]byte) []byte {
	body := []byte(listType)
	for _, chunk := range chunks {
		body = append(body, chunk...)
	}
	return riffChunk("LIST", body)
}

func riffFile(formType string, chunks ...[]byte) []byte {
	body := []byte(formType)
	for _, chunk := range chunks {
		body = append(body, chunk...)
	}
	return riffChunk("RIFF", body)
}
