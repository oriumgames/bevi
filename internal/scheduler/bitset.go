package scheduler

import "math/bits"

// BitSet is a compact set of non-negative integers implemented as a slice of
// 64-bit words. It supports fast set algebra and membership checks with low
// memory overhead. All operations are safe for negative indexes (they are
// treated as no-ops or return false).
//
// This implementation focuses on zero-allocation hot paths once capacity is
// provisioned. It exposes helpers to reduce allocations and to trim capacity
// when needed.
//
// Concurrency: BitSet is not thread-safe for concurrent writers. Readers that
// only call Has/IsDisjoint/Count concurrently with writers may see torn reads.
// Guard with a mutex at a higher level if you need synchronization.
type BitSet struct {
	words []uint64
}

// NewBitSet creates a new empty BitSet. If capacityWords > 0, the internal
// storage is pre-allocated to capacityWords 64-bit words (capacityWords*64 bits).
func NewBitSet(capacityWords int) *BitSet {
	if capacityWords < 0 {
		capacityWords = 0
	}
	return &BitSet{
		words: make([]uint64, capacityWords),
	}
}

// FromIndices constructs a new BitSet with all provided indices set.
func FromIndices(idxs ...int) *BitSet {
	b := &BitSet{}
	for _, i := range idxs {
		b.Set(i)
	}
	return b
}

// Clone returns a deep copy of the bitset.
func (b *BitSet) Clone() *BitSet {
	if b == nil {
		return nil
	}
	cp := &BitSet{words: make([]uint64, len(b.words))}
	copy(cp.words, b.words)
	return cp
}

// Reset clears all bits but retains capacity.
func (b *BitSet) Reset() {
	for i := range b.words {
		b.words[i] = 0
	}
}

// TrimRight trims trailing zero words to minimize memory footprint.
// Returns the number of words removed.
func (b *BitSet) TrimRight() int {
	i := len(b.words) - 1
	for i >= 0 && b.words[i] == 0 {
		i--
	}
	removed := len(b.words) - (i + 1)
	if removed > 0 {
		b.words = b.words[:i+1]
	}
	return removed
}

// ensure ensures there is capacity for index i.
func (b *BitSet) ensure(i int) {
	if i < 0 {
		return
	}
	w := wordIndex(i)
	if w < len(b.words) {
		return
	}
	n := w + 1
	// Grow exponentially to amortize allocations.
	capacity := len(b.words)
	if capacity == 0 {
		capacity = 1
	}
	for capacity < n {
		capacity <<= 1
	}
	newWords := make([]uint64, capacity)
	copy(newWords, b.words)
	b.words = newWords
}

// Set sets bit i to 1.
func (b *BitSet) Set(i int) {
	if i < 0 {
		return
	}
	b.ensure(i)
	w := wordIndex(i)
	b.words[w] |= bitMask(i)
}

// Clear resets bit i to 0.
func (b *BitSet) Clear(i int) {
	if i < 0 {
		return
	}
	w := wordIndex(i)
	if w >= len(b.words) {
		return
	}
	b.words[w] &^= bitMask(i)
}

// Toggle flips bit i.
func (b *BitSet) Toggle(i int) {
	if i < 0 {
		return
	}
	b.ensure(i)
	w := wordIndex(i)
	b.words[w] ^= bitMask(i)
}

// Has reports whether bit i is set.
func (b *BitSet) Has(i int) bool {
	if i < 0 {
		return false
	}
	w := wordIndex(i)
	if w >= len(b.words) {
		return false
	}
	return (b.words[w] & bitMask(i)) != 0
}

// SetRange sets all bits in [lo, hi] inclusive. No-ops if the range is empty
// or invalid (lo < 0 or hi < lo).
func (b *BitSet) SetRange(lo, hi int) {
	if lo < 0 || hi < lo {
		return
	}
	b.ensure(hi)
	loW, hiW := wordIndex(lo), wordIndex(hi)
	loOff, hiOff := uint(lo&63), uint(hi&63)
	if loW == hiW {
		mask := maskRange(loOff, hiOff)
		b.words[loW] |= mask
		return
	}
	// First partial word
	b.words[loW] |= ^uint64(0) << loOff
	// Middle full words
	for w := loW + 1; w < hiW; w++ {
		b.words[w] = ^uint64(0)
	}
	// Last partial word
	b.words[hiW] |= (uint64(1)<<(hiOff+1) - 1)
}

