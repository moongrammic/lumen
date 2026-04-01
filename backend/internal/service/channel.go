package service

import (
	"context"
	"errors"
	"lumen/internal/domain"

	"github.com/google/uuid"
)

type ChannelRepository interface {
	Create(ctx context.Context, channel *domain.Channel) (*domain.Channel, error)
	ListByGuild(ctx context.Context, guildID uint) ([]domain.Channel, error)
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
