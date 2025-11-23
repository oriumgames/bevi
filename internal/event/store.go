package event

import (
	"sync"
	"sync/atomic"
)

var closedCh = func() chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}()

// entry represents a single emitted event and tracks its lifecycle.
//
//   - pending: number of readers currently registered to process this entry.
//   - cancelled: set to true if any reader cancels while processing this entry.
//   - done: completion signal, closed exactly once when pending reaches zero or when
//     the system determines there will be no readers for the current frame.
//   - state: atomic bitset to guarantee single close without sync.Once.
type entry[T any] struct {
	val       T
	pending   atomic.Int32
	cancelled atomic.Bool
	done      chan struct{}
	doneMu    sync.Mutex
	state     atomic.Uint32 // bit0: 1 = completed (done closed)
}

func (s *store[T]) newEntry(v T, wantDone bool) *entry[T] {
	if x := s.entryPool.Get(); x != nil {
		e := x.(*entry[T])
		// reset fields
		var zero T
		e.val = zero
		e.val = v
		e.pending.Store(0)
		e.cancelled.Store(false)
		e.state.Store(0)
		// optionally create a fresh channel for completion signaling
		if wantDone {
			e.done = make(chan struct{})
		} else {
			e.done = nil
		}
		return e
	}
	if wantDone {
		return &entry[T]{
			val:  v,
			done: make(chan struct{}),
		}
	}
	return &entry[T]{val: v}
}

// dec decrements the pending reader count.
// Completion is deferred to advance() at frame end.
func (e *entry[T]) dec() {
	e.pending.Add(-1)
}

// markCancelled sets the cancellation flag.
// Readers should call this when they intend to cancel the current event.
func (e *entry[T]) markCancelled() {
	e.cancelled.Store(true)
}

// IsDone reports whether the entry has completed (its done channel has been signaled).
func (e *entry[T]) IsDone() bool {
	return e.state.Load()&1 == 1
}

// ensureDoneChan lazily creates a done channel if it doesn't exist.
// If the entry is already done, it sets a pre-closed channel to allow immediate wakeups.
func (e *entry[T]) ensureDoneChan() chan struct{} {
	if e.done != nil {
		return e.done
	}
	e.doneMu.Lock()
	if e.done == nil {
		if e.IsDone() {
			e.done = closedCh
		} else {
			e.done = make(chan struct{})
		}
	}
	ch := e.done
	e.doneMu.Unlock()
	return ch
}

// store is the per-type container for events.
// It is double-buffered: writers append to writeEnt, while readers iterate readEnt.
type store[T any] struct {
	mu        sync.RWMutex
	readEnt   []*entry[T]
	writeEnt  []*entry[T]
	entryPool sync.Pool // pools *entry[T] to reduce allocations
	name      string
	diag      Diagnostics
}

// appendEntry appends an event to the current write buffer and returns its entry.
func (s *store[T]) appendEntry(v T) *entry[T] {
	if s.diag != nil {
		s.diag.EventEmit(s.name, 1)
	}
	ent := s.newEntry(v, false)

	s.mu.Lock()
	s.writeEnt = append(s.writeEnt, ent)
	s.mu.Unlock()

	return ent
}

// appendMany appends multiple events without returning result handles.
func (s *store[T]) appendMany(vals []T) {
	if len(vals) == 0 {
		return
	}

	if s.diag != nil {
		s.diag.EventEmit(s.name, len(vals))
	}

	s.mu.Lock()
	for _, v := range vals {
		s.writeEnt = append(s.writeEnt, s.newEntry(v, false))
	}
	s.mu.Unlock()
}

// drain returns the read values and clears the read buffers.
// Prefer Reader.Iter for proper completion semantics; Drain is for special cases.
func (s *store[T]) drain() []T {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.readEnt) == 0 {
		return nil
	}
	out := make([]T, len(s.readEnt))
	for i, ent := range s.readEnt {
		out[i] = ent.val
	}
	// Do not clear readEnt here; CompleteNoReader needs entries to close completion for drained events.
	// It will be cleared during Advance after completion is ensured.
	return out
}

// snapshotEntries returns the current read entries slice without copying.
// Callers must treat the returned slice as read-only and should not retain it
// across Advance(), as the store may recycle or mutate entries at frame boundaries.
func (s *store[T]) snapshotEntries() []*entry[T] {
	s.mu.RLock()
	out := s.readEnt
	s.mu.RUnlock()
	return out
}

// advance swaps write/read buffers and clears the new write buffers.
// Minor perf tweak: reuse slice capacities by slicing to zero length.
func (s *store[T]) advance() {
	s.mu.Lock()

	for _, e := range s.readEnt {
		if e.state.CompareAndSwap(0, 1) {
			if e.done != nil {
				close(e.done)
			} else {
				e.doneMu.Lock()
				if e.done == nil {
					e.done = closedCh
				}
				e.doneMu.Unlock()
			}
		}
	}

	s.readEnt, s.writeEnt = s.writeEnt, s.readEnt

	if len(s.writeEnt) > 0 {
		for i := range s.writeEnt {
			e := s.writeEnt[i]
			var zero T
			e.val = zero
			s.entryPool.Put(e)
		}
		s.writeEnt = s.writeEnt[:0]
	}
	s.mu.Unlock()
}
