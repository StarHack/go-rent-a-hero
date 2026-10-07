package audio

import (
	"fmt"
	"io"
	"sync/atomic"

	"github.com/ebitengine/oto/v3"
)

// bytesPerFrame is fixed by our chosen output format: stereo, 16-bit
// signed LE (2 channels * 2 bytes).
const bytesPerFrame = 4

// OtoEngine plays sound through github.com/ebitengine/oto/v3.
type OtoEngine struct {
	ctx        *oto.Context
	sampleRate int
	volume     [categoryCount]float64

	// active keeps every currently-playing *oto.Player reachable for the
	// garbage collector: oto.Player.Close's doc says the underlying
	// player "is called by the finalizer" when the wrapper becomes
	// unreachable, i.e. an unreferenced Player can be silently stopped by
	// GC mid-playback. A fire-and-forget caller (PlaySFX, PlayMusic)
	// discards the Handle Play returns, so without this the engine would
	// go silent partway through -- most noticeable on long ambient
	// tracks, since a short SFX often finishes before the next GC cycle
	// even runs.
	active           []*oto.Player
	activeCategories map[*oto.Player]Category
	activeHandles    map[*oto.Player]*otoHandle
}

// NewOtoEngine creates an Oto context at sampleRate (e.g. 44100) and
// blocks until it is ready.
func NewOtoEngine(sampleRate int) (*OtoEngine, error) {
	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   sampleRate,
		ChannelCount: 2,
		Format:       oto.FormatSignedInt16LE,
	})
	if err != nil {
		return nil, fmt.Errorf("audio: oto.NewContext: %w", err)
	}
	<-ready

	e := &OtoEngine{ctx: ctx, sampleRate: sampleRate, activeCategories: make(map[*oto.Player]Category), activeHandles: make(map[*oto.Player]*otoHandle)}
	for i := range e.volume {
		e.volume[i] = 1.0
	}

	return e, nil
}

func (e *OtoEngine) Play(sound *Sound, category Category) Handle {
	return e.play(sound, category, false, 1)
}

func (e *OtoEngine) PlayWithVolume(sound *Sound, category Category, volume float64) Handle {
	return e.play(sound, category, false, volume)
}

// PlayLooping implements audio.Engine: see its own doc comment. The
// underlying trackingReader simply wraps back to byte 0 instead of
// returning io.EOF once it reaches the end of the clip, so oto's player
// itself never observes an end of stream and keeps calling Read
// indefinitely -- exactly like a real one-shot playback, just never
// finishing on its own.
func (e *OtoEngine) PlayLooping(sound *Sound, category Category) Handle {
	return e.play(sound, category, true, 1)
}

func (e *OtoEngine) PlayLoopingWithVolume(sound *Sound, category Category, volume float64) Handle {
	return e.play(sound, category, true, volume)
}

func (e *OtoEngine) play(sound *Sound, category Category, loop bool, volume float64) Handle {
	mono := sound.Mono
	if sound.SampleRate != e.sampleRate {
		mono = resampleLinear(mono, sound.SampleRate, e.sampleRate)
	}

	data := toStereoBytes(mono)
	reader := &trackingReader{data: data, loop: loop}

	player := e.ctx.NewPlayer(reader)
	h := &otoHandle{
		player:         player,
		reader:         reader,
		totalBytes:     len(data),
		bytesPerSecond: float64(e.sampleRate * bytesPerFrame),
		categoryVolume: clamp01(e.volume[category]),
		volume:         max(volume, 0),
	}
	h.applyVolume()
	player.Play()

	e.reapFinishedPlayers()
	e.active = append(e.active, player)
	e.activeCategories[player] = category
	e.activeHandles[player] = h

	return h
}

