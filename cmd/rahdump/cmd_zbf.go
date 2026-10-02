package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/wok/rent-a-hero/internal/formats/zbf"
)

func cmdZBF(args []string) error {
	fs := flag.NewFlagSet("zbf", flag.ExitOnError)
	pngOut := fs.String("png", "", "write a debug visualization to this PNG path")
	mode := fs.String("mode", "gray", "visualization mode: gray|walk|depth")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}

	if fs.NArg() != 1 {
		return fmt.Errorf("usage: rahdump zbf <file.ZBF> [--png out.png] [--mode gray|walk|depth]")
	}

	path := fs.Arg(0)

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	r, err := zbf.Parse(data)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	var nonZero int
	var minV, maxV byte = 255, 0
	unique := map[byte]bool{}

	for _, v := range r.Pixels {
		unique[v] = true
		if v != 0 {
			nonZero++
		}
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}

	fmt.Printf("file:        %s\n", path)
	fmt.Printf("size:        %d bytes\n", len(data))
	fmt.Printf("header size: %d\n", r.Header.HeaderSize)
	fmt.Printf("param1:      %v\n", r.Header.Param1)
	fmt.Printf("min value:   %d (header) / %d (observed)\n", r.Header.MinValue, minV)
	fmt.Printf("param2:      %v\n", r.Header.Param2)
	fmt.Printf("max value:   %d (header) / %d (observed)\n", r.Header.MaxValue, maxV)
	fmt.Printf("dims:        %dx%d\n", r.Width, r.Height)
	fmt.Printf("non-zero:    %d / %d\n", nonZero, len(r.Pixels))
	fmt.Printf("unique vals: %d\n", len(unique))

	if *pngOut != "" {
		pixels := renderZBFMode(r, *mode)
		if err := writeRGBAPNG(*pngOut, r.Width, r.Height, pixels); err != nil {
			return fmt.Errorf("write png: %w", err)
		}
		fmt.Printf("wrote:       %s (mode=%s)\n", *pngOut, *mode)
	}

	return nil
}

func renderZBFMode(r *zbf.Raster, mode string) []byte {
	pixels := make([]byte, r.Width*r.Height*4)

	for i, v := range r.Pixels {
		var rr, gg, bb byte

		switch mode {
		case "walk":
			if v != 0 {
				rr, gg, bb = 255, 255, 255
			}
		case "depth":
			if v == 255 {
				// Mark the sentinel/background value distinctly (magenta)
				// instead of treating it as an ordinary far value.
				rr, gg, bb = 255, 0, 255
			} else {
				rr, gg, bb = v, v, v
			}
		default: // gray
			rr, gg, bb = v, v, v
		}

		pixels[i*4+0] = rr
		pixels[i*4+1] = gg
		pixels[i*4+2] = bb
		pixels[i*4+3] = 255
	}

	return pixels
}
