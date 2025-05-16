package bot

import (
	"embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

//go:embed messages.json
var messageFS embed.FS

type Messages struct {
	Plus        []string `json:"plus"`
	PlusPoints  []string `json:"plus_points"`
	Minus       []string `json:"minus"`
	MinusPoints []string `json:"minus_points"`
	Equals      []string `json:"equals"`
	Self        []string `json:"self"`
}

type MessageType int

const (
	PlusPointsMessage MessageType = iota
	MinusPointsMessage
	EqualsMessage
	SelfMessage
)

var (
	messages Messages
	rnd      = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func init() {
	// Load messages from embedded JSON file
	data, err := messageFS.ReadFile("messages.json")
	if err != nil {
		panic(err)
	}

	if err := json.Unmarshal(data, &messages); err != nil {
		panic(err)
	}
}

func getFormattedMessage(messageType MessageType, userID string, points int) string {
	var reaction, template string
	switch messageType {
	case PlusPointsMessage:
		reaction = messages.Plus[rnd.Intn(len(messages.Plus))]
		template = messages.PlusPoints[rnd.Intn(len(messages.PlusPoints))]
	case MinusPointsMessage:
		reaction = messages.Minus[rnd.Intn(len(messages.Minus))]
		template = messages.MinusPoints[rnd.Intn(len(messages.MinusPoints))]
	case EqualsMessage:
		reaction = ""
		template = messages.Equals[rnd.Intn(len(messages.Equals))]
	case SelfMessage:
		reaction = ""
		template = messages.Self[rnd.Intn(len(messages.Self))]
	}

	pointsStr := fmt.Sprintf("%d points", points)
	username := fmt.Sprintf("<@%s>", userID)

	message := strings.ReplaceAll(template, "{thing}", username)
	message = strings.ReplaceAll(message, "{points_string}", pointsStr)
	if reaction != "" {
		message = fmt.Sprintf("%s %s", reaction, message)
	}
	return message
}
