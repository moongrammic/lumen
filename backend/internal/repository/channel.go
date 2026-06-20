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

func (r *ChannelRepository) GetByID(ctx context.Context, channelID uint) (*domain.Channel, error) {
	var channel domain.Channel
	if err := r.db.WithContext(ctx).First(&channel, "id = ?", channelID).Error; err != nil {
		return nil, err
	}
	return &channel, nil
}

func (r *ChannelRepository) UpdateFields(ctx context.Context, channelID uint, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&domain.Channel{}).
		Where("id = ?", channelID).
		Updates(fields).Error
}

func (r *ChannelRepository) Delete(ctx context.Context, channelID uint) error {
	return r.db.WithContext(ctx).Delete(&domain.Channel{}, channelID).Error
}
