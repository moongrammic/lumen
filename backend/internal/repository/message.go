package repository

import (
	"context"
	"lumen/internal/domain"

	"gorm.io/gorm"
)

type MessageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Create(ctx context.Context, message *domain.Message) (*domain.Message, error) {
	if err := r.db.WithContext(ctx).Create(message).Error; err != nil {
		return nil, err
	}

	var created domain.Message
	if err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Attachments").
		First(&created, message.ID).Error; err != nil {
		return nil, err
	}

	return &created, nil
}

func (r *MessageRepository) ListByChannel(
	ctx context.Context,
	channelID uint,
	beforeID *uint,
	limit int,
) ([]domain.Message, *uint, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := r.db.WithContext(ctx).
		Preload("User").
		Preload("Attachments").
		Where("channel_id = ?", channelID).
		Order("id DESC").
		Limit(limit + 1)

	if beforeID != nil {
		query = query.Where("id < ?", *beforeID)
	}

	var messages []domain.Message
	if err := query.Find(&messages).Error; err != nil {
		return nil, nil, err
	}

	var nextCursor *uint
	if len(messages) > limit {
		cursor := messages[limit-1].ID
		nextCursor = &cursor
		messages = messages[:limit]
	}

	return messages, nextCursor, nil
}
