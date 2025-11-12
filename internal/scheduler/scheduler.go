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

// job is an internal struct for dispatching system execution to the worker pool.
type job struct {
	ctx  context.Context
	sys  *System
	w    any
	diag Diagnostics
	wg   *sync.WaitGroup
}

// systemSorter implements sort.Interface for []*System to avoid closure allocations.
type systemSorter struct {
	systems []*System
}

func (s *systemSorter) Len() int           { return len(s.systems) }
func (s *systemSorter) Swap(i, j int)      { s.systems[i], s.systems[j] = s.systems[j], s.systems[i] }
func (s *systemSorter) Less(i, j int) bool { return s.systems[i].Name < s.systems[j].Name }

// Scheduler manages system execution order and parallelization.
type Scheduler struct {
	mu      sync.RWMutex
	systems map[Stage][]*System
	batches map[Stage][][]*System

	// Worker pool
	maxWorkers    int
	work          chan *job
	workersWG     sync.WaitGroup
	startOnce     sync.Once
	jobPool       sync.Pool
	waitGroupPool sync.Pool

	// Reusable data structures to avoid allocations
	sorter     *systemSorter
	nameToSys  map[string]*System
	setMembers map[string][]*System
	outgoing   map[*System]map[*System]bool
	inDegree   map[*System]int
}

