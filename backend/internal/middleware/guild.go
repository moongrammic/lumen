package middleware

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type GuildAccessChecker interface {
	IsMember(ctx context.Context, guildID uint, userID uuid.UUID) (bool, error)
}

func GuildAccess(checker GuildAccessChecker) fiber.Handler {
	return func(c *fiber.Ctx) error {
		guildID64, err := strconv.ParseUint(c.Params("guildID"), 10, 32)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid guild id",
			})
		}

		userID, err := ExtractUserIDFromClaims(c.Locals("user"))
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		ok, err := checker.IsMember(c.UserContext(), uint(guildID64), userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to check guild access",
			})
		}
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "access denied",
			})
		}

		return c.Next()
	}
}
