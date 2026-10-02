package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/wok/rent-a-hero/internal/formats/bmp"
)

func cmdBMP(args []string) error {
	fs := flag.NewFlagSet("bmp", flag.ExitOnError)
	pngOut := fs.String("png", "", "write decoded image to this PNG path")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return fmt.Errorf("usage: rahdump bmp <file.BMP> [--png out.png]")
	}

	path := fs.Arg(0)

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	img, err := bmp.Decode(data)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	fmt.Printf("file: %s\n", path)
	fmt.Printf("size: %d bytes\n", len(data))
	fmt.Printf("dims: %dx%d\n", img.Width, img.Height)

	if *pngOut != "" {
		if err := writeRGBAPNG(*pngOut, img.Width, img.Height, img.Pixels); err != nil {
			return fmt.Errorf("write png: %w", err)
		}
		fmt.Printf("wrote: %s\n", *pngOut)
	}

	return nil
}
