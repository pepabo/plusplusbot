package repository

import (
	"context"
	"fmt"
	"os"
	"time"

	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/guregu/dynamo/v2"
)

// UserPoints represents a user's points in DynamoDB
type UserPoints struct {
	UserID       string    `dynamo:"user_id,hash"`
	Points       int       `dynamo:"points"`
	IsUser       bool      `dynamo:"is_user"`
	LastModified time.Time `dynamo:"last_modified"`
}

// DynamoDBRepository implements the UserPointsRepository interface using DynamoDB
type DynamoDBRepository struct {
	db        *dynamo.DB
	tableName string
	logger    *slog.Logger
}

// NewDynamoDBRepository creates a new DynamoDBRepository instance
func NewDynamoDBRepository(tableName string, isLocal bool, logger *slog.Logger) (*DynamoDBRepository, error) {
	var db *dynamo.DB

	if isLocal {
		logger.Info("Using local DynamoDB instance")
		cfg, err := config.LoadDefaultConfig(context.TODO(),
			config.WithRegion("dummy"),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("dummy", "dummy", "dummy")),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load configuration: %v", err)
		}

		db = dynamo.New(cfg, func(o *dynamodb.Options) {
			o.BaseEndpoint = aws.String("http://localhost:8000")
		})

		err = setupDynamoDBSchema(db, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to setup schema: %v", err)
		}
	} else {
		logger.Info("Using AWS DynamoDB service")
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("failed to load configuration: %v", err)
		}

		// Use specific region if provided
		if region := os.Getenv("AWS_REGION"); region != "" {
			cfg.Region = region
		}

		db = dynamo.New(cfg)
	}

	return &DynamoDBRepository{
		db:        db,
		tableName: tableName,
		logger:    logger,
	}, nil
}

// setupDynamoDBSchema creates the DynamoDB table if it doesn't exist
func setupDynamoDBSchema(db *dynamo.DB, tableName string) error {
	t := db.Table(tableName)
	_, err := t.Describe().Run(context.TODO())
	if err != nil {
		input := db.CreateTable(tableName, UserPoints{}).
			Provision(10, 10)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		return input.Run(ctx)
	}
	return nil
}

// AddPoints adds points to a user
func (r *DynamoDBRepository) AddPoints(ctx context.Context, userID string, points int, isUser bool) error {
	// First try to get the current user points
	var userPoints UserPoints
	err := r.db.Table(r.tableName).Get("user_id", userID).One(ctx, &userPoints)

	if err != nil {
		if err != dynamo.ErrNotFound {
			return err
		}

		// User not found, create new record
		userPoints = UserPoints{
			UserID:       userID,
			Points:       points,
			IsUser:       isUser,
			LastModified: time.Now(),
		}

		return r.db.Table(r.tableName).Put(userPoints).Run(ctx)
	}

	// Update existing user
	userPoints.Points += points
	userPoints.IsUser = isUser
	userPoints.LastModified = time.Now()

	return r.db.Table(r.tableName).Put(userPoints).Run(ctx)
}

// GetPoints gets the current points for a user
func (r *DynamoDBRepository) GetPoints(ctx context.Context, userID string) (int, error) {
	var userPoints UserPoints
	err := r.db.Table(r.tableName).Get("user_id", userID).One(ctx, &userPoints)

	if err != nil {
		if err == dynamo.ErrNotFound {
			return 0, nil
		}
		return 0, err
	}

	return userPoints.Points, nil
}

// Close is a no-op for DynamoDB as it doesn't require explicit connection closing
func (r *DynamoDBRepository) Close() error {
	// DynamoDB doesn't require explicit connection closing
	return nil
}
