// Package emitter provides interfaces and implementations for publishing check events to analytics pipelines.
package emitter

import "sentinel/internal/analytics/event"

// EventEmitter is the contract for publishing check events asynchronously.
type EventEmitter interface {
	// Emit queues a check event for processing (non-blocking).
	Emit(evt event.CheckEvent)
}

// NoopEmitter is a no-op implementation that discards all events.
type NoopEmitter struct{}

// Emit discards the event.
func (NoopEmitter) Emit(event.CheckEvent) {}
