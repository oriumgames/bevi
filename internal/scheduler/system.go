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

	// Compact bitset representation using a global TypeIndex
	readsBits       *bitset
	writesBits      *bitset
	resReadsBits    *bitset
	resWritesBits   *bitset
	eventReadsBits  *bitset
	eventWritesBits *bitset
}

// PrepareSets precomputes lookup sets from the slice fields for faster conflict checks.
// It also builds compact bitsets using a global TypeIndex for very fast set algebra.
func (a *AccessMeta) PrepareSets() {
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
	buildBits := func(src []reflect.Type) *bitset {
		if len(src) == 0 {
			return nil
		}
		b := &bitset{}
		for _, t := range src {
			idx := globalTypeIndex.indexOf(t)
			b.set(idx)
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
	Fn          any // func(context.Context, *World)
	Meta        SystemMeta
	lastRunUnix atomic.Int64
	LastRun     time.Time
}

// ShouldRun checks if the system should run based on its Every constraint.
func (s *System) ShouldRun(now time.Time) bool {
	if s.Meta.Every == 0 {
		return true
	}
	// Fast, lock-free read using atomic timestamp
	lastUnix := s.lastRunUnix.Load()
	var last time.Time
	if lastUnix != 0 {
		last = time.Unix(0, lastUnix)
	} else {
		// Fallback for code paths that set LastRun directly (e.g., tests)
		last = s.LastRun
		if !last.IsZero() {
			s.lastRunUnix.Store(last.UnixNano())
		}
	}
	return now.Sub(last) >= s.Meta.Every
}

// MarkRun updates the last run timestamp.
func (s *System) MarkRun(now time.Time) {
	s.lastRunUnix.Store(now.UnixNano())
	s.LastRun = now
}

// Global compact type index for reflect.Type -> small int mapping.
// This enables compact bitset representations for AccessMeta.
var globalTypeIndex typeIndex

type typeIndex struct {
	mu sync.Mutex
	m  map[reflect.Type]int
}

func (ti *typeIndex) ensure() {
	if ti.m == nil {
		ti.m = make(map[reflect.Type]int)
	}
}

func (ti *typeIndex) indexOf(t reflect.Type) int {
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

// Minimal bitset for compact membership and fast intersection checks.
type bitset struct {
	words []uint64
}

func (b *bitset) set(i int) {
	if i < 0 {
		return
	}
	w := i >> 6
	off := uint(i & 63)
	if w >= len(b.words) {
		nw := make([]uint64, w+1)
		copy(nw, b.words)
		b.words = nw
	}
	b.words[w] |= 1 << off
}

func (b *bitset) anyIntersect(other *bitset) bool {
	if b == nil || other == nil {
		return false
	}
	n := len(b.words)
	if len(other.words) < n {
		n = len(other.words)
	}
	for i := 0; i < n; i++ {
		if (b.words[i] & other.words[i]) != 0 {
			return true
		}
	}
	return false
}
