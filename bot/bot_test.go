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

func TestDetectOperationAndTarget(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		wantOp     PointOperation
		wantTarget string
		wantIsUser bool
	}{
		// Normal cases: user mention
		{
			name:       "User point up",
			text:       "<@U123456>++",
			wantOp:     PointUp,
			wantTarget: "U123456",
			wantIsUser: true,
		},
		{
			name:       "User point down",
			text:       "<@U123456>--",
			wantOp:     PointDown,
			wantTarget: "U123456",
			wantIsUser: true,
		},
		{
			name:       "User point check",
			text:       "<@U123456>==",
			wantOp:     PointCheck,
			wantTarget: "U123456",
			wantIsUser: true,
		},
		{
			name:       "User with space",
			text:       "<@U123456> ++",
			wantOp:     PointUp,
			wantTarget: "U123456",
			wantIsUser: true,
		},
		{
			name:       "User with full-width space",
			text:       "<@U123456>　++",
			wantOp:     PointUp,
			wantTarget: "U123456",
			wantIsUser: true,
		},
		// Normal cases: emoji
		{
			name:       "Emoji point up",
			text:       ":sake: ++",
			wantOp:     PointUp,
			wantTarget: "sake",
			wantIsUser: false,
		},
		{
			name:       "Emoji point down",
			text:       ":sake: --",
			wantOp:     PointDown,
			wantTarget: "sake",
			wantIsUser: false,
		},
		{
			name:       "Emoji point check",
			text:       ":sake: ==",
			wantOp:     PointCheck,
			wantTarget: "sake",
			wantIsUser: false,
		},
		{
			name:       "Emoji no space",
			text:       ":sake:++",
			wantOp:     PointUp,
			wantTarget: "sake",
			wantIsUser: false,
		},
		{
			name:       "Emoji with underscores",
			text:       ":beer_mug: ++",
			wantOp:     PointUp,
			wantTarget: "beer_mug",
			wantIsUser: false,
		},
		{
			name:       "Emoji with numbers",
			text:       ":beer2: ++",
			wantOp:     PointUp,
			wantTarget: "beer2",
			wantIsUser: false,
		},
		{
			name:       "Emoji with hyphens",
			text:       ":haruotsu-no1: ++",
			wantOp:     PointUp,
			wantTarget: "haruotsu-no1",
			wantIsUser: false,
		},
		// Text with characters between target and operator should NOT match
		{
			name:       "Emoji with text before operator (plus)",
			text:       ":haruotsu-no1:hogehogehogege  ++",
			wantOp:     NoOperation,
			wantTarget: "",
			wantIsUser: false,
		},
		{
			name:       "Emoji with text before operator (minus)",
			text:       ":haruotsu-no1:hogehogehogege  --",
			wantOp:     NoOperation,
			wantTarget: "",
			wantIsUser: false,
		},
		{
			name:       "Emoji with text directly before operator",
			text:       ":sake:hogehoge++",
			wantOp:     NoOperation,
			wantTarget: "",
			wantIsUser: false,
		},
		{
			name:       "User with text before operator",
			text:       "<@U123456>hogehoge ++",
			wantOp:     NoOperation,
			wantTarget: "",
			wantIsUser: false,
		},
		// Multiple emoji patterns - should match the correct one
		{
			name:       "Multiple emojis, second one has operator",
			text:       "hello :foo:bar :baz: ++",
			wantOp:     PointUp,
			wantTarget: "baz",
			wantIsUser: false,
		},
		// When both user and emoji exist, the one directly followed by the operator wins
		{
			name:       "Emoji directly before operator takes precedence",
			text:       "<@U123456> :sake: ++",
			wantOp:     PointUp,
			wantTarget: "sake",
			wantIsUser: false,
		},
		{
			name:       "User directly before operator takes precedence",
			text:       ":sake: <@U123456> ++",
			wantOp:     PointUp,
			wantTarget: "U123456",
			wantIsUser: true,
		},
		// No operation
		{
			name:       "No operation",
			text:       "Hello world",
			wantOp:     NoOperation,
			wantTarget: "",
			wantIsUser: false,
		},
		{
			name:       "Newline between target and operator",
			text:       "<@U123456>\n++",
			wantOp:     NoOperation,
			wantTarget: "",
			wantIsUser: false,
		},
		{
			name:       "Invalid emoji format",
			text:       ":sake +",
			wantOp:     NoOperation,
			wantTarget: "",
			wantIsUser: false,
		},
		// Prefix text should be fine
		{
			name:       "Text before emoji operation",
			text:       "nice work :sake: ++",
			wantOp:     PointUp,
			wantTarget: "sake",
			wantIsUser: false,
		},
		{
			name:       "Text before user operation",
			text:       "good job <@U123456> ++",
			wantOp:     PointUp,
			wantTarget: "U123456",
			wantIsUser: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOp, gotTarget, gotIsUser := detectOperationAndTarget(tt.text)
			if gotOp != tt.wantOp {
				t.Errorf("detectOperationAndTarget() op = %v, want %v", gotOp, tt.wantOp)
			}
			if gotTarget != tt.wantTarget {
				t.Errorf("detectOperationAndTarget() target = %v, want %v", gotTarget, tt.wantTarget)
			}
			if gotIsUser != tt.wantIsUser {
				t.Errorf("detectOperationAndTarget() isUser = %v, want %v", gotIsUser, tt.wantIsUser)
			}
		})
	}
}

