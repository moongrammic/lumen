package service

import (
	"context"
	"errors"
	"lumen/internal/domain"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChannelRepository interface {
	Create(ctx context.Context, channel *domain.Channel) (*domain.Channel, error)
	ListByGuild(ctx context.Context, guildID uint) ([]domain.Channel, error)
	GetByID(ctx context.Context, channelID uint) (*domain.Channel, error)
	UpdateFields(ctx context.Context, channelID uint, fields map[string]any) error
	Delete(ctx context.Context, channelID uint) error
}

type ChannelAccessChecker interface {
	IsMember(ctx context.Context, guildID uint, userID uuid.UUID) (bool, error)
	GetMemberPermissions(ctx context.Context, guildID uint, userID uuid.UUID) (uint64, error)
}

type ChannelService struct {
	repo   ChannelRepository
	access ChannelAccessChecker
}

type ChannelDTO struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	GuildID uint   `json:"guild_id"`
	Type    string `json:"type"`
}

var ErrChannelAccessDenied = errors.New("channel access denied")
var ErrMissingManageChannels = errors.New("missing manage channels permission")
var ErrChannelNotFoundInGuild = errors.New("channel not found in guild")

func NewChannelService(repo ChannelRepository, access ChannelAccessChecker) *ChannelService {
	return &ChannelService{repo: repo, access: access}
}

func (s *ChannelService) Create(ctx context.Context, guildID uint, userID uuid.UUID, name string, channelType string) (*ChannelDTO, error) {
	if name == "" {
		return nil, errors.New("channel name is required")
	}
	if channelType == "" {
		channelType = "text"
	}

	if err := s.ensureCanManageChannels(ctx, guildID, userID); err != nil {
		return nil, err
	}

	channel, err := s.repo.Create(ctx, &domain.Channel{
		Name:    name,
		GuildID: guildID,
		Type:    channelType,
	})
	if err != nil {
		return nil, err
	}
	return &ChannelDTO{
		ID:      channel.ID,
		Name:    channel.Name,
		GuildID: channel.GuildID,
		Type:    channel.Type,
	}, nil
}

func (s *ChannelService) ListByGuild(ctx context.Context, guildID uint, userID uuid.UUID) ([]ChannelDTO, error) {
	isMember, err := s.access.IsMember(ctx, guildID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrChannelAccessDenied
	}

	channels, err := s.repo.ListByGuild(ctx, guildID)
	if err != nil {
		return nil, err
	}

	result := make([]ChannelDTO, 0, len(channels))
	for _, channel := range channels {
		result = append(result, ChannelDTO{
			ID:      channel.ID,
			Name:    channel.Name,
			GuildID: channel.GuildID,
			Type:    channel.Type,
		})
	}
	return result, nil
}

func (s *ChannelService) UpdateChannel(ctx context.Context, guildID uint, channelID uint, userID uuid.UUID, name *string, channelType *string) (*ChannelDTO, error) {
	if err := s.ensureCanManageChannels(ctx, guildID, userID); err != nil {
		return nil, err
	}

	channel, err := s.repo.GetByID(ctx, channelID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrChannelNotFoundInGuild
		}
		return nil, err
	}
	if channel.GuildID != guildID {
		return nil, ErrChannelNotFoundInGuild
	}

	fields := map[string]any{}
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return nil, errors.New("channel name cannot be empty")
		}
		fields["name"] = trimmed
	}
	if channelType != nil {
		if *channelType != "text" && *channelType != "voice" {
			return nil, errors.New("channel type must be text or voice")
		}
		fields["type"] = *channelType
	}

	if err := s.repo.UpdateFields(ctx, channelID, fields); err != nil {
		return nil, err
	}

	updated, err := s.repo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	return &ChannelDTO{
		ID:      updated.ID,
		Name:    updated.Name,
		GuildID: updated.GuildID,
		Type:    updated.Type,
	}, nil
}

func (s *ChannelService) DeleteChannel(ctx context.Context, guildID uint, channelID uint, userID uuid.UUID) error {
	if err := s.ensureCanManageChannels(ctx, guildID, userID); err != nil {
		return err
	}

	channel, err := s.repo.GetByID(ctx, channelID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrChannelNotFoundInGuild
		}
		return err
	}
	if channel.GuildID != guildID {
		return ErrChannelNotFoundInGuild
	}

	return s.repo.Delete(ctx, channelID)
}

func (s *ChannelService) ensureCanManageChannels(ctx context.Context, guildID uint, userID uuid.UUID) error {
	isMember, err := s.access.IsMember(ctx, guildID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrChannelAccessDenied
	}
	perms, err := s.access.GetMemberPermissions(ctx, guildID, userID)
	if err != nil {
		return err
	}
	if perms&domain.PermManageChannels == 0 && perms&domain.PermManageGuild == 0 {
		return ErrMissingManageChannels
	}
	return nil
}
