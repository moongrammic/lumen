package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"lumen/internal/middleware"
	"lumen/internal/repository"
	"lumen/internal/service"
	"lumen/internal/ws"
	"lumen/pkg/apierr"
	"os"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	validate := validator.New()

	// 1. Инициализация БД
	db, err := repository.InitDB()
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)
	guildRepo := repository.NewGuildRepository(db)
	guildService := service.NewGuildService(guildRepo)
	messageRepo := repository.NewMessageRepository(db)

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

	// 3. Роуты
	api := app.Group("/api")
	hub := ws.NewHub()
	chatService := service.NewChatService(messageRepo, hub)
	go hub.Run()

	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "db": "connected"})
	})

	api.Get("/me", middleware.JWTProtected(), func(c *fiber.Ctx) error {
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

	api.Post("/guilds", middleware.JWTProtected(), func(c *fiber.Ctx) error {
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

	api.Post("/guilds/join", middleware.JWTProtected(), func(c *fiber.Ctx) error {
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

	api.Get("/guilds/:guildID/channels/:channelID/messages", middleware.JWTProtected(), middleware.GuildAccess(guildRepo), func(c *fiber.Ctx) error {
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

		result, err := chatService.ListMessages(c.UserContext(), uint(channelID), beforeID, limit)
		if err != nil {
			return apierr.Write(c, fiber.StatusInternalServerError, "messages_list_failed", "Не удалось получить историю сообщений")
		}

		return c.JSON(result)
	})

	app.Get("/ws", middleware.JWTProtected(), limiter.New(limiter.Config{
		Max:        30,
		Expiration: 60 * time.Second,
	}), websocket.New(func(c *websocket.Conn) {
		userID, err := middleware.ExtractUserIDFromClaims(c.Locals("user"))
		if err != nil {
			_ = c.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","payload":{"message":"unauthorized"}}`))
			return
		}

		hub.Register(c)
		defer hub.Unregister(c)

		for {
			_, msg, readErr := c.ReadMessage()
			if readErr != nil {
				return
			}

			if processErr := chatService.HandleIncomingEvent(context.Background(), userID, msg); processErr != nil {
				_ = c.WriteMessage(
					websocket.TextMessage,
					[]byte(fmt.Sprintf(`{"type":"error","payload":{"message":"%s"}}`, processErr.Error())),
				)
			}
		}
	}))

	// 4. Запуск
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := app.Listen(":" + port); err != nil {
		slog.Error("fiber listen failed", "port", port, "error", err)
		os.Exit(1)
	}
}

type CreateGuildDTO struct {
	Name string `json:"name" validate:"required,min=3,max=32"`
}

type JoinGuildDTO struct {
	InviteCode string `json:"invite_code" validate:"required,min=6,max=64"`
}
