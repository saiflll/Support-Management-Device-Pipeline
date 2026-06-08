package telegram

import (
	timed "IoTT/internal/time"
	"crypto/md5"
	"fmt"
	"log"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	bot              *tgbotapi.BotAPI
	lastMessageTime  time.Time
	messageMutex     sync.Mutex
	lastMessageHash  string
	lastSameHashTime time.Time
)

const (
	messageInterval     = 2 * time.Second // Anti-spam interval untuk pesan berbeda
	sameMessageInterval = 5 * time.Minute // Interval untuk pesan yang sama
)

func InitBot() error {
	if BotToken == "" || ChatID == 0 {
		return nil
	}

	var err error
	bot, err = tgbotapi.NewBotAPI(BotToken)
	if err != nil {
		return fmt.Errorf("gagal membuat instance bot Telegram: %w", err)
	}

	lastMessageTime = timed.Now().Add(-messageInterval)
	lastSameHashTime = timed.Now().Add(-sameMessageInterval)
	return nil
}

// hashMessage creates a simple hash of the message for comparison
func hashMessage(text string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(text)))
}

func SendAlert(messageText string) {
	if bot == nil || ChatID == 0 {
		return
	}

	go func(msgTxt string) {
		messageMutex.Lock()
		defer messageMutex.Unlock()

		currentHash := hashMessage(msgTxt)
		now := timed.Now()

		// Jika pesan sama dengan pesan terakhir
		if currentHash == lastMessageHash {
			// Cek apakah sudah lewat 5 menit sejak pesan sama terakhir dikirim
			if now.Sub(lastSameHashTime) < sameMessageInterval {
				log.Printf("⏭️ Notifikasi yang sama diabaikan (tunggu %v lagi)",
					sameMessageInterval-now.Sub(lastSameHashTime))
				return
			}
			log.Printf("⏰ 5 menit telah berlalu, mengirim notifikasi yang sama lagi")
		} else {
			// Pesan berbeda, reset timer
			lastMessageHash = currentHash
			log.Printf("📨 Notifikasi baru terdeteksi")
		}

		// Rate limiting untuk menghindari spam
		if now.Sub(lastMessageTime) < messageInterval {
			time.Sleep(messageInterval - now.Sub(lastMessageTime))
		}

		msg := tgbotapi.NewMessage(ChatID, msgTxt)
		msg.ParseMode = tgbotapi.ModeMarkdown
		msg.DisableWebPagePreview = true

		if _, err := bot.Send(msg); err != nil {
			log.Printf("❌ Error mengirim pesan Telegram: %v", err)
		} else {
			log.Printf("✅ Notifikasi Telegram terkirim")
			lastSameHashTime = now
		}

		lastMessageTime = now
	}(messageText)
}
