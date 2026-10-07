package main

import (
	"archive/zip"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
)

type zipImageSource struct {
	archive *zip.ReadCloser
	inner   sourceFS
	reader  *zipReaderAt
}

type zipReaderAt struct {
	file   *zip.File
	mu     sync.Mutex
	reader io.ReadCloser
	data   []byte
	eof    bool
}

func openZIPSource(name string) (sourceFS, error) {
	zr, err := zip.OpenReader(name)
	if err != nil {
		return nil, err
	}

	var isoFile *zip.File
	var rawFile *zip.File
	var cueFile *zip.File
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		switch strings.ToLower(filepath.Ext(f.Name)) {
		case ".iso":
			if isoFile == nil {
				isoFile = f
			}
		case ".bin", ".img":
			if rawFile == nil {
				rawFile = f
			}
		case ".cue":
			if cueFile == nil {
				cueFile = f
			}
		}
	}

	selected := isoFile
	kind := "iso"
	if selected == nil {
		selected = rawFile
		kind = "raw"
	}
	if selected == nil {
		zr.Close()
		return nil, fmt.Errorf("zip contains no ISO, BIN, or IMG image")
	}

	reader := &zipReaderAt{file: selected}
	var inner sourceFS
	if kind == "iso" {
		inner, err = newISOSource(nil, reader, int64(selected.UncompressedSize64))
	} else {
		var cue []byte
		if cueFile != nil {
			cue, err = readZIPFile(cueFile)
			if err != nil {
				zr.Close()
				return nil, err
			}
		}
		inner, err = openBINReader(reader, int64(selected.UncompressedSize64), filepath.Base(selected.Name), cue)
	}
	if err != nil {
		reader.Close()
		zr.Close()
		return nil, err
	}
	return &zipImageSource{archive: zr, inner: inner, reader: reader}, nil
}

func readZIPFile(f *zip.File) ([]byte, error) {
	r, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func (r *zipReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, fmt.Errorf("negative read offset")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	end := off + int64(len(p))
	if err := r.fill(end); err != nil && err != io.EOF {
		return 0, err
	}
	if off >= int64(len(r.data)) {
		return 0, io.EOF
	}

	n := copy(p, r.data[off:])
	if n != len(p) {
		return n, io.EOF
	}
	return n, nil
}

func (r *zipReaderAt) fill(end int64) error {
	if end <= int64(len(r.data)) {
		return nil
	}
	if r.eof {
		return io.EOF
	}
	if r.reader == nil {
		reader, err := r.file.Open()
		if err != nil {
			return err
		}
		r.reader = reader
	}

	buf := make([]byte, 256*1024)
	for int64(len(r.data)) < end {
		want := len(buf)
		remaining := end - int64(len(r.data))
		if remaining < int64(want) {
			want = int(remaining)
		}
		n, err := r.reader.Read(buf[:want])
		if n > 0 {
			r.data = append(r.data, buf[:n]...)
		}
		if err != nil {
			if err == io.EOF {
				r.eof = true
				r.reader.Close()
				r.reader = nil
			}
			return err
		}
		if n == 0 {
			return io.ErrNoProgress
		}
	}
	return nil
}

func (r *zipReaderAt) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.reader != nil {
		err := r.reader.Close()
		r.reader = nil
		return err
	}
	return nil
}

func (z *zipImageSource) Open(name string) (io.ReadSeekCloser, error) {
	return z.inner.Open(name)
}

func (z *zipImageSource) ReadDir(name string) ([]sourceEntry, error) {
	return z.inner.ReadDir(name)
}

func (z *zipImageSource) Close() error {
	if z.inner != nil {
		z.inner.Close()
	}
	if z.reader != nil {
		z.reader.Close()
	}
	if z.archive != nil {
		return z.archive.Close()
	}
	return nil
}
