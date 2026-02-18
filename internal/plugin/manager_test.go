package plugin

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/inelson/finguard/internal/stream"
	pluginpkg "github.com/inelson/finguard/pkg/plugin"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

type mockPlugin struct {
	meta   *pluginpkg.Metadata
	events chan *pluginpkg.Event
}

func newMockPlugin(name string) *mockPlugin {
	return &mockPlugin{
		meta: &pluginpkg.Metadata{
			Name:        name,
			Version:     "0.1.0",
			Description: "test plugin",
			Type:        "cost",
			Topics:      []string{"test.event"},
			Routes: []pluginpkg.Route{
				{Method: "GET", Path: "/test", Description: "test endpoint"},
			},
		},
		events: make(chan *pluginpkg.Event, 10),
	}
}

func (m *mockPlugin) GetMetadata() (*pluginpkg.Metadata, error) {
	return m.meta, nil
}

func (m *mockPlugin) Initialize(_ context.Context, _ *pluginpkg.InitRequest) error {
	return nil
}

func (m *mockPlugin) Execute(_ context.Context, req *pluginpkg.ExecuteRequest) (*pluginpkg.ExecuteResponse, error) {
	data, _ := json.Marshal(map[string]string{"action": req.Action, "status": "ok"})
	return &pluginpkg.ExecuteResponse{
		Data:        data,
		ContentType: "application/json",
		StatusCode:  http.StatusOK,
	}, nil
}

func (m *mockPlugin) StreamEvents(_ context.Context) (<-chan *pluginpkg.Event, error) {
	return m.events, nil
}

func (m *mockPlugin) Shutdown(_ context.Context) error {
	close(m.events)
	return nil
}

func TestManager_Register(t *testing.T) {
	hub := stream.NewHub(testLogger())
	mgr := NewManager(hub, testLogger())

	p := newMockPlugin("test-plugin")
	if err := mgr.Register(p); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	plugins := mgr.ListPlugins()
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].Name != "test-plugin" {
		t.Errorf("expected name 'test-plugin', got %q", plugins[0].Name)
	}
}

func TestManager_RegisterDuplicate(t *testing.T) {
	hub := stream.NewHub(testLogger())
	mgr := NewManager(hub, testLogger())

	p := newMockPlugin("dup")
	mgr.Register(p)

	p2 := newMockPlugin("dup")
	if err := mgr.Register(p2); err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestManager_InitializeAll(t *testing.T) {
	hub := stream.NewHub(testLogger())
	mgr := NewManager(hub, testLogger())

	p := newMockPlugin("init-test")
	mgr.Register(p)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := mgr.InitializeAll(ctx, "http://localhost:9003"); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}
}

func TestManager_EventBridge(t *testing.T) {
	hub := stream.NewHub(testLogger())
	mgr := NewManager(hub, testLogger())

	p := newMockPlugin("event-test")
	mgr.Register(p)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Register a hub client to receive bridged events
	client := hub.Register(ctx, nil)

	mgr.InitializeAll(ctx, "http://localhost:9003")

	// Give the bridge goroutine a moment to start
	time.Sleep(50 * time.Millisecond)

	// Send an event from the plugin
	p.events <- &pluginpkg.Event{
		Type:      "test",
		Topic:     "test.event",
		Timestamp: time.Now(),
		Source:    "event-test",
		Payload:   []byte(`{}`),
	}

	select {
	case msg := <-client.Send():
		if len(msg) == 0 {
			t.Fatal("received empty bridged event")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for bridged event")
	}

	cancel()
	hub.Unregister(client)
}

func TestManager_ShutdownAll(t *testing.T) {
	hub := stream.NewHub(testLogger())
	mgr := NewManager(hub, testLogger())

	p := newMockPlugin("shutdown-test")
	mgr.Register(p)

	ctx := context.Background()
	mgr.InitializeAll(ctx, "http://localhost:9003")
	mgr.ShutdownAll(ctx)

	// After shutdown, the event channel should be closed
	_, ok := <-p.events
	if ok {
		t.Error("expected events channel to be closed after shutdown")
	}
}
