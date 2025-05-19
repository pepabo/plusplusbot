package repository

import (
	"fmt"
	"os"
	"testing"

	"log/slog"
	"plusplusbot/infra/config"
)

func TestNewRepository(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	tests := []struct {
		name        string
		config      *config.Config
		wantType    string
		wantErr     bool
		skipIfNoEnv string
	}{
		{
			name: "SQLite repository",
			config: &config.Config{
				RepositoryType: config.SQLiteRepository,
				SQLiteDBPath:   ":memory:",
			},
			wantType: "*repository.SQLiteRepository",
			wantErr:  false,
		},
		{
			name: "DynamoDB repository",
			config: &config.Config{
				RepositoryType:    config.DynamoDBRepository,
				DynamoDBTableName: "test_table",
				DynamoDBLocal:     true,
			},
			wantType:    "*repository.DynamoDBRepository",
			wantErr:     false,
			skipIfNoEnv: "DYNAMO_LOCAL",
		},
		{
			name: "Unsupported repository type",
			config: &config.Config{
				RepositoryType: "unsupported",
			},
			wantType: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipIfNoEnv != "" && os.Getenv(tt.skipIfNoEnv) == "" {
				t.Skipf("Skipping test: %s not set", tt.skipIfNoEnv)
			}

			repo, err := NewRepository(tt.config, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRepository() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				gotType := fmt.Sprintf("%T", repo)
				if gotType != tt.wantType {
					t.Errorf("NewRepository() = %v, want %v", gotType, tt.wantType)
				}
			}
		})
	}
}
