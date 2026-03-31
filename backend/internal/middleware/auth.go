package middleware

import (
	"lumen/pkg/apierr"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
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

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "unexpected signing method")
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			return apierr.Write(c, fiber.StatusUnauthorized, "invalid_token", "Invalid or expired token")
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Locals("user", claims)
		}

		return c.Next()
	}
}
