package bot

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"plusplusbot/infra/repository"
	"regexp"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

// logAdapter converts slog.Logger to log.Logger
type logAdapter struct {
	logger *slog.Logger
}

func (l *logAdapter) Write(p []byte) (n int, err error) {
	l.logger.Debug(string(p))
	return len(p), nil
}

// Bot represents a Slack bot instance
type Bot struct {
	api          *slack.Client
	socketClient *socketmode.Client
	verbose      bool
	logger       *slog.Logger
	repo         repository.UserPointsRepository
}

// New creates a new Slack bot instance
func New(botToken, appToken string, repo repository.UserPointsRepository, verbose bool, logger *slog.Logger) (*Bot, error) {
	if botToken == "" || appToken == "" {
		return nil, fmt.Errorf("SLACK_BOT_TOKEN or SLACK_APP_TOKEN is not set")
	}

	api := slack.New(
		botToken,
		slack.OptionAppLevelToken(appToken),
	)

	// Create log adapter for socketmode
	adapter := &logAdapter{logger: logger}
	socketLogger := log.New(adapter, "socketmode: ", log.Lshortfile|log.LstdFlags)

	socketClient := socketmode.New(
		api,
		socketmode.OptionDebug(verbose),
		socketmode.OptionLog(socketLogger),
	)

	return &Bot{
		api:          api,
		socketClient: socketClient,
		verbose:      verbose,
		logger:       logger,
		repo:         repo,
	}, nil
}

// Start starts the Slack bot
func (b *Bot) Start() {
	b.logger.Debug("Starting bot(version: " + Version + ")...")
	go b.handleEvents()
	b.logger.Debug("Starting socket mode client...")
	if err := b.socketClient.Run(); err != nil {
		b.logger.Error("Error running socket client", "error", err)
	}
}

type PointOperation int

const (
	NoOperation PointOperation = iota
	PointUp
	PointDown
	PointCheck
)

// Pre-compiled regexes for detecting point operations with targets
var (
	// User mention pattern: <@U123456> ++ (captures user ID and operator)
	userOperationPattern = regexp.MustCompile(`<@([A-Z0-9]+)>[ 　]*(\+\+|-{2}|={2})`)
	// Emoji pattern: :emoji: ++ (captures emoji name and operator)
	emojiOperationPattern = regexp.MustCompile(`:([a-zA-Z0-9_+-]+):[ 　]*(\+\+|-{2}|={2})`)
)

// parseOperator converts an operator string to a PointOperation
func parseOperator(op string) PointOperation {
	switch op {
	case "++":
		return PointUp
	case "--":
		return PointDown
	case "==":
		return PointCheck
	default:
		return NoOperation
	}
}

// detectOperationAndTarget detects the point operation and extracts the target in one step.
// This ensures the detected operation is associated with the correct target.
func detectOperationAndTarget(text string) (PointOperation, string, bool) {
	// User mentions have priority
	if matches := userOperationPattern.FindStringSubmatch(text); len(matches) >= 3 {
		return parseOperator(matches[2]), matches[1], true
	}
	if matches := emojiOperationPattern.FindStringSubmatch(text); len(matches) >= 3 {
		return parseOperator(matches[2]), matches[1], false
	}
	return NoOperation, "", false
}

// detectPointOperation checks if the message contains a point operation (++, --, ==)
func (b *Bot) detectPointOperation(text string) PointOperation {
	op, _, _ := detectOperationAndTarget(text)
	return op
}

// isUser checks if the given user ID belongs to a user
func (b *Bot) isUser(userID string) (bool, error) {
	user, err := b.api.GetUserInfo(userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user info: %w", err)
	}
	b.logger.Debug("User info", "user", user)
	return !user.IsBot, nil
}

