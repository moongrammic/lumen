package apierr

import "github.com/gofiber/fiber/v2"

type Response struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func Write(c *fiber.Ctx, status int, code string, message string) error {
	return c.Status(status).JSON(Response{
		Error:   code,
		Message: message,
	})
}
