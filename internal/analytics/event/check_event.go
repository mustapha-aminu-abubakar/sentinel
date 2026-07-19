// Package event defines the analytics check-event schema and serialization.
package event

import (
	"encoding/json"
	"errors"
	"time"
)

// TopicCheckEvents is the Kafka topic name for check events.
const TopicCheckEvents = "sentinel.check_events"

const (
	// StatusAllowed marks a check that passed rate limits.
	StatusAllowed = "allowed"
	// StatusRejected marks a check that exceeded rate limits.
	StatusRejected = "rejected"
)

// CheckEvent records a single rate-limit check for async analytics processing.
type CheckEvent struct {
	ClientID  string    `json:"client_id"`
	API       string    `json:"api"`
	Timestamp time.Time `json:"timestamp"`
	LatencyMS int       `json:"latency"`
	Status    string    `json:"status"`
}

// Encode serializes a CheckEvent to JSON bytes.
func Encode(evt CheckEvent) ([]byte, error) {
	return json.Marshal(evt)
}

// Decode deserializes JSON bytes into a CheckEvent, validating required fields.
func Decode(data []byte) (CheckEvent, error) {
	var evt CheckEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		return CheckEvent{}, err
	}
	if evt.ClientID == "" || evt.API == "" || evt.Status == "" {
		return CheckEvent{}, errors.New("event: missing required field")
	}
	return evt, nil
}
