package middleware

import (
	"context"
	"lumen/pkg/apierr"
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
			return apierr.Write(c, fiber.StatusBadRequest, "invalid_guild_id", "Guild ID has invalid format")
		}

		userID, err := ExtractUserIDFromClaims(c.Locals("user"))
		if err != nil {
			return apierr.Write(c, fiber.StatusUnauthorized, "invalid_token_claims", err.Error())
		}

		ok, err := checker.IsMember(c.UserContext(), uint(guildID64), userID)
		if err != nil {
			return apierr.Write(c, fiber.StatusInternalServerError, "guild_access_check_failed", "Failed to verify guild membership")
		}
		if !ok {
			return apierr.Write(c, fiber.StatusForbidden, "guild_access_denied", "You are not a member of this guild")
		}

		return c.Next()
	}
}
