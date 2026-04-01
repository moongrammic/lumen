package service

import (
	"context"
	"encoding/json"
	"errors"
	"lumen/internal/domain"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChatRepository interface {
	Create(ctx context.Context, message *domain.Message) (*domain.Message, error)
	ListByChannel(ctx context.Context, channelID uint, beforeID *uint, limit int) ([]domain.Message, *uint, error)
}

type ChatBroadcaster interface {
	Broadcast(event any) error
	SetPresence(ctx context.Context, userID string, status string, ttl time.Duration) error
}

type ChatAccessChecker interface {
	IsMember(ctx context.Context, guildID uint, userID uuid.UUID) (bool, error)
	GetMemberPermissions(ctx context.Context, guildID uint, userID uuid.UUID) (uint64, error)
	GetChannelGuildID(ctx context.Context, channelID uint) (uint, error)
}

type ChatService struct {
	repo   ChatRepository
	access ChatAccessChecker
	hub    ChatBroadcaster
}

type AuthorDTO struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Avatar   string    `json:"avatar"`
}

type MessagePayload struct {
	ID          uint      `json:"id"`
	ChannelID   uint      `json:"channel_id"`
	Content     string    `json:"content"`
	Author      AuthorDTO `json:"author"`
	Attachments []string  `json:"attachments"`
}

type Event struct {
	Op      int         `json:"op"`
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
}

type IncomingEvent struct {
	Op      int             `json:"op"`
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

type IncomingMessageCreatePayload struct {
	ChannelID uint   `json:"channel_id"`
	Content   string `json:"content"`
}

type IncomingTypingPayload struct {
	ChannelID uint `json:"channel_id"`
}

type PresencePayload struct {
	UserID string `json:"user_id"`
	Status string `json:"status"`
}

type ListMessagesResult struct {
	Messages   []MessagePayload `json:"messages"`
	NextCursor *uint            `json:"next_cursor,omitempty"`
}

var ErrUnsupportedEventType = errors.New("unsupported event type")
var ErrInvalidMessagePayload = errors.New("invalid message payload")
var ErrChatAccessDenied = errors.New("chat access denied")
var ErrMissingSendPermission = errors.New("missing send messages permission")
var ErrChannelNotFound = errors.New("channel not found")

func NewChatService(repo ChatRepository, access ChatAccessChecker, hub ChatBroadcaster) *ChatService {
	return &ChatService{repo: repo, access: access, hub: hub}
}

func (s *ChatService) HandleIncomingEvent(ctx context.Context, authorID uuid.UUID, raw []byte) error {
	var incoming IncomingEvent
	if err := json.Unmarshal(raw, &incoming); err != nil {
		return ErrInvalidMessagePayload
	}

	switch incoming.Event {
	case "MESSAGE_CREATE":
		var payload IncomingMessageCreatePayload
		if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
			return ErrInvalidMessagePayload
		}

		_, err := s.CreateMessage(ctx, authorID, payload.ChannelID, payload.Content)
		return err
	case "TYPING_START":
		var payload IncomingTypingPayload
		if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
			return ErrInvalidMessagePayload
		}
		return s.BroadcastTyping(ctx, authorID, payload.ChannelID)
	case "PRESENCE_UPDATE":
		var payload PresencePayload
		if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
			return ErrInvalidMessagePayload
		}
		return s.UpdatePresence(ctx, authorID, payload.Status, 60*time.Second)
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
	if err := s.ensureCanSendMessage(ctx, authorID, channelID); err != nil {
		return nil, err
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
		Op:      0,
		Event:   "MESSAGE_CREATE",
		Payload: payload,
	}); err != nil {
		return nil, err
	}

	return &payload, nil
}

func (s *ChatService) BroadcastTyping(ctx context.Context, authorID uuid.UUID, channelID uint) error {
	return s.hub.Broadcast(Event{
		Op:    0,
		Event: "TYPING_START",
		Payload: map[string]interface{}{
			"channel_id": channelID,
			"user_id":    authorID.String(),
		},
	})
}

func (s *ChatService) UpdatePresence(ctx context.Context, authorID uuid.UUID, status string, ttl time.Duration) error {
	if status == "" {
		status = "online"
	}
	if err := s.hub.SetPresence(ctx, authorID.String(), status, ttl); err != nil {
		return err
	}
	return s.hub.Broadcast(Event{
		Op:    0,
		Event: "PRESENCE_UPDATE",
		Payload: PresencePayload{
			UserID: authorID.String(),
			Status: status,
		},
	})
}

func (s *ChatService) ListMessages(
	ctx context.Context,
	userID uuid.UUID,
	channelID uint,
	beforeID *uint,
	limit int,
) (*ListMessagesResult, error) {
	if err := s.ensureCanReadChannel(ctx, userID, channelID); err != nil {
		return nil, err
	}

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

func (s *ChatService) GetRecentMessages(ctx context.Context, userID uuid.UUID, channelID uint) ([]domain.Message, error) {
	if err := s.ensureCanReadChannel(ctx, userID, channelID); err != nil {
		return nil, err
	}

	messages, _, err := s.repo.ListByChannel(ctx, channelID, nil, 50)
	if err != nil {
		return nil, err
	}
	return messages, nil
}

func (s *ChatService) ensureCanReadChannel(ctx context.Context, userID uuid.UUID, channelID uint) error {
	guildID, err := s.access.GetChannelGuildID(ctx, channelID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrChannelNotFound
		}
		return err
	}
	isMember, err := s.access.IsMember(ctx, guildID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrChatAccessDenied
	}
	return nil
}

func (s *ChatService) ensureCanSendMessage(ctx context.Context, userID uuid.UUID, channelID uint) error {
	guildID, err := s.access.GetChannelGuildID(ctx, channelID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrChannelNotFound
		}
		return err
	}
	isMember, err := s.access.IsMember(ctx, guildID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrChatAccessDenied
	}

	perms, err := s.access.GetMemberPermissions(ctx, guildID, userID)
	if err != nil {
		return err
	}
	if perms&domain.PermSendMessages == 0 {
		return ErrMissingSendPermission
	}
	return nil
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
		Attachments: attachmentsToURLs(message.Attachments),
	}
}

func attachmentsToURLs(items []domain.Attachment) []string {
	urls := make([]string, 0, len(items))
	for _, item := range items {
		urls = append(urls, item.URL)
	}
	return urls
}
