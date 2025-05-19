package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"log/slog"
)

func setupTestDynamoDBRepository(t *testing.T) (*DynamoDBRepository, func()) {
	// Skip test if DYNAMO_LOCAL is not set
	if os.Getenv("DYNAMO_LOCAL") == "" {
		t.Skip("Skipping DynamoDB test: DYNAMO_LOCAL not set")
	}

	// Generate a unique table name for this test
	tableName := "user_points_test_" + time.Now().Format("20060102150405")

	// Create a logger that only shows error level logs
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	repo, err := NewDynamoDBRepository(tableName, true, logger)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	// Return cleanup function
	cleanup := func() {
		// Delete the table
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := repo.db.Table(tableName).DeleteTable().Run(ctx)
		if err != nil {
			t.Logf("Failed to delete test table: %v", err)
		}
	}

	return repo, cleanup
}

func TestDynamoDBAddPoints(t *testing.T) {
	repo, cleanup := setupTestDynamoDBRepository(t)
	defer cleanup()

	ctx := context.Background()

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
			err := repo.AddPoints(ctx, tt.userID, tt.points, tt.isUser)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddPoints() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify points were added correctly
			got, err := repo.GetPoints(ctx, tt.userID)
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

func TestDynamoDBGetPoints(t *testing.T) {
	repo, cleanup := setupTestDynamoDBRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Test getting points for non-existent user
	points, err := repo.GetPoints(ctx, "nonexistent")
	if err != nil {
		t.Errorf("GetPoints() error = %v", err)
	}
	if points != 0 {
		t.Errorf("GetPoints() = %v, want 0", points)
	}

	// Add points to a user and verify
	err = repo.AddPoints(ctx, "user1", 20, true)
	if err != nil {
		t.Errorf("AddPoints() error = %v", err)
	}

	points, err = repo.GetPoints(ctx, "user1")
	if err != nil {
		t.Errorf("GetPoints() error = %v", err)
	}
	if points != 20 {
		t.Errorf("GetPoints() = %v, want 20", points)
	}
}

func TestDynamoDBTableCreation(t *testing.T) {
	// Skip test if DYNAMO_LOCAL is not set
	if os.Getenv("DYNAMO_LOCAL") == "" {
		t.Skip("Skipping DynamoDB test: DYNAMO_LOCAL not set")
	}

	// Generate a unique table name for this test
	tableName := "user_points_test_creation_" + time.Now().Format("20060102150405")

	// Create a logger that only shows error level logs
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	repo, err := NewDynamoDBRepository(tableName, true, logger)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}
	defer func() {
		// Delete the table
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := repo.db.Table(tableName).DeleteTable().Run(ctx)
		if err != nil {
			t.Logf("Failed to delete test table: %v", err)
		}
	}()

	// Check if table exists
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = repo.db.Table(tableName).Describe().Run(ctx)
	if err != nil {
		t.Errorf("Table was not created: %v", err)
	}
}
