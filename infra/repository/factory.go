package repository

import (
	"fmt"
	"log/slog"

	"plusplusbot/infra/config"
)

// NewRepository creates a new repository based on the configuration
func NewRepository(cfg *config.Config, logger *slog.Logger) (UserPointsRepository, error) {
	switch cfg.RepositoryType {
	case config.SQLiteRepository:
		return NewSQLiteRepository(cfg.SQLiteDBPath, logger)
	case config.DynamoDBRepository:
		return NewDynamoDBRepository(cfg.DynamoDBTableName, cfg.DynamoDBLocal, logger)
	default:
		return nil, fmt.Errorf("unsupported repository type: %s", cfg.RepositoryType)
	}
}
