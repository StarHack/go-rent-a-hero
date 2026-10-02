package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wok/rent-a-hero/internal/assets"
	"github.com/wok/rent-a-hero/internal/formats/a16"
	"github.com/wok/rent-a-hero/internal/formats/acs"
	"github.com/wok/rent-a-hero/internal/formats/bmp"
	"github.com/wok/rent-a-hero/internal/formats/szn"
	"github.com/wok/rent-a-hero/internal/formats/tcc"
	"github.com/wok/rent-a-hero/internal/formats/zbf"
)

// scanStats accumulates aggregate findings across every file under the
// scanned root. It is intentionally printed even when individual files fail
// to parse, per agents/IMPLEMENTATION.md: "rahdump should be able to scan
// all asset files and report validation failures without stopping at the
// first file."
type scanStats struct {
	extCounts     map[string]int
	tccFlags      map[uint32]int
	tccDims       map[string]int
	a16FrameCount map[int]int
	zbfHeaders    map[string]int
	acsRates      map[float64]int
	sznKeysByKind map[string]map[string]bool // section kind -> set of keys seen
	bmpDims       map[string]int
	referenced    map[string]bool // lowercased filenames referenced from SZN
	errors        []string
}

func newScanStats() *scanStats {
	return &scanStats{
		extCounts:     map[string]int{},
		tccFlags:      map[uint32]int{},
		tccDims:       map[string]int{},
		a16FrameCount: map[int]int{},
		zbfHeaders:    map[string]int{},
		acsRates:      map[float64]int{},
		sznKeysByKind: map[string]map[string]bool{},
		bmpDims:       map[string]int{},
		referenced:    map[string]bool{},
	}
}

func cmdScan(args []string) error {
	fs2 := flag.NewFlagSet("scan", flag.ExitOnError)
	if err := fs2.Parse(args); err != nil {
		return err
	}

	if fs2.NArg() != 1 {
		return fmt.Errorf("usage: rahdump scan <asset-root>")
	}

	root := fs2.Arg(0)

	idx, err := assets.NewIndex(root)
	if err != nil {
		return fmt.Errorf("building asset index: %w", err)
	}

	stats := newScanStats()

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			stats.errors = append(stats.errors, fmt.Sprintf("%s: %v", path, err))
			return nil
		}

		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		stats.extCounts[ext]++

		switch ext {
		case ".tcc":
			scanTCC(stats, path)
		case ".a16":
			scanA16(stats, path)
		case ".zbf":
			scanZBF(stats, path)
		case ".acs":
			scanACS(stats, path)
		case ".bmp":
			scanBMP(stats, path)
		case ".szn":
			scanSZN(stats, path)
		}

		return nil
	})
	if err != nil {
		return err
	}

	printScanReport(stats, idx)

	return nil
}

func recordErr(s *scanStats, path string, err error) {
	s.errors = append(s.errors, fmt.Sprintf("%s: %v", path, err))
}

func scanTCC(s *scanStats, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		recordErr(s, path, err)
		return
	}

	h, err := tcc.ParseHeader(data)
	if err != nil {
		recordErr(s, path, err)
		return
	}

	s.tccFlags[h.Flags]++
	s.tccDims[fmt.Sprintf("%dx%d", h.Width, h.Height)]++

	if _, err := tcc.Decode(data); err != nil {
		recordErr(s, path, fmt.Errorf("decode: %w", err))
	}
}

func scanA16(s *scanStats, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		recordErr(s, path, err)
		return
	}

	anim, err := a16.Parse(data)
	if err != nil {
		recordErr(s, path, err)
		return
	}

	s.a16FrameCount[anim.Count()]++

	for _, f := range anim.Frames {
		s.tccFlags[f.Header.Flags]++
	}
}

func scanZBF(s *scanStats, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		recordErr(s, path, err)
		return
	}

	r, err := zbf.Parse(data)
	if err != nil {
		recordErr(s, path, err)
		return
	}

	key := fmt.Sprintf("headerSize=%d param1=%v min=%d param2=%v max=%d", r.Header.HeaderSize, r.Header.Param1, r.Header.MinValue, r.Header.Param2, r.Header.MaxValue)
	s.zbfHeaders[key]++
}

func scanACS(s *scanStats, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		recordErr(s, path, err)
		return
	}

	track, err := acs.Parse(data)
	if err != nil {
		recordErr(s, path, err)
		return
	}

	s.acsRates[track.Rate]++
}

