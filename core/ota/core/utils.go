package core

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

func FloatPtr(f float64) *float64 {
	return &f
}

func GetEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func SendTelegramMessage(psn string) {
	urlStr := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", TelegramBotToken)

	res, err := http.PostForm(urlStr, url.Values{
		"chat_id":    {TelegramChatID},
		"text":       {psn},
		"parse_mode": {"Markdown"},
	})

	if err != nil {
		HndlErr("Error sending Telegram message", err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		dt, _ := io.ReadAll(res.Body)
		HndlErr("Failed to send Telegram message", fmt.Errorf("status: %s, response: %s", res.Status, string(dt)))
	}
}

func LoadInitialFiles(dr string) {
	FileMutex.Lock()
	defer FileMutex.Unlock()

	ent, err := os.ReadDir(dr)
	if err != nil {
		HndlErr(fmt.Sprintf("could not read upload directory %s", dr), err)
		return
	}

	for _, e := range ent {
		if !e.IsDir() {
			inf, err := e.Info()
			if err == nil {
				nm := inf.Name()
				FileInfos[nm] = FileInfo{
					Name:       nm,
					URL:        "/files/" + nm,
					UploadTime: inf.ModTime(),
					Size:       inf.Size(),
				}
			}
		}
	}
}
