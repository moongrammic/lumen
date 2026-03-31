package main

import (
	"log"
	"lumen/internal/middleware"
	"lumen/internal/repository"
	"lumen/internal/ws"
	"os"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// 1. Инициализация БД
	_, err := repository.InitDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	app := fiber.New(fiber.Config{
		AppName: "Lumen API v1.0",
	})

	// 2. Middlewares
	app.Use(logger.New())
	app.Use(recover.New())

	// 3. Роуты
	api := app.Group("/api")
	hub := ws.NewHub()
	go hub.Run()

	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "db": "connected"})
	})

	api.Get("/me", middleware.JWTProtected(), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"user": c.Locals("user")})
	})

	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		hub.HandleConnection(c)
	}))

	// 4. Запуск
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(app.Listen(":" + port))
}