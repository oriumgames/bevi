package event

import (
	"reflect"
	"sync"
)

// Diagnostics is the interface for event system diagnostics.
type Diagnostics interface {
	EventEmit(name string, count int)
}

// Bus is a high-performance, per-type event system with frame-based delivery.
type Bus struct {
	stores sync.Map // key: reflect.Type, value: *store[T]
	diag   Diagnostics
}

// NewBus constructs a Bus.
func NewBus() *Bus {
	return &Bus{}
}

// SetDiagnostics sets the diagnostics implementation.
func (b *Bus) SetDiagnostics(d Diagnostics) {
	b.diag = d
	b.stores.Range(func(_, v any) bool {
		if dgn, ok := v.(diagnoser); ok {
			dgn.setDiagnostics(d)
		}
		return true
	})
}

// Advance flips write->read buffers for all event types.
func (b *Bus) Advance() {
	b.stores.Range(func(_, v any) bool {
		if adv, ok := v.(advancer); ok {
			adv.advance()
		}
		return true
	})
}

// WriterFor returns a type-safe writer bound to this bus.
func WriterFor[T any](b *Bus) Writer[T] {
	return Writer[T]{store: ensureStore[T](b)}
}

// ReaderFor returns a type-safe reader bound to this bus.
func ReaderFor[T any](b *Bus) Reader[T] {
	return Reader[T]{store: ensureStore[T](b)}
}

// advancer and completer are implemented by the per-type store to support
// frame advancement and completion handling.
type advancer interface{ advance() }
type diagnoser interface{ setDiagnostics(Diagnostics) }

func (s *store[T]) setDiagnostics(d Diagnostics) {
	s.diag = d
}

// ensureStore fetches or creates the per-type store for T.
func ensureStore[T any](b *Bus) *store[T] {
	t := baseType(reflect.TypeOf((*T)(nil)).Elem())

	if v, ok := b.stores.Load(t); ok {
		return v.(*store[T])
	}
	st := &store[T]{
		name: t.String(),
		diag: b.diag,
	}
	actual, _ := b.stores.LoadOrStore(t, st)
	return actual.(*store[T])
}

func baseType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}
