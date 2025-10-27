package scheduler

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"sort"
	"sync"
	"time"
)

// Scheduler manages system execution order and parallelization.
type Scheduler struct {
	mu      sync.RWMutex
	systems map[Stage][]*System
	batches map[Stage][][]*System
}

// NewScheduler creates a new scheduler.
func NewScheduler() *Scheduler {
	return &Scheduler{
		systems: make(map[Stage][]*System),
		batches: make(map[Stage][][]*System),
	}
}

// AddSystem registers a system for the given stage.
func (s *Scheduler) AddSystem(sys *System) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Precompute access sets for faster conflict checks
	sys.Meta.Access.PrepareSets()

	// Cache typed function if signature matches to avoid repeated type assertions at runtime
	if fn, ok := sys.Fn.(func(context.Context, any)); ok {
		sys.Fn = fn
	} else {
		name := sys.Name
		sys.Fn = func(context.Context, any) {
			panic(fmt.Sprintf("invalid system function signature for %s", name))
		}
	}

	s.systems[sys.Stage] = append(s.systems[sys.Stage], sys)
	s.batches[sys.Stage] = nil // Invalidate batches
}

// Build computes the execution order and parallel batches for all stages.
func (s *Scheduler) Build() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	newBatches := make(map[Stage][][]*System, len(s.systems))
	for stage, systems := range s.systems {
		// Validate dependencies first (detect cycles)
		if _, err := s.topologicalSort(systems); err != nil {
			return fmt.Errorf("stage %v: %w", stage, err)
		}
		// Build dependency-aware batches
		newBatches[stage] = s.computeBatches(systems)
	}
	s.batches = newBatches

	return nil
}

// topologicalSort orders systems based on Before/After constraints (deterministic).
func (s *Scheduler) topologicalSort(systems []*System) ([]*System, error) {
	// Build name and set maps
	nameToSys := make(map[string]*System, len(systems))
	setMembers := make(map[string][]*System)
	for _, sys := range systems {
		nameToSys[sys.Name] = sys
		if sys.Meta.Set != "" {
			setMembers[sys.Meta.Set] = append(setMembers[sys.Meta.Set], sys)
		}
	}

	// Build adjacency and indegree
	outgoing := make(map[*System]map[*System]bool, len(systems))
	inDegree := make(map[*System]int, len(systems))
	ensure := func(m map[*System]map[*System]bool, k *System) map[*System]bool {
		if m[k] == nil {
			m[k] = make(map[*System]bool)
		}
		return m[k]
	}
	for _, sys := range systems {
		outgoing[sys] = make(map[*System]bool)
		inDegree[sys] = 0
	}
	addEdge := func(a, b *System) {
		if !outgoing[a][b] {
			ensure(outgoing, a)[b] = true
			inDegree[b]++
		}
	}

	for _, sys := range systems {
		// sys must run before targets
		for _, target := range sys.Meta.Before {
			if targetSys, ok := nameToSys[target]; ok {
				addEdge(sys, targetSys)
			} else if members, ok := setMembers[target]; ok {
				for _, member := range members {
					addEdge(sys, member)
				}
			}
		}
		// sys must run after deps
		for _, dep := range sys.Meta.After {
			if depSys, ok := nameToSys[dep]; ok {
				addEdge(depSys, sys)
			} else if members, ok := setMembers[dep]; ok {
				for _, member := range members {
					addEdge(member, sys)
				}
			}
		}
	}

	// Zero in-degree queue (deterministic by name)
	var zero []*System
	for _, sys := range systems {
		if inDegree[sys] == 0 {
			zero = append(zero, sys)
		}
	}
	sort.Slice(zero, func(i, j int) bool { return zero[i].Name < zero[j].Name })

	var result []*System
	for len(zero) > 0 {
		cur := zero[0]
		zero = zero[1:]
		result = append(result, cur)

		for neigh := range outgoing[cur] {
			inDegree[neigh]--
			if inDegree[neigh] == 0 {
				zero = append(zero, neigh)
			}
		}
		// keep deterministic
		sort.Slice(zero, func(i, j int) bool { return zero[i].Name < zero[j].Name })
	}

	if len(result) != len(systems) {
		return nil, fmt.Errorf("cyclic dependency detected")
	}
	return result, nil
}

