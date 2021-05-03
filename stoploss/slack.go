package stoploss

import (
	"log"

	slackwebhookapi "github.com/ashwanthkumar/slack-go-webhook"
)

type Slack struct {
	webhookUrl string
	logger     *log.Logger
}

// Create Slack instance
func NewSlack(slackToken string, logger *log.Logger) *Slack {
	return &Slack{slackToken, logger}
}

// Send message
func (slack *Slack) Send(message string) {
	slack.logger.Println(message)

	if slack.webhookUrl == "" {
		return
	}

	payload := slackwebhookapi.Payload{
		Text:      message,
		Username:  "binance",
		Channel:   "#stoploss",
		IconEmoji: ":monkey_face:",
	}

	err := slackwebhookapi.Send(slack.webhookUrl, "", payload)
	if len(err) > 0 {
		slack.logger.Printf("Cannot send message to Slack: %s\n", err)
	}
}
