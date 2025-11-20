package scheduler

import (
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

// Stage represents a scheduling stage.
type Stage int

// SystemMeta describes system scheduling metadata.
type SystemMeta struct {
	Access AccessMeta
	Set    string
	Before []string
	After  []string
	Every  time.Duration
}

// AccessMeta describes what resources a system reads or writes.
type AccessMeta struct {
	Reads       []reflect.Type
	Writes      []reflect.Type
	ResReads    []reflect.Type
	ResWrites   []reflect.Type
	EventReads  []reflect.Type
	EventWrites []reflect.Type

	// Precomputed sets for fast conflict checks
	readsSet       map[reflect.Type]struct{}
	writesSet      map[reflect.Type]struct{}
	resReadsSet    map[reflect.Type]struct{}
	resWritesSet   map[reflect.Type]struct{}
	eventReadsSet  map[reflect.Type]struct{}
	eventWritesSet map[reflect.Type]struct{}

	// Compact bitset representation using a TypeIndex
	readsBits       *BitSet
	writesBits      *BitSet
	resReadsBits    *BitSet
	resWritesBits   *BitSet
	eventReadsBits  *BitSet
	eventWritesBits *BitSet
}

// PrepareSets precomputes lookup sets from the slice fields for faster conflict checks.
// It also builds compact bitsets using the provided TypeIndex for very fast set algebra.
func (a *AccessMeta) PrepareSets(ti *TypeIndex) {
	build := func(dst *map[reflect.Type]struct{}, src []reflect.Type) {
		if len(src) == 0 {
			*dst = nil
			return
		}
		m := make(map[reflect.Type]struct{}, len(src))
		for _, t := range src {
			m[t] = struct{}{}
		}
		*dst = m
	}
	build(&a.readsSet, a.Reads)
	build(&a.writesSet, a.Writes)
	build(&a.resReadsSet, a.ResReads)
	build(&a.resWritesSet, a.ResWrites)
	build(&a.eventReadsSet, a.EventReads)
	build(&a.eventWritesSet, a.EventWrites)

	// Build compact bitsets
	buildBits := func(src []reflect.Type) *BitSet {
		if len(src) == 0 {
			return nil
		}
		b := &BitSet{}
		for _, t := range src {
			idx := ti.indexOf(t)
			b.Set(idx)
		}
		return b
	}
	a.readsBits = buildBits(a.Reads)
	a.writesBits = buildBits(a.Writes)
	a.resReadsBits = buildBits(a.ResReads)
	a.resWritesBits = buildBits(a.ResWrites)
	a.eventReadsBits = buildBits(a.EventReads)
	a.eventWritesBits = buildBits(a.EventWrites)
}

// System represents a registered system with its metadata.
type System struct {
	Name        string
	Stage       Stage
	Fn          any // func(context.Context, *ecs.World)
	Meta        SystemMeta
	lastRunUnix atomic.Int64
	LastRun     time.Time
	nextRunUnix atomic.Int64
}

// ShouldRun checks if the system should run based on its Every constraint.
func (s *System) ShouldRun(now time.Time) bool {
	if s.Meta.Every == 0 {
		return true
	}

	next := s.nextRunUnix.Load()
	if next != 0 {
		return now.UnixNano() >= next
	}

	// First-time check (next == 0). Initialize from last run time to preserve
	// existing behavior for systems that might have LastRun set manually.
	lastUnix := s.lastRunUnix.Load()
	var last time.Time
	if lastUnix != 0 {
		last = time.Unix(0, lastUnix)
	} else {
		last = s.LastRun
	}

	if last.IsZero() {
		// No last run time, so it runs now. nextRunUnix will be set in MarkRun.
		return true
	}

	// Has a last run time. Schedule the *correct* first drift-free time.
	firstDeadline := last.Add(s.Meta.Every).UnixNano()
	s.nextRunUnix.Store(firstDeadline)

	return now.UnixNano() >= firstDeadline
}

// MarkRun updates the last run timestamp.
func (s *System) MarkRun(now time.Time) {
	s.lastRunUnix.Store(now.UnixNano())
	s.LastRun = now

	if s.Meta.Every <= 0 {
		return
	}

	nowNanos := now.UnixNano()
	lastScheduled := s.nextRunUnix.Load()

	// If this was the first run for a system with no initial LastRun,
	// lastScheduled will be 0. We base the next run on now.
	if lastScheduled == 0 {
		lastScheduled = nowNanos
	}

	// Drift-free update: the next deadline is calculated from the previous deadline.
	next := lastScheduled + s.Meta.Every.Nanoseconds()

	// Reset schedule if we are lagging to prevent catch-up bursts.
	if next < nowNanos {
		next = nowNanos + s.Meta.Every.Nanoseconds()
	}

	s.nextRunUnix.Store(next)
}

// TypeIndex maps reflect.Type -> small int for compact bitsets.
type TypeIndex struct {
	mu sync.Mutex
	m  map[reflect.Type]int
}

func (ti *TypeIndex) ensure() {
	if ti.m == nil {
		ti.m = make(map[reflect.Type]int)
	}
}

func (ti *TypeIndex) indexOf(t reflect.Type) int {
	ti.mu.Lock()
	defer ti.mu.Unlock()
	ti.ensure()
	if idx, ok := ti.m[t]; ok {
		return idx
	}
	idx := len(ti.m)
	ti.m[t] = idx
	return idx
}
