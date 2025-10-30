package event

import (
	"reflect"
	"sync"
)

// Bus is a high-performance, per-type event system with frame-based delivery.
type Bus struct {
	stores sync.Map // key: reflect.Type, value: *store[T]
}

// NewBus constructs a Bus.
func NewBus() *Bus {
	return &Bus{}
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

// CompleteNoReader closes completion signals for events with no pending readers.
// Call once after a frame's systems have run and before Advance().
func (b *Bus) CompleteNoReader() {
	b.stores.Range(func(_, v any) bool {
		if cmp, ok := v.(completer); ok {
			cmp.completeNoReader()
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
type completer interface{ completeNoReader() }

// ensureStore fetches or creates the per-type store for T.
func ensureStore[T any](b *Bus) *store[T] {
	t := baseType(reflect.TypeOf((*T)(nil)).Elem())

	if v, ok := b.stores.Load(t); ok {
		return v.(*store[T])
	}
	st := &store[T]{}
	actual, _ := b.stores.LoadOrStore(t, st)
	return actual.(*store[T])
}

func baseType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}
