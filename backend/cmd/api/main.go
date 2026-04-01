package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"lumen/internal/config"
	"lumen/internal/middleware"
	"lumen/internal/repository"
	"lumen/internal/service"
	"lumen/internal/ws"
	"lumen/pkg/apierr"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/redis/go-redis/v9"
)

func main() {
	validate := validator.New()
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// 1. Инициализация БД
	db, err := repository.InitDB(cfg.DB)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)
	authService := service.NewAuthService(userRepo, cfg.JWT.Secret)
	guildRepo := repository.NewGuildRepository(db)
	guildService := service.NewGuildService(guildRepo)
	messageRepo := repository.NewMessageRepository(db)
	channelRepo := repository.NewChannelRepository(db)

	app := fiber.New(fiber.Config{
		AppName: "Lumen API v1.0",
	})

	// 2. Middlewares
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use("/api", limiter.New(limiter.Config{
		Max:        120,
		Expiration: 60 * time.Second,
	}))
	app.Use("/api/auth", limiter.New(limiter.Config{
		Max:        20,
		Expiration: 60 * time.Second,
	}))

	// 3. Роуты
	api := app.Group("/api")
	hub := ws.NewHub(cfg.Redis)
	rateLimiterRedis := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	messageRateLimiter := service.NewRedisMessageRateLimiter(rateLimiterRedis, cfg.RateLimit)
	chatService := service.NewChatService(messageRepo, guildRepo, hub, messageRateLimiter)
	channelService := service.NewChannelService(channelRepo, guildRepo)
	voiceService := service.NewVoiceService(guildRepo, hub, cfg.LiveKit.APIKey, cfg.LiveKit.APISecret)
	go hub.Run()

	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "db": "connected"})
	})

	api.Post("/auth/register", func(c *fiber.Ctx) error {
		var req RegisterDTO
		if err := c.BodyParser(&req); err != nil {
			return apierr.Write(c, fiber.StatusBadRequest, "invalid_body", "Некорректный JSON запроса")
		}
		if err := validate.Struct(req); err != nil {
			return apierr.Write(c, fiber.StatusBadRequest, "validation_failed", err.Error())
		}

		resp, err := authService.Register(c.UserContext(), service.RegisterInput{
			Username: req.Username,
			Email:    req.Email,
			Password: req.Password,
		})
		if err != nil {
			if errors.Is(err, service.ErrEmailAlreadyExists) {
				return apierr.Write(c, fiber.StatusConflict, "email_already_exists", "Пользователь с таким email уже существует")
			}
			if errors.Is(err, service.ErrUsernameAlreadyExists) {
				return apierr.Write(c, fiber.StatusConflict, "username_already_exists", "Пользователь с таким username уже существует")
			}
			return apierr.Write(c, fiber.StatusInternalServerError, "register_failed", "Не удалось зарегистрировать пользователя")
		}

		return c.Status(fiber.StatusCreated).JSON(resp)
	})

	api.Post("/auth/login", func(c *fiber.Ctx) error {
		var req LoginDTO
		if err := c.BodyParser(&req); err != nil {
			return apierr.Write(c, fiber.StatusBadRequest, "invalid_body", "Некорректный JSON запроса")
		}
		if err := validate.Struct(req); err != nil {
			return apierr.Write(c, fiber.StatusBadRequest, "validation_failed", err.Error())
		}

		resp, err := authService.Login(c.UserContext(), service.LoginInput{
			Email:    req.Email,
			Password: req.Password,
		})
		if err != nil {
			if errors.Is(err, service.ErrInvalidCredentials) {
				return apierr.Write(c, fiber.StatusUnauthorized, "invalid_credentials", "Неверный email или пароль")
			}
			return apierr.Write(c, fiber.StatusInternalServerError, "login_failed", "Не удалось выполнить вход")
		}

		return c.JSON(resp)
	})

	api.Get("/me", middleware.JWTProtected(cfg.JWT.Secret), func(c *fiber.Ctx) error {
		userID, err := middleware.ExtractUserIDFromClaims(c.Locals("user"))
		if err != nil {
			return apierr.Write(c, fiber.StatusUnauthorized, "invalid_token_claims", err.Error())
		}

		me, err := userService.GetMe(c.UserContext(), userID)
		if err != nil {
			if errors.Is(err, service.ErrUserNotFound) {
				return apierr.Write(c, fiber.StatusNotFound, "user_not_found", "Пользователь не найден")
			}
			return apierr.Write(c, fiber.StatusInternalServerError, "user_load_failed", "Не удалось получить профиль пользователя")
		}

		return c.JSON(me)
	})

	api.Post("/guilds", middleware.JWTProtected(cfg.JWT.Secret), func(c *fiber.Ctx) error {
		var req CreateGuildDTO
		if err := c.BodyParser(&req); err != nil {
			return apierr.Write(c, fiber.StatusBadRequest, "invalid_body", "Некорректный JSON запроса")
		}
		if err := validate.Struct(req); err != nil {
			return apierr.Write(c, fiber.StatusBadRequest, "validation_failed", err.Error())
		}

		userID, err := middleware.ExtractUserIDFromClaims(c.Locals("user"))
		if err != nil {
			return apierr.Write(c, fiber.StatusUnauthorized, "invalid_token_claims", err.Error())
		}

		guild, err := guildService.Create(c.UserContext(), req.Name, userID)
		if err != nil {
			return apierr.Write(c, fiber.StatusBadRequest, "guild_create_failed", err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(guild)
	})

	api.Post("/guilds/join", middleware.JWTProtected(cfg.JWT.Secret), func(c *fiber.Ctx) error {
		var req JoinGuildDTO
		if err := c.BodyParser(&req); err != nil {
			return apierr.Write(c, fiber.StatusBadRequest, "invalid_body", "Некорректный JSON запроса")
		}
		if err := validate.Struct(req); err != nil {
			return apierr.Write(c, fiber.StatusBadRequest, "validation_failed", err.Error())
		}

		userID, err := middleware.ExtractUserIDFromClaims(c.Locals("user"))
		if err != nil {
			return apierr.Write(c, fiber.StatusUnauthorized, "invalid_token_claims", err.Error())
		}

		guild, err := guildService.JoinByInvite(c.UserContext(), req.InviteCode, userID)
		if err != nil {
			if errors.Is(err, service.ErrGuildNotFound) {
				return apierr.Write(c, fiber.StatusNotFound, "guild_not_found", "Гильдия с таким invite code не существует")
			}
			return apierr.Write(c, fiber.StatusBadRequest, "guild_join_failed", err.Error())
		}
		return c.JSON(guild)
	})

	api.Post("/guilds/:guildID/channels", middleware.JWTProtected(cfg.JWT.Secret), middleware.GuildAccess(guildRepo), func(c *fiber.Ctx) error {
		guildID64, err := strconv.ParseUint(c.Params("guildID"), 10, 32)
		if err != nil {
			return apierr.Write(c, fiber.StatusBadRequest, "invalid_guild_id", "Guild ID has invalid format")
		}

		var req CreateChannelDTO
		if err := c.BodyParser(&req); err != nil {
			return apierr.Write(c, fiber.StatusBadRequest, "invalid_body", "Некорректный JSON запроса")
		}
		if err := validate.Struct(req); err != nil {
			return apierr.Write(c, fiber.StatusBadRequest, "validation_failed", err.Error())
		}

		userID, err := middleware.ExtractUserIDFromClaims(c.Locals("user"))
		if err != nil {
			return apierr.Write(c, fiber.StatusUnauthorized, "invalid_token_claims", err.Error())
		}

		channel, err := channelService.Create(c.UserContext(), uint(guildID64), userID, req.Name, req.Type)
		if err != nil {
			if errors.Is(err, service.ErrChannelAccessDenied) {
				return apierr.Write(c, fiber.StatusForbidden, "channel_access_denied", "Пользователь не состоит в гильдии")
			}
			if errors.Is(err, service.ErrMissingManageChannels) {
				return apierr.Write(c, fiber.StatusForbidden, "missing_manage_channels_permission", "Недостаточно прав для создания канала")
			}
			return apierr.Write(c, fiber.StatusBadRequest, "channel_create_failed", err.Error())
		}

		return c.Status(fiber.StatusCreated).JSON(channel)
	})

	api.Get("/guilds/:guildID/channels", middleware.JWTProtected(cfg.JWT.Secret), middleware.GuildAccess(guildRepo), func(c *fiber.Ctx) error {
		guildID64, err := strconv.ParseUint(c.Params("guildID"), 10, 32)
		if err != nil {
			return apierr.Write(c, fiber.StatusBadRequest, "invalid_guild_id", "Guild ID has invalid format")
		}
		userID, err := middleware.ExtractUserIDFromClaims(c.Locals("user"))
		if err != nil {
			return apierr.Write(c, fiber.StatusUnauthorized, "invalid_token_claims", err.Error())
		}

		channels, err := channelService.ListByGuild(c.UserContext(), uint(guildID64), userID)
		if err != nil {
			if errors.Is(err, service.ErrChannelAccessDenied) {
				return apierr.Write(c, fiber.StatusForbidden, "channel_access_denied", "Пользователь не состоит в гильдии")
			}
			return apierr.Write(c, fiber.StatusInternalServerError, "channels_list_failed", "Не удалось получить список каналов")
		}
		return c.JSON(fiber.Map{"channels": channels})
	})

	api.Get("/guilds/:guildID/channels/:channelID/messages", middleware.JWTProtected(cfg.JWT.Secret), middleware.GuildAccess(guildRepo), func(c *fiber.Ctx) error {
		guildID, err := strconv.ParseUint(c.Params("guildID"), 10, 32)
		if err != nil {
			return apierr.Write(c, fiber.StatusBadRequest, "invalid_guild_id", "Guild ID has invalid format")
		}

		channelID, err := strconv.ParseUint(c.Params("channelID"), 10, 32)
		if err != nil {
			return apierr.Write(c, fiber.StatusBadRequest, "invalid_channel_id", "Channel ID has invalid format")
		}

		belongs, err := guildRepo.ChannelBelongsToGuild(c.UserContext(), uint(channelID), uint(guildID))
		if err != nil {
			return apierr.Write(c, fiber.StatusInternalServerError, "channel_access_check_failed", "Не удалось проверить принадлежность канала")
		}
		if !belongs {
			return apierr.Write(c, fiber.StatusForbidden, "channel_not_in_guild", "Канал не принадлежит указанной гильдии")
		}

		var beforeID *uint
		if rawBefore := c.Query("before"); rawBefore != "" {
			parsedBefore, parseErr := strconv.ParseUint(rawBefore, 10, 32)
			if parseErr != nil {
				return apierr.Write(c, fiber.StatusBadRequest, "invalid_before_cursor", "Параметр before имеет неверный формат")
			}
			cursor := uint(parsedBefore)
			beforeID = &cursor
		}

		limit := 50
		if rawLimit := c.Query("limit"); rawLimit != "" {
			parsedLimit, parseErr := strconv.Atoi(rawLimit)
			if parseErr != nil || parsedLimit <= 0 {
				return apierr.Write(c, fiber.StatusBadRequest, "invalid_limit", "Параметр limit должен быть положительным числом")
			}
			limit = parsedLimit
		}

		userID, err := middleware.ExtractUserIDFromClaims(c.Locals("user"))
		if err != nil {
			return apierr.Write(c, fiber.StatusUnauthorized, "invalid_token_claims", err.Error())
		}
		result, err := chatService.ListMessages(c.UserContext(), userID, uint(channelID), beforeID, limit)
		if err != nil {
			if errors.Is(err, service.ErrChatAccessDenied) {
				return apierr.Write(c, fiber.StatusForbidden, "chat_access_denied", "Нет доступа к каналу")
			}
			if errors.Is(err, service.ErrChannelNotFound) {
				return apierr.Write(c, fiber.StatusNotFound, "channel_not_found", "Канал не найден")
			}
			return apierr.Write(c, fiber.StatusInternalServerError, "messages_list_failed", "Не удалось получить историю сообщений")
		}

		return c.JSON(result)
	})

	api.Post("/channels/:channelID/messages", middleware.JWTProtected(cfg.JWT.Secret), func(c *fiber.Ctx) error {
		channelID64, err := strconv.ParseUint(c.Params("channelID"), 10, 32)
		if err != nil {
			return apierr.Write(c, fiber.StatusBadRequest, "invalid_channel_id", "Channel ID has invalid format")
		}
		var req CreateMessageDTO
		if err := c.BodyParser(&req); err != nil {
			return apierr.Write(c, fiber.StatusBadRequest, "invalid_body", "Некорректный JSON запроса")
		}
		if err := validate.Struct(req); err != nil {
			return apierr.Write(c, fiber.StatusBadRequest, "validation_failed", err.Error())
		}
		userID, err := middleware.ExtractUserIDFromClaims(c.Locals("user"))
		if err != nil {
			return apierr.Write(c, fiber.StatusUnauthorized, "invalid_token_claims", err.Error())
		}
		msg, err := chatService.CreateMessage(c.UserContext(), userID, uint(channelID64), req.Content)
		if err != nil {
			if errors.Is(err, service.ErrChatAccessDenied) {
				return apierr.Write(c, fiber.StatusForbidden, "chat_access_denied", "Нет доступа к каналу")
			}
			if errors.Is(err, service.ErrInsufficientPermissions) {
				return apierr.Write(c, fiber.StatusForbidden, "insufficient_permissions", "Недостаточно прав для отправки сообщений")
			}
			if errors.Is(err, service.ErrRateLimitExceeded) {
				c.Set("Retry-After", "10")
				return apierr.Write(c, fiber.StatusTooManyRequests, "rate_limit_exceeded", "Слишком много сообщений, попробуйте позже")
			}
			if errors.Is(err, service.ErrChannelNotFound) {
				return apierr.Write(c, fiber.StatusNotFound, "channel_not_found", "Канал не найден")
			}
			return apierr.Write(c, fiber.StatusBadRequest, "message_create_failed", err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(msg)
	})

	api.Get("/channels/:channelID/messages", middleware.JWTProtected(cfg.JWT.Secret), func(c *fiber.Ctx) error {
		channelID64, err := strconv.ParseUint(c.Params("channelID"), 10, 32)
		if err != nil {
			return apierr.Write(c, fiber.StatusBadRequest, "invalid_channel_id", "Channel ID has invalid format")
		}
		userID, err := middleware.ExtractUserIDFromClaims(c.Locals("user"))
		if err != nil {
			return apierr.Write(c, fiber.StatusUnauthorized, "invalid_token_claims", err.Error())
		}
		messages, err := chatService.GetRecentMessages(c.UserContext(), userID, uint(channelID64))
		if err != nil {
			if errors.Is(err, service.ErrChatAccessDenied) {
				return apierr.Write(c, fiber.StatusForbidden, "chat_access_denied", "Нет доступа к каналу")
			}
			if errors.Is(err, service.ErrChannelNotFound) {
				return apierr.Write(c, fiber.StatusNotFound, "channel_not_found", "Канал не найден")
			}
			return apierr.Write(c, fiber.StatusInternalServerError, "messages_recent_failed", "Не удалось получить сообщения")
		}
		return c.JSON(fiber.Map{"messages": messages})
	})

	api.Post("/voice/join-token", middleware.JWTProtected(cfg.JWT.Secret), func(c *fiber.Ctx) error {
		var req VoiceJoinTokenDTO
		if err := c.BodyParser(&req); err != nil {
			return apierr.Write(c, fiber.StatusBadRequest, "invalid_body", "Некорректный JSON запроса")
		}
		if err := validate.Struct(req); err != nil {
			return apierr.Write(c, fiber.StatusBadRequest, "validation_failed", err.Error())
		}

		userID, err := middleware.ExtractUserIDFromClaims(c.Locals("user"))
		if err != nil {
			return apierr.Write(c, fiber.StatusUnauthorized, "invalid_token_claims", err.Error())
		}

		token, err := voiceService.JoinRoom(c.UserContext(), userID, req.GuildID, req.RoomName)
		if err != nil {
			if err.Error() == "voice access denied" {
				return apierr.Write(c, fiber.StatusForbidden, "voice_access_denied", "Пользователь не состоит в гильдии")
			}
			return apierr.Write(c, fiber.StatusBadRequest, "voice_token_failed", err.Error())
		}

		return c.JSON(fiber.Map{
			"token":       token,
			"livekit_url": cfg.LiveKit.URL,
		})
	})

	api.Post("/voice/leave", middleware.JWTProtected(cfg.JWT.Secret), func(c *fiber.Ctx) error {
		var req VoiceLeaveDTO
		if err := c.BodyParser(&req); err != nil {
			return apierr.Write(c, fiber.StatusBadRequest, "invalid_body", "Некорректный JSON запроса")
		}
		if err := validate.Struct(req); err != nil {
			return apierr.Write(c, fiber.StatusBadRequest, "validation_failed", err.Error())
		}

		userID, err := middleware.ExtractUserIDFromClaims(c.Locals("user"))
		if err != nil {
			return apierr.Write(c, fiber.StatusUnauthorized, "invalid_token_claims", err.Error())
		}

		if err := voiceService.LeaveRoom(c.UserContext(), userID, req.GuildID, req.RoomName); err != nil {
			if err.Error() == "voice access denied" {
				return apierr.Write(c, fiber.StatusForbidden, "voice_access_denied", "Пользователь не состоит в гильдии")
			}
			return apierr.Write(c, fiber.StatusBadRequest, "voice_leave_failed", err.Error())
		}

		return c.JSON(fiber.Map{"ok": true})
	})

	app.Get("/ws", limiter.New(limiter.Config{
		Max:        30,
		Expiration: 60 * time.Second,
	}), websocket.New(func(c *websocket.Conn) {
		userID, err := waitWebSocketIdentify(c, cfg.JWT.Secret, 15*time.Second)
		if err != nil {
			_ = c.WriteJSON(fiber.Map{
				"op":      0,
				"event":   "ERROR",
				"payload": fiber.Map{"code": "IDENTIFY_FAILED", "message": err.Error()},
			})
			return
		}
		if err := c.WriteJSON(fiber.Map{"op": 0, "event": "READY", "payload": fiber.Map{}}); err != nil {
			return
		}

		hub.Register(c)
		defer hub.Unregister(c)
		_ = chatService.UpdatePresence(context.Background(), userID, "online", cfg.Presence.TTL)
		defer func() {
			_ = chatService.UpdatePresence(context.Background(), userID, "offline", 5*time.Second)
		}()

		for {
			_, msg, readErr := c.ReadMessage()
			if readErr != nil {
				return
			}

			var incoming service.IncomingEvent
			if err := json.Unmarshal(msg, &incoming); err != nil {
				_ = c.WriteMessage(websocket.TextMessage, []byte(`{"event":"ERROR","payload":{"message":"invalid websocket payload"}}`))
				continue
			}

			switch incoming.Event {
			case "IDENTIFY":
				_ = c.WriteJSON(fiber.Map{
					"op":      0,
					"event":   "ERROR",
					"payload": fiber.Map{"code": "ALREADY_IDENTIFIED", "message": "session already authenticated"},
				})
				continue
			case "SUBSCRIBE_CHANNEL":
				channelID, err := extractChannelID(incoming.Payload)
				if err != nil {
					_ = c.WriteMessage(websocket.TextMessage, []byte(`{"event":"ERROR","payload":{"message":"invalid channel_id"}}`))
					continue
				}
				guildID, err := guildRepo.GetChannelGuildID(context.Background(), channelID)
				if err != nil {
					_ = c.WriteMessage(websocket.TextMessage, []byte(`{"event":"ERROR","payload":{"message":"channel not found"}}`))
					continue
				}
				isMember, err := guildRepo.IsMember(context.Background(), guildID, userID)
				if err != nil || !isMember {
					_ = c.WriteMessage(websocket.TextMessage, []byte(`{"event":"ERROR","payload":{"message":"access denied"}}`))
					continue
				}
				if err := hub.Subscribe(c, channelID); err != nil {
					_ = c.WriteMessage(websocket.TextMessage, []byte(`{"event":"ERROR","payload":{"message":"subscribe failed"}}`))
					continue
				}
				_ = c.WriteJSON(fiber.Map{"event": "CHANNEL_SUBSCRIBED", "payload": fiber.Map{"channel_id": channelID}})
				continue
			case "UNSUBSCRIBE_CHANNEL":
				channelID, err := extractChannelID(incoming.Payload)
				if err != nil {
					_ = c.WriteMessage(websocket.TextMessage, []byte(`{"event":"ERROR","payload":{"message":"invalid channel_id"}}`))
					continue
				}
				if err := hub.Unsubscribe(c, channelID); err != nil {
					_ = c.WriteMessage(websocket.TextMessage, []byte(`{"event":"ERROR","payload":{"message":"unsubscribe failed"}}`))
					continue
				}
				_ = c.WriteJSON(fiber.Map{"event": "CHANNEL_UNSUBSCRIBED", "payload": fiber.Map{"channel_id": channelID}})
				continue
			}

			if processErr := chatService.HandleIncomingEvent(context.Background(), userID, msg); processErr != nil {
				if errors.Is(processErr, service.ErrInsufficientPermissions) {
					_ = c.WriteJSON(fiber.Map{
						"event": "ERROR",
						"payload": fiber.Map{
							"code":    "INSUFFICIENT_PERMISSIONS",
							"message": "You don't have permission to send messages in this channel",
						},
					})
					continue
				}
				if errors.Is(processErr, service.ErrRateLimitExceeded) {
					_ = c.WriteJSON(fiber.Map{
						"event": "ERROR",
						"payload": fiber.Map{
							"code":        "RATE_LIMIT_EXCEEDED",
							"message":     "Too many messages, slow down",
							"retry_after": 10,
						},
					})
					continue
				}
				_ = c.WriteMessage(
					websocket.TextMessage,
					[]byte(fmt.Sprintf(`{"type":"error","payload":{"message":"%s"}}`, processErr.Error())),
				)
			}
		}
	}))

	// 4. Запуск
	if err := app.Listen(":" + cfg.App.Port); err != nil {
		slog.Error("fiber listen failed", "port", cfg.App.Port, "error", err)
		os.Exit(1)
	}
}

type CreateGuildDTO struct {
	Name string `json:"name" validate:"required,min=3,max=32"`
}

type RegisterDTO struct {
	Username string `json:"username" validate:"required,min=3,max=32"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type LoginDTO struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type JoinGuildDTO struct {
	InviteCode string `json:"invite_code" validate:"required,min=6,max=64"`
}

type CreateChannelDTO struct {
	Name string `json:"name" validate:"required,min=1,max=64"`
	Type string `json:"type" validate:"omitempty,oneof=text voice"`
}

type CreateMessageDTO struct {
	Content string `json:"content" validate:"required,min=1,max=2000"`
}

type VoiceJoinTokenDTO struct {
	GuildID  uint   `json:"guild_id" validate:"required"`
	RoomName string `json:"room_name" validate:"required,min=2,max=128"`
}

type VoiceLeaveDTO struct {
	GuildID  uint   `json:"guild_id" validate:"required"`
	RoomName string `json:"room_name" validate:"required,min=2,max=128"`
}

func waitWebSocketIdentify(c *websocket.Conn, secret string, timeout time.Duration) (uuid.UUID, error) {
	if err := c.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return uuid.Nil, err
	}
	defer func() { _ = c.SetReadDeadline(time.Time{}) }()

	_, msg, err := c.ReadMessage()
	if err != nil {
		return uuid.Nil, err
	}

	var incoming service.IncomingEvent
	if err := json.Unmarshal(msg, &incoming); err != nil {
		return uuid.Nil, errors.New("invalid websocket payload")
	}
	if incoming.Event != "IDENTIFY" {
		return uuid.Nil, fmt.Errorf("expected IDENTIFY first, got %q", incoming.Event)
	}

	var identify struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(incoming.Payload, &identify); err != nil || strings.TrimSpace(identify.Token) == "" {
		return uuid.Nil, errors.New("missing token in IDENTIFY payload")
	}

	claims, err := middleware.ParseJWTFromString(secret, identify.Token)
	if err != nil {
		return uuid.Nil, err
	}
	return middleware.ExtractUserIDFromClaims(claims)
}

func extractChannelID(raw json.RawMessage) (uint, error) {
	var payload struct {
		ChannelID uint `json:"channel_id"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return 0, err
	}
	if payload.ChannelID == 0 {
		return 0, errors.New("invalid channel id")
	}
	return payload.ChannelID, nil
}
