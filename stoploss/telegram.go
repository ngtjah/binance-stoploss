package stoploss

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type Telegram struct {
	tlgToken string
	chatID   int64
	logger   *log.Logger
}

// Create Telegram instance
func NewTelegram(telegramToken string, channelID int64, logger *log.Logger) *Telegram {
	return &Telegram{telegramToken, channelID, logger}
}

// Send message
func (telegram *Telegram) Send(message string) {
	telegram.logger.Println(message)

	if telegram.tlgToken == "" {
		return
	}

	bot, err := tgbotapi.NewBotAPI(telegram.tlgToken)
	if err != nil {
		telegram.logger.Printf("Cannot connect to Telegram:", err.Error())

		return
	}

	msg := tgbotapi.NewMessage(telegram.chatID, message)

	_, err = bot.Send(msg)

	if err != nil {
		telegram.logger.Printf("Cannot send message to telegram:", err.Error())
	}
}
