package service

import (
	"context"
	"errors"
	"lumen/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GuildRepository interface {
	Create(ctx context.Context, guild *domain.Guild, ownerID uuid.UUID) (*domain.Guild, error)
	FindByInviteCode(ctx context.Context, inviteCode string) (*domain.Guild, error)
	AddMemberIfNotExists(ctx context.Context, guildID uint, userID uuid.UUID) error
}

type GuildService struct {
	repo GuildRepository
}

type GuildDTO struct {
	ID         uint      `json:"id"`
	Name       string    `json:"name"`
	InviteCode string    `json:"invite_code"`
	OwnerID    uuid.UUID `json:"owner_id"`
}

var ErrGuildNotFound = errors.New("guild not found")

func NewGuildService(repo GuildRepository) *GuildService {
	return &GuildService{repo: repo}
}

func (s *GuildService) Create(ctx context.Context, name string, ownerID uuid.UUID) (*GuildDTO, error) {
	if name == "" {
		return nil, errors.New("guild name is required")
	}

	inviteCode := uuid.NewString()[:8]
	guild, err := s.repo.Create(ctx, &domain.Guild{
		Name:       name,
		InviteCode: inviteCode,
		OwnerID:    ownerID,
	}, ownerID)
	if err != nil {
		return nil, err
	}

	return &GuildDTO{
		ID:         guild.ID,
		Name:       guild.Name,
		InviteCode: guild.InviteCode,
		OwnerID:    guild.OwnerID,
	}, nil
}

func (s *GuildService) JoinByInvite(ctx context.Context, inviteCode string, userID uuid.UUID) (*GuildDTO, error) {
	if inviteCode == "" {
		return nil, errors.New("invite code is required")
	}

	guild, err := s.repo.FindByInviteCode(ctx, inviteCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGuildNotFound
		}
		return nil, err
	}

	if err := s.repo.AddMemberIfNotExists(ctx, guild.ID, userID); err != nil {
		return nil, err
	}

	return &GuildDTO{
		ID:         guild.ID,
		Name:       guild.Name,
		InviteCode: guild.InviteCode,
		OwnerID:    guild.OwnerID,
	}, nil
}
