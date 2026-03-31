package repository

import (
	"context"
	"lumen/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GuildRepository struct {
	db *gorm.DB
}

func NewGuildRepository(db *gorm.DB) *GuildRepository {
	return &GuildRepository{db: db}
}

func (r *GuildRepository) Create(ctx context.Context, guild *domain.Guild, ownerID uuid.UUID) (*domain.Guild, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(guild).Error; err != nil {
			return err
		}
		member := domain.GuildMember{
			GuildID: guild.ID,
			UserID:  ownerID,
			Role:    "owner",
		}
		return tx.Create(&member).Error
	})
	if err != nil {
		return nil, err
	}
	return guild, nil
}

func (r *GuildRepository) FindByInviteCode(ctx context.Context, inviteCode string) (*domain.Guild, error) {
	var guild domain.Guild
	if err := r.db.WithContext(ctx).First(&guild, "invite_code = ?", inviteCode).Error; err != nil {
		return nil, err
	}
	return &guild, nil
}

func (r *GuildRepository) AddMemberIfNotExists(ctx context.Context, guildID uint, userID uuid.UUID) error {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&domain.GuildMember{}).
		Where("guild_id = ? AND user_id = ?", guildID, userID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	member := domain.GuildMember{
		GuildID: guildID,
		UserID:  userID,
		Role:    "member",
	}
	return r.db.WithContext(ctx).Create(&member).Error
}

func (r *GuildRepository) IsMember(ctx context.Context, guildID uint, userID uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&domain.GuildMember{}).
		Where("guild_id = ? AND user_id = ?", guildID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *GuildRepository) ChannelBelongsToGuild(ctx context.Context, channelID uint, guildID uint) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&domain.Channel{}).
		Where("id = ? AND guild_id = ?", channelID, guildID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
