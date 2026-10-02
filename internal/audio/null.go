package audio

// NullEngine discards all playback. Used as a safe fallback when no audio
// device is available, so the rest of the game can run headless.
type NullEngine struct{}

// NewNullEngine returns a NullEngine.
func NewNullEngine() *NullEngine { return &NullEngine{} }

func (*NullEngine) Play(*Sound, Category) Handle                           { return doneHandle{} }
func (*NullEngine) PlayWithVolume(*Sound, Category, float64) Handle        { return doneHandle{} }
func (*NullEngine) PlayLooping(*Sound, Category) Handle                    { return doneHandle{} }
func (*NullEngine) PlayLoopingWithVolume(*Sound, Category, float64) Handle { return doneHandle{} }
func (*NullEngine) SetCategoryVolume(Category, float64)                    {}
func (*NullEngine) StopCategory(Category)                                  {}
func (*NullEngine) StopAll()                                               {}
func (*NullEngine) Close() error                                           { return nil }

// doneHandle reports playback as already finished, so tasks waiting on
// completion (e.g. a speech task) don't block forever with no audio
// backend.
type doneHandle struct{}

func (doneHandle) IsPlaying() bool   { return false }
func (doneHandle) Elapsed() float64  { return 0 }
func (doneHandle) Stop()             {}
func (doneHandle) SetVolume(float64) {}
