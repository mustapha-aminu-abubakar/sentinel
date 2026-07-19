package emitter

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"sentinel/internal/analytics/event"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/kafka-go"
)

var (
	emitterDropped = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "sentinel_emitter_dropped_total",
		Help: "Total number of check events dropped due to full channel.",
	})
	emitterWriteErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "sentinel_emitter_write_errors_total",
		Help: "Total number of Kafka write errors.",
	})
)

func init() {
	prometheus.MustRegister(emitterDropped, emitterWriteErrors)
}

// kafkaWriter abstracts the Kafka writer for testability.
type kafkaWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

// KafkaEmitter emits check events to a Kafka topic with buffered, async writes.
type KafkaEmitter struct {
	ch        chan event.CheckEvent
	writer    kafkaWriter
	topic     string
	workers   sync.WaitGroup
	closeOnce sync.Once
	cancel    context.CancelFunc
}

// NewKafkaEmitter creates a KafkaEmitter connecting to the given brokers and topic.
func NewKafkaEmitter(brokers []string, topic string, channelSize, workerCount int) *KafkaEmitter {
	w := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Balancer: &kafka.LeastBytes{},
		Async:    true,
	}
	return NewKafkaEmitterWithWriter(w, topic, channelSize, workerCount)
}

// NewKafkaEmitterWithWriter creates a KafkaEmitter with a caller-provided writer (useful for tests).
func NewKafkaEmitterWithWriter(w kafkaWriter, topic string, channelSize, workerCount int) *KafkaEmitter {
	if channelSize <= 0 {
		channelSize = 10000
	}
	if workerCount <= 0 {
		workerCount = 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	e := &KafkaEmitter{
		ch:     make(chan event.CheckEvent, channelSize),
		writer: w,
		topic:  topic,
		cancel: cancel,
	}

	for i := 0; i < workerCount; i++ {
		e.workers.Add(1)
		go e.worker(ctx)
	}

	return e
}

// Emit queues a check event; drops if the channel is full.
func (e *KafkaEmitter) Emit(evt event.CheckEvent) {
	select {
	case e.ch <- evt:
	default:
		emitterDropped.Inc()
	}
}

// Close drains the channel and shuts down worker goroutines with a 5-second timeout.
func (e *KafkaEmitter) Close() {
	e.closeOnce.Do(func() {
		close(e.ch)

		done := make(chan struct{})
		go func() {
			e.workers.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			e.cancel()
			slog.Warn("kafka emitter: worker shutdown timed out, forcing close")
		}

		if err := e.writer.Close(); err != nil {
			slog.Error("kafka emitter: failed to close writer", "error", err)
		}
	})
}

// worker serialises events from the channel and writes them to Kafka.
func (e *KafkaEmitter) worker(ctx context.Context) {
	defer e.workers.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-e.ch:
			if !ok {
				return
			}

			data, err := event.Encode(evt)
			if err != nil {
				slog.Error("kafka emitter: failed to encode event", "error", err)
				continue
			}

			msg := kafka.Message{
				Topic: e.topic,
				Value: data,
			}

			if err := e.writer.WriteMessages(ctx, msg); err != nil {
				if ctx.Err() != nil {
					return
				}
				emitterWriteErrors.Inc()
				slog.Warn("kafka emitter: write error", "error", err)
			}
		}
	}
}
