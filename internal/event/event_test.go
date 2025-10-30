package event_test

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oriumgames/bevi/internal/event"
)

type testEvent struct {
	ID int
}

type cancelEvent struct {
	Msg string
}

func collect[T any](r event.Reader[T]) []T {
	var out []T
	for v := range r.Iter() {
		out = append(out, v)
	}
	return out
}

func TestEmitIterOrderAndAdvance(t *testing.T) {
	b := event.NewBus()
	w := event.WriterFor[int](b)
	r := event.ReaderFor[int](b)

	// Emit but don't advance yet -> readers should see nothing.
	w.Emit(1)
	w.Emit(2)

	gotBefore := collect(r)
	if len(gotBefore) != 0 {
		t.Fatalf("expected no events before Advance, got %v", gotBefore)
	}

	// After Advance -> readers see the events in order.
	b.Advance()
	got := collect(r)
	want := []int{1, 2}
	if len(got) != len(want) {
		t.Fatalf("expected %d events, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("event[%d] = %v, want %v", i, got[i], want[i])
		}
	}

	// Iterating again on the same frame should yield none (entries considered done).
	gotAfter := collect(r)
	if len(gotAfter) != 0 {
		t.Fatalf("expected no events after first iteration, got %v", gotAfter)
	}
}

func TestCancelAndWaitCancelledFast(t *testing.T) {
	b := event.NewBus()
	w := event.WriterFor[cancelEvent](b)
	r := event.ReaderFor[cancelEvent](b)

	res := w.EmitResult(cancelEvent{Msg: "please-cancel"})
	b.Advance()

	started := make(chan struct{})
	done := make(chan struct{})
	var wasCancelled bool
	var took time.Duration

	go func() {
		defer close(done)
		close(started)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		begin := time.Now()
		wasCancelled = res.WaitCancelled(ctx)
		took = time.Since(begin)
	}()

	<-started
	// Reader cancels the event while iterating.
	for e := range r.Iter() {
		if e.Msg != "please-cancel" {
			t.Fatalf("unexpected event payload: %v", e.Msg)
		}
		r.Cancel()
		// simulate some processing time
		time.Sleep(300 * time.Microsecond)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("WaitCancelled did not return in time")
	}

	if !wasCancelled {
		t.Fatalf("expected WaitCancelled to observe cancellation, got false")
	}
	if took > 50*time.Millisecond {
		t.Fatalf("WaitCancelled took too long: %v (should be fast)", took)
	}

	// After reader finished, Wait should return quickly as well with cancellation=true.
	ctx2, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if !res.Wait(ctx2) {
		t.Fatalf("Wait should return cancelled=true")
	}
}

func TestCompleteNoReader(t *testing.T) {
	b := event.NewBus()
	w := event.WriterFor[string](b)

	res := w.EmitResult("foo")
	b.Advance()

	// No reader called Iter() this frame. CompleteNoReader should complete the event.
	b.CompleteNoReader()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	cancelled := res.Wait(ctx)
	if cancelled {
		t.Fatalf("unexpected cancellation, expected false")
	}
}

func TestDrainRequiresCompleteNoReader(t *testing.T) {
	b := event.NewBus()
	w := event.WriterFor[int](b)
	r := event.ReaderFor[int](b)

	res := w.EmitResult(10)
	b.Advance()

	vals := r.Drain()
	if len(vals) != 1 || vals[0] != 10 {
		t.Fatalf("Drain returned %v, want [10]", vals)
	}

	// Waiting before CompleteNoReader should block. Verify it doesn't complete early.
	waitDone := make(chan struct{})
	go func() {
		defer close(waitDone)
		_ = res.Wait(context.Background())
	}()

	select {
	case <-waitDone:
		t.Fatalf("Wait completed before CompleteNoReader; expected to block")
	case <-time.After(20 * time.Millisecond):
		// good, still waiting
	}

	// Now complete and ensure the waiter finishes.
	b.CompleteNoReader()
	select {
	case <-waitDone:
	case <-time.After(time.Second):
		t.Fatalf("Wait didn't complete after CompleteNoReader")
	}

	// Not cancelled.
	if res.Cancelled() {
		t.Fatalf("unexpected cancellation after CompleteNoReader")
	}
}