// NewScheduler creates a new scheduler.
func NewScheduler() *Scheduler {
	return &Scheduler{
		systems:    make(map[Stage][]*System),
		batches:    make(map[Stage][][]*System),
		maxWorkers: max(runtime.GOMAXPROCS(0), 1),
		jobPool: sync.Pool{
			New: func() any { return new(job) },
		},
		waitGroupPool: sync.Pool{
			New: func() any { return new(sync.WaitGroup) },
		},
		sorter:     &systemSorter{},
		nameToSys:  make(map[string]*System),
		setMembers: make(map[string][]*System),
		outgoing:   make(map[*System]map[*System]bool),
		inDegree:   make(map[*System]int),
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
		// Clear reusable data structures for this stage.
		for k := range s.nameToSys {
			delete(s.nameToSys, k)
		}
		for k := range s.setMembers {
			delete(s.setMembers, k)
		}
		for k := range s.outgoing {
			delete(s.outgoing, k)
		}
		for k := range s.inDegree {
			delete(s.inDegree, k)
		}

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

// Startup initializes the persistent worker pool. It is safe to call multiple times.
// It is called automatically by the first RunStage execution.
func (s *Scheduler) Startup() {
	s.startOnce.Do(func() {
		s.work = make(chan *job)
		s.workersWG.Add(s.maxWorkers)
		for i := 0; i < s.maxWorkers; i++ {
			go func() {
				defer s.workersWG.Done()
				for j := range s.work {
					s.runSystem(j.ctx, j.sys, j.w, j.diag)
					j.wg.Done()
					// Reset job and return to pool to avoid allocations.
					*j = job{} // j.wg is overwritten on next Get, no need to nil it.
					s.jobPool.Put(j)
				}
			}()
		}
	})
}

// Shutdown gracefully stops the worker pool and waits for all workers to exit.
func (s *Scheduler) Shutdown() {
	// Check if the pool was ever started
	if s.work == nil {
		return
	}
	close(s.work)
	s.workersWG.Wait()
}

// topologicalSort orders systems based on Before/After constraints (deterministic).
func (s *Scheduler) topologicalSort(systems []*System) ([]*System, error) {
	// Build name and set maps using reusable fields.
	for _, sys := range systems {
		s.nameToSys[sys.Name] = sys
		if sys.Meta.Set != "" {
			s.setMembers[sys.Meta.Set] = append(s.setMembers[sys.Meta.Set], sys)
		}
	}

	// Build adjacency and indegree using reusable fields.
	ensure := func(m map[*System]map[*System]bool, k *System) map[*System]bool {
		if m[k] == nil {
			m[k] = make(map[*System]bool)
		}
		return m[k]
	}
	for _, sys := range systems {
		s.outgoing[sys] = make(map[*System]bool)
		s.inDegree[sys] = 0
	}
	addEdge := func(a, b *System) {
		if !s.outgoing[a][b] {
			ensure(s.outgoing, a)[b] = true
			s.inDegree[b]++
		}
	}

	for _, sys := range systems {
		// sys must run before targets
		for _, target := range sys.Meta.Before {
			if targetSys, ok := s.nameToSys[target]; ok {
				addEdge(sys, targetSys)
			} else if members, ok := s.setMembers[target]; ok {
				for _, member := range members {
					addEdge(sys, member)
				}
			}
		}
		// sys must run after deps
		for _, dep := range sys.Meta.After {
			if depSys, ok := s.nameToSys[dep]; ok {
				addEdge(depSys, sys)
			} else if members, ok := s.setMembers[dep]; ok {
				for _, member := range members {
					addEdge(member, sys)
				}
			}
		}
	}

	// Zero in-degree queue (deterministic by name)
	var zero []*System
	for _, sys := range systems {
		if s.inDegree[sys] == 0 {
			zero = append(zero, sys)
		}
	}
	sort.Slice(zero, func(i, j int) bool { return zero[i].Name < zero[j].Name })

	var result []*System
	for len(zero) > 0 {
		cur := zero[0]
		zero = zero[1:]
		result = append(result, cur)

		for neigh := range s.outgoing[cur] {
			s.inDegree[neigh]--
			if s.inDegree[neigh] == 0 {
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
	// Rebuild dependency graph for this stage using shared maps
	// nameToSys and setMembers were already populated by topologicalSort
	// We must re-calculate outgoing and inDegree as topologicalSort consumes them
	ensure := func(m map[*System]map[*System]bool, k *System) map[*System]bool {
		if m[k] == nil {
			m[k] = make(map[*System]bool)
		}
		return m[k]
	}
	for _, sys := range systems {
		s.outgoing[sys] = make(map[*System]bool)
		s.inDegree[sys] = 0
	}
	addDep := func(a, b *System) {
		// a -> b (b depends on a)
		if !s.outgoing[a][b] {
			ensure(s.outgoing, a)[b] = true
			s.inDegree[b]++
		}
	}
	for _, sys := range systems {
		// sys must run after deps
		for _, dep := range sys.Meta.After {
			if depSys, ok := s.nameToSys[dep]; ok {
				addDep(depSys, sys)
			} else if members, ok := s.setMembers[dep]; ok {
				for _, m := range members {
					addDep(m, sys)
				}
			}
		}
		// targets must run after sys
		for _, target := range sys.Meta.Before {
			if tgtSys, ok := s.nameToSys[target]; ok {
				addDep(sys, tgtSys)
			} else if members, ok := s.setMembers[target]; ok {
				for _, m := range members {
					addDep(sys, m)
				}
			}
		}
	}

	// Initialize ready list (zero in-degree), deterministic by name
	var ready []*System
	for _, sys := range systems {
		if s.inDegree[sys] == 0 {
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
				if s.inDegree[sys] > 0 {
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
				for neigh := range s.outgoing[sys] {
					s.inDegree[neigh]--
					if s.inDegree[neigh] == 0 {
						nextReadySet[neigh] = true
					}
				}
				// mark removed
				s.inDegree[sys] = -1
				remaining--
			}

			ready = ready[:0]
			for n := range nextReadySet {
				if s.inDegree[n] == 0 {
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
	// Ensure the worker pool is running. This is safe to call multiple times
	s.Startup()

	s.mu.RLock()
	batches := s.batches[stage]
	s.mu.RUnlock()

	for _, batch := range batches {
		// Allow cancellation between batches
		if err := ctx.Err(); err != nil {
			return
		}

		// Systems within a batch are dispatched in a deterministic (sorted) order
		s.sorter.systems = batch
		sort.Sort(s.sorter)

		batchWG := s.waitGroupPool.Get().(*sync.WaitGroup)
		for _, sys := range batch {
			if !sys.ShouldRun(time.Now()) {
				continue
			}
			batchWG.Add(1)
			j := s.jobPool.Get().(*job)
			j.ctx = ctx
			j.sys = sys
			j.w = w
			j.diag = diag
			j.wg = batchWG
			s.work <- j
		}
		batchWG.Wait()
		s.waitGroupPool.Put(batchWG)
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
		end := time.Now()

		r := recover()
		if r != nil {
			runErr = fmt.Errorf("panic: %v\n%s", r, debug.Stack())
		}

		if diag != nil {
			diag.SystemEnd(sys.Name, sys.Stage, runErr, end.Sub(start))
		}

		// Use actual end time for gating accuracy
		sys.MarkRun(end)

		if r != nil {
			panic(r)
		}
	}()

	fn := sys.Fn.(func(context.Context, any))
	fn(ctx, w)
}
