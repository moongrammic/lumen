package ws

import (
	"context"
	"log"
	"os"

	"github.com/gofiber/contrib/websocket"
	"github.com/redis/go-redis/v9"
)

type Hub struct {
	ctx        context.Context
	clients    map[*websocket.Conn]bool
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	broadcast  chan []byte
	redis      *redis.Client
	channel    string
}

func NewHub() *Hub {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}

	redisChannel := os.Getenv("REDIS_CHANNEL")
	if redisChannel == "" {
		redisChannel = "lumen:chat"
	}

	return &Hub{
		ctx:        context.Background(),
		clients:    make(map[*websocket.Conn]bool),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		broadcast:  make(chan []byte),
		redis:      redis.NewClient(&redis.Options{Addr: redisAddr}),
		channel:    redisChannel,
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
				log.Printf("redis publish error: %v", err)
				h.deliver(message)
			}
		}
	}
}

func (h *Hub) consumeRedisMessages() {
	pubsub := h.redis.Subscribe(h.ctx, h.channel)
	defer func() {
		_ = pubsub.Close()
	}()

	for msg := range pubsub.Channel() {
		h.deliver([]byte(msg.Payload))
	}
}

func (h *Hub) deliver(message []byte) {
	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("ws write error: %v", err)
			delete(h.clients, conn)
			_ = conn.Close()
		}
	}
}

func (h *Hub) HandleConnection(conn *websocket.Conn) {
	h.register <- conn
	defer func() {
		h.unregister <- conn
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		h.broadcast <- msg
	}
}