func TestEmitManyAndDrainTo(t *testing.T) {
	b := event.NewBus()
	w := event.WriterFor[int](b)
	r := event.ReaderFor[int](b)

	w.EmitMany([]int{1, 2, 3})
	b.Advance()

	col := collect(r)
	want := []int{1, 2, 3}
	if len(col) != len(want) {
		t.Fatalf("EmitMany -> got %d events, want %d", len(col), len(want))
	}
	for i := range want {
		if col[i] != want[i] {
			t.Fatalf("event[%d] = %v, want %v", i, col[i], want[i])
		}
	}

	// DrainTo with a pre-allocated buffer.
	w.EmitMany([]int{4, 5, 6, 7})
	b.Advance()
	buf := make([]int, 3)
	n := r.DrainTo(buf)
	if n != 3 {
		t.Fatalf("DrainTo wrote %d, want 3", n)
	}
	if buf[0] != 4 || buf[1] != 5 || buf[2] != 6 {
		t.Fatalf("DrainTo buffer unexpected: %v", buf)
	}
}

func TestLazyDoneImmediateAfterComplete(t *testing.T) {
	b := event.NewBus()
	w := event.WriterFor[int](b)

	res := w.EmitResult(42)
	b.Advance()
	// No readers -> mark complete this frame.
	b.CompleteNoReader()

	// WaitCancelled should return immediately (not cancelled).
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	begin := time.Now()
	c := res.WaitCancelled(ctx)
	elapsed := time.Since(begin)
	if c {
		t.Fatalf("unexpected cancellation")
	}
	if elapsed > 10*time.Millisecond {
		t.Fatalf("WaitCancelled took too long after completion: %v", elapsed)
	}

	// Wait should be immediate as well.
	begin = time.Now()
	_ = res.Wait(context.Background())
	elapsed = time.Since(begin)
	if elapsed > 10*time.Millisecond {
		t.Fatalf("Wait took too long after completion: %v", elapsed)
	}
}

func TestMultipleReadersCancelVisibilityAndCompletion(t *testing.T) {
	b := event.NewBus()
	w := event.WriterFor[int](b)
	r1 := event.ReaderFor[int](b)
	r2 := event.ReaderFor[int](b)

	res := w.EmitResult(7)
	b.Advance()

	var wg sync.WaitGroup
	wg.Add(2)

	// Synchronize r1 cancel -> r2 sees cancellation.
	canceled := make(chan struct{})
	seen := make(chan struct{})
	observed := make(chan struct{}, 1)

	go func() {
		defer wg.Done()
		for range r1.Iter() {
			// r1 cancels then waits for r2 to observe it.
			r1.Cancel()
			close(canceled)
			<-seen
		}
	}()

	go func() {
		defer wg.Done()
		for range r2.Iter() {
			<-canceled
			if r2.IsCancelled() {
				observed <- struct{}{}
			}
			close(seen)
		}
	}()

	wg.Wait()

	select {
	case <-observed:
	default:
		t.Fatalf("r2 did not see cancellation from r1")
	}

	// Writer sees cancellation and completion.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if !res.Wait(ctx) {
		t.Fatalf("writer didn't observe cancellation via Wait")
	}
}

func TestNoAdvanceNoEvents(t *testing.T) {
	b := event.NewBus()
	w := event.WriterFor[string](b)
	r := event.ReaderFor[string](b)

	w.Emit("a")
	w.Emit("b")

	got := collect(r)
	if len(got) != 0 {
		t.Fatalf("expected no events without Advance, got %v", got)
	}
}

func TestStressPoolingNoWait(t *testing.T) {
	b := event.NewBus()
	w := event.WriterFor[testEvent](b)
	r := event.ReaderFor[testEvent](b)

	// Warm-up to grow internal buffers and pools.
	const warmFrames = 8
	const warmPerFrame = 256
	for range warmFrames {
		for i := range warmPerFrame {
			w.Emit(testEvent{ID: i})
		}
		b.Advance()
		_ = r.Drain()
		b.CompleteNoReader()
	}

	// Stress pass: many frames and events without waiting; just ensure correctness and no deadlocks.
	const frames = 16
	const perFrame = 512

	total := 0
	for f := range frames {
		for i := range perFrame {
			w.Emit(testEvent{ID: i})
		}
		b.Advance()
		col := collect(r)
		if len(col) != perFrame {
			t.Fatalf("frame %d: got %d events, want %d", f, len(col), perFrame)
		}
		total += len(col)
		b.CompleteNoReader()
	}
	if total != frames*perFrame {
		t.Fatalf("total events mismatch: %d vs %d", total, frames*perFrame)
	}
}

func TestEmitAndWaitConvenience(t *testing.T) {
	b := event.NewBus()
	w := event.WriterFor[int](b)
	r := event.ReaderFor[int](b)

	go func() {
		// Advance first so EmitAndWait waits on events in this frame.
		b.Advance()
		for range r.Iter() {
			// consume but don't cancel
		}
	}()

	// EmitAndWait returns false (not cancelled).
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	cancelled := w.EmitAndWait(ctx, 123)
	if cancelled {
		t.Fatalf("unexpected cancellation from EmitAndWait")
	}
}

