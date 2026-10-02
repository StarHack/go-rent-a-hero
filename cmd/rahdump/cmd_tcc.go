package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/wok/rent-a-hero/internal/formats/tcc"
)

func cmdTCC(args []string) error {
	fs := flag.NewFlagSet("tcc", flag.ExitOnError)
	pngOut := fs.String("png", "", "write decoded image to this PNG path")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return fmt.Errorf("usage: rahdump tcc <file.TCC> [--png out.png]")
	}

	path := fs.Arg(0)

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	h, err := tcc.ParseHeader(data)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	size, err := tcc.FrameSize(h)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	fmt.Printf("file:    %s\n", path)
	fmt.Printf("size:    %d bytes (header says %d)\n", len(data), size)
	fmt.Printf("flags:   0x%x\n", h.Flags)
	fmt.Printf("streams: %d %d %d %d\n", h.StreamSize0, h.StreamSize1, h.StreamSize2, h.StreamSize3)
	fmt.Printf("dims:    %dx%d\n", h.Width, h.Height)

	if size != len(data) {
		fmt.Printf("WARNING: computed frame size %d does not match file size %d\n", size, len(data))
	}

	img, err := tcc.Decode(data)
	if err != nil {
		return fmt.Errorf("%s: decode: %w", path, err)
	}

	fmt.Printf("decoded: ok\n")

	if *pngOut != "" {
		if err := writeRGBAPNG(*pngOut, img.Width, img.Height, img.Pixels); err != nil {
			return fmt.Errorf("write png: %w", err)
		}
		fmt.Printf("wrote:   %s\n", *pngOut)
	}

	return nil
}
