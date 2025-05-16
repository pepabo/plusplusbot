package main

import (
	"log/slog"
	"os"

	"plusplusbot/bot"
)

func main() {
	botToken := os.Getenv("SLACK_BOT_TOKEN")
	appToken := os.Getenv("SLACK_APP_TOKEN")
	dbConnStr := os.Getenv("DATABASE_URL")

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

	if dbConnStr == "" {
		logger.Error("DATABASE_URL environment variable is not set")
		os.Exit(1)
	}

	verbose := os.Getenv("DEBUG") != ""
	bot, err := bot.New(botToken, appToken, dbConnStr, verbose, logger)
	if err != nil {
		logger.Error("Failed to create bot", "error", err)
		os.Exit(1)
	}

	bot.Start()
}
