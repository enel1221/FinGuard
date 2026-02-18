package stream

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/inelson/finguard/pkg/event"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestHub_RegisterUnregister(t *testing.T) {
	hub := NewHub(testLogger())

	if hub.Len() != 0 {
		t.Fatalf("expected 0 clients, got %d", hub.Len())
	}

	ctx := context.Background()
	c := hub.Register(ctx, nil)

	if hub.Len() != 1 {
		t.Fatalf("expected 1 client, got %d", hub.Len())
	}

	hub.Unregister(c)

	if hub.Len() != 0 {
		t.Fatalf("expected 0 clients after unregister, got %d", hub.Len())
	}
}

func TestHub_PublishToSubscribedClient(t *testing.T) {
	hub := NewHub(testLogger())
	ctx := context.Background()
	c := hub.Register(ctx, nil)
	defer hub.Unregister(c)

	c.Subscribe("test.topic")

	evt := &event.Event{
		Type:      "test",
		Topic:     "test.topic",
		Timestamp: time.Now(),
		Source:    "test",
		Payload:   []byte(`{"msg":"hello"}`),
	}

	hub.Publish(evt)

	select {
	case msg := <-c.Send():
		if len(msg) == 0 {
			t.Fatal("received empty message")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestHub_PublishFiltersByTopic(t *testing.T) {
	hub := NewHub(testLogger())
	ctx := context.Background()
	c := hub.Register(ctx, nil)
	defer hub.Unregister(c)

	c.Subscribe("wanted.topic")

	evt := &event.Event{
		Type:  "test",
		Topic: "unwanted.topic",
	}
	hub.Publish(evt)

	select {
	case <-c.Send():
		t.Fatal("should not receive message for unsubscribed topic")
	case <-time.After(100 * time.Millisecond):
		// correct: no message received
	}
}

func TestHub_BroadcastReachesAllClients(t *testing.T) {
	hub := NewHub(testLogger())
	ctx := context.Background()

	c1 := hub.Register(ctx, nil)
	c2 := hub.Register(ctx, nil)
	defer hub.Unregister(c1)
	defer hub.Unregister(c2)

	c1.Subscribe("specific.topic")
	// c2 has no subscriptions (receives all)

	evt := &event.Event{
		Type:  "broadcast",
		Topic: "system",
	}
	hub.Broadcast(evt)

	for _, c := range []*Client{c1, c2} {
		select {
		case msg := <-c.Send():
			if len(msg) == 0 {
				t.Fatal("received empty broadcast")
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for broadcast")
		}
	}
}

func TestClient_NoFilterReceivesAll(t *testing.T) {
	hub := NewHub(testLogger())
	ctx := context.Background()
	c := hub.Register(ctx, nil)
	defer hub.Unregister(c)

	// No subscriptions = receive all topics
	evt := &event.Event{
		Type:  "test",
		Topic: "any.topic",
	}
	hub.Publish(evt)

	select {
	case msg := <-c.Send():
		if len(msg) == 0 {
			t.Fatal("received empty message")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestHub_DoubleUnregisterNoPanic(t *testing.T) {
	hub := NewHub(testLogger())
	ctx := context.Background()
	c := hub.Register(ctx, nil)

	hub.Unregister(c)
	// Second call must not panic
	hub.Unregister(c)

	if hub.Len() != 0 {
		t.Errorf("expected 0 clients, got %d", hub.Len())
	}
}
