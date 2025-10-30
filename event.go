package bevi

import (
	"context"

	"github.com/oriumgames/bevi/internal/event"
)

// EventBus is the public alias for the internal events.Bus.
type EventBus = event.Bus

// EventWriter is the public alias for the internal events.Writer[T].
type EventWriter[T any] = event.Writer[T]

// EventReader is the public alias for the internal events.Reader[T].
type EventReader[T any] = event.Reader[T]

// EventResult is the public alias for the internal events.EventResult[T].
type EventResult[T any] = event.EventResult[T]

// NewEventBus constructs a new event bus.
func NewEventBus() *EventBus {
	return event.NewBus()
}

// WriterFor returns a typed EventWriter bound to the given bus.
func WriterFor[T any](bus *EventBus) EventWriter[T] {
	return event.WriterFor[T](bus)
}

// ReaderFor returns a typed EventReader bound to the given bus.
func ReaderFor[T any](bus *EventBus) EventReader[T] {
	return event.ReaderFor[T](bus)
}

// WithEventBus attaches the EventBus to the provided context.
func WithEventBus(parent context.Context, bus *EventBus) context.Context {
	return context.WithValue(parent, eventBusCtxKey{}, bus)
}

// EventBusFrom extracts the EventBus from the context if present, or nil.
func EventBusFrom(ctx context.Context) *EventBus {
	if v := ctx.Value(eventBusCtxKey{}); v != nil {
		if b, ok := v.(*event.Bus); ok {
			return b
		}
	}
	return nil
}

// WriterFromContext fetches a typed EventWriter from context if a bus is present.
// Returns a zero-value writer if none is found.
func WriterFromContext[T any](ctx context.Context) EventWriter[T] {
	if bus := EventBusFrom(ctx); bus != nil {
		return WriterFor[T](bus)
	}
	var zero EventWriter[T]
	return zero
}

// ReaderFromContext fetches a typed EventReader from context if a bus is present.
// Returns a zero-value reader if none is found.
func ReaderFromContext[T any](ctx context.Context) EventReader[T] {
	if bus := EventBusFrom(ctx); bus != nil {
		return ReaderFor[T](bus)
	}
	var zero EventReader[T]
	return zero
}

type eventBusCtxKey struct{}
