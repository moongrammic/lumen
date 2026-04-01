package main

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func main() {
	// Берем секрет из переменных окружения контейнера
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "your_super_secret_key" // fallback если переменная не подцепилась
	}

	userID := uuid.New().String()
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		panic(err)
	}
	fmt.Println("\n--- ВАШ ТОКЕН ---")
	fmt.Println(signed)
	fmt.Println("-----------------\n")
}
