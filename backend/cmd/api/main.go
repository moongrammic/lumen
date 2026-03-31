package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"lumen/internal/middleware"
	"lumen/internal/repository"
	"lumen/internal/service"
	"lumen/internal/ws"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// 1. Инициализация БД
	db, err := repository.InitDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
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
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		me, err := userService.GetMe(c.UserContext(), userID)
		if err != nil {
			if errors.Is(err, service.ErrUserNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "user not found",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to load user",
			})
		}

		return c.JSON(me)
	})

	api.Post("/guilds", middleware.JWTProtected(), func(c *fiber.Ctx) error {
		type createGuildRequest struct {
			Name string `json:"name"`
		}

		var req createGuildRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
		}

		userID, err := middleware.ExtractUserIDFromClaims(c.Locals("user"))
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		guild, err := guildService.Create(c.UserContext(), req.Name, userID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusCreated).JSON(guild)
	})

	api.Post("/guilds/join", middleware.JWTProtected(), func(c *fiber.Ctx) error {
		type joinGuildRequest struct {
			InviteCode string `json:"invite_code"`
		}

		var req joinGuildRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
		}

		userID, err := middleware.ExtractUserIDFromClaims(c.Locals("user"))
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		guild, err := guildService.JoinByInvite(c.UserContext(), req.InviteCode, userID)
		if err != nil {
			if errors.Is(err, service.ErrGuildNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "guild not found"})
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(guild)
	})

	api.Get("/guilds/:guildID/channels/:channelID/messages", middleware.JWTProtected(), middleware.GuildAccess(guildRepo), func(c *fiber.Ctx) error {
		guildID, err := strconv.ParseUint(c.Params("guildID"), 10, 32)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid guild id"})
		}

		channelID, err := strconv.ParseUint(c.Params("channelID"), 10, 32)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid channel id"})
		}

		belongs, err := guildRepo.ChannelBelongsToGuild(c.UserContext(), uint(channelID), uint(guildID))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to verify channel access"})
		}
		if !belongs {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "channel does not belong to guild"})
		}

		var beforeID *uint
		if rawBefore := c.Query("before"); rawBefore != "" {
			parsedBefore, parseErr := strconv.ParseUint(rawBefore, 10, 32)
			if parseErr != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid before cursor"})
			}
			cursor := uint(parsedBefore)
			beforeID = &cursor
		}

		limit := 50
		if rawLimit := c.Query("limit"); rawLimit != "" {
			parsedLimit, parseErr := strconv.Atoi(rawLimit)
			if parseErr != nil || parsedLimit <= 0 {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid limit"})
			}
			limit = parsedLimit
		}

		result, err := chatService.ListMessages(c.UserContext(), uint(channelID), beforeID, limit)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list messages"})
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
	log.Fatal(app.Listen(":" + port))
}
