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

func GetEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func SendTelegramMessage(psn string) {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", TelegramBotToken)

	rsp, err := http.PostForm(apiURL, url.Values{
		"chat_id":    {TelegramChatID},
		"text":       {psn},
		"parse_mode": {"Markdown"},
	})

	if err != nil {
		HndlErr("Error sending Telegram message", err)
		return
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		bdy, _ := io.ReadAll(rsp.Body)
		HndlErr("Failed to send Telegram message", fmt.Errorf("status: %s, response: %s", rsp.Status, string(bdy)))
	}
}

func LoadInitialFiles(dir string) {
	FileMutex.Lock()
	defer FileMutex.Unlock()

	entries, err := os.ReadDir(dir)
	if err != nil {
		HndlErr(fmt.Sprintf("could not read upload directory %s", dir), err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			info, err := entry.Info()
			if err == nil {
				name := info.Name()
				FileInfos[name] = FileInfo{
					Name:       name,
					URL:        "/files/" + name,
					UploadTime: info.ModTime(),
					Size:       info.Size(),
				}
			}
		}
	}
}
