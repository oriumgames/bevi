package scheduler

import "slices"

// Conflicts returns true if this access conflicts with another.
func (a AccessMeta) Conflicts(other AccessMeta) bool {
	// Fast path: use compact bitsets if available.
	// Components
	if a.writesBits != nil && other.readsBits != nil && a.writesBits.anyIntersect(other.readsBits) {
		return true
	}
	if a.writesBits != nil && other.writesBits != nil && a.writesBits.anyIntersect(other.writesBits) {
		return true
	}
	if a.readsBits != nil && other.writesBits != nil && a.readsBits.anyIntersect(other.writesBits) {
		return true
	}
	// Resources
	if a.resWritesBits != nil && other.resReadsBits != nil && a.resWritesBits.anyIntersect(other.resReadsBits) {
		return true
	}
	if a.resWritesBits != nil && other.resWritesBits != nil && a.resWritesBits.anyIntersect(other.resWritesBits) {
		return true
	}
	if a.resReadsBits != nil && other.resWritesBits != nil && a.resReadsBits.anyIntersect(other.resWritesBits) {
		return true
	}
	// Events
	if a.eventWritesBits != nil && other.eventReadsBits != nil && a.eventWritesBits.anyIntersect(other.eventReadsBits) {
		return true
	}
	if a.eventWritesBits != nil && other.eventWritesBits != nil && a.eventWritesBits.anyIntersect(other.eventWritesBits) {
		return true
	}
	if a.eventReadsBits != nil && other.eventWritesBits != nil && a.eventReadsBits.anyIntersect(other.eventWritesBits) {
		return true
	}

	// Fallbacks using precomputed map sets when available (no extra allocations).
	// Components
	if other.readsSet != nil {
		for _, w := range a.Writes {
			if _, ok := other.readsSet[w]; ok {
				return true
			}
		}
	} else {
		for _, w := range a.Writes {
			if slices.Contains(other.Reads, w) {
				return true
			}
		}
	}
	if other.writesSet != nil {
		for _, w := range a.Writes {
			if _, ok := other.writesSet[w]; ok {
				return true
			}
		}
		for _, r := range a.Reads {
			if _, ok := other.writesSet[r]; ok {
				return true
			}
		}
	} else {
		for _, w := range a.Writes {
			if slices.Contains(other.Writes, w) {
				return true
			}
		}
		for _, r := range a.Reads {
			if slices.Contains(other.Writes, r) {
				return true
			}
		}
	}

	// Resources
	if other.resReadsSet != nil {
		for _, w := range a.ResWrites {
			if _, ok := other.resReadsSet[w]; ok {
				return true
			}
		}
	} else {
		for _, w := range a.ResWrites {
			if slices.Contains(other.ResReads, w) {
				return true
			}
		}
	}
	if other.resWritesSet != nil {
		for _, w := range a.ResWrites {
			if _, ok := other.resWritesSet[w]; ok {
				return true
			}
		}
		for _, r := range a.ResReads {
			if _, ok := other.resWritesSet[r]; ok {
				return true
			}
		}
	} else {
		for _, w := range a.ResWrites {
			if slices.Contains(other.ResWrites, w) {
				return true
			}
		}
		for _, r := range a.ResReads {
			if slices.Contains(other.ResWrites, r) {
				return true
			}
		}
	}

	// Events
	if other.eventReadsSet != nil {
		for _, w := range a.EventWrites {
			if _, ok := other.eventReadsSet[w]; ok {
				return true
			}
		}
	} else {
		for _, w := range a.EventWrites {
			if slices.Contains(other.EventReads, w) {
				return true
			}
		}
	}
	if other.eventWritesSet != nil {
		for _, w := range a.EventWrites {
			if _, ok := other.eventWritesSet[w]; ok {
				return true
			}
		}
		for _, r := range a.EventReads {
			if _, ok := other.eventWritesSet[r]; ok {
				return true
			}
		}
	} else {
		for _, w := range a.EventWrites {
			if slices.Contains(other.EventWrites, w) {
				return true
			}
		}
		for _, r := range a.EventReads {
			if slices.Contains(other.EventWrites, r) {
				return true
			}
		}
	}

	return false
}
