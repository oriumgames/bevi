package event

// Reader iterates over the current read buffer snapshot (the previous frame's writes).
// It supports per-event cancellation via Cancel() during iteration and exposes the
// current event's cancellation state via IsCancelled(). For batch extraction, use
// Drain or DrainTo.
type Reader[T any] struct {
	store *store[T]
	cur   *entry[T] // current entry for Cancel()/IsCancelled()
}

// Cancel marks the current event as cancelled. Call inside the ForEach() callback.
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
// It is safe to call from within the ForEach callback.
// If no iteration is active, it returns false.
func (r *Reader[T]) IsCancelled() bool {
	if r.cur == nil {
		return false
	}
	return r.cur.cancelled.Load()
}

// ForEach iterates the current read buffer snapshot with a callback.
// This is the recommended, zero-allocation iteration method. The callback
// should return `false` to stop iteration early.
//
// Example:
//
//	reader.ForEach(func(e MyEvent) bool {
//	    if shouldCancel(e) {
//	        reader.Cancel()
//	    }
//	    return true // return false to stop
//	})
//
// It safely handles completion tracking, even with early exits.
func (r *Reader[T]) ForEach(yield func(T) bool) {
	entries := r.store.snapshotEntries()
	if len(entries) == 0 {
		return
	}

	// First, register this reader for all non-completed entries. This is done
	// in a separate pass to ensure that even if the loop breaks early, all
	// events that *could* have been seen are accounted for.
	for _, ent := range entries {
		if !ent.IsDone() {
			ent.pending.Add(1)
		}
	}

	// Now, iterate and process.
	for i, ent := range entries {
		r.cur = ent
		// Only yield if the entry is not done; otherwise, just clean it up.
		if !ent.IsDone() {
			if !yield(ent.val) {
				ent.dec()
				// If we stopped early, we still need to decrement the pending count
				// for all the remaining entries we registered but didn't process.
				// The current entry `ent` has already been decremented.
				for j := i + 1; j < len(entries); j++ {
					entries[j].dec()
				}
				break
			}
		}
		ent.dec()
	}
	r.cur = nil
}

// Drain returns the values of the current read buffer and clears it.
// Prefer ForEach() for proper completion semantics; Drain is provided for special cases
// and does not register readers, so writers may rely on CompleteNoReader to resolve.
func (r Reader[T]) Drain() []T {
	return r.store.drain()
}

// DrainTo fills the provided buffer with events from the current read buffer and clears it.
// It returns the number of events written to dst. If dst is smaller than the number
// of available events, only the first len(dst) are copied.
// Prefer ForEach() for proper completion semantics; DrainTo is for special cases.
func (r Reader[T]) DrainTo(dst []T) int {
	if len(dst) == 0 {
		return 0
	}
	vals := r.store.drain()
	n := min(len(vals), len(dst))
	copy(dst, vals[:n])
	return n
}
