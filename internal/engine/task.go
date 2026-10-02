package engine

// Task is a resumable, steppable unit of gameplay work. Update returns true
// when the task has completed.
type Task interface {
	Update(dt float64) bool
}

// TaskFunc adapts a plain function to Task.
type TaskFunc func(dt float64) bool

func (f TaskFunc) Update(dt float64) bool { return f(dt) }

// sequenceTask runs a list of tasks one after another.
type sequenceTask struct {
	tasks []Task
	index int
}

// Sequence returns a Task that runs each of tasks in order, one at a time.
func Sequence(tasks ...Task) Task {
	return &sequenceTask{tasks: tasks}
}

func (s *sequenceTask) Update(dt float64) bool {
	for s.index < len(s.tasks) {
		if !s.tasks[s.index].Update(dt) {
			return false
		}
		s.index++
	}
	return true
}

// parallelTask runs every task concurrently (within one Update call each
// frame) until all have completed.
type parallelTask struct {
	tasks []Task
	done  []bool
}

// Parallel returns a Task that runs every task at once, completing when all
// of them have completed.
func Parallel(tasks ...Task) Task {
	return &parallelTask{tasks: tasks, done: make([]bool, len(tasks))}
}

func (p *parallelTask) Update(dt float64) bool {
	allDone := true

	for i, t := range p.tasks {
		if p.done[i] {
			continue
		}
		if t.Update(dt) {
			p.done[i] = true
		} else {
			allDone = false
		}
	}

	return allDone
}

// waitTask completes after a fixed number of seconds have elapsed.
type waitTask struct {
	remaining float64
}

// Wait returns a Task that completes after seconds have elapsed.
func Wait(seconds float64) Task {
	return &waitTask{remaining: seconds}
}

func (w *waitTask) Update(dt float64) bool {
	w.remaining -= dt
	return w.remaining <= 0
}

// immediateTask runs a plain side-effecting function once and completes on
// its first Update.
type immediateTask struct {
	fn   func()
	done bool
}

// Immediate returns a Task that runs fn once (e.g. a flag/visibility
// mutation) and completes on its first Update -- useful for embedding a
// single non-blocking gameplay effect inside a Sequence.
func Immediate(fn func()) Task {
	return &immediateTask{fn: fn}
}

func (t *immediateTask) Update(dt float64) bool {
	if !t.done {
		t.fn()
		t.done = true
	}
	return true
}

// PlayLayerOnce sets layer to play its animation once (from frame 0) and
// completes when it reaches its last frame, per
// agents/GAMEPLAY.md's PlayLayerOnce example. A static (fps<=0 or
// single-frame) layer completes immediately.
//
// This is the counterpart to FreezeLayer: it sets Playing=true so a layer
// previously frozen as a dormant "special action" prop actually animates
// while this task runs, then freezes it again (Playing=false) once done,
// holding the final frame rather than looping.
func PlayLayerOnce(layer *Layer) Task {
	layer.Mode = AnimOnce
	layer.Frame = 0
	layer.Accumulator = 0
	layer.Playing = true
	layer.TaskDriven = true
	return &playLayerOnceTask{layer: layer}
}

type playLayerOnceTask struct{ layer *Layer }

func (t *playLayerOnceTask) Update(dt float64) bool {
	if t.layer.Source == nil {
		return true
	}

	frames := t.layer.Source.Frames()
	if frames <= 1 {
		return true
	}

	t.layer.AdvanceScripted(dt)

	done := t.layer.Frame >= frames-1
	if done {
		t.layer.Playing = false
		t.layer.TaskDriven = false
	}

	return done
}

// FreezeLayer pins layer at frame (no animation) until something -- usually
// PlayLayerOnce -- re-enables it. Use this in a Controller's Enter hook for
// "special action" layers that must not auto-loop from scene load, per
// agents/GAMEPLAY.md "Scene entry state reconstruction".
func FreezeLayer(layer *Layer, frame int) Task {
	return Immediate(func() {
		layer.Playing = false
		layer.Frame = frame
		layer.Accumulator = 0
	})
}
