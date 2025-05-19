package config

import (
	"os"
)

// RepositoryType represents the type of repository to use
type RepositoryType string

const (
	// SQLiteRepository represents a SQLite repository
	SQLiteRepository RepositoryType = "sqlite"

	// DynamoDBRepository represents a DynamoDB repository
	DynamoDBRepository RepositoryType = "dynamodb"
)

// Config holds the configuration for the repositories
type Config struct {
	// RepositoryType is the type of repository to use
	RepositoryType RepositoryType

	// SQLiteDBPath is the path to the SQLite database
	SQLiteDBPath string

	// DynamoDBTableName is the name of the DynamoDB table
	DynamoDBTableName string

	// DynamoDBLocal indicates whether to use a local DynamoDB instance
	DynamoDBLocal bool
}

// NewConfig creates a new Config instance from environment variables
func NewConfig() *Config {
	repoType := RepositoryType(os.Getenv("REPOSITORY_TYPE"))
	if repoType == "" {
		repoType = SQLiteRepository
	}

	dbPath := os.Getenv("DATABASE_URL")

	tableName := os.Getenv("DYNAMO_USER_POINTS_TABLE")
	if tableName == "" {
		tableName = "user_points"
	}

	dynamoLocal := os.Getenv("DYNAMO_LOCAL") != ""

	return &Config{
		RepositoryType:    repoType,
		SQLiteDBPath:      dbPath,
		DynamoDBTableName: tableName,
		DynamoDBLocal:     dynamoLocal,
	}
}
