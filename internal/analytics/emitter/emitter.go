package emitter

import "sentinel/internal/analytics/event"

type EventEmitter interface {
	Emit(evt event.CheckEvent)
}

type NoopEmitter struct{}

func (NoopEmitter) Emit(event.CheckEvent) {}
