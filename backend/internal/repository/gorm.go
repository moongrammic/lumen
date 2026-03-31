package repository

import (
	"fmt"
	"lumen/internal/domain"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB() (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=5432 sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Автомиграция сущностей
	err = db.AutoMigrate(&domain.User{}, &domain.Guild{}, &domain.GuildMember{}, &domain.Channel{}, &domain.Message{})
	return db, err
}
