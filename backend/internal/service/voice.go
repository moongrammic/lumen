package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	lkauth "github.com/livekit/protocol/auth"
)

type VoiceAccessChecker interface {
	IsMember(ctx context.Context, guildID uint, userID uuid.UUID) (bool, error)
}

type VoiceService struct {
	guilds    VoiceAccessChecker
	hub       ChatBroadcaster
	apiKey    string
	apiSecret string
}

func NewVoiceService(guilds VoiceAccessChecker, hub ChatBroadcaster, apiKey string, apiSecret string) *VoiceService {
	return &VoiceService{
		guilds:    guilds,
		hub:       hub,
		apiKey:    apiKey,
		apiSecret: apiSecret,
	}
}

func (s *VoiceService) GenerateJoinToken(ctx context.Context, userID uuid.UUID, guildID uint, roomName string) (string, error) {
	if s.apiKey == "" || s.apiSecret == "" {
		return "", errors.New("livekit credentials are not configured")
	}
	if roomName == "" {
		return "", errors.New("room name is required")
	}

	ok, err := s.guilds.IsMember(ctx, guildID, userID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errors.New("voice access denied")
	}

	token := lkauth.NewAccessToken(s.apiKey, s.apiSecret)
	token.SetIdentity(userID.String())
	token.SetName(fmt.Sprintf("user-%s", userID.String()[:8]))
	token.SetVideoGrant(&lkauth.VideoGrant{
		RoomJoin: true,
		Room:     roomName,
	})

	return token.ToJWT()
}

func (s *VoiceService) JoinRoom(ctx context.Context, userID uuid.UUID, guildID uint, roomName string) (string, error) {
	token, err := s.GenerateJoinToken(ctx, userID, guildID, roomName)
	if err != nil {
		return "", err
	}

	if err := s.hub.Broadcast(Event{
		Op:    0,
		Event: "VOICE_STATE_UPDATE",
		Payload: map[string]interface{}{
			"user_id":   userID.String(),
			"guild_id":  guildID,
			"room_name": roomName,
			"status":    "joined",
		},
	}); err != nil {
		return "", err
	}

	return token, nil
}

func (s *VoiceService) LeaveRoom(ctx context.Context, userID uuid.UUID, guildID uint, roomName string) error {
	ok, err := s.guilds.IsMember(ctx, guildID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("voice access denied")
	}

	return s.hub.Broadcast(Event{
		Op:    0,
		Event: "VOICE_STATE_UPDATE",
		Payload: map[string]interface{}{
			"user_id":   userID.String(),
			"guild_id":  guildID,
			"room_name": roomName,
			"status":    "left",
		},
	})
}