// reapFinishedPlayers drops every tracked player that has finished, so
// active doesn't grow unbounded over a long play session; once dropped
// here (and not still held by some Handle a caller kept), a finished
// player is simply left for normal GC. Called opportunistically from
// Play, which is enough since every caller in this codebase runs on the
// single game-update goroutine.
func (e *OtoEngine) reapFinishedPlayers() {
	kept := e.active[:0]
	for _, p := range e.active {
		if p.IsPlaying() {
			kept = append(kept, p)
			continue
		}
		delete(e.activeCategories, p)
		delete(e.activeHandles, p)
	}
	e.active = kept
}

func (e *OtoEngine) SetCategoryVolume(category Category, volume float64) {
	volume = clamp01(volume)
	e.volume[category] = volume
	for player, activeCategory := range e.activeCategories {
		if activeCategory != category {
			continue
		}
		if h := e.activeHandles[player]; h != nil {
			h.categoryVolume = volume
			h.applyVolume()
		}
	}
}

func (e *OtoEngine) StopCategory(category Category) {
	kept := e.active[:0]
	for _, p := range e.active {
		if e.activeCategories[p] == category {
			p.Pause()
			delete(e.activeCategories, p)
			delete(e.activeHandles, p)
			continue
		}
		kept = append(kept, p)
	}
	e.active = kept
}

// StopAll pauses every currently tracked player (music, SFX, and speech
// alike -- e.active tracks all of them in the same slice, per Play) and
// drops them from tracking, so a fire-and-forget PlayMusic/PlaySFX bed from
// a scene just left doesn't keep playing into the next one.
func (e *OtoEngine) StopAll() {
	for _, p := range e.active {
		p.Pause()
	}
	e.active = e.active[:0]
	clear(e.activeCategories)
	clear(e.activeHandles)
}

func (e *OtoEngine) Close() error {
	e.active = nil
	clear(e.activeCategories)
	clear(e.activeHandles)
	return e.ctx.Suspend()
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// trackingReader is an io.Reader/io.Seeker over a fixed byte buffer that
// records how many bytes have been handed to the player so far, safely
// under concurrent access from Oto's internal playback goroutine. When loop
// is true (see OtoEngine.PlayLooping), Read wraps back to the start instead
// of ever returning io.EOF, so the player it feeds never observes an end of
// stream and keeps playing indefinitely.
type trackingReader struct {
	data []byte
	loop bool
	pos  int64
}

func (r *trackingReader) Read(p []byte) (int, error) {
	pos := atomic.LoadInt64(&r.pos)
	if pos >= int64(len(r.data)) {
		if !r.loop || len(r.data) == 0 {
			return 0, io.EOF
		}
		pos = 0
		atomic.StoreInt64(&r.pos, 0)
	}

	n := copy(p, r.data[pos:])
	atomic.AddInt64(&r.pos, int64(n))

	return n, nil
}

func (r *trackingReader) Seek(offset int64, whence int) (int64, error) {
	var newPos int64

	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = atomic.LoadInt64(&r.pos) + offset
	case io.SeekEnd:
		newPos = int64(len(r.data)) + offset
	default:
		return 0, fmt.Errorf("audio: invalid whence %d", whence)
	}

	atomic.StoreInt64(&r.pos, newPos)

	return newPos, nil
}

func (r *trackingReader) BytesRead() int64 {
	return atomic.LoadInt64(&r.pos)
}

// otoHandle adapts an *oto.Player to the Handle interface.
type otoHandle struct {
	player         *oto.Player
	reader         *trackingReader
	totalBytes     int
	bytesPerSecond float64
	categoryVolume float64
	volume         float64
}

func (h *otoHandle) IsPlaying() bool { return h.player.IsPlaying() }

func (h *otoHandle) Elapsed() float64 {
	consumed := max(h.reader.BytesRead()-int64(h.player.BufferedSize()), 0)
	return float64(consumed) / h.bytesPerSecond
}

func (h *otoHandle) Stop() {
	h.player.Pause()
}

func (h *otoHandle) SetVolume(volume float64) {
	if volume < 0 {
		volume = 0
	}
	h.volume = volume
	h.applyVolume()
}

func (h *otoHandle) applyVolume() {
	h.player.SetVolume(h.categoryVolume * h.volume)
}
