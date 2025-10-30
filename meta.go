package bevi

import (
	"reflect"
	"time"

	"github.com/oriumgames/bevi/internal/scheduler"
)

// AccessMeta describes what resources a system reads or writes.
type AccessMeta struct {
	Reads       []reflect.Type
	Writes      []reflect.Type
	ResReads    []reflect.Type
	ResWrites   []reflect.Type
	EventReads  []reflect.Type
	EventWrites []reflect.Type
}

// NewAccess creates a new empty AccessMeta.
func NewAccess() AccessMeta {
	return AccessMeta{
		Reads:       make([]reflect.Type, 0),
		Writes:      make([]reflect.Type, 0),
		ResReads:    make([]reflect.Type, 0),
		ResWrites:   make([]reflect.Type, 0),
		EventReads:  make([]reflect.Type, 0),
		EventWrites: make([]reflect.Type, 0),
	}
}

// AccessRead adds a component read access.
func AccessRead[T any](acc *AccessMeta) {
	typ := baseType(reflect.TypeOf((*T)(nil)).Elem())
	acc.Reads = append(acc.Reads, typ)
}

// AccessWrite adds a component write access.
func AccessWrite[T any](acc *AccessMeta) {
	typ := baseType(reflect.TypeOf((*T)(nil)).Elem())
	acc.Writes = append(acc.Writes, typ)
}

// AccessResRead adds a resource read access.
func AccessResRead[T any](acc *AccessMeta) {
	typ := baseType(reflect.TypeOf((*T)(nil)).Elem())
	acc.ResReads = append(acc.ResReads, typ)
}

// AccessResWrite adds a resource write access.
func AccessResWrite[T any](acc *AccessMeta) {
	typ := baseType(reflect.TypeOf((*T)(nil)).Elem())
	acc.ResWrites = append(acc.ResWrites, typ)
}

// AccessEventRead adds an event read access.
func AccessEventRead[E any](acc *AccessMeta) {
	typ := reflect.TypeOf((*E)(nil)).Elem()
	acc.EventReads = append(acc.EventReads, typ)
}

// AccessEventWrite adds an event write access.
func AccessEventWrite[E any](acc *AccessMeta) {
	typ := reflect.TypeOf((*E)(nil)).Elem()
	acc.EventWrites = append(acc.EventWrites, typ)
}

// MergeAccess merges src into dst.
func MergeAccess(dst, src *AccessMeta) {
	dst.Reads = append(dst.Reads, src.Reads...)
	dst.Writes = append(dst.Writes, src.Writes...)
	dst.ResReads = append(dst.ResReads, src.ResReads...)
	dst.ResWrites = append(dst.ResWrites, src.ResWrites...)
	dst.EventReads = append(dst.EventReads, src.EventReads...)
	dst.EventWrites = append(dst.EventWrites, src.EventWrites...)
}

func (a AccessMeta) toInternal() scheduler.AccessMeta {
	return scheduler.AccessMeta{
		Reads:       a.Reads,
		Writes:      a.Writes,
		ResReads:    a.ResReads,
		ResWrites:   a.ResWrites,
		EventReads:  a.EventReads,
		EventWrites: a.EventWrites,
	}
}

// SystemMeta describes system scheduling metadata.
type SystemMeta struct {
	Access AccessMeta
	Set    string
	Before []string
	After  []string
	Every  time.Duration
}

func (a SystemMeta) toInternal() scheduler.SystemMeta {
	return scheduler.SystemMeta{
		Access: a.Access.toInternal(),
		Set:    a.Set,
		Before: a.Before,
		After:  a.After,
		Every:  a.Every,
	}
}

// baseType returns the non-pointer base reflect.Type and is the canonical helper for this package.
func baseType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}
