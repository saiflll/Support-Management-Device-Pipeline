package telegram

import (
	"log"
	"os"
	"strconv"
)

var (
	BotToken string
	ChatID   int64
)

func LoadConfig() {
	BotToken = os.Getenv("TELE_BOT_ALRT")
	chatIDStr := os.Getenv("TELEGRAM_CHAT_ID")

	if BotToken == "" {
		log.Println("Peringatan: TELE_BOT_ALRT tidak diatur. Fitur notifikasi telegram tidak akan berfungsi.")
	}

	if chatIDStr == "" {
		log.Println("Peringatan: TELEGRAM_CHAT_ID tidak diatur. Fitur notifikasi telegram tidak akan berfungsi.")
	} else {
		var err error
		ChatID, err = strconv.ParseInt(chatIDStr, 10, 64)
		if err != nil {
			log.Fatalf("Error parsing TELEGRAM_CHAT_ID: %v", err)
		}
	}
}
