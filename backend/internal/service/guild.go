package service

import (
	"context"
	"errors"
	"lumen/internal/domain"
	"lumen/internal/repository"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GuildRepository interface {
	Create(ctx context.Context, guild *domain.Guild, ownerID uuid.UUID) (*domain.Guild, error)
	FindByInviteCode(ctx context.Context, inviteCode string) (*domain.Guild, error)
	AddMemberIfNotExists(ctx context.Context, guildID uint, userID uuid.UUID) error
	GetUserGuilds(ctx context.Context, userID uuid.UUID) ([]*domain.Guild, error)
	GetByID(ctx context.Context, guildID uint) (*domain.Guild, error)
	UpdateFields(ctx context.Context, guildID uint, fields map[string]any) error
	Delete(ctx context.Context, guildID uint) error
	IsMember(ctx context.Context, guildID uint, userID uuid.UUID) (bool, error)
	GetMemberPermissions(ctx context.Context, guildID uint, userID uuid.UUID) (uint64, error)
	ListMembers(ctx context.Context, guildID uint) ([]repository.GuildMemberRow, error)
	RemoveMember(ctx context.Context, guildID uint, userID uuid.UUID) (int64, error)
}

type GuildUserFinder interface {
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
}

type GuildService struct {
	repo  GuildRepository
	users GuildUserFinder
}

type GuildDTO struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	IconURL     string    `json:"icon_url"`
	Description string    `json:"description"`
	InviteCode  string    `json:"invite_code"`
	OwnerID     uuid.UUID `json:"owner_id"`
}

type MemberDTO struct {
	UserID      uuid.UUID `json:"user_id"`
	Username    string    `json:"username"`
	Role        string    `json:"role"`
	Permissions uint64    `json:"permissions"`
	IsOwner     bool      `json:"is_owner"`
}

type UpdateGuildInput struct {
	Name        *string
	IconURL     *string
	Description *string
}

var ErrGuildNotFound = errors.New("guild not found")
var ErrGuildAccessDenied = errors.New("guild access denied")
var ErrMissingManageGuild = errors.New("missing manage guild permission")
var ErrNotGuildOwner = errors.New("only the owner can perform this action")
var ErrMemberUserNotFound = errors.New("user not found")
var ErrMemberNotFound = errors.New("member not found")
var ErrCannotRemoveOwner = errors.New("cannot remove the guild owner")

func NewGuildService(repo GuildRepository, users GuildUserFinder) *GuildService {
	return &GuildService{repo: repo, users: users}
}

func toGuildDTO(guild *domain.Guild) *GuildDTO {
	return &GuildDTO{
		ID:          guild.ID,
		Name:        guild.Name,
		IconURL:     guild.IconURL,
		Description: guild.Description,
		InviteCode:  guild.InviteCode,
		OwnerID:     guild.OwnerID,
	}
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

	return toGuildDTO(guild), nil
}

func (s *GuildService) ListByUser(ctx context.Context, userID uuid.UUID) ([]GuildDTO, error) {
	guilds, err := s.repo.GetUserGuilds(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]GuildDTO, 0, len(guilds))
	for _, g := range guilds {
		result = append(result, *toGuildDTO(g))
	}
	return result, nil
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

	return toGuildDTO(guild), nil
}

func (s *GuildService) UpdateGuild(ctx context.Context, guildID uint, userID uuid.UUID, input UpdateGuildInput) (*GuildDTO, error) {
	if err := s.ensureCanManageGuild(ctx, guildID, userID); err != nil {
		return nil, err
	}

	fields := map[string]any{}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, errors.New("guild name cannot be empty")
		}
		fields["name"] = name
	}
	if input.IconURL != nil {
		fields["icon_url"] = strings.TrimSpace(*input.IconURL)
	}
	if input.Description != nil {
		fields["description"] = strings.TrimSpace(*input.Description)
	}

	if err := s.repo.UpdateFields(ctx, guildID, fields); err != nil {
		return nil, err
	}

	guild, err := s.repo.GetByID(ctx, guildID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGuildNotFound
		}
		return nil, err
	}
	return toGuildDTO(guild), nil
}

func (s *GuildService) DeleteGuild(ctx context.Context, guildID uint, userID uuid.UUID) error {
	guild, err := s.repo.GetByID(ctx, guildID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrGuildNotFound
		}
		return err
	}
	if guild.OwnerID != userID {
		return ErrNotGuildOwner
	}
	return s.repo.Delete(ctx, guildID)
}

func (s *GuildService) ListMembers(ctx context.Context, guildID uint, userID uuid.UUID) ([]MemberDTO, error) {
	isMember, err := s.repo.IsMember(ctx, guildID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrGuildAccessDenied
	}

	guild, err := s.repo.GetByID(ctx, guildID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGuildNotFound
		}
		return nil, err
	}

	rows, err := s.repo.ListMembers(ctx, guildID)
	if err != nil {
		return nil, err
	}

	result := make([]MemberDTO, 0, len(rows))
	for _, row := range rows {
		result = append(result, MemberDTO{
			UserID:      row.UserID,
			Username:    row.Username,
			Role:        row.Role,
			Permissions: row.Permissions,
			IsOwner:     row.UserID == guild.OwnerID,
		})
	}
	return result, nil
}

func (s *GuildService) AddMember(ctx context.Context, guildID uint, actorID uuid.UUID, username string) (*MemberDTO, error) {
	if err := s.ensureCanManageGuild(ctx, guildID, actorID); err != nil {
		return nil, err
	}

	username = strings.TrimSpace(username)
	if username == "" {
		return nil, errors.New("username is required")
	}

	user, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMemberUserNotFound
		}
		return nil, err
	}

	if err := s.repo.AddMemberIfNotExists(ctx, guildID, user.ID); err != nil {
		return nil, err
	}

	return &MemberDTO{
		UserID:   user.ID,
		Username: user.Username,
		Role:     "member",
	}, nil
}

func (s *GuildService) RemoveMember(ctx context.Context, guildID uint, actorID uuid.UUID, targetID uuid.UUID) error {
	if err := s.ensureCanManageGuild(ctx, guildID, actorID); err != nil {
		return err
	}

	guild, err := s.repo.GetByID(ctx, guildID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrGuildNotFound
		}
		return err
	}
	if guild.OwnerID == targetID {
		return ErrCannotRemoveOwner
	}

	affected, err := s.repo.RemoveMember(ctx, guildID, targetID)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrMemberNotFound
	}
	return nil
}

func (s *GuildService) ensureCanManageGuild(ctx context.Context, guildID uint, userID uuid.UUID) error {
	isMember, err := s.repo.IsMember(ctx, guildID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrGuildAccessDenied
	}
	perms, err := s.repo.GetMemberPermissions(ctx, guildID, userID)
	if err != nil {
		return err
	}
	member := domain.GuildMember{Permissions: perms}
	if !member.HasPermission(domain.PermManageGuild) {
		return ErrMissingManageGuild
	}
	return nil
}
