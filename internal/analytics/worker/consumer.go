// Package worker implements the Kafka-consuming analytics worker that writes check events to Postgres.
package worker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/kafka-go"

	"sentinel/internal/analytics/event"
)

var (
	pgUnavailable = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "sentinel_worker_pg_unavailable_total",
		Help: "Total number of times Postgres was unavailable during analytics write.",
	})
	eventsConsumed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "sentinel_worker_events_consumed_total",
		Help: "Total number of check events consumed by the analytics worker.",
	})
	eventsWritten = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "sentinel_worker_events_written_total",
		Help: "Total number of check events successfully written to Postgres.",
	})
)

func init() {
	prometheus.MustRegister(pgUnavailable, eventsConsumed, eventsWritten)
}

// Consumer reads check events from Kafka and writes them to Postgres.
type Consumer struct {
	reader *kafka.Reader
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewConsumer creates a Kafka consumer that writes analytics events to the given pool.
func NewConsumer(brokers []string, pool *pgxpool.Pool) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       event.TopicCheckEvents,
		GroupID:     "sentinel-analytics-workers",
		StartOffset: kafka.FirstOffset,
		MinBytes:    1,
		MaxBytes:    10e6,
	})

	return &Consumer{
		reader: reader,
		pool:   pool,
		logger: slog.With("component", "analytics-worker"),
	}
}

// Run starts the consumer loop, fetching, decoding, writing, and committing messages.
func (c *Consumer) Run(ctx context.Context) error {
	c.logger.Info("starting consumer loop",
		"topic", event.TopicCheckEvents,
		"group", "sentinel-analytics-workers",
	)

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("fetch message: %w", err)
		}

		eventsConsumed.Inc()

		evt, err := event.Decode(msg.Value)
		if err != nil {
			c.logger.Error("failed to decode event", "error", err, "offset", msg.Offset)
			if cErr := c.reader.CommitMessages(ctx, msg); cErr != nil {
				c.logger.Error("failed to commit after decode error", "error", cErr)
			}
			continue
		}

		if err := Write(ctx, c.pool, evt); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("write event: %w", err)
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("commit offset: %w", err)
		}

		eventsWritten.Inc()
	}
}

// Close shuts down the underlying Kafka reader.
func (c *Consumer) Close() error {
	return c.reader.Close()
}
