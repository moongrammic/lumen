package repository

import (
	"context"
	"lumen/internal/domain"

	"gorm.io/gorm"
)

type ChannelRepository struct {
	db *gorm.DB
}

func NewChannelRepository(db *gorm.DB) *ChannelRepository {
	return &ChannelRepository{db: db}
}

func (r *ChannelRepository) Create(ctx context.Context, channel *domain.Channel) (*domain.Channel, error) {
	if err := r.db.WithContext(ctx).Create(channel).Error; err != nil {
		return nil, err
	}
	return channel, nil
}

func (r *ChannelRepository) ListByGuild(ctx context.Context, guildID uint) ([]domain.Channel, error) {
	var channels []domain.Channel
	if err := r.db.WithContext(ctx).
		Where("guild_id = ?", guildID).
		Order("id ASC").
		Find(&channels).Error; err != nil {
		return nil, err
	}
	return channels, nil
}
