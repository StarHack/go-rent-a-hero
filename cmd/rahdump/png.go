package main

import (
	"image"
	"image/png"
	"os"
)

// writeRGBAPNG writes width x height RGBA8 pixel data (R,G,B,A per pixel,
// row-major top-down) to path as a PNG file.
func writeRGBAPNG(path string, width, height int, pixels []byte) error {
	img := &image.RGBA{
		Pix:    pixels,
		Stride: width * 4,
		Rect:   image.Rect(0, 0, width, height),
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		return err
	}

	return f.Close()
}
