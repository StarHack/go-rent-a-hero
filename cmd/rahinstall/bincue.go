package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var rawCDSync = []byte{
	0x00,
	0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff,
	0x00,
}

type rawCDReader struct {
	reader        io.ReaderAt
	rawSize       int64
	sectorSize    int64
	payloadOffset int64
	trackSector   int64
}

func isRawCDSync(data []byte) bool {
	if len(data) < len(rawCDSync) {
		return false
	}

	for i := range rawCDSync {
		if data[i] != rawCDSync[i] {
			return false
		}
	}

	return true
}

func openBINSource(name string) (sourceFS, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	trackSector, payloadOffset, err := detectBINLayout(name, f, info.Size())
	if err != nil {
		f.Close()
		return nil, err
	}

	raw := &rawCDReader{
		reader:        f,
		rawSize:       info.Size(),
		sectorSize:    2352,
		payloadOffset: payloadOffset,
		trackSector:   trackSector,
	}

	logicalSectors := info.Size()/2352 - trackSector
	if logicalSectors <= 16 {
		f.Close()
		return nil, fmt.Errorf("BIN data track is too small")
	}

	src, err := newISOSource(f, raw, logicalSectors*isoSectorSize)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("BIN does not contain a supported ISO9660 data track: %w", err)
	}

	return src, nil
}

func detectBINLayout(name string, f *os.File, size int64) (int64, int64, error) {
	if cueTrack, cueMode, ok := readSiblingCUE(name); ok {
		switch cueMode {
		case "MODE1/2352":
			return cueTrack, 16, nil
		case "MODE2/2352":
			return cueTrack, 24, nil
		}
	}

	candidates := []int64{0}

	for _, sector := range candidates {
		var header [24]byte
		offset := sector * 2352

		if _, err := f.ReadAt(header[:], offset); err != nil {
			continue
		}

		if !isRawCDSync(header[:12]) {
			continue
		}

		switch header[15] {
		case 1:
			return sector, 16, nil
		case 2:
			return sector, 24, nil
		}
	}

	maxSectors := size / 2352
	limit := maxSectors
	if limit > 45000 {
		limit = 45000
	}

	var header [24]byte

	for sector := int64(0); sector < limit; sector++ {
		offset := sector * 2352

		if _, err := f.ReadAt(header[:], offset); err != nil {
			break
		}

		if !isRawCDSync(header[:12]) {
			continue
		}

		if header[15] != 1 && header[15] != 2 {
			continue
		}

		payloadOffset := int64(16)
		if header[15] == 2 {
			payloadOffset = 24
		}

		var pvd [6]byte
		pvdOffset := (sector+16)*2352 + payloadOffset

		if _, err := f.ReadAt(pvd[:], pvdOffset); err != nil {
			continue
		}

		if pvd[0] == 1 && string(pvd[1:6]) == "CD001" {
			return sector, payloadOffset, nil
		}
	}

	return 0, 0, fmt.Errorf("unable to locate MODE1/2352 or MODE2/2352 ISO track")
}

func readSiblingCUE(binName string) (int64, string, bool) {
	base := strings.TrimSuffix(binName, filepath.Ext(binName))

	candidates := []string{
		base + ".cue",
		base + ".CUE",
	}

	for _, cueName := range candidates {
		f, err := os.Open(cueName)
		if err != nil {
			continue
		}

		sector, mode, ok := parseCUE(f, filepath.Base(binName))
		f.Close()

		if ok {
			return sector, mode, true
		}
	}

	return 0, "", false
}

func parseCUE(r io.Reader, binBase string) (int64, string, bool) {
	scanner := bufio.NewScanner(r)

	currentFile := ""
	currentMode := ""
	matchesFile := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := splitCUEFields(line)

		if len(fields) == 0 {
			continue
		}

		switch strings.ToUpper(fields[0]) {
		case "FILE":
			if len(fields) >= 2 {
				currentFile = fields[1]
				matchesFile = strings.EqualFold(filepath.Base(currentFile), binBase)
				currentMode = ""
			}

		case "TRACK":
			if matchesFile && len(fields) >= 3 {
				mode := strings.ToUpper(fields[2])
				if mode == "MODE1/2352" || mode == "MODE2/2352" {
					currentMode = mode
				} else {
					currentMode = ""
				}
			}

		case "INDEX":
			if matchesFile && currentMode != "" && len(fields) >= 3 && fields[1] == "01" {
				sector, err := cueTimeToSector(fields[2])
				if err == nil {
					return sector, currentMode, true
				}
			}
		}
	}

	return 0, "", false
}

func splitCUEFields(line string) []string {
	var fields []string

	for len(line) > 0 {
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		if line[0] == '"' {
			end := strings.IndexByte(line[1:], '"')
			if end < 0 {
				fields = append(fields, line[1:])
				break
			}

			end++
			fields = append(fields, line[1:end])
			line = line[end+1:]
			continue
		}

		end := strings.IndexAny(line, " \t")
		if end < 0 {
			fields = append(fields, line)
			break
		}

		fields = append(fields, line[:end])
		line = line[end:]
	}

	return fields
}

func cueTimeToSector(value string) (int64, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid CUE time %q", value)
	}

	minutes, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}

	seconds, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, err
	}

	frames, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return 0, err
	}

	if seconds >= 60 || frames >= 75 {
		return 0, fmt.Errorf("invalid CUE time %q", value)
	}

	return (minutes*60+seconds)*75 + frames, nil
}

func (r *rawCDReader) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, fmt.Errorf("negative read offset")
	}

	total := 0

	for len(p) > 0 {
		logicalSector := off / isoSectorSize
		inSector := off % isoSectorSize

		rawSector := r.trackSector + logicalSector
		rawOffset := rawSector*r.sectorSize + r.payloadOffset + inSector

		if rawOffset >= r.rawSize {
			if total == 0 {
				return 0, io.EOF
			}
			return total, io.EOF
		}

		count := int64(len(p))
		remaining := int64(isoSectorSize) - inSector

		if count > remaining {
			count = remaining
		}

		n, err := r.reader.ReadAt(p[:count], rawOffset)

		total += n
		p = p[n:]
		off += int64(n)

		if err != nil {
			return total, err
		}

		if int64(n) != count {
			return total, io.ErrUnexpectedEOF
		}
	}

	return total, nil
}
