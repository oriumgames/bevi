package event

// Reader iterates over the current read buffer snapshot (the previous frame's writes).
// It supports per-event cancellation via Cancel() during iteration and exposes the
// current event's cancellation state via IsCancelled(). For batch extraction, use
// Drain or DrainTo.
type Reader[T any] struct {
	store *store[T]
	cur   *entry[T] // current entry for Cancel()/IsCancelled()
}

// Cancel marks the current event as cancelled. Call inside the Iter() callback.
// Cancellation is visible to writers via EventResult.Cancelled/Wait/WaitCancelled
// and to other readers via IsCancelled while iterating the same event.
func (r *Reader[T]) Cancel() {
	if r.cur != nil {
		r.cur.cancelled.Store(true)
	}
}

// IsCancelled reports whether the current event has been cancelled by any reader.
// This can be used by readers to adapt how they process the current event or
// decide behavior for subsequent events.
// It is safe to call from within the Iter callback.
// If no iteration is active, it returns false.
func (r *Reader[T]) IsCancelled() bool {
	if r.cur == nil {
		return false
	}
	return r.cur.cancelled.Load()
}

// Iter returns a Go 1.22+ rangefunc iterator over the current read buffer snapshot.
// Example:
//
//	for e := range reader.Iter() {
//	    if shouldCancel(e) {
//	        reader.Cancel() // makes cancellation visible immediately
//	    }
//	    if reader.IsCancelled() {
//	        // adapt behavior for e, or decide on subsequent events
//	    }
//	}
//
// Behavior:
//   - The iterator registers this reader for completion only on entries that are not
//     already completed.
//   - For each processed entry, it decrements the reader count to potentially complete
//     the event for waiting writers.
//   - If the consumer stops early (yield returns false), any remaining registered entries
//     are decremented so writers won't wait forever.
func (r *Reader[T]) Iter() func(func(T) bool) {
	entries := r.store.snapshotEntries()

	// Register this reader only for entries that haven't already completed.
	var registered []*entry[T]
	for _, ent := range entries {
		if ent.IsDone() {
			// already completed; skip registration
			continue
		}
		ent.pending.Add(1)
		registered = append(registered, ent)
	}

	return func(yield func(T) bool) {
		// Iterate registered entries and decrement immediately after processing each.
		for i, ent := range registered {
			r.cur = ent
			cont := yield(ent.val)
			ent.decAndMaybeClose()
			if !cont {
				// Decrement any remaining entries we registered but didn't process.
				for j := i + 1; j < len(registered); j++ {
					registered[j].decAndMaybeClose()
				}
				break
			}
		}
		r.cur = nil
	}
}

// Drain returns the values of the current read buffer and clears it.
// Prefer Iter() for proper completion semantics; Drain is provided for special cases
// and does not register readers, so writers may rely on CompleteNoReader to resolve.
func (r Reader[T]) Drain() []T {
	return r.store.drain()
}

// DrainTo fills the provided buffer with events from the current read buffer and clears it.
// It returns the number of events written to dst. If dst is smaller than the number
// of available events, only the first len(dst) are copied.
// Prefer Iter() for proper completion semantics; DrainTo is for special cases.
func (r Reader[T]) DrainTo(dst []T) int {
	if len(dst) == 0 {
		return 0
	}
	vals := r.store.drain()
	n := min(len(vals), len(dst))
	copy(dst, vals[:n])
	return n
}
