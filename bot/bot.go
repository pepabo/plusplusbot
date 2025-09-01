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

// detectPointOperation checks if the message contains a point operation (++, --, ==)
func (b *Bot) detectPointOperation(text string) PointOperation {
	// Support both user mentions and emoji patterns
	// User mention patterns: <@U123456>++
	userPlusPattern := `.*<@[A-Z0-9]+>[ 　]*\+\+.*`
	userMinusPattern := `.*<@[A-Z0-9]+>[ 　]*\-\-.*`
	userEqualsPattern := `.*<@[A-Z0-9]+>[ 　]*\=\=.*`
	
	// Emoji patterns: :emoji: ++
	emojiPlusPattern := `.*:[a-zA-Z0-9_+-]+:[ 　]*\+\+.*`
	emojiMinusPattern := `.*:[a-zA-Z0-9_+-]+:[ 　]*\-\-.*`
	emojiEqualsPattern := `.*:[a-zA-Z0-9_+-]+:[ 　]*\=\=.*`

	// Check for plus patterns
	userPlusMatched, err := regexp.MatchString(userPlusPattern, text)
	if err != nil {
		b.logger.Error("Error matching user plus pattern", "error", err)
		return NoOperation
	}
	emojiPlusMatched, err := regexp.MatchString(emojiPlusPattern, text)
	if err != nil {
		b.logger.Error("Error matching emoji plus pattern", "error", err)
		return NoOperation
	}
	if userPlusMatched || emojiPlusMatched {
		return PointUp
	}

	// Check for minus patterns
	userMinusMatched, err := regexp.MatchString(userMinusPattern, text)
	if err != nil {
		b.logger.Error("Error matching user minus pattern", "error", err)
		return NoOperation
	}
	emojiMinusMatched, err := regexp.MatchString(emojiMinusPattern, text)
	if err != nil {
		b.logger.Error("Error matching emoji minus pattern", "error", err)
		return NoOperation
	}
	if userMinusMatched || emojiMinusMatched {
		return PointDown
	}

	// Check for equals patterns
	userEqualsMatched, err := regexp.MatchString(userEqualsPattern, text)
	if err != nil {
		b.logger.Error("Error matching user equals pattern", "error", err)
		return NoOperation
	}
	emojiEqualsMatched, err := regexp.MatchString(emojiEqualsPattern, text)
	if err != nil {
		b.logger.Error("Error matching emoji equals pattern", "error", err)
		return NoOperation
	}
	if userEqualsMatched || emojiEqualsMatched {
		return PointCheck
	}

	return NoOperation
}

func extractUserID(text string) string {
	re := regexp.MustCompile(`<@([A-Z0-9]+)>`)
	matches := re.FindStringSubmatch(text)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

// extractEmojiName extracts emoji name from text (e.g., ":sake:" -> "sake")
func extractEmojiName(text string) string {
	re := regexp.MustCompile(`:([a-zA-Z0-9_+-]+):`)
	matches := re.FindStringSubmatch(text)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

// extractTargetFromText extracts either user ID or emoji name from text
func extractTargetFromText(text string) (string, bool) {
	// Try to extract user ID first
	userID := extractUserID(text)
	if userID != "" {
		return userID, true // true indicates it's a user ID
	}
	
	// If no user ID, try to extract emoji name
	emojiName := extractEmojiName(text)
	if emojiName != "" {
		return emojiName, false // false indicates it's an emoji name
	}
	
	return "", false
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
func (b *Bot) handlePointChangeMessage(ev *slackevents.MessageEvent, operation PointOperation) {
	// Extract target (user ID or emoji name) from the message
	target, isUser := extractTargetFromText(ev.Text)
	if target == "" {
		b.logger.Error("No target found in message")
		return
	}

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

func (b *Bot) handlePointCheckMessage(ev *slackevents.MessageEvent) {
	// Extract target (user ID or emoji name) from the message
	target, isUser := extractTargetFromText(ev.Text)
	if target == "" {
		b.logger.Error("No target found in message")
		return
	}

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
	operation := b.detectPointOperation(ev.Text)
	if operation != NoOperation {
		b.logger.Info("Point operation detected", "text", ev.Text)
		if operation == PointCheck {
			b.handlePointCheckMessage(ev)
		} else {
			b.handlePointChangeMessage(ev, operation)
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

			b.socketClient.Ack(*evt.Request)

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
