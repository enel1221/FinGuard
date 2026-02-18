package event

import (
	"encoding/json"
	"time"
)

type Event struct {
	Type      string          `json:"type"`
	Topic     string          `json:"topic"`
	Timestamp time.Time       `json:"timestamp"`
	Source    string          `json:"source"`
	Payload   json.RawMessage `json:"payload"`
}

func New(eventType, topic, source string, payload any) (*Event, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &Event{
		Type:      eventType,
		Topic:     topic,
		Timestamp: time.Now().UTC(),
		Source:    source,
		Payload:   data,
	}, nil
}

const (
	TopicCostAllocation = "cost.allocation"
	TopicCostIdle       = "cost.idle.detected"
	TopicBudgetWarning  = "budget.warning"
	TopicBudgetExceeded = "budget.exceeded"
	TopicClusterChange  = "cluster.change"
	TopicPluginStatus   = "plugin.status"
	TopicSystem         = "system"
)
