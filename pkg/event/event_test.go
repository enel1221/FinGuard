package event

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	payload := map[string]string{"key": "value"}
	evt, err := New("test_type", TopicSystem, "test_source", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if evt.Type != "test_type" {
		t.Errorf("expected type 'test_type', got %q", evt.Type)
	}
	if evt.Topic != TopicSystem {
		t.Errorf("expected topic %q, got %q", TopicSystem, evt.Topic)
	}
	if evt.Source != "test_source" {
		t.Errorf("expected source 'test_source', got %q", evt.Source)
	}
	if time.Since(evt.Timestamp) > time.Minute {
		t.Error("timestamp is too old")
	}

	var p map[string]string
	if err := json.Unmarshal(evt.Payload, &p); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if p["key"] != "value" {
		t.Errorf("expected payload key 'value', got %q", p["key"])
	}
}

func TestNew_MarshalError(t *testing.T) {
	_, err := New("test", "topic", "source", make(chan int))
	if err == nil {
		t.Error("expected error for unmarshalable payload")
	}
}

func TestEvent_JSONRoundTrip(t *testing.T) {
	evt, _ := New("type1", TopicCostAllocation, "src", "data")
	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Type != evt.Type {
		t.Errorf("type mismatch: %q vs %q", decoded.Type, evt.Type)
	}
	if decoded.Topic != evt.Topic {
		t.Errorf("topic mismatch: %q vs %q", decoded.Topic, evt.Topic)
	}
}