func TestWaitCancelledTimeoutWhenNoReaders(t *testing.T) {
	b := event.NewBus()
	w := event.WriterFor[int](b)

	res := w.EmitResult(1)
	b.Advance()

	// No readers; WaitCancelled should return false after context timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	begin := time.Now()
	c := res.WaitCancelled(ctx)
	dur := time.Since(begin)
	if c {
		t.Fatalf("unexpected cancellation when no readers")
	}
	// It should roughly honor context timeouts (we allow a wide margin).
	if dur < 5*time.Millisecond || dur > 500*time.Millisecond {
		t.Fatalf("WaitCancelled duration unexpected: %v", dur)
	}
}

func TestReaderEarlyStopDecrementsPending(t *testing.T) {
	b := event.NewBus()
	w := event.WriterFor[int](b)
	r := event.ReaderFor[int](b)

	res := w.EmitResult(1)
	res2 := w.EmitResult(2)
	b.Advance()

	// Stop early after first event; pending for the second should be decremented by the reader as well.
	i := 0
	for range r.Iter() {
		i++
		if i == 1 {
			break
		}
	}

	// No more readers this frame -> CompleteNoReader should complete both events.
	b.CompleteNoReader()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if res.Wait(ctx) {
		t.Fatalf("first event shouldn't be cancelled")
	}
	if res2.Wait(ctx) {
		t.Fatalf("second event shouldn't be cancelled")
	}
}

func TestConcurrentReadersAndWriters(t *testing.T) {
	b := event.NewBus()
	r := event.ReaderFor[int](b)

	const writers = 4
	const perWriter = 500

	var wg sync.WaitGroup
	wg.Add(writers)
	start := make(chan struct{})

	for w := range writers {
		go func(id int) {
			defer wg.Done()
			<-start
			wr := event.WriterFor[int](b)
			for i := range perWriter {
				wr.Emit(i + id*100000)
				// yield to increase interleaving
				if i%50 == 0 {
					runtime.Gosched()
				}
			}
		}(w)
	}

	close(start)
	// Collect over several frames.
	total := 0
	seen := make(map[int]struct{})
	for range 6 {
		time.Sleep(2 * time.Millisecond)
		b.Advance()
		for v := range r.Iter() {
			seen[v] = struct{}{}
			total++
		}
		b.CompleteNoReader()
		if total >= writers*perWriter {
			break
		}
	}
	wg.Wait()

	// Drain remaining
	for range 2 {
		b.Advance()
		for range r.Iter() {
			total++
		}
		b.CompleteNoReader()
	}

	if total == 0 || len(seen) == 0 {
		t.Fatalf("no events observed")
	}
}

func TestEventResultValidAndCancelledAccessors(t *testing.T) {
	b := event.NewBus()
	w := event.WriterFor[int](b)

	var zero event.EventResult[int]
	if zero.Valid() {
		t.Fatalf("zero EventResult should be invalid")
	}
	if zero.Cancelled() {
		t.Fatalf("zero EventResult Cancelled should be false")
	}

	res := w.EmitResult(5)
	if res.Cancelled() {
		t.Fatalf("cancelled should be false before any reader runs")
	}
	b.Advance()
	b.CompleteNoReader()
	if res.Cancelled() {
		t.Fatalf("cancelled should be false after completion with no readers")
	}
}

func TestCancelFromMultipleReadersOnlySetsFlagOnce(t *testing.T) {
	b := event.NewBus()
	w := event.WriterFor[int](b)
	r1 := event.ReaderFor[int](b)
	r2 := event.ReaderFor[int](b)
	r3 := event.ReaderFor[int](b)

	res := w.EmitResult(1)
	b.Advance()

	var cancels atomic.Int32
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		for range r1.Iter() {
			r1.Cancel()
			cancels.Add(1)
		}
	}()
	go func() {
		defer wg.Done()
		for range r2.Iter() {
			if !r2.IsCancelled() {
				// If r2 wins the race, it might cancel as well.
				r2.Cancel()
				cancels.Add(1)
			}
		}
	}()
	go func() {
		defer wg.Done()
		for range r3.Iter() {
			// don't cancel
		}
	}()

	wg.Wait()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if !res.Wait(ctx) {
		t.Fatalf("expected cancellation to be observed by writer")
	}
	if cancels.Load() < 1 {
		t.Fatalf("no reader cancelled, expected at least one")
	}
}
