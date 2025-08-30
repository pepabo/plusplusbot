package bot

import (
	"os"
	"testing"

	"log/slog"
	"plusplusbot/infra/repository"
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

	// Create a SQLite repository for testing
	repo, err := repository.NewSQLiteRepository(tempFile.Name(), logger)
	if err != nil {
		if err := os.Remove(tempFile.Name()); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
		t.Fatalf("Failed to create test repository: %v", err)
	}

	// Create a test bot with dummy tokens
	bot, err := New("dummy-bot-token", "dummy-app-token", repo, false, logger)
	if err != nil {
		if err := os.Remove(tempFile.Name()); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
		t.Fatalf("Failed to create test bot: %v", err)
	}

	// Return cleanup function
	cleanup := func() {
		if err := repo.Close(); err != nil {
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

	// Create a real SQLite repository for testing
	repo, err := repository.NewSQLiteRepository(":memory:", logger)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}
	defer func() {
		err := repo.Close()
		if err != nil {
			t.Fatalf("Failed to close test repository: %v", err)
		}
	}()

	tests := []struct {
		name     string
		botToken string
		appToken string
		verbose  bool
		wantErr  bool
	}{
		{
			name:     "Valid tokens",
			botToken: "valid-bot-token",
			appToken: "valid-app-token",
			verbose:  false,
			wantErr:  false,
		},
		{
			name:     "Empty bot token",
			botToken: "",
			appToken: "valid-app-token",
			verbose:  false,
			wantErr:  true,
		},
		{
			name:     "Empty app token",
			botToken: "valid-bot-token",
			appToken: "",
			verbose:  false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bot, err := New(tt.botToken, tt.appToken, repo, tt.verbose, logger)
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
			name: "Emoji point up",
			text: ":sake: ++",
			want: PointUp,
		},
		{
			name: "Emoji point down",
			text: ":sake: --",
			want: PointDown,
		},
		{
			name: "Emoji point check",
			text: ":sake: ==",
			want: PointCheck,
		},
		{
			name: "Emoji with underscores",
			text: ":beer_mug: ++",
			want: PointUp,
		},
		{
			name: "Emoji with numbers",
			text: ":beer2: ++",
			want: PointUp,
		},
		{
			name: "Emoji with spaces",
			text: ":sake:   ++",
			want: PointUp,
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
		{
			name: "Invalid emoji format",
			text: ":sake +",
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

func TestExtractEmojiName(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "Valid emoji",
			text: ":sake: ++",
			want: "sake",
		},
		{
			name: "Emoji with underscores",
			text: ":beer_mug: ++",
			want: "beer_mug",
		},
		{
			name: "Emoji with numbers",
			text: ":beer2: ++",
			want: "beer2",
		},
		{
			name: "No emoji",
			text: "Hello world",
			want: "",
		},
		{
			name: "Invalid emoji format",
			text: ":sake ++",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractEmojiName(tt.text)
			if got != tt.want {
				t.Errorf("extractEmojiName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractTargetFromText(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		wantTarget   string
		wantIsUser   bool
	}{
		{
			name:         "User mention",
			text:         "<@U123456>++",
			wantTarget:   "U123456",
			wantIsUser:   true,
		},
		{
			name:         "Emoji",
			text:         ":sake: ++",
			wantTarget:   "sake",
			wantIsUser:   false,
		},
		{
			name:         "No target",
			text:         "Hello world",
			wantTarget:   "",
			wantIsUser:   false,
		},
		{
			name:         "User mention has priority",
			text:         "<@U123456> :sake: ++",
			wantTarget:   "U123456",
			wantIsUser:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTarget, gotIsUser := extractTargetFromText(tt.text)
			if gotTarget != tt.wantTarget {
				t.Errorf("extractTargetFromText() gotTarget = %v, want %v", gotTarget, tt.wantTarget)
			}
			if gotIsUser != tt.wantIsUser {
				t.Errorf("extractTargetFromText() gotIsUser = %v, want %v", gotIsUser, tt.wantIsUser)
			}
		})
	}
}
