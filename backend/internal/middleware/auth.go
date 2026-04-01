package middleware

import (
	"lumen/pkg/apierr"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func JWTProtected(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if secret == "" {
			return apierr.Write(c, fiber.StatusInternalServerError, "jwt_not_configured", "JWT secret is not configured")
		}

		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return apierr.Write(c, fiber.StatusUnauthorized, "missing_auth_header", "Missing Authorization header")
		}

		tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if tokenString == "" || tokenString == authHeader {
			return apierr.Write(c, fiber.StatusUnauthorized, "invalid_bearer_format", "Invalid Bearer token format")
		}

		claims, err := ParseJWTFromString(secret, tokenString)
		if err != nil {
			return apierr.Write(c, fiber.StatusUnauthorized, "invalid_token", err.Error())
		}
		c.Locals("user", claims)
		return c.Next()
	}
}
