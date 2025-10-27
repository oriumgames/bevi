package scheduler

import (
	"context"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

type captureDiag struct {
	mu      sync.Mutex
	starts  map[string]time.Time
	ends    map[string]time.Time
	errs    map[string]error
	durs    map[string]time.Duration
	ordered []string
}

func newCaptureDiag() *captureDiag {
	return &captureDiag{
		starts: make(map[string]time.Time),
		ends:   make(map[string]time.Time),
		errs:   make(map[string]error),
		durs:   make(map[string]time.Duration),
	}
}

func (c *captureDiag) SystemStart(name string, stage Stage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	c.starts[name] = now
	c.ordered = append(c.ordered, "start:"+name)
}

func (c *captureDiag) SystemEnd(name string, stage Stage, err error, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	c.ends[name] = now
	if err != nil {
		c.errs[name] = err
	}
	c.durs[name] = duration
	c.ordered = append(c.ordered, "end:"+name)
}

func overlaps(aStart, aEnd, bStart, bEnd time.Time) bool {
	return aStart.Before(bEnd) && bStart.Before(aEnd)
}

func TestComplexExecutionWithDiagnostics(t *testing.T) {
	// Ensure at least 2 OS threads so parallel tests can overlap.
	prevProcs := runtime.GOMAXPROCS(2)
	defer runtime.GOMAXPROCS(prevProcs)

	s := NewScheduler()

	intType := reflect.TypeOf(0)
	strType := reflect.TypeOf("")
	resType := reflect.TypeOf(struct{ R int }{})
	evtType := reflect.TypeOf(struct{ E string }{})

	// Barriers to force certain systems to overlap to test parallelism deterministically.
	var rBarrier sync.WaitGroup
	rBarrier.Add(2)
	var gBarrier sync.WaitGroup
	gBarrier.Add(2)

	// Parallel, non-conflicting systems (should overlap): R1 and R2
	sysR1 := &System{
		Name:  "R1",
		Stage: 0,
		Fn: func(ctx context.Context, w any) {
			rBarrier.Done()
			rBarrier.Wait()
			time.Sleep(50 * time.Millisecond)
		},
		Meta: SystemMeta{
			Access: AccessMeta{
				Reads: []reflect.Type{intType},
			},
		},
	}
	sysR2 := &System{
		Name:  "R2",
		Stage: 0,
		Fn: func(ctx context.Context, w any) {
			rBarrier.Done()
			rBarrier.Wait()
			time.Sleep(50 * time.Millisecond)
		},
		Meta: SystemMeta{
			Access: AccessMeta{
				Reads: []reflect.Type{strType},
			},
		},
	}

	// Conflicting with R1 (should NOT overlap): WInt writes int
	sysWInt := &System{
		Name:  "WInt",
		Stage: 0,
		Fn: func(ctx context.Context, w any) {
			time.Sleep(30 * time.Millisecond)
		},
		Meta: SystemMeta{
			Access: AccessMeta{
				Writes: []reflect.Type{intType},
			},
		},
	}

	// Group G1 and G2 (same Set "G"), AfterG must run after both complete
	sysG1 := &System{
		Name:  "G1",
		Stage: 0,
		Fn: func(ctx context.Context, w any) {
			gBarrier.Done()
			gBarrier.Wait()
			time.Sleep(20 * time.Millisecond)
		},
		Meta: SystemMeta{
			Set: "G",
		},
	}
	sysG2 := &System{
		Name:  "G2",
		Stage: 0,
		Fn: func(ctx context.Context, w any) {
			gBarrier.Done()
			gBarrier.Wait()
			time.Sleep(20 * time.Millisecond)
		},
		Meta: SystemMeta{
			Set: "G",
		},
	}
	sysAfterG := &System{
		Name:  "AfterG",
		Stage: 0,
		Fn: func(ctx context.Context, w any) {
			time.Sleep(10 * time.Millisecond)
		},
		Meta: SystemMeta{
			After: []string{"G"},
		},
	}

	// Resource conflict: ResW (write) and ResR (read) must not overlap
	sysResW := &System{
		Name:  "ResW",
		Stage: 0,
		Fn: func(ctx context.Context, w any) {
			time.Sleep(15 * time.Millisecond)
		},
		Meta: SystemMeta{
			Access: AccessMeta{
				ResWrites: []reflect.Type{resType},
			},
		},
	}
	sysResR := &System{
		Name:  "ResR",
		Stage: 0,
		Fn: func(ctx context.Context, w any) {
			time.Sleep(15 * time.Millisecond)
		},
		Meta: SystemMeta{
			Access: AccessMeta{
				ResReads: []reflect.Type{resType},
			},
		},
	}

	// Event conflict: EvtW (write) and EvtR (read) must not overlap
	sysEvtW := &System{
		Name:  "EvtW",
		Stage: 0,
		Fn: func(ctx context.Context, w any) {
			time.Sleep(15 * time.Millisecond)
		},
		Meta: SystemMeta{
			Access: AccessMeta{
				EventWrites: []reflect.Type{evtType},
			},
		},
	}
	sysEvtR := &System{
		Name:  "EvtR",
		Stage: 0,
		Fn: func(ctx context.Context, w any) {
			time.Sleep(15 * time.Millisecond)
		},
		Meta: SystemMeta{
			Access: AccessMeta{
				EventReads: []reflect.Type{evtType},
			},
		},
	}

	// Gated system: should not run on first pass, will run on second pass
	sysGated := &System{
		Name:  "Gated",
		Stage: 0,
		Fn: func(ctx context.Context, w any) {
			time.Sleep(5 * time.Millisecond)
		},
		Meta: SystemMeta{
			Every: 80 * time.Millisecond,
		},
		LastRun: time.Now(),
	}

	// Panic system: must not crash scheduler, should appear as error in diagnostics
	sysPanic := &System{
		Name:  "PanicSys",
		Stage: 0,
		Fn: func(ctx context.Context, w any) {
			panic("boom")
		},
	}

	// Register systems
	for _, sys := range []*System{
		sysR1, sysR2,
		sysWInt,
		sysG1, sysG2, sysAfterG,
		sysResW, sysResR,
		sysEvtW, sysEvtR,
		sysGated,
		sysPanic,
	} {
		s.AddSystem(sys)
	}

	if err := s.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// First run: Gated should not execute
	diag1 := newCaptureDiag()
	s.RunStage(context.Background(), 0, nil, diag1)

	// Validate panic captured
	if err, ok := diag1.errs["PanicSys"]; !ok || err == nil || !strings.Contains(err.Error(), "panic:") {
		t.Fatalf("expected panic error captured for PanicSys, got: %v (present=%v)", err, ok)
	}

	// Validate presence of required systems in diag
	requireTimes := func(name string) (time.Time, time.Time) {
		start, okS := diag1.starts[name]
		end, okE := diag1.ends[name]
		if !okS || !okE {
			t.Fatalf("expected start/end times for %s, got start=%v end=%v", name, okS, okE)
		}
		return start, end
	}

	// R1 and R2 should overlap (parallel non-conflicting)
	r1s, r1e := requireTimes("R1")
	r2s, r2e := requireTimes("R2")
	if !overlaps(r1s, r1e, r2s, r2e) {
		t.Fatalf("expected R1 and R2 to overlap, got intervals [%v,%v] and [%v,%v]", r1s, r1e, r2s, r2e)
	}

	// WInt must not overlap with R1 (conflicting write-read on int)
	wis, wie := requireTimes("WInt")
	if overlaps(r1s, r1e, wis, wie) {
		t.Fatalf("expected WInt and R1 NOT to overlap, got intervals [%v,%v] and [%v,%v]", r1s, r1e, wis, wie)
	}

	// Group G1 and G2 should overlap (same set, no access conflicts)
	g1s, g1e := requireTimes("G1")
	g2s, g2e := requireTimes("G2")
	if !overlaps(g1s, g1e, g2s, g2e) {
		t.Fatalf("expected G1 and G2 to overlap, got intervals [%v,%v] and [%v,%v]", g1s, g1e, g2s, g2e)
	}
	// AfterG must start after both G1 and G2 end
	ags, _ := requireTimes("AfterG")
	maxGE := g1e
	if g2e.After(maxGE) {
		maxGE = g2e
	}
	if !ags.After(maxGE) && !ags.Equal(maxGE) {
		t.Fatalf("expected AfterG to start after both G1 and G2 completed, AfterG start=%v, G1 end=%v, G2 end=%v", ags, g1e, g2e)
	}

	// Resource conflict: ResW and ResR must not overlap
	rws, rwe := requireTimes("ResW")
	rrs, rre := requireTimes("ResR")
	if overlaps(rws, rwe, rrs, rre) {
		t.Fatalf("expected ResW and ResR NOT to overlap, got intervals [%v,%v] and [%v,%v]", rws, rwe, rrs, rre)
	}

	// Event conflict: EvtW and EvtR must not overlap
	ews, ewe := requireTimes("EvtW")
	ers, ere := requireTimes("EvtR")
	if overlaps(ews, ewe, ers, ere) {
		t.Fatalf("expected EvtW and EvtR NOT to overlap, got intervals [%v,%v] and [%v,%v]", ews, ewe, ers, ere)
	}

	// Gated system should not have run in first pass
	if _, ok := diag1.starts["Gated"]; ok {
		t.Fatalf("expected Gated not to run on first pass due to Every gating")
	}

	// Second run: after enough time, Gated should run once
	time.Sleep(90 * time.Millisecond)
	diag2 := newCaptureDiag()
	s.RunStage(context.Background(), 0, nil, diag2)

	// Gated should have executed now
	if _, ok := diag2.starts["Gated"]; !ok {
		t.Fatalf("expected Gated to run on second pass after gating interval elapsed")
	}

	// Sanity: other systems also executed again
	names := []string{"R1", "R2", "WInt", "G1", "G2", "AfterG", "ResW", "ResR", "EvtW", "EvtR", "PanicSys"}
	for _, n := range names {
		if _, ok := diag2.starts[n]; !ok {
			t.Fatalf("expected %s to run on second pass", n)
		}
	}
}
