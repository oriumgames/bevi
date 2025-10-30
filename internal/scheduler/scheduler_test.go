package scheduler_test

import (
	"context"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oriumgames/bevi/internal/scheduler"
)

const Startup = scheduler.Stage(0)
const Update = scheduler.Stage(1)

// Test that stages isolate systems and that Before/After ordering constraints
// are honored in a single RunStage execution.
func TestStageSeparationAndOrder(t *testing.T) {
	s := scheduler.NewScheduler()

	var order []string
	record := func(name string) func(context.Context, any) {
		return func(ctx context.Context, _ any) {
			order = append(order, name)
		}
	}

	// Startup system should not run when we run Update stage below.
	sysStartup := &scheduler.System{
		Name:  "S-Startup",
		Stage: Startup,
		Fn:    record("S-Startup"),
		Meta:  scheduler.SystemMeta{},
	}
	// Define three systems in Update with explicit ordering:
	// A -> B -> C (via After constraints)
	sysA := &scheduler.System{
		Name:  "A",
		Stage: Update,
		Fn:    record("A"),
		Meta: scheduler.SystemMeta{
			Set:    "grp",
			After:  nil,
			Before: []string{
				// could also have constraints here, but After for B is sufficient
			},
		},
	}
	sysB := &scheduler.System{
		Name:  "B",
		Stage: Update,
		Fn:    record("B"),
		Meta: scheduler.SystemMeta{
			Set:    "grp",
			After:  []string{"A"},
			Before: []string{
				// After B for C is defined on C
			},
		},
	}
	sysC := &scheduler.System{
		Name:  "C",
		Stage: Update,
		Fn:    record("C"),
		Meta: scheduler.SystemMeta{
			Set:   "grp",
			After: []string{"B"},
		},
	}

	s.AddSystem(sysStartup)
	s.AddSystem(sysB) // add out of order on purpose
	s.AddSystem(sysC)
	s.AddSystem(sysA)

	if err := s.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	ctx := context.Background()
	world := struct{}{}

	// Running Update should run only A, B, C in the expected order.
	s.RunStage(ctx, Update, &world, nil)

	want := []string{"A", "B", "C"}
	if len(order) != len(want) {
		t.Fatalf("expected %d systems to run, got %d: %v", len(want), len(order), order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order[%d] = %q, want %q; full order: %v", i, order[i], want[i], order)
		}
	}

	// Running Startup should run only the startup system.
	order = order[:0]
	s.RunStage(ctx, Startup, &world, nil)
	if len(order) != 1 || order[0] != "S-Startup" {
		t.Fatalf("expected only startup system to run, got: %v", order)
	}
}

// Test that conflicting access metadata does not prevent systems from running,
// and both are executed once when scheduled in the same stage.
// This does not assert on parallelism, only on execution semantics.
func TestAccessConflictsExecution(t *testing.T) {
	s := scheduler.NewScheduler()

	var c1, c2 int32
	intType := reflect.TypeOf((*int)(nil)).Elem()

	sys1 := &scheduler.System{
		Name:  "Writer1",
		Stage: Update,
		Fn: func(ctx context.Context, _ any) {
			atomic.AddInt32(&c1, 1)
		},
		Meta: scheduler.SystemMeta{
			Access: scheduler.AccessMeta{
				ResWrites: []reflect.Type{intType}, // simulate resource write conflict
			},
		},
	}
	sys2 := &scheduler.System{
		Name:  "Writer2",
		Stage: Update,
		Fn: func(ctx context.Context, _ any) {
			atomic.AddInt32(&c2, 1)
		},
		Meta: scheduler.SystemMeta{
			Access: scheduler.AccessMeta{
				ResWrites: []reflect.Type{intType}, // same resource -> conflict
			},
		},
	}

	s.AddSystem(sys1)
	s.AddSystem(sys2)
	if err := s.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	ctx := context.Background()
	world := struct{}{}
	s.RunStage(ctx, Update, &world, nil)

	if atomic.LoadInt32(&c1) != 1 || atomic.LoadInt32(&c2) != 1 {
		t.Fatalf("expected both conflicting systems to run once, got c1=%d c2=%d", c1, c2)
	}
}

