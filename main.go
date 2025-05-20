package main

import (
	"log/slog"
	"os"

	"plusplusbot/bot"
	"plusplusbot/infra/config"
	"plusplusbot/infra/repository"
)

func main() {
	botToken := os.Getenv("SLACK_BOT_TOKEN")
	appToken := os.Getenv("SLACK_APP_TOKEN")

	// Initialize logger
	var level slog.Level
	if os.Getenv("DEBUG") != "" {
		level = slog.LevelDebug
	} else {
		level = slog.LevelInfo
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))

	// Load configuration
	cfg := config.NewConfig()

	if cfg.RepositoryType == config.SQLiteRepository && cfg.SQLiteDBPath == "" {
		logger.Error("DATABASE_URL environment variable is not set")
		os.Exit(1)
	}

	// Initialize repository
	repo, err := repository.NewRepository(cfg, logger)
	if err != nil {
		logger.Error("Failed to create repository", "error", err)
		os.Exit(1)
	}

	// Initialize bot
	verbose := os.Getenv("DEBUG") != ""
	bot, err := bot.New(botToken, appToken, repo, verbose, logger)
	if err != nil {
		logger.Error("Failed to create bot", "error", err)
		os.Exit(1)
	}

	bot.Start()
}
