package repository

import (
	"context"
)

// UserPointsRepository defines the interface for user points storage operations
type UserPointsRepository interface {
	// AddPoints adds points to a user
	AddPoints(ctx context.Context, userID string, points int, isUser bool) error

	// GetPoints gets the current points for a user
	GetPoints(ctx context.Context, userID string) (int, error)

	// Close closes the repository connection
	Close() error
}
