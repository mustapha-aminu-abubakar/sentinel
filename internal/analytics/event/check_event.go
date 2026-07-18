package event

import (
	"encoding/json"
	"errors"
	"time"
)

const TopicCheckEvents = "sentinel.check_events"

const (
	StatusAllowed  = "allowed"
	StatusRejected = "rejected"
)

type CheckEvent struct {
	ClientID  string    `json:"client_id"`
	API       string    `json:"api"`
	Timestamp time.Time `json:"timestamp"`
	LatencyMS int       `json:"latency"`
	Status    string    `json:"status"`
}

func Encode(evt CheckEvent) ([]byte, error) {
	return json.Marshal(evt)
}

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
