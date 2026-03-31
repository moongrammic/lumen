package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	App      AppConfig
	DB       DBConfig
	JWT      JWTConfig
	Redis    RedisConfig
	LiveKit  LiveKitConfig
	Presence PresenceConfig
}

type AppConfig struct {
	Port string `env:"PORT" env-default:"8080"`
}

type DBConfig struct {
	Host     string `env:"DB_HOST" env-default:"postgres"`
	User     string `env:"DB_USER" env-default:"lumen"`
	Password string `env:"DB_PASSWORD" env-default:"lumen"`
	Name     string `env:"DB_NAME" env-default:"lumen"`
	Port     int    `env:"DB_PORT" env-default:"5432"`
	SSLMode  string `env:"DB_SSLMODE" env-default:"disable"`
}

type JWTConfig struct {
	Secret string `env:"JWT_SECRET" env-required:"true"`
}

type RedisConfig struct {
	Addr       string `env:"REDIS_ADDR" env-default:"redis:6379"`
	Channel    string `env:"REDIS_CHANNEL" env-default:"lumen:chat"`
	Password   string `env:"REDIS_PASSWORD" env-default:""`
	DB         int    `env:"REDIS_DB" env-default:"0"`
	PresenceTT int    `env:"PRESENCE_TTL_SECONDS" env-default:"60"`
}

type LiveKitConfig struct {
	URL       string `env:"LIVEKIT_URL" env-default:""`
	APIKey    string `env:"LIVEKIT_API_KEY" env-default:""`
	APISecret string `env:"LIVEKIT_API_SECRET" env-default:""`
}

type PresenceConfig struct {
	TTL time.Duration
}

func Load() (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}
	cfg.Presence.TTL = time.Duration(cfg.Redis.PresenceTT) * time.Second
	return &cfg, nil
}

func (c DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		c.Host, c.User, c.Password, c.Name, c.Port, c.SSLMode,
	)
}
