package service

import (
	"context"
	"encoding/json"
	"errors"
	"lumen/internal/domain"

	"github.com/google/uuid"
)

type ChatRepository interface {
	Create(ctx context.Context, message *domain.Message) (*domain.Message, error)
	ListByChannel(ctx context.Context, channelID uint, beforeID *uint, limit int) ([]domain.Message, *uint, error)
}

type ChatBroadcaster interface {
	Broadcast(event any) error
}

type ChatService struct {
	repo ChatRepository
	hub  ChatBroadcaster
}

type AuthorDTO struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Avatar   string    `json:"avatar"`
}

type MessagePayload struct {
	ID        uint      `json:"id"`
	ChannelID uint      `json:"channel_id"`
	Content   string    `json:"content"`
	Author    AuthorDTO `json:"author"`
}

type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type IncomingEvent struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type IncomingMessageCreatePayload struct {
	ChannelID uint   `json:"channel_id"`
	Content   string `json:"content"`
}

type ListMessagesResult struct {
	Messages   []MessagePayload `json:"messages"`
	NextCursor *uint            `json:"next_cursor,omitempty"`
}

var ErrUnsupportedEventType = errors.New("unsupported event type")
var ErrInvalidMessagePayload = errors.New("invalid message payload")

func NewChatService(repo ChatRepository, hub ChatBroadcaster) *ChatService {
	return &ChatService{repo: repo, hub: hub}
}

func (s *ChatService) HandleIncomingEvent(ctx context.Context, authorID uuid.UUID, raw []byte) error {
	var incoming IncomingEvent
	if err := json.Unmarshal(raw, &incoming); err != nil {
		return ErrInvalidMessagePayload
	}

	switch incoming.Type {
	case "message_create":
		var payload IncomingMessageCreatePayload
		if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
			return ErrInvalidMessagePayload
		}

		_, err := s.CreateMessage(ctx, authorID, payload.ChannelID, payload.Content)
		return err
	default:
		return ErrUnsupportedEventType
	}
}

func (s *ChatService) CreateMessage(
	ctx context.Context,
	authorID uuid.UUID,
	channelID uint,
	content string,
) (*MessagePayload, error) {
	if channelID == 0 || content == "" {
		return nil, ErrInvalidMessagePayload
	}

	message, err := s.repo.Create(ctx, &domain.Message{
		Content:   content,
		UserID:    authorID,
		ChannelID: channelID,
	})
	if err != nil {
		return nil, err
	}

	payload := toMessagePayload(*message)
	if err := s.hub.Broadcast(Event{
		Type:    "message_create",
		Payload: payload,
	}); err != nil {
		return nil, err
	}

	return &payload, nil
}

func (s *ChatService) ListMessages(
	ctx context.Context,
	channelID uint,
	beforeID *uint,
	limit int,
) (*ListMessagesResult, error) {
	messages, nextCursor, err := s.repo.ListByChannel(ctx, channelID, beforeID, limit)
	if err != nil {
		return nil, err
	}

	result := make([]MessagePayload, 0, len(messages))
	for _, message := range messages {
		result = append(result, toMessagePayload(message))
	}

	return &ListMessagesResult{
		Messages:   result,
		NextCursor: nextCursor,
	}, nil
}

func toMessagePayload(message domain.Message) MessagePayload {
	return MessagePayload{
		ID:        message.ID,
		ChannelID: message.ChannelID,
		Content:   message.Content,
		Author: AuthorDTO{
			ID:       message.User.ID,
			Username: message.User.Username,
			Avatar:   "",
		},
	}
}
