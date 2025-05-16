package bot

import (
	"strings"
	"testing"
)

func TestGetFormattedMessage(t *testing.T) {
	tests := []struct {
		name         string
		messageType  MessageType
		userID       string
		points       int
		wantContains []string
	}{
		{
			name:         "PlusPointsMessage",
			messageType:  PlusPointsMessage,
			userID:       "U123456",
			points:       5,
			wantContains: []string{"<@U123456>", "5 points"},
		},
		{
			name:         "MinusPointsMessage",
			messageType:  MinusPointsMessage,
			userID:       "U789012",
			points:       -3,
			wantContains: []string{"<@U789012>", "-3 points"},
		},
		{
			name:         "EqualsMessage",
			messageType:  EqualsMessage,
			userID:       "U345678",
			points:       0,
			wantContains: []string{"<@U345678>", "0 points"},
		},
		{
			name:         "SelfMessage",
			messageType:  SelfMessage,
			userID:       "U901234",
			points:       0,
			wantContains: []string{"<@U901234>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getFormattedMessage(tt.messageType, tt.userID, tt.points)

			// For PlusPointsMessage and MinusPointsMessage, check if there's a reaction
			if tt.messageType == PlusPointsMessage || tt.messageType == MinusPointsMessage {
				// Check if the message contains all required elements
				for _, want := range tt.wantContains {
					if !strings.Contains(got, want) {
						t.Errorf("getFormattedMessage() = %v, want to contain %v", got, want)
					}
				}

				parts := strings.Split(got, " ")
				if len(parts) < 2 {
					t.Errorf("getFormattedMessage() should contain a reaction and a message, got: %v", got)
				}
			}

			// For SelfMessage, check if it's one of the expected messages
			if tt.messageType == SelfMessage {
				validMessages := []string{
					"Nice try, <@U901234>",
					"We've got a cheater over here!",
					"Don't even try me",
					"Great! You now have -∞ points!",
					"<@U901234> has been banned from Slack.",
					"<@U901234>--",
				}
				found := false
				for _, valid := range validMessages {
					if got == valid {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("getFormattedMessage() = %v, want one of %v", got, validMessages)
				}
			}
		})
	}
}
