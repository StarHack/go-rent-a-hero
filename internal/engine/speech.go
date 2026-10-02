package engine

import (
	"fmt"
	"os"

	"github.com/wok/rent-a-hero/internal/assets"
	"github.com/wok/rent-a-hero/internal/audio"
	"github.com/wok/rent-a-hero/internal/formats/acs"
	"github.com/wok/rent-a-hero/internal/formats/wav"
)

// SpeechTask owns one spoken line: WAV playback, optional ACS mouth track,
// a localized subtitle, and the speaker's talk state. See
// agents/IMPLEMENTATION.md "Speech".
type SpeechTask struct {
	actor    *Actor
	handle   audio.Handle
	track    *acs.Track // nil if the line has no ACS mouth track
	subtitle string
	elapsed  float64
	duration float64
	skipped  bool
}

// LoadSound resolves gamePath (e.g. "Gen_StepLeft.wav") through idx and
// decodes it into backend-independent PCM ready for an audio.Engine.
func LoadSound(idx *assets.Index, gamePath string) (*audio.Sound, error) {
	path, err := idx.Resolve(gamePath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	sound, err := wav.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	return &audio.Sound{SampleRate: sound.SampleRate, Mono: sound.ToInt16Mono()}, nil
}

func LoadSoundByBaseName(idx *assets.Index, gamePath string) (*audio.Sound, error) {
	path, err := idx.ResolveBaseName(gamePath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	sound, err := wav.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	return &audio.Sound{SampleRate: sound.SampleRate, Mono: sound.ToInt16Mono()}, nil
}

func talkFrame(actor *Actor, mouth uint16) int {
	if actor == nil || actor.TalkSource == nil {
		return 0
	}

	frames := actor.TalkSource.Frames()
	if frames <= 0 {
		return 0
	}

	if frames >= talkCycleFrameCount {
		band := walkCycleBands[actor.Facing]
		viseme := int(mouth)
		if viseme < 0 || band.start+viseme > band.end {
			viseme = 0
		}
		frame := band.start + viseme
		if frame < frames {
			return frame
		}
	}

	if int(mouth) < frames {
		return int(mouth)
	}
	return 0
}

// NewSpeech resolves baseName+".WAV" (required) and baseName+".ACS"
// (optional) through idx -- matching the "046_ROD_01.WAV" /
// "046_ROD_01.ACS" naming convention -- and immediately starts playback on
// eng's speech category.
func NewSpeech(idx *assets.Index, eng audio.Engine, actor *Actor, baseName string, subtitle string) (*SpeechTask, error) {
	pcm, err := LoadSoundByBaseName(idx, baseName+".WAV")
	if err != nil {
		return nil, fmt.Errorf("speech: %w", err)
	}

	var track *acs.Track
	if acsPath, err := idx.ResolveBaseName(baseName + ".ACS"); err == nil {
		if acsData, err := os.ReadFile(acsPath); err == nil {
			if t, err := acs.Parse(acsData); err == nil {
				track = t
			}
		}
	}

	handle := eng.Play(pcm, audio.CategorySpeech)

	actor.State = ActorTalking
	actor.MouthState = 0
	actor.SetFacingImmediate(actor.Facing)
	if actor.TalkSource != nil {
		actor.TalkFrame = talkFrame(actor, 0)
	}

	return &SpeechTask{actor: actor, handle: handle, track: track, subtitle: subtitle, duration: pcm.Duration()}, nil
}

// Subtitle returns this line's localized display text.
func (s *SpeechTask) Subtitle() string {
	return s.subtitle
}

// Skip stops playback immediately, as if the line finished naturally
// (supports click/keyboard dialogue skipping per agents/GAMEPLAY.md).
func (s *SpeechTask) Skip() {
	s.skipped = true
	s.handle.Stop()
}

// Update drives the mouth state from the audio playback clock (not an
// independent timer, so skip/pause stay in sync) and completes the task
// exactly when the underlying WAV finishes playing.
func (s *SpeechTask) Update(dt float64) bool {
	if s.skipped {
		s.actor.State = ActorIdle
		s.actor.MouthState = 0
		s.actor.TalkFrame = 0
		s.actor.SetFacingImmediate(s.actor.Facing)
		return true
	}

	s.elapsed += dt
	if s.elapsed < 0 {
		s.elapsed = 0
	}

	if s.track != nil {
		s.actor.MouthState = s.track.MouthStateAt(s.elapsed)
	}
	if s.actor.TalkSource != nil {
		s.actor.TalkFrame = talkFrame(s.actor, s.actor.MouthState)
	}

	if s.duration <= 0 || s.elapsed >= s.duration {
		s.handle.Stop()
		s.actor.State = ActorIdle
		s.actor.MouthState = 0
		s.actor.TalkFrame = 0
		s.actor.SetFacingImmediate(s.actor.Facing)
		return true
	}

	return false
}
