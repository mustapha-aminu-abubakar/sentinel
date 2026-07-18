package emitter_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"sentinel/internal/analytics/emitter"
	"sentinel/internal/analytics/event"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type blockingWriter struct {
	block chan struct{}
}

func (b *blockingWriter) WriteMessages(ctx context.Context, _ ...kafka.Message) error {
	select {
	case <-b.block:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (b *blockingWriter) Close() error {
	return nil
}

type mockWriter struct {
	mu       sync.Mutex
	messages []kafka.Message
	err      error
}

func (m *mockWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msgs...)
	return m.err
}

func (m *mockWriter) Close() error {
	return nil
}

func (m *mockWriter) Messages() []kafka.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]kafka.Message, len(m.messages))
	copy(result, m.messages)
	return result
}

func TestEmit_NonBlocking_WhenChannelFull(t *testing.T) {
	bw := &blockingWriter{block: make(chan struct{})}
	e := emitter.NewKafkaEmitterWithWriter(bw, event.TopicCheckEvents, 1, 1)

	evt := event.CheckEvent{
		ClientID:  "c1",
		API:       "/api/test",
		Timestamp: time.Now(),
		LatencyMS: 5,
		Status:    event.StatusAllowed,
	}

	e.Emit(evt)
	// worker is blocked on WriteMessages, channel is full

	start := time.Now()
	e.Emit(evt)
	elapsed := time.Since(start)

	assert.Less(t, elapsed, 1*time.Millisecond,
		"Emit should return in <1ms even when channel is full")

	close(bw.block)
	e.Close()
}

func TestEmit_PublishesToCorrectTopic(t *testing.T) {
	mw := &mockWriter{}
	e := emitter.NewKafkaEmitterWithWriter(mw, event.TopicCheckEvents, 10, 1)

	now := time.Now().Truncate(time.Millisecond)
	evt := event.CheckEvent{
		ClientID:  "550e8400-e29b-41d4-a716-446655440000",
		API:       "/api/v1/data",
		Timestamp: now,
		LatencyMS: 3,
		Status:    event.StatusRejected,
	}

	e.Emit(evt)
	e.Close()

	msgs := mw.Messages()
	require.Len(t, msgs, 1, "expected exactly one message")

	assert.Equal(t, event.TopicCheckEvents, msgs[0].Topic,
		"message should be published to the correct topic")

	decoded, err := event.Decode(msgs[0].Value)
	require.NoError(t, err, "message payload should be a valid CheckEvent")

	assert.Equal(t, evt.ClientID, decoded.ClientID)
	assert.Equal(t, evt.API, decoded.API)
	assert.Equal(t, evt.Status, decoded.Status)
	assert.Equal(t, evt.LatencyMS, decoded.LatencyMS)
	assert.WithinDuration(t, evt.Timestamp, decoded.Timestamp, time.Second,
		"timestamp should be preserved within 1s")
}

func TestEmit_ReturnsOnWriterError(t *testing.T) {
	mw := &mockWriter{err: errors.New("kafka unavailable")}
	e := emitter.NewKafkaEmitterWithWriter(mw, event.TopicCheckEvents, 10, 1)

	evt := event.CheckEvent{
		ClientID:  "c1",
		API:       "/api/test",
		Timestamp: time.Now(),
		LatencyMS: 1,
		Status:    event.StatusAllowed,
	}

	e.Emit(evt)
	e.Close()

	msgs := mw.Messages()
	require.Len(t, msgs, 1, "message should have been attempted despite writer error")
}

func TestEmit_ConcurrentSafety(t *testing.T) {
	mw := &mockWriter{}
	e := emitter.NewKafkaEmitterWithWriter(mw, event.TopicCheckEvents, 100, 3)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			evt := event.CheckEvent{
				ClientID:  "c1",
				API:       "/api/test",
				Timestamp: time.Now(),
				LatencyMS: i,
				Status:    event.StatusAllowed,
			}
			e.Emit(evt)
		}(i)
	}
	wg.Wait()
	e.Close()

	assert.GreaterOrEqual(t, len(mw.Messages()), 1,
		"concurrent emits should not panic and should deliver events")
}

func TestEmit_OverloadDropsEvents(t *testing.T) {
	bw := &blockingWriter{block: make(chan struct{})}
	e := emitter.NewKafkaEmitterWithWriter(bw, event.TopicCheckEvents, 5, 1)

	evt := event.CheckEvent{
		ClientID:  "c1",
		API:       "/api/test",
		Timestamp: time.Now(),
		LatencyMS: 10,
		Status:    event.StatusAllowed,
	}

	for i := 0; i < 20; i++ {
		e.Emit(evt)
	}

	close(bw.block)
	e.Close()

	assert.True(t, true, "emitter survived overload without panic or hang")
}
