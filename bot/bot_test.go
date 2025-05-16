package bot

import (
	"os"
	"testing"

	"log/slog"
)

func setupTestBot(t *testing.T) (*Bot, func()) {
	// Create a temporary file for the test database
	tempFile, err := os.CreateTemp("", "plusplusbot-test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Create a logger that only shows error level logs
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// Create a test bot with dummy tokens
	bot, err := New("dummy-bot-token", "dummy-app-token", tempFile.Name(), false, logger)
	if err != nil {
		if err := os.Remove(tempFile.Name()); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
		t.Fatalf("Failed to create test bot: %v", err)
	}

	// Return cleanup function
	cleanup := func() {
		if err := bot.db.(*SQLiteDB).db.Close(); err != nil {
			t.Logf("Failed to close database: %v", err)
		}
		if err := os.Remove(tempFile.Name()); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}

	return bot, cleanup
}

func TestNew(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	tests := []struct {
		name      string
		botToken  string
		appToken  string
		dbConnStr string
		verbose   bool
		wantErr   bool
	}{
		{
			name:      "Valid tokens",
			botToken:  "valid-bot-token",
			appToken:  "valid-app-token",
			dbConnStr: ":memory:",
			verbose:   false,
			wantErr:   false,
		},
		{
			name:      "Empty bot token",
			botToken:  "",
			appToken:  "valid-app-token",
			dbConnStr: ":memory:",
			verbose:   false,
			wantErr:   true,
		},
		{
			name:      "Empty app token",
			botToken:  "valid-bot-token",
			appToken:  "",
			dbConnStr: ":memory:",
			verbose:   false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bot, err := New(tt.botToken, tt.appToken, tt.dbConnStr, tt.verbose, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && bot == nil {
				t.Error("New() returned nil bot when no error was expected")
			}
		})
	}
}

func TestDetectPointOperation(t *testing.T) {
	bot, cleanup := setupTestBot(t)
	defer cleanup()

	tests := []struct {
		name string
		text string
		want PointOperation
	}{
		{
			name: "Point up",
			text: "<@U123456>++",
			want: PointUp,
		},
		{
			name: "Point down",
			text: "<@U123456>--",
			want: PointDown,
		},
		{
			name: "Point check",
			text: "<@U123456>==",
			want: PointCheck,
		},
		{
			name: "No operation",
			text: "Hello world",
			want: NoOperation,
		},
		{
			name: "Invalid format",
			text: "<@U123456>+",
			want: NoOperation,
		},
		{
			name: "Invalid format with newline",
			text: "<@U123456>\n++",
			want: NoOperation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bot.detectPointOperation(tt.text)
			if got != tt.want {
				t.Errorf("detectPointOperation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractUserID(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "Valid user ID",
			text: "<@U123456>++",
			want: "U123456",
		},
		{
			name: "No user ID",
			text: "Hello world",
			want: "",
		},
		{
			name: "Invalid format",
			text: "<U123456>++",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractUserID(tt.text)
			if got != tt.want {
				t.Errorf("extractUserID() = %v, want %v", got, tt.want)
			}
		})
	}
}
