package render

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"unsafe"

	"github.com/jupiterrider/purego-sdl3/sdl"
)

// CaptureRGBA reads back the current render target (the window's actual
// pixel content, at whatever real size it's currently presented at) as an
// unpremultiplied RGBA image. Used by both SavePNG and the recording
// indicator's frame-by-frame video capture (see cmd/rah's "8" key).
func (r *Renderer) CaptureRGBA() (*image.RGBA, error) {
	surface := sdl.RenderReadPixels(r.renderer, nil)
	if surface == nil {
		return nil, fmt.Errorf("render: RenderReadPixels: %s", sdl.GetError())
	}
	defer sdl.DestroySurface(surface)

	converted := sdl.ConvertSurface(surface, sdl.PixelFormatRGBA32)
	if converted == nil {
		return nil, fmt.Errorf("render: ConvertSurface: %s", sdl.GetError())
	}
	defer sdl.DestroySurface(converted)

	w, h, pitch := int(converted.W), int(converted.H), int(converted.Pitch)

	src := unsafe.Slice((*byte)(converted.Pixels), pitch*h)

	pixels := make([]byte, w*h*4)
	for y := range h {
		copy(pixels[y*w*4:(y+1)*w*4], src[y*pitch:y*pitch+w*4])
	}

	return &image.RGBA{
		Pix:    pixels,
		Stride: w * 4,
		Rect:   image.Rect(0, 0, w, h),
	}, nil
}

// SavePNG reads back the current render target and writes it to path as a
// PNG. Intended for developer verification and for the scene screenshot
// regression test described in agents/IMPLEMENTATION.md.
func (r *Renderer) SavePNG(path string) error {
	img, err := r.CaptureRGBA()
	if err != nil {
		return err
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