// handlePointChangeMessage processes a point up or down message
func (b *Bot) handlePointChangeMessage(ev *slackevents.MessageEvent, operation PointOperation, target string, isUser bool) {

	// Check if user is trying to point themselves (only applies to user targets)
	if isUser && target == ev.User {
		message := getFormattedMessage(SelfMessage, target, 0, true)
		_, _, err := b.api.PostMessage(ev.Channel, slack.MsgOptionText(message, false), slack.MsgOptionTS(ev.ThreadTimeStamp))
		if err != nil {
			b.logger.Error("Error sending message", "error", err)
		}
		return
	}

	// For user targets, check if they are bots
	is_user_target := false
	if isUser {
		var err error
		is_user_target, err = b.isUser(target)
		if err != nil {
			b.logger.Error("Error checking if user is bot", "error", err)
			return
		}
	} else {
		// For emoji targets, treat as non-user (similar to bot behavior)
		is_user_target = false
	}

	pointsChange := 1
	if operation == PointDown {
		pointsChange = -1
	}

	// Add points to the target
	ctx := context.Background()
	if err := b.repo.AddPoints(ctx, target, pointsChange, is_user_target); err != nil {
		b.logger.Error("Error adding points", "error", err)
		return
	}

	// Get current points
	points, err := b.repo.GetPoints(ctx, target)
	if err != nil {
		b.logger.Error("Error getting points", "error", err)
		return
	}

	// Send messages
	var message string
	switch operation {
	case PointDown:
		message = getFormattedMessage(MinusPointsMessage, target, points, isUser)
	case PointUp:
		message = getFormattedMessage(PlusPointsMessage, target, points, isUser)
	}

	_, _, err = b.api.PostMessage(ev.Channel, slack.MsgOptionText(message, false), slack.MsgOptionTS(ev.ThreadTimeStamp))
	if err != nil {
		b.logger.Error("Error sending message", "error", err)
		return
	}

	b.logger.Debug("Reply sent", "message", message)
}

func (b *Bot) handlePointCheckMessage(ev *slackevents.MessageEvent, target string, isUser bool) {
	ctx := context.Background()
	points, err := b.repo.GetPoints(ctx, target)
	if err != nil {
		b.logger.Error("Error getting points", "error", err)
		return
	}

	message := getFormattedMessage(EqualsMessage, target, points, isUser)

	_, _, err = b.api.PostMessage(ev.Channel, slack.MsgOptionText(message, false), slack.MsgOptionTS(ev.ThreadTimeStamp))
	if err != nil {
		b.logger.Error("Error sending message", "error", err)
	}
}

// handleMessageEvent processes a message event
func (b *Bot) handleMessageEvent(ev *slackevents.MessageEvent) {
	b.logger.Debug("Received message event", "event", ev)
	operation, target, isUser := detectOperationAndTarget(ev.Text)
	if operation != NoOperation && target != "" {
		b.logger.Info("Point operation detected", "text", ev.Text, "target", target, "isUser", isUser)
		if operation == PointCheck {
			b.handlePointCheckMessage(ev, target, isUser)
		} else {
			b.handlePointChangeMessage(ev, operation, target, isUser)
		}
	}
}

func (b *Bot) handleEvents() {
	for evt := range b.socketClient.Events {
		switch evt.Type {
		case socketmode.EventTypeConnecting:
			b.logger.Debug("Establishing connection with Slack...")
		case socketmode.EventTypeConnectionError:
			b.logger.Error("Connection error", "data", evt.Data)
		case socketmode.EventTypeConnected:
			b.logger.Info("Connection established with Slack")
		case socketmode.EventTypeEventsAPI:
			eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
			if !ok {
				b.logger.Error("Unexpected event type", "data", evt.Data)
				continue
			}

			if err := b.socketClient.Ack(*evt.Request); err != nil {
				b.logger.Error("Failed to acknowledge event", "error", err)
				continue
			}

			switch eventsAPIEvent.Type {
			case slackevents.CallbackEvent:
				innerEvent := eventsAPIEvent.InnerEvent
				switch ev := innerEvent.Data.(type) {
				case *slackevents.MessageEvent:
					b.handleMessageEvent(ev)
				}
			}
		}
	}
}
