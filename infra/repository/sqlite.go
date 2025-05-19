package repository

import (
	"context"
	"database/sql"
	"log/slog"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// SQLiteRepository implements the UserPointsRepository interface using SQLite
type SQLiteRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewSQLiteRepository creates a new SQLiteRepository instance
func NewSQLiteRepository(dbPath string, logger *slog.Logger) (*SQLiteRepository, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	sqliteRepo := &SQLiteRepository{
		db:     db,
		logger: logger,
	}

	// Check if table exists
	var exists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT name FROM sqlite_master
			WHERE type='table' AND name='user_points'
		)
	`).Scan(&exists)
	if err != nil {
		return nil, err
	}

	if !exists {
		logger.Info("Table 'user_points' does not exist. Creating...")
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS user_points (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id TEXT NOT NULL UNIQUE,
				points INTEGER DEFAULT 0,
				is_user BOOLEAN DEFAULT 1,
				last_modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			return nil, err
		}
		logger.Info("Table 'user_points' created successfully")
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return sqliteRepo, nil
}

// AddPoints adds points to a user
func (s *SQLiteRepository) AddPoints(ctx context.Context, userID string, points int, isUser bool) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_points (user_id, points, is_user)
		VALUES (?, ?, ?)
		ON CONFLICT (user_id)
		DO UPDATE SET
			points = points + ?,
			is_user = ?,
			last_modified = CURRENT_TIMESTAMP
	`, userID, points, isUser, points, isUser)
	return err
}

// GetPoints gets the current points for a user
func (s *SQLiteRepository) GetPoints(ctx context.Context, userID string) (int, error) {
	var points int
	err := s.db.QueryRowContext(ctx, "SELECT points FROM user_points WHERE user_id = ?", userID).Scan(&points)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return points, err
}

// Close closes the database connection
func (s *SQLiteRepository) Close() error {
	return s.db.Close()
}
