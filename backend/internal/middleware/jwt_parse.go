package middleware

import (
	"errors"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// ParseJWTFromString parses and validates an HMAC-signed access token (same rules as JWTProtected).
func ParseJWTFromString(secret, tokenString string) (jwt.MapClaims, error) {
	if secret == "" {
		return nil, errors.New("jwt secret not configured")
	}
	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		return nil, errors.New("empty token")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}
