package stream

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/coder/websocket"

	"github.com/inelson/finguard/pkg/event"
)

type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
	logger  *slog.Logger
}

type Client struct {
	conn   *websocket.Conn
	topics map[string]struct{} // empty map means all topics
	send   chan []byte
	ctx    context.Context
	cancel context.CancelFunc
	closed sync.Once
}

func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		clients: make(map[*Client]struct{}),
		logger:  logger,
	}
}

func (h *Hub) Register(ctx context.Context, conn *websocket.Conn) *Client {
	cctx, cancel := context.WithCancel(ctx)
	c := &Client{
		conn:   conn,
		topics: make(map[string]struct{}),
		send:   make(chan []byte, 256),
		ctx:    cctx,
		cancel: cancel,
	}
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()

	h.logger.Info("client connected", "total", h.Len())
	return c
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	_, existed := h.clients[c]
	delete(h.clients, c)
	h.mu.Unlock()
	if !existed {
		return
	}
	c.cancel()
	c.closed.Do(func() { close(c.send) })
	h.logger.Info("client disconnected", "total", h.Len())
}

func (h *Hub) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) Publish(e *event.Event) {
	data, err := json.Marshal(e)
	if err != nil {
		h.logger.Error("failed to marshal event", "error", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		if !c.subscribedTo(e.Topic) {
			continue
		}
		select {
		case c.send <- data:
		default:
			h.logger.Warn("client send buffer full, dropping event", "topic", e.Topic)
		}
	}
}

func (h *Hub) Broadcast(e *event.Event) {
	data, err := json.Marshal(e)
	if err != nil {
		h.logger.Error("failed to marshal event", "error", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		select {
		case c.send <- data:
		default:
			h.logger.Warn("client send buffer full, dropping broadcast")
		}
	}
}

func (c *Client) Subscribe(topics ...string) {
	for _, t := range topics {
		c.topics[t] = struct{}{}
	}
}

func (c *Client) subscribedTo(topic string) bool {
	if len(c.topics) == 0 {
		return true // no filter = all topics
	}
	_, ok := c.topics[topic]
	return ok
}

func (c *Client) Send() <-chan []byte {
	return c.send
}

func (c *Client) Context() context.Context {
	return c.ctx
}

func (c *Client) Conn() *websocket.Conn {
	return c.conn
}
