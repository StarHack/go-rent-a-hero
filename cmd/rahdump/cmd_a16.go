package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wok/rent-a-hero/internal/formats/a16"
)

func cmdA16(args []string) error {
	fs := flag.NewFlagSet("a16", flag.ExitOnError)
	framesOut := fs.String("frames", "", "write every decoded frame as PNG into this directory")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return fmt.Errorf("usage: rahdump a16 <file.a16> [--frames out-dir/]")
	}

	path := fs.Arg(0)

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	anim, err := a16.Parse(data)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	fmt.Printf("file:   %s\n", path)
	fmt.Printf("size:   %d bytes\n", len(data))
	fmt.Printf("frames: %d\n", anim.Count())

	for i, f := range anim.Frames {
		fmt.Printf("  [%2d] offset=%-8d flags=0x%02x dims=%dx%d\n", i, f.Offset, f.Header.Flags, f.Header.Width, f.Header.Height)
	}

	if *framesOut != "" {
		if err := os.MkdirAll(*framesOut, 0o755); err != nil {
			return err
		}

		for i := range anim.Count() {
			img, err := anim.Decode(i)
			if err != nil {
				return fmt.Errorf("frame %d: %w", i, err)
			}

			out := filepath.Join(*framesOut, fmt.Sprintf("frame%03d.png", i))
			if err := writeRGBAPNG(out, img.Width, img.Height, img.Pixels); err != nil {
				return fmt.Errorf("frame %d: write png: %w", i, err)
			}
		}

		fmt.Printf("wrote:  %d PNG frames to %s\n", anim.Count(), *framesOut)
	}

	return nil
}
