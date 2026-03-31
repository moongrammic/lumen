package middleware

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func ExtractUserIDFromClaims(claims any) (uuid.UUID, error) {
	mapClaims, ok := claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, errors.New("invalid token claims")
	}

	candidates := []string{"sub", "user_id", "id"}
	for _, key := range candidates {
		rawValue, exists := mapClaims[key]
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
