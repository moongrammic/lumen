package main

import (
	"errors"
	"log"
	"lumen/internal/middleware"
	"lumen/internal/repository"
	"lumen/internal/service"
	"lumen/internal/ws"
	"os"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func main() {
	// 1. Инициализация БД
	db, err := repository.InitDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)

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
	go hub.Run()

	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "db": "connected"})
	})

	api.Get("/me", middleware.JWTProtected(), func(c *fiber.Ctx) error {
		userID, err := extractUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		me, err := userService.GetMe(c.Context(), userID)
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

	app.Get("/ws", limiter.New(limiter.Config{
		Max:        30,
		Expiration: 60 * time.Second,
	}), websocket.New(func(c *websocket.Conn) {
		hub.HandleConnection(c)
	}))

	// 4. Запуск
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(app.Listen(":" + port))
}

func extractUserID(c *fiber.Ctx) (uuid.UUID, error) {
	claims, ok := c.Locals("user").(jwt.MapClaims)
	if !ok {
		return uuid.Nil, errors.New("invalid token claims")
	}

	candidates := []string{"sub", "user_id", "id"}
	for _, key := range candidates {
		rawValue, exists := claims[key]
		if !exists {
			continue
		}
		strValue, ok := rawValue.(string)
		if !ok || strValue == "" {
			continue
		}
		parsed, err := uuid.Parse(strValue)
		if err == nil {
			return parsed, nil
		}
	}

	return uuid.Nil, errors.New("token missing user id")
}