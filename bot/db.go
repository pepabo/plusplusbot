package bot

import (
	"database/sql"
	"log/slog"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// Database interface defines the methods for database operations
type Database interface {
	AddPoints(userID string, points int, is_user bool) error
	GetPoints(userID string) (int, error)
}

// SQLiteDB implements the Database interface
type SQLiteDB struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewSQLiteDB creates a new SQLiteDB instance
func DatabaseNew(dbPath string, logger *slog.Logger) (*SQLiteDB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	sqlitedb := &SQLiteDB{
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

	return sqlitedb, nil
}

// AddPoints adds points to a user
func (s *SQLiteDB) AddPoints(userID string, points int, is_user bool) error {
	_, err := s.db.Exec(`
		INSERT INTO user_points (user_id, points, is_user)
		VALUES (?, ?, ?)
		ON CONFLICT (user_id)
		DO UPDATE SET
			points = points + ?,
			is_user = ?,
			last_modified = CURRENT_TIMESTAMP
	`, userID, points, is_user, points, is_user)
	return err
}

// GetPoints gets the current points for a user
func (s *SQLiteDB) GetPoints(userID string) (int, error) {
	var points int
	err := s.db.QueryRow("SELECT points FROM user_points WHERE user_id = ?", userID).Scan(&points)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return points, err
}
