package event

import (
	"context"
	"runtime"
	"time"
)

// Writer appends events to the current frame's write buffer.
// Use EmitResult/EmitAndWait to observe completion and cancellation; Emit is fire-and-forget.
type Writer[T any] struct {
	store *store[T]
}

// Emit appends an event (fire-and-forget).
func (w Writer[T]) Emit(v T) {
	if w.store == nil {
		return
	}
	_ = w.store.appendEntry(v)
}

// EmitResult appends an event and returns a handle to wait for completion/cancellation.
func (w Writer[T]) EmitResult(v T) EventResult[T] {
	if w.store == nil {
		return EventResult[T]{}
	}
	ent := w.store.appendEntry(v)
	return EventResult[T]{ent: ent}
}

// EmitAndWait convenience to emit and wait on completion; it returns true if cancelled.
func (w Writer[T]) EmitAndWait(ctx context.Context, v T) bool {
	return w.EmitResult(v).Wait(ctx)
}

// EmitMany appends multiple events in a single critical section to reduce contention and allocations.
// It is safe to pass a nil or empty slice.
func (w Writer[T]) EmitMany(vals []T) {
	if w.store == nil || len(vals) == 0 {
		return
	}
	w.store.appendMany(vals)
}

// EventResult is a handle to observe completion and cancellation for a single emitted event.
type EventResult[T any] struct {
	ent *entry[T]
}

// Valid reports whether this result is non-zero.
func (r EventResult[T]) Valid() bool {
	return r.ent != nil
}

// Cancelled reports the current cancellation state without waiting.
// This may return false even if a reader has not yet had a chance to run.
func (r EventResult[T]) Cancelled() bool {
	if r.ent == nil {
		return false
	}
	return r.ent.cancelled.Load()
}

// Wait blocks until the event has been processed by all readers that started for the frame,
// or until ctx is done. It returns true if any reader cancelled the event.
func (r EventResult[T]) Wait(ctx context.Context) bool {
	if r.ent == nil {
		return false
	}
	// Fast path: already done via atomic state.
	if r.ent.IsDone() {
		return r.ent.cancelled.Load()
	}

	// Primary wait using the completion channel for blocking.
	done := r.ent.ensureDoneChan()
	select {
	case <-done:
		return r.ent.cancelled.Load()
	case <-ctx.Done():
		// If event finishes concurrently with ctx.Done(), prefer the final state via atomic flag.
		if r.ent.IsDone() {
			return r.ent.cancelled.Load()
		}
		return r.ent.cancelled.Load()
	}
}

// WaitCancelled returns as soon as either a reader cancels the event, the event completes,
// or ctx is done. The return value is the current cancellation state.
// This allows a fast "was it cancelled?" answer while the event may still continue processing.
func (r EventResult[T]) WaitCancelled(ctx context.Context) bool {
	if r.ent == nil {
		return false
	}
	// Fast checks.
	if r.ent.cancelled.Load() {
		return true
	}
	if r.ent.IsDone() {
		return r.ent.cancelled.Load()
	}

	// Spin briefly to catch very near-term updates without allocating timers.
	const spins = 4
	for range spins {
		if r.ent.cancelled.Load() {
			return true
		}
		if r.ent.IsDone() {
			return r.ent.cancelled.Load()
		}
		if ctx.Err() != nil {
			return r.ent.cancelled.Load()
		}
		runtime.Gosched()
	}

	// Fallback to light polling with blocking wait on the completion channel.
	ticker := time.NewTicker(250 * time.Microsecond)
	defer ticker.Stop()

	for {
		if r.ent.cancelled.Load() {
			return true
		}
		if r.ent.IsDone() {
			return r.ent.cancelled.Load()
		}
		done := r.ent.ensureDoneChan()
		select {
		case <-ctx.Done():
			return r.ent.cancelled.Load()
		case <-done:
			return r.ent.cancelled.Load()
		case <-ticker.C:
			// re-check loop
		}
	}
}