// Test that Every throttles execution frequency under repeated RunStage calls.
// We run Update in a loop with sleeps and expect the system to run roughly
// according to its period (with loose bounds to avoid flakes).
func TestEveryUnderLoad(t *testing.T) {
	s := scheduler.NewScheduler()

	var count int32
	period := 30 * time.Millisecond

	sys := &scheduler.System{
		Name:  "Periodic",
		Stage: Update,
		Fn: func(ctx context.Context, _ any) {
			atomic.AddInt32(&count, 1)
		},
		Meta: scheduler.SystemMeta{
			Every: period,
		},
	}

	s.AddSystem(sys)
	if err := s.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	ctx := context.Background()
	world := struct{}{}

	start := time.Now()
	// Run for ~220ms, sleeping 10ms between frames to simulate frames.
	for i := 0; i < 22; i++ {
		s.RunStage(ctx, Update, &world, nil)
		time.Sleep(10 * time.Millisecond)
	}
	elapsed := time.Since(start)

	got := atomic.LoadInt32(&count)
	// Expect approximately elapsed/period executions, within a loose tolerance.
	exp := float64(elapsed) / float64(period)
	lo := int32(exp * 0.5) // 50% lower bound
	hi := int32(exp * 1.8) // generous upper bound for timing variance

	if got < lo || got > hi {
		t.Fatalf("Every period mismatch: got=%d, expected around %.1f (bounds %d..%d) over %v",
			got, exp, lo, hi, elapsed)
	}
}

// Test mixed ordering constraints with Before and After combined across multiple
// systems to ensure the scheduler computes a valid topological order.
func TestComplexOrderConstraints(t *testing.T) {
	s := scheduler.NewScheduler()
	var order []string
	record := func(name string) func(context.Context, any) {
		return func(ctx context.Context, _ any) { order = append(order, name) }
	}

	// Intended order: Init -> Load -> Process -> Finalize
	init := &scheduler.System{
		Name:  "Init",
		Stage: Update,
		Fn:    record("Init"),
		Meta:  scheduler.SystemMeta{Set: "pipeline"},
	}
	load := &scheduler.System{
		Name:  "Load",
		Stage: Update,
		Fn:    record("Load"),
		Meta: scheduler.SystemMeta{
			Set:   "pipeline",
			After: []string{"Init"},
		},
	}
	process := &scheduler.System{
		Name:  "Process",
		Stage: Update,
		Fn:    record("Process"),
		Meta: scheduler.SystemMeta{
			Set:    "pipeline",
			After:  []string{"Load"},
			Before: []string{"Finalize"},
		},
	}
	finalize := &scheduler.System{
		Name:  "Finalize",
		Stage: Update,
		Fn:    record("Finalize"),
		Meta: scheduler.SystemMeta{
			Set:   "pipeline",
			After: []string{"Process"},
		},
	}

	// Add intentionally out of order.
	s.AddSystem(process)
	s.AddSystem(init)
	s.AddSystem(finalize)
	s.AddSystem(load)

	if err := s.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	ctx := context.Background()
	world := struct{}{}
	s.RunStage(ctx, Update, &world, nil)

	want := []string{"Init", "Load", "Process", "Finalize"}
	if len(order) != len(want) {
		t.Fatalf("expected %d systems, got %d: %v", len(want), len(order), order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("at %d: %q != %q; full order: %v", i, order[i], want[i], order)
		}
	}
}

// Test that a system with zero Every runs on every frame, and that adding a large
// amount of other systems (load) doesn't prevent correct execution ordering.
func TestZeroEveryAndLoad(t *testing.T) {
	s := scheduler.NewScheduler()

	var baseline int32
	base := &scheduler.System{
		Name:  "Baseline",
		Stage: Update,
		Fn: func(ctx context.Context, _ any) {
			atomic.AddInt32(&baseline, 1)
		},
		Meta: scheduler.SystemMeta{
			Every: 0, // should run every frame
		},
	}
	s.AddSystem(base)

	// Add many other systems with no particular constraints.
	const extra = 50
	for i := 0; i < extra; i++ {
		sys := &scheduler.System{
			Name:  "Extra-" + string(rune('A'+(i%26))),
			Stage: Update,
			Fn: func(ctx context.Context, _ any) {
				// no-op
			},
			Meta: scheduler.SystemMeta{},
		}
		s.AddSystem(sys)
	}

	if err := s.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	ctx := context.Background()
	world := struct{}{}

	// Run several frames quickly.
	const frames = 10
	for i := 0; i < frames; i++ {
		s.RunStage(ctx, Update, &world, nil)
	}

	if got := atomic.LoadInt32(&baseline); got != frames {
		t.Fatalf("Baseline ran %d times, want %d", got, frames)
	}
}
