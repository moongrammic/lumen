package main

import (
	"errors"
	"fmt"
	"log/slog"
	"lumen/internal/config"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if len(os.Args) < 2 {
		slog.Error("missing migration command", "usage", "go run ./cmd/migrate <up|down|force>")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	m, err := migrate.New("file://migrations", "postgres://"+pgConn(cfg)+"/"+cfg.DB.Name+"?sslmode="+cfg.DB.SSLMode)
	if err != nil {
		slog.Error("failed to initialize migrate", "error", err)
		os.Exit(1)
	}
	defer func() {
		_, _ = m.Close()
	}()

	cmd := os.Args[1]
	switch cmd {
	case "up":
		err = m.Up()
	case "down":
		err = m.Down()
	case "force":
		if len(os.Args) < 3 {
			slog.Error("missing version for force", "usage", "go run ./cmd/migrate force <version>")
			os.Exit(1)
		}
		var version int
		_, scanErr := fmt.Sscanf(os.Args[2], "%d", &version)
		if scanErr != nil {
			slog.Error("invalid version", "error", scanErr)
			os.Exit(1)
		}
		err = m.Force(version)
	default:
		slog.Error("unsupported migration command", "command", cmd)
		os.Exit(1)
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		slog.Error("migration command failed", "command", cmd, "error", err)
		os.Exit(1)
	}

	slog.Info("migration command completed", "command", cmd)
}

func pgConn(cfg *config.Config) string {
	return fmt.Sprintf("%s:%s@%s:%d", cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port)
}
