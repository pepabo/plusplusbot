package bot

import (
	"os"
	"testing"

	"log/slog"
)

func setupTestDB(t *testing.T) (*SQLiteDB, func()) {
	// Create a temporary file for the test database
	tempFile, err := os.CreateTemp("", "plusplusbot-test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempFile.Close() // Close the file as SQLite will open it

	// Create a logger that only shows error level logs
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	db, err := DatabaseNew(tempFile.Name(), logger)
	if err != nil {
		os.Remove(tempFile.Name())
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Return cleanup function
	cleanup := func() {
		db.db.Close()
		os.Remove(tempFile.Name())
	}

	return db, cleanup
}

func TestAddPoints(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name    string
		userID  string
		points  int
		isUser  bool
		wantErr bool
	}{
		{
			name:    "Add points to new user",
			userID:  "user1",
			points:  10,
			isUser:  true,
			wantErr: false,
		},
		{
			name:    "Add points to existing user",
			userID:  "user1",
			points:  5,
			isUser:  true,
			wantErr: false,
		},
		{
			name:    "Add points to bot",
			userID:  "bot1",
			points:  3,
			isUser:  false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.AddPoints(tt.userID, tt.points, tt.isUser)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddPoints() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify points were added correctly
			got, err := db.GetPoints(tt.userID)
			if err != nil {
				t.Errorf("GetPoints() error = %v", err)
			}

			// For the first test case, points should be 10
			// For the second test case, points should be 15 (10 + 5)
			// For the third test case, points should be 3
			expected := tt.points
			if tt.userID == "user1" && tt.points == 5 {
				expected = 15
			}

			if got != expected {
				t.Errorf("GetPoints() = %v, want %v", got, expected)
			}
		})
	}
}

func TestGetPoints(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Test getting points for non-existent user
	points, err := db.GetPoints("nonexistent")
	if err != nil {
		t.Errorf("GetPoints() error = %v", err)
	}
	if points != 0 {
		t.Errorf("GetPoints() = %v, want 0", points)
	}

	// Add points to a user and verify
	err = db.AddPoints("user1", 20, true)
	if err != nil {
		t.Errorf("AddPoints() error = %v", err)
	}

	points, err = db.GetPoints("user1")
	if err != nil {
		t.Errorf("GetPoints() error = %v", err)
	}
	if points != 20 {
		t.Errorf("GetPoints() = %v, want 20", points)
	}
}