// ClearRange clears all bits in [lo, hi] inclusive. No-ops if the range is empty
// or invalid (lo < 0 or hi < lo).
func (b *BitSet) ClearRange(lo, hi int) {
	if lo < 0 || hi < lo {
		return
	}
	loW, hiW := wordIndex(lo), wordIndex(hi)
	if loW >= len(b.words) {
		return
	}
	hiW = min(hiW, len(b.words)-1)
	loOff, hiOff := uint(lo&63), uint(hi&63)
	if loW == hiW {
		mask := maskRange(loOff, hiOff)
		b.words[loW] &^= mask
		return
	}
	// First partial word
	b.words[loW] &^= ^uint64(0) << loOff
	// Middle full words
	for w := loW + 1; w < hiW; w++ {
		b.words[w] = 0
	}
	// Last partial word
	b.words[hiW] &^= (uint64(1)<<(hiOff+1) - 1)
}

// Union sets b = b ∪ other.
func (b *BitSet) Union(other *BitSet) {
	if other == nil {
		return
	}
	n := len(other.words)
	if n > len(b.words) {
		// Expand b to fit other
		newWords := make([]uint64, n)
		copy(newWords, b.words)
		b.words = newWords
	}
	for i := range n {
		b.words[i] |= other.words[i]
	}
}

// Intersect sets b = b ∩ other.
func (b *BitSet) Intersect(other *BitSet) {
	if other == nil {
		// Intersect with empty => empty
		for i := range b.words {
			b.words[i] = 0
		}
		return
	}
	minLen := min(len(b.words), len(other.words))
	for i := range minLen {
		b.words[i] &= other.words[i]
	}
	for i := minLen; i < len(b.words); i++ {
		b.words[i] = 0
	}
}

// Difference sets b = b \\ other.
func (b *BitSet) Difference(other *BitSet) {
	if other == nil {
		return
	}
	minLen := min(len(b.words), len(other.words))
	for i := range minLen {
		b.words[i] &^= other.words[i]
	}
	// higher words remain unchanged
}

// SymmetricDifference sets b = (b ∪ other) \\ (b ∩ other).
func (b *BitSet) SymmetricDifference(other *BitSet) {
	if other == nil {
		return
	}
	n := len(other.words)
	if n > len(b.words) {
		newWords := make([]uint64, n)
		copy(newWords, b.words)
		b.words = newWords
	}
	for i := range n {
		b.words[i] ^= other.words[i]
	}
}

// IsDisjoint reports whether b and other have no elements in common.
func (b *BitSet) IsDisjoint(other *BitSet) bool {
	if b == nil || other == nil {
		return true
	}
	minLen := min(len(b.words), len(other.words))
	for i := range minLen {
		if (b.words[i] & other.words[i]) != 0 {
			return false
		}
	}
	return true
}

// IsEmpty reports whether no bits are set.
func (b *BitSet) IsEmpty() bool {
	for i := range b.words {
		if b.words[i] != 0 {
			return false
		}
	}
	return true
}

// Count returns the number of set bits.
func (b *BitSet) Count() int {
	count := 0
	for _, w := range b.words {
		count += bits.OnesCount64(w)
	}
	return count
}

// NextSet returns the index of the next set bit at or after 'from'.
// If none exists, it returns -1.
func (b *BitSet) NextSet(from int) int {
	if from < 0 {
		from = 0
	}
	wIdx := wordIndex(from)
	if wIdx >= len(b.words) {
		return -1
	}
	// Mask off bits below 'from' in the first word.
	w := b.words[wIdx]
	bitOff := uint(from & 63)
	if bitOff != 0 {
		w &= ^((uint64(1) << bitOff) - 1)
	}
	for {
		if w != 0 {
			// Find least significant set bit index within this word.
			v := w
			pos := 0
			for (v & 1) == 0 {
				v >>= 1
				pos++
			}
			return wIdx*64 + pos
		}
		wIdx++
		if wIdx >= len(b.words) {
			return -1
		}
		w = b.words[wIdx]
	}
}

// ForEach iterates all set bits in ascending order and calls fn(idx).
// If fn returns false, iteration stops early.
func (b *BitSet) ForEach(fn func(idx int) bool) {
	if fn == nil {
		return
	}
	for wi, w := range b.words {
		for w != 0 {
			// Isolate least significant set bit.
			lsb := w & -w
			// Compute its bit position by counting shifts. This loop runs at most 64 times per word,
			// but in practice much fewer as we skip whole chunks by clearing lsb each iteration.
			pos := 0
			v := lsb
			for (v & 1) == 0 {
				v >>= 1
				pos++
			}
			idx := wi*64 + pos
			if !fn(idx) {
				return
			}
			// Clear the least significant set bit.
			w &= w - 1
		}
	}
}

// Internal helpers

func wordIndex(i int) int  { return i >> 6 } // divide by 64
func bitMask(i int) uint64 { return 1 << (uint(i) & 63) }

func maskRange(lo, hi uint) uint64 {
	if hi >= 63 {
		return ^uint64(0) << lo
	}
	return ((uint64(1) << (hi + 1)) - 1) &^ ((uint64(1) << lo) - 1)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
