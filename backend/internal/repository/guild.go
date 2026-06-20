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
			Permissions: domain.PermSendMessages |
				domain.PermViewChannel |
				domain.PermManageChannels |
				domain.PermManageGuild,
		}
		if err := tx.Create(&member).Error; err != nil {
			return err
		}
		return tx.Create(&domain.Channel{
			Name:    "general",
			GuildID: guild.ID,
			Type:    "text",
		}).Error
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
		Permissions: domain.PermSendMessages |
			domain.PermViewChannel,
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

func (r *GuildRepository) GetMemberPermissions(ctx context.Context, guildID uint, userID uuid.UUID) (uint64, error) {
	var member domain.GuildMember
	if err := r.db.WithContext(ctx).
		First(&member, "guild_id = ? AND user_id = ?", guildID, userID).Error; err != nil {
		return 0, err
	}
	return member.Permissions, nil
}

func (r *GuildRepository) GetChannelGuildID(ctx context.Context, channelID uint) (uint, error) {
	var channel domain.Channel
	if err := r.db.WithContext(ctx).First(&channel, "id = ?", channelID).Error; err != nil {
		return 0, err
	}
	return channel.GuildID, nil
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

func (r *GuildRepository) GetUserGuilds(ctx context.Context, userID uuid.UUID) ([]*domain.Guild, error) {
	var guilds []*domain.Guild
	if err := r.db.WithContext(ctx).
		Joins("JOIN guild_members ON guild_members.guild_id = guilds.id").
		Where("guild_members.user_id = ?", userID).
		Find(&guilds).Error; err != nil {
		return nil, err
	}
	return guilds, nil
}

func (r *GuildRepository) GetByID(ctx context.Context, guildID uint) (*domain.Guild, error) {
	var guild domain.Guild
	if err := r.db.WithContext(ctx).First(&guild, "id = ?", guildID).Error; err != nil {
		return nil, err
	}
	return &guild, nil
}

func (r *GuildRepository) UpdateFields(ctx context.Context, guildID uint, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&domain.Guild{}).
		Where("id = ?", guildID).
		Updates(fields).Error
}

func (r *GuildRepository) Delete(ctx context.Context, guildID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("guild_id = ?", guildID).Delete(&domain.Channel{}).Error; err != nil {
			return err
		}
		if err := tx.Where("guild_id = ?", guildID).Delete(&domain.GuildMember{}).Error; err != nil {
			return err
		}
		return tx.Delete(&domain.Guild{}, guildID).Error
	})
}

func (r *GuildRepository) RemoveMember(ctx context.Context, guildID uint, userID uuid.UUID) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("guild_id = ? AND user_id = ?", guildID, userID).
		Delete(&domain.GuildMember{})
	return res.RowsAffected, res.Error
}

type GuildMemberRow struct {
	UserID      uuid.UUID
	Username    string
	Role        string
	Permissions uint64
}

func (r *GuildRepository) ListMembers(ctx context.Context, guildID uint) ([]GuildMemberRow, error) {
	var rows []GuildMemberRow
	if err := r.db.WithContext(ctx).
		Table("guild_members").
		Select("guild_members.user_id, users.username, guild_members.role, guild_members.permissions").
		Joins("JOIN users ON users.id = guild_members.user_id").
		Where("guild_members.guild_id = ? AND guild_members.deleted_at IS NULL AND users.deleted_at IS NULL", guildID).
		Order("guild_members.created_at ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