func scanBMP(s *scanStats, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		recordErr(s, path, err)
		return
	}

	img, err := bmp.Decode(data)
	if err != nil {
		recordErr(s, path, err)
		return
	}

	s.bmpDims[fmt.Sprintf("%dx%d", img.Width, img.Height)]++
}

func scanSZN(s *scanStats, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		recordErr(s, path, err)
		return
	}

	def, err := szn.Load(path, data)
	if err != nil {
		recordErr(s, path, err)
		return
	}

	recordKeys := func(kind string, extra map[string]string) {
		set, ok := s.sznKeysByKind[kind]
		if !ok {
			set = map[string]bool{}
			s.sznKeysByKind[kind] = set
		}
		for k := range extra {
			set[k] = true
		}
	}

	for _, name := range def.General.Backgrounds {
		b := def.Backgrounds[name]
		recordKeys("background", b.Extra)
		s.referenced[strings.ToLower(b.Filename)] = true
		s.referenced[strings.ToLower(b.ZBuf)] = true
		s.referenced[strings.ToLower(b.WBuf)] = true
	}

	for _, name := range def.General.Characters {
		c := def.Characters[name]
		recordKeys("character", c.Extra)
		s.referenced[strings.ToLower(c.Filename)] = true
		s.referenced[strings.ToLower(c.SFXLeft)] = true
		s.referenced[strings.ToLower(c.SFXRight)] = true
		s.referenced[strings.ToLower(c.SFXAlt)] = true
	}

	for _, name := range def.General.Layers {
		l := def.Layers[name]
		recordKeys("layer", l.Extra)
		s.referenced[strings.ToLower(l.Filename)] = true
	}

	for _, name := range def.General.Areas {
		a := def.Areas[name]
		recordKeys("area", a.Extra)
	}

	delete(s.referenced, "")
}

func printScanReport(s *scanStats, idx *assets.Index) {
	fmt.Println("== extensions ==")
	printSortedCounts(mapKeys(s.extCounts), s.extCounts)

	fmt.Println("\n== TCC flags ==")
	for flag, count := range s.tccFlags {
		fmt.Printf("  0x%02x: %d\n", flag, count)
	}

	fmt.Println("\n== TCC/A16 dimensions ==")
	printSortedCounts(mapKeys(s.tccDims), s.tccDims)

	fmt.Println("\n== A16 frame counts ==")
	frameCountKeys := make([]string, 0, len(s.a16FrameCount))
	frameCountVals := map[string]int{}
	for k, v := range s.a16FrameCount {
		key := fmt.Sprintf("%d", k)
		frameCountKeys = append(frameCountKeys, key)
		frameCountVals[key] = v
	}
	printSortedCounts(frameCountKeys, frameCountVals)

	fmt.Println("\n== ZBF header variants ==")
	printSortedCounts(mapKeys(s.zbfHeaders), s.zbfHeaders)

	fmt.Println("\n== ACS rates ==")
	rateKeys := make([]string, 0, len(s.acsRates))
	rateVals := map[string]int{}
	for k, v := range s.acsRates {
		key := fmt.Sprintf("%v", k)
		rateKeys = append(rateKeys, key)
		rateVals[key] = v
	}
	printSortedCounts(rateKeys, rateVals)

	fmt.Println("\n== BMP dimensions ==")
	printSortedCounts(mapKeys(s.bmpDims), s.bmpDims)

	fmt.Println("\n== SZN unknown keys by section kind ==")
	kinds := make([]string, 0, len(s.sznKeysByKind))
	for k := range s.sznKeysByKind {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	for _, kind := range kinds {
		keys := make([]string, 0, len(s.sznKeysByKind[kind]))
		for k := range s.sznKeysByKind[kind] {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Printf("  %s: %v\n", kind, keys)
	}

	fmt.Println("\n== unresolved SZN-referenced filenames ==")
	var unresolved []string
	for name := range s.referenced {
		if !idx.Has(name) {
			unresolved = append(unresolved, name)
		}
	}
	sort.Strings(unresolved)
	if len(unresolved) == 0 {
		fmt.Println("  (none)")
	}
	for _, name := range unresolved {
		fmt.Printf("  %s\n", name)
	}

	fmt.Printf("\n== errors (%d) ==\n", len(s.errors))
	for _, e := range s.errors {
		fmt.Printf("  %s\n", e)
	}
}

func mapKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func printSortedCounts(keys []string, counts map[string]int) {
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %s: %d\n", k, counts[k])
	}
}
