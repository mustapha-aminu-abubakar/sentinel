package event

import (
	"encoding/json"
	"testing"
	"time"
)

func fixture() CheckEvent {
	return CheckEvent{
		ClientID:  "client_123",
		API:       "openai",
		Timestamp: time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC),
		LatencyMS: 120,
		Status:    StatusAllowed,
	}
}

func briefExampleJSON() []byte {
	return []byte(`{
		"client_id": "client_123",
		"api": "openai",
		"timestamp": "2026-07-18T12:00:00Z",
		"latency": 120,
		"status": "allowed"
	}`)
}

func TestCheckEventWireShape(t *testing.T) {
	evt := fixture()
	data, err := Encode(evt)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	wanted := []string{"client_id", "api", "timestamp", "latency", "status"}
	for _, key := range wanted {
		if _, ok := m[key]; !ok {
			t.Errorf("missing key %q in encoded JSON", key)
		}
	}

	if len(m) != len(wanted) {
		t.Errorf("expected %d keys, got %d: %v", len(wanted), len(m), m)
	}
}

func TestDecodeBriefExample(t *testing.T) {
	evt, err := Decode(briefExampleJSON())
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if evt.ClientID != "client_123" {
		t.Errorf("ClientID = %q, want %q", evt.ClientID, "client_123")
	}
	if evt.API != "openai" {
		t.Errorf("API = %q, want %q", evt.API, "openai")
	}
	if evt.LatencyMS != 120 {
		t.Errorf("LatencyMS = %d, want %d", evt.LatencyMS, 120)
	}
	if evt.Status != StatusAllowed {
		t.Errorf("Status = %q, want %q", evt.Status, StatusAllowed)
	}
}

func TestRoundTripLossless(t *testing.T) {
	original := fixture()
	data, err := Encode(original)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoded, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if decoded.ClientID != original.ClientID {
		t.Errorf("ClientID: got %q, want %q", decoded.ClientID, original.ClientID)
	}
	if decoded.API != original.API {
		t.Errorf("API: got %q, want %q", decoded.API, original.API)
	}
	if !decoded.Timestamp.Equal(original.Timestamp) {
		t.Errorf("Timestamp: got %v, want %v", decoded.Timestamp, original.Timestamp)
	}
	if decoded.LatencyMS != original.LatencyMS {
		t.Errorf("LatencyMS: got %d, want %d", decoded.LatencyMS, original.LatencyMS)
	}
	if decoded.Status != original.Status {
		t.Errorf("Status: got %q, want %q", decoded.Status, original.Status)
	}
}

func TestDecodeRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{"empty object", `{}`},
		{"missing client_id", `{"api":"openai","timestamp":"2026-07-18T12:00:00Z","latency":120,"status":"allowed"}`},
		{"missing api", `{"client_id":"c1","timestamp":"2026-07-18T12:00:00Z","latency":120,"status":"allowed"}`},
		{"missing status", `{"client_id":"c1","api":"openai","timestamp":"2026-07-18T12:00:00Z","latency":120}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decode([]byte(tt.json))
			if err == nil {
				t.Error("expected error for missing field, got nil")
			}
		})
	}
}

func TestStatusConstants(t *testing.T) {
	if StatusAllowed != "allowed" {
		t.Errorf("StatusAllowed = %q, want %q", StatusAllowed, "allowed")
	}
	if StatusRejected != "rejected" {
		t.Errorf("StatusRejected = %q, want %q", StatusRejected, "rejected")
	}
	if TopicCheckEvents != "sentinel.check_events" {
		t.Errorf("TopicCheckEvents = %q, want %q", TopicCheckEvents, "sentinel.check_events")
	}
}
