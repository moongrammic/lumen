package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"lumen/internal/config"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/redis/go-redis/v9"
)

type Hub struct {
	ctx        context.Context
	clients    map[*websocket.Conn]bool
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	broadcast  chan []byte
	incoming   chan []byte
	redis      *redis.Client
	channel    string
}

func NewHub(cfg config.RedisConfig) *Hub {
	return &Hub{
		ctx:        context.Background(),
		clients:    make(map[*websocket.Conn]bool),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		broadcast:  make(chan []byte),
		incoming:   make(chan []byte),
		redis:      redis.NewClient(&redis.Options{Addr: cfg.Addr, Password: cfg.Password, DB: cfg.DB}),
		channel:    cfg.Channel,
	}
}

func (h *Hub) Run() {
	go h.consumeRedisMessages()

	for {
		select {
		case conn := <-h.register:
			h.clients[conn] = true
		case conn := <-h.unregister:
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				_ = conn.Close()
			}
		case message := <-h.broadcast:
			if err := h.redis.Publish(h.ctx, h.channel, message).Err(); err != nil {
				slog.Error("redis publish failed", "channel", h.channel, "error", err)
				h.deliver(message)
			}
		case message := <-h.incoming:
			h.deliver(message)
		}
	}
}

func (h *Hub) consumeRedisMessages() {
	pubsub := h.redis.Subscribe(h.ctx, h.channel)
	defer func() {
		_ = pubsub.Close()
	}()

	for msg := range pubsub.Channel() {
		h.incoming <- []byte(msg.Payload)
	}
}

func (h *Hub) deliver(message []byte) {
	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			slog.Error("ws write failed", "error", err)
			delete(h.clients, conn)
			_ = conn.Close()
		}
	}
}

func (h *Hub) HandleConnection(conn *websocket.Conn) {
	h.Register(conn)
}

func (h *Hub) Register(conn *websocket.Conn) {
	h.register <- conn
}

func (h *Hub) Unregister(conn *websocket.Conn) {
	h.unregister <- conn
}

func (h *Hub) Broadcast(event any) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	select {
	case h.broadcast <- data:
		return nil
	default:
		return errors.New("hub broadcast queue is full")
	}
}

func (h *Hub) SetPresence(ctx context.Context, userID string, status string, ttl time.Duration) error {
	key := "presence:user:" + userID
	return h.redis.Set(ctx, key, status, ttl).Err()
}
