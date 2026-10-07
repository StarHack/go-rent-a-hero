package audio

// FakeEngine is a deterministic Engine test double: it never touches a
// real audio device, and its handles only advance when the test explicitly
// calls Advance, making speech/mouth-sync tests fast and reproducible.
type FakeEngine struct {
	Plays  []*FakePlay
	Volume [categoryCount]float64
}

// FakePlay records one Play call and its controllable handle.
type FakePlay struct {
	Sound      *Sound
	Category   Category
	BaseVolume float64
	Handle     *FakeHandle
}

// NewFakeEngine returns a FakeEngine.
func NewFakeEngine() *FakeEngine {
	e := &FakeEngine{}
	for i := range e.Volume {
		e.Volume[i] = 1.0
	}
	return e
}

func (e *FakeEngine) Play(sound *Sound, category Category) Handle {
	return e.play(sound, category, false, 1)
}

func (e *FakeEngine) PlayWithVolume(sound *Sound, category Category, volume float64) Handle {
	return e.play(sound, category, false, volume)
}

// PlayLooping implements audio.Engine: the returned handle reports
// IsPlaying() == true forever (regardless of Advance) until Stop is called,
// matching OtoEngine.PlayLooping's real never-ending playback -- a test
// checking "does this bed ever fall silent on its own" can drive Advance by
// an arbitrarily large amount and still see it playing.
func (e *FakeEngine) PlayLooping(sound *Sound, category Category) Handle {
	return e.play(sound, category, true, 1)
}

func (e *FakeEngine) PlayLoopingWithVolume(sound *Sound, category Category, volume float64) Handle {
	return e.play(sound, category, true, volume)
}

func (e *FakeEngine) play(sound *Sound, category Category, looping bool, volume float64) Handle {
	if volume < 0 {
		volume = 0
	}
	h := &FakeHandle{duration: sound.Duration(), looping: looping, volume: volume * e.Volume[category]}
	e.Plays = append(e.Plays, &FakePlay{Sound: sound, Category: category, BaseVolume: volume, Handle: h})
	return h
}

func (e *FakeEngine) SetCategoryVolume(category Category, volume float64) {
	if volume < 0 {
		volume = 0
	}
	if volume > 1 {
		volume = 1
	}
	e.Volume[category] = volume
	for _, p := range e.Plays {
		if p.Category == category && p.Handle.IsPlaying() {
			p.Handle.SetVolume(p.BaseVolume * volume)
		}
	}
}

func (e *FakeEngine) StopCategory(category Category) {
	for _, p := range e.Plays {
		if p.Category == category {
			p.Handle.Stop()
		}
	}
}

// StopAll stops every handle this engine has ever produced (matching
// OtoEngine's real behavior of pausing every tracked player), so a test can
// assert a scene-transition call actually reached the audio layer.
func (e *FakeEngine) StopAll() {
	for _, p := range e.Plays {
		p.Handle.Stop()
	}
}

func (e *FakeEngine) Close() error { return nil }

// FakeHandle is a manually-advanceable Handle.
type FakeHandle struct {
	duration float64
	elapsed  float64
	stopped  bool
	looping  bool
	volume   float64
}

func (h *FakeHandle) IsPlaying() bool  { return !h.stopped && (h.looping || h.elapsed < h.duration) }
func (h *FakeHandle) Elapsed() float64 { return h.elapsed }
func (h *FakeHandle) Stop()            { h.stopped = true }
func (h *FakeHandle) SetVolume(volume float64) {
	if volume < 0 {
		volume = 0
	}
	h.volume = volume
}
func (h *FakeHandle) Volume() float64 { return h.volume }

// Advance moves this handle's playback clock forward by dt seconds.
func (h *FakeHandle) Advance(dt float64) { h.elapsed += dt }
