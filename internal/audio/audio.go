// Package audio defines a backend-independent playback interface plus a
// real Oto v3 implementation, so gameplay logic (speech tasks, footstep
// triggers) can be tested without an audio device.
//
// See agents/IMPLEMENTATION.md "Audio": Oto v3 is used for cross-platform
// PCM playback without cgo. Categories (speech/sfx/music) get independent
// volumes, and speech playback exposes elapsed time for ACS mouth sync plus
// a completion signal for scripts.
package audio

// Category is an independent-volume playback bus.
type Category int

const (
	CategorySpeech Category = iota
	CategorySFX
	CategoryMusic

	categoryCount
)

// Sound is decoded, backend-independent mono PCM at its original sample
// rate. Concrete Engine implementations convert it to their own output
// format at play time.
type Sound struct {
	SampleRate int
	Mono       []int16
}

// Duration returns the sound's playback length in seconds.
func (s *Sound) Duration() float64 {
	if s == nil || s.SampleRate == 0 {
		return 0
	}
	return float64(len(s.Mono)) / float64(s.SampleRate)
}

// Handle controls and reports on one in-flight playback.
type Handle interface {
	// IsPlaying reports whether playback has not yet finished (or been
	// stopped).
	IsPlaying() bool

	// Elapsed returns playback position in seconds. It keeps advancing
	// based on actual output consumption, not wall-clock time since Play
	// was called, so it stays meaningful even under buffering latency.
	Elapsed() float64

	// Stop ends playback immediately (e.g. for a skipped dialogue line).
	Stop()

	SetVolume(volume float64)
}

// Engine plays sounds on independent-volume categories.
type Engine interface {
	// Play starts sound once on category, stopping on its own once the
	// clip ends -- used for speech and one-shot SFX.
	Play(sound *Sound, category Category) Handle
	PlayWithVolume(sound *Sound, category Category, volume float64) Handle

	// PlayLooping starts sound on category and restarts it from the
	// beginning every time it reaches the end, indefinitely, until Stop is
	// called on the returned Handle (or StopAll runs). Used for background
	// music/ambient beds (Context.PlayMusic, the intro's own ambient bed):
	// a location's music must keep playing for as long as the player stays
	// there, not fall silent once the clip's own runtime elapses.
	PlayLooping(sound *Sound, category Category) Handle
	PlayLoopingWithVolume(sound *Sound, category Category, volume float64) Handle

	SetCategoryVolume(category Category, volume float64)
	StopCategory(category Category)

	// StopAll immediately stops every currently in-flight playback across
	// every category (music, SFX, speech). Callers use this at a hard scene
	// boundary -- most importantly changing locations -- so a background
	// music bed, ambient loop, or voiceover from the place just left never
	// bleeds into the next one; nothing in this package stops sounds on its
	// own otherwise, since ctx.PlayMusic/PlaySFX are deliberately
	// fire-and-forget and keep no handle a scene-transition caller could
	// stop individually.
	StopAll()

	Close() error
}
