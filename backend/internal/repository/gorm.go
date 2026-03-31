package repository

import (
	"lumen/internal/config"
	"lumen/internal/domain"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB(cfg config.DBConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Keep domain imports alive for compile-time model checks.
	_ = domain.User{}
	_ = domain.Guild{}
	_ = domain.GuildMember{}
	_ = domain.Channel{}
	_ = domain.Message{}
	_ = domain.Attachment{}

	return db, nil
}
