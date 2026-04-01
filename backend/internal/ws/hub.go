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
	ctx          context.Context
	clients      map[*websocket.Conn]bool
	connChannels map[*websocket.Conn]map[uint]bool
	channelSubs  map[uint]map[*websocket.Conn]bool
	register     chan *websocket.Conn
	unregister   chan *websocket.Conn
	subscribe    chan subscribeRequest
	unsubscribe  chan subscribeRequest
	broadcast    chan []byte
	incoming     chan []byte
	redis        *redis.Client
	channel      string
}

type subscribeRequest struct {
	conn      *websocket.Conn
	channelID uint
	result    chan error
}

func NewHub(cfg config.RedisConfig) *Hub {
	return &Hub{
		ctx:          context.Background(),
		clients:      make(map[*websocket.Conn]bool),
		connChannels: make(map[*websocket.Conn]map[uint]bool),
		channelSubs:  make(map[uint]map[*websocket.Conn]bool),
		register:     make(chan *websocket.Conn),
		unregister:   make(chan *websocket.Conn),
		subscribe:    make(chan subscribeRequest),
		unsubscribe:  make(chan subscribeRequest),
		broadcast:    make(chan []byte),
		incoming:     make(chan []byte),
		redis:        redis.NewClient(&redis.Options{Addr: cfg.Addr, Password: cfg.Password, DB: cfg.DB}),
		channel:      cfg.Channel,
	}
}

func (h *Hub) Run() {
	go h.consumeRedisMessages()

	for {
		select {
		case conn := <-h.register:
			h.clients[conn] = true
			h.connChannels[conn] = make(map[uint]bool)
		case conn := <-h.unregister:
			if _, ok := h.clients[conn]; ok {
				h.unsubscribeAll(conn)
				delete(h.clients, conn)
				delete(h.connChannels, conn)
				_ = conn.Close()
			}
		case req := <-h.subscribe:
			h.subscribeConn(req.conn, req.channelID)
			req.result <- nil
		case req := <-h.unsubscribe:
			h.unsubscribeConn(req.conn, req.channelID)
			req.result <- nil
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
	targetConns := h.resolveTargets(message)
	for conn := range h.clients {
		if targetConns != nil {
			if _, ok := targetConns[conn]; !ok {
				continue
			}
		}
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			slog.Error("ws write failed", "error", err)
			h.unregister <- conn
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

func (h *Hub) Subscribe(conn *websocket.Conn, channelID uint) error {
	result := make(chan error, 1)
	h.subscribe <- subscribeRequest{conn: conn, channelID: channelID, result: result}
	return <-result
}

func (h *Hub) Unsubscribe(conn *websocket.Conn, channelID uint) error {
	result := make(chan error, 1)
	h.unsubscribe <- subscribeRequest{conn: conn, channelID: channelID, result: result}
	return <-result
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

func (h *Hub) subscribeConn(conn *websocket.Conn, channelID uint) {
	if _, ok := h.channelSubs[channelID]; !ok {
		h.channelSubs[channelID] = make(map[*websocket.Conn]bool)
	}
	h.channelSubs[channelID][conn] = true
	if _, ok := h.connChannels[conn]; !ok {
		h.connChannels[conn] = make(map[uint]bool)
	}
	h.connChannels[conn][channelID] = true
}

func (h *Hub) unsubscribeConn(conn *websocket.Conn, channelID uint) {
	if subs, ok := h.channelSubs[channelID]; ok {
		delete(subs, conn)
		if len(subs) == 0 {
			delete(h.channelSubs, channelID)
		}
	}
	if channels, ok := h.connChannels[conn]; ok {
		delete(channels, channelID)
	}
}

func (h *Hub) unsubscribeAll(conn *websocket.Conn) {
	channels, ok := h.connChannels[conn]
	if !ok {
		return
	}
	for channelID := range channels {
		h.unsubscribeConn(conn, channelID)
	}
}

func (h *Hub) resolveTargets(message []byte) map[*websocket.Conn]bool {
	var envelope struct {
		Event   string                 `json:"event"`
		Payload map[string]interface{} `json:"payload"`
	}
	if err := json.Unmarshal(message, &envelope); err != nil {
		return nil
	}
	if envelope.Event != "MESSAGE_CREATE" && envelope.Event != "TYPING_START" {
		return nil
	}

	rawChannelID, ok := envelope.Payload["channel_id"]
	if !ok {
		return nil
	}
	channelID, ok := toUint(rawChannelID)
	if !ok {
		return nil
	}

	if subs, exists := h.channelSubs[channelID]; exists {
		return subs
	}
	return map[*websocket.Conn]bool{}
}

func toUint(v interface{}) (uint, bool) {
	switch t := v.(type) {
	case float64:
		return uint(t), true
	case int:
		return uint(t), true
	case int64:
		return uint(t), true
	case uint:
		return t, true
	default:
		return 0, false
	}
}
