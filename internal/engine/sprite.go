package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wok/rent-a-hero/internal/assets"
	"github.com/wok/rent-a-hero/internal/formats/a16"
	"github.com/wok/rent-a-hero/internal/formats/bmp"
	"github.com/wok/rent-a-hero/internal/formats/tcc"
	"github.com/wok/rent-a-hero/internal/video"
)

// tccSource adapts a single decoded TCC image to SpriteSource (one frame).
type tccSource struct {
	img *tcc.Image
}

func (s *tccSource) Width() int  { return s.img.Width }
func (s *tccSource) Height() int { return s.img.Height }
func (s *tccSource) Frames() int { return 1 }

func (s *tccSource) RGBA(frame int) ([]byte, error) {
	if frame != 0 {
		return nil, fmt.Errorf("tccSource: frame %d out of range [0,1)", frame)
	}
	return s.img.Pixels, nil
}

// bmpSource adapts a single decoded BMP image to SpriteSource (one frame).
type bmpSource struct {
	img *bmp.Image
}

func (s *bmpSource) Width() int  { return s.img.Width }
func (s *bmpSource) Height() int { return s.img.Height }
func (s *bmpSource) Frames() int { return 1 }

func (s *bmpSource) RGBA(frame int) ([]byte, error) {
	if frame != 0 {
		return nil, fmt.Errorf("bmpSource: frame %d out of range [0,1)", frame)
	}
	return s.img.Pixels, nil
}

// a16Source adapts an A16 animation to SpriteSource, decoding frames lazily
// and caching the result since frame boundaries are already indexed.
type a16Source struct {
	anim  *a16.Animation
	cache map[int][]byte
}

func newA16Source(anim *a16.Animation) *a16Source {
	return &a16Source{anim: anim, cache: map[int][]byte{}}
}

// Width and Height report frame 0's dimensions. Real A16 assets observed so
// far are frame-uniform; per agents/A16.md this is not guaranteed in
// general, but the SpriteSource interface (agents/IMPLEMENTATION.md) only
// exposes one Width/Height for the whole source.
func (s *a16Source) Width() int  { return int(s.anim.Frames[0].Header.Width) }
func (s *a16Source) Height() int { return int(s.anim.Frames[0].Header.Height) }
func (s *a16Source) Frames() int { return s.anim.Count() }

func (s *a16Source) RGBA(frame int) ([]byte, error) {
	if cached, ok := s.cache[frame]; ok {
		return cached, nil
	}

	img, err := s.anim.Decode(frame)
	if err != nil {
		return nil, err
	}

	s.cache[frame] = img.Pixels

	return img.Pixels, nil
}

// LoadSpriteSource resolves gamePath (a bare filename as it appears in an
// SZN section, e.g. "back7b.bmp") through idx and decodes it into the
// appropriate SpriteSource based on its extension.
func LoadSpriteSource(idx *assets.Index, gamePath string) (SpriteSource, error) {
	real, err := idx.Resolve(gamePath)
	if err != nil {
		return nil, err
	}

	return loadSpriteSourceFile(real, gamePath)
}

func LoadSpriteSourceByBaseName(idx *assets.Index, gamePath string) (SpriteSource, error) {
	real, err := idx.ResolveBaseName(gamePath)
	if err != nil {
		return nil, err
	}

	return loadSpriteSourceFile(real, gamePath)
}

func loadSpriteSourceFile(real, gamePath string) (SpriteSource, error) {
	ext := strings.ToLower(filepath.Ext(gamePath))

	// Some locations (e.g. Location 35's glider scene, see
	// agents/scenes/02_DRAGON_BLASTER_DELUXE.MD) declare an AVI directly as
	// a normal SZN Layer's Filename, at its own X/Y/Z/Zoom like any other
	// sprite -- not the launch intro's full-screen PlayVideo overlay.
	// video.Source already implements SpriteSource, so it plugs in here
	// directly; handled before the generic os.ReadFile below since
	// video.Load reads the (potentially large) file itself.
	if ext == ".avi" {
		src, err := video.Load(real)
		if err != nil {
			return nil, fmt.Errorf("decode avi %s: %w", real, err)
		}
		return src, nil
	}

	data, err := os.ReadFile(real)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", real, err)
	}

	switch ext {
	case ".tcc":
		img, err := tcc.Decode(data)
		if err != nil {
			return nil, fmt.Errorf("decode tcc %s: %w", real, err)
		}
		return &tccSource{img: img}, nil

	case ".a16":
		anim, err := a16.Parse(data)
		if err != nil {
			return nil, fmt.Errorf("decode a16 %s: %w", real, err)
		}
		if anim.Count() == 0 {
			return nil, fmt.Errorf("decode a16 %s: no frames", real)
		}
		return newA16Source(anim), nil

	case ".bmp":
		img, err := bmp.Decode(data)
		if err != nil {
			return nil, fmt.Errorf("decode bmp %s: %w", real, err)
		}
		return &bmpSource{img: img}, nil

	default:
		return nil, fmt.Errorf("unsupported sprite source extension for %q", gamePath)
	}
}

func LoadSpriteSourceByStem(idx *assets.Index, stem string, extensions ...string) (SpriteSource, string, error) {
	real, name, err := idx.ResolveByStem(stem, extensions...)
	if err != nil {
		return nil, "", err
	}
	source, err := loadSpriteSourceFile(real, name)
	if err != nil {
		return nil, "", err
	}
	return source, real, nil
}