// computeBatches groups systems into parallel batches based on access conflicts
// while respecting Before/After constraints using DAG levels.
func (s *Scheduler) computeBatches(systems []*System) [][]*System {
	// Build name and set indexes
	nameToSys := make(map[string]*System, len(systems))
	setMembers := make(map[string][]*System)
	for _, sys := range systems {
		nameToSys[sys.Name] = sys
		if sys.Meta.Set != "" {
			setMembers[sys.Meta.Set] = append(setMembers[sys.Meta.Set], sys)
		}
	}

	// Build dependency graph
	outgoing := make(map[*System]map[*System]bool, len(systems))
	inDegree := make(map[*System]int, len(systems))
	ensure := func(m map[*System]map[*System]bool, k *System) map[*System]bool {
		if m[k] == nil {
			m[k] = make(map[*System]bool)
		}
		return m[k]
	}
	for _, sys := range systems {
		outgoing[sys] = make(map[*System]bool)
		inDegree[sys] = 0
	}
	addDep := func(a, b *System) {
		// a -> b (b depends on a)
		if !outgoing[a][b] {
			ensure(outgoing, a)[b] = true
			inDegree[b]++
		}
	}
	for _, sys := range systems {
		// sys must run after deps
		for _, dep := range sys.Meta.After {
			if depSys, ok := nameToSys[dep]; ok {
				addDep(depSys, sys)
			} else if members, ok := setMembers[dep]; ok {
				for _, m := range members {
					addDep(m, sys)
				}
			}
		}
		// targets must run after sys
		for _, target := range sys.Meta.Before {
			if tgtSys, ok := nameToSys[target]; ok {
				addDep(sys, tgtSys)
			} else if members, ok := setMembers[target]; ok {
				for _, m := range members {
					addDep(sys, m)
				}
			}
		}
	}

	// Initialize ready list (zero in-degree), deterministic by name
	var ready []*System
	for _, sys := range systems {
		if inDegree[sys] == 0 {
			ready = append(ready, sys)
		}
	}
	sort.Slice(ready, func(i, j int) bool { return ready[i].Name < ready[j].Name })

	remaining := len(systems)
	var batches [][]*System

	for remaining > 0 {
		// Safety: If no ready nodes remain (shouldn't happen after Build), try to drain sequentially.
		if len(ready) == 0 {
			// Pick any node with positive indegree to break a cycle
			var any *System
			for _, sys := range systems {
				if inDegree[sys] > 0 {
					any = sys
					break
				}
			}
			if any == nil {
				break
			}
			ready = []*System{any}
		}

		// Make as many conflict-free batches as needed from current ready set
		current := append([]*System(nil), ready...)
		used := make([]bool, len(current))

		for {
			var batch []*System
			for i, sys := range current {
				if used[i] {
					continue
				}
				canAdd := true
				for _, other := range batch {
					if sys.Meta.Access.Conflicts(other.Meta.Access) {
						canAdd = false
						break
					}
				}
				if canAdd {
					batch = append(batch, sys)
					used[i] = true
				}
			}

			if len(batch) == 0 {
				break
			}

			batches = append(batches, batch)

			// Update in-degrees and build next ready set
			nextReadySet := make(map[*System]bool)
			for i, sys := range current {
				if !used[i] {
					nextReadySet[sys] = true
				}
			}
			for _, sys := range batch {
				for neigh := range outgoing[sys] {
					inDegree[neigh]--
					if inDegree[neigh] == 0 {
						nextReadySet[neigh] = true
					}
				}
				// mark removed
				inDegree[sys] = -1
				remaining--
			}

			ready = ready[:0]
			for n := range nextReadySet {
				if inDegree[n] == 0 {
					ready = append(ready, n)
				}
			}
			sort.Slice(ready, func(i, j int) bool { return ready[i].Name < ready[j].Name })

			current = append([]*System(nil), ready...)
			used = make([]bool, len(current))
		}
	}

	return batches
}

// Diagnostics is the interface for system execution diagnostics.
type Diagnostics interface {
	SystemStart(name string, stage Stage)
	SystemEnd(name string, stage Stage, err error, duration time.Duration)
}

// RunStage executes all systems for the given stage.
func (s *Scheduler) RunStage(ctx context.Context, stage Stage, w any, diag Diagnostics) {
	s.mu.RLock()
	batches := s.batches[stage]
	s.mu.RUnlock()

	// Bounded worker pool reused across batches to reduce goroutine churn
	type job struct {
		sys  *System
		done func()
	}

	work := make(chan job)
	maxWorkers := max(runtime.GOMAXPROCS(0), 1)

	var workersWG sync.WaitGroup
	workersWG.Add(maxWorkers)
	for range maxWorkers {
		go func() {
			defer workersWG.Done()
			for j := range work {
				s.runSystem(ctx, j.sys, w, diag)
				j.done()
			}
		}()
	}
	defer func() {
		close(work)
		workersWG.Wait()
	}()

	for _, batch := range batches {
		sort.Slice(batch, func(i, j int) bool { return batch[i].Name < batch[j].Name })
		// Allow cancellation between batches
		if err := ctx.Err(); err != nil {
			return
		}

		var batchWG sync.WaitGroup
		for _, sys := range batch {
			if !sys.ShouldRun(time.Now()) {
				continue
			}
			batchWG.Add(1)
			work <- job{sys: sys, done: batchWG.Done}
		}
		batchWG.Wait()
	}
}

// runSystem executes a single system with diagnostics and error handling.
func (s *Scheduler) runSystem(ctx context.Context, sys *System, w any, diag Diagnostics) {
	if diag != nil {
		diag.SystemStart(sys.Name, sys.Stage)
	}

	start := time.Now()
	var runErr error

	defer func() {
		if r := recover(); r != nil {
			runErr = fmt.Errorf("panic: %v\n%s", r, debug.Stack())
		}
		if diag != nil {
			diag.SystemEnd(sys.Name, sys.Stage, runErr, time.Since(start))
		}
		// Use actual end time for gating accuracy
		sys.MarkRun(time.Now())
	}()

	fn := sys.Fn.(func(context.Context, any))
	fn(ctx, w)
}
