package bot

import (
	"context"
	"log/slog"

	"plusplusbot/infra/repository"
)

// Database interface defines the methods for database operations
// This is kept for backward compatibility
type Database interface {
	AddPoints(userID string, points int, is_user bool) error
	GetPoints(userID string) (int, error)
}

// RepositoryAdapter adapts the UserPointsRepository to the Database interface
type RepositoryAdapter struct {
	repo   repository.UserPointsRepository
	logger *slog.Logger
}

// NewRepositoryAdapter creates a new RepositoryAdapter
func NewRepositoryAdapter(repo repository.UserPointsRepository, logger *slog.Logger) *RepositoryAdapter {
	return &RepositoryAdapter{
		repo:   repo,
		logger: logger,
	}
}

// AddPoints adds points to a user
func (a *RepositoryAdapter) AddPoints(userID string, points int, is_user bool) error {
	ctx := context.Background()
	return a.repo.AddPoints(ctx, userID, points, is_user)
}

// GetPoints gets the current points for a user
func (a *RepositoryAdapter) GetPoints(userID string) (int, error) {
	ctx := context.Background()
	return a.repo.GetPoints(ctx, userID)
}
