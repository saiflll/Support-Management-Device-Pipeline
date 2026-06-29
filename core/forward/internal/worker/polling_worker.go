package worker

import (
	"IoTT/internal/logger"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// retryPipelineClients menyimpan referensi MQTT client per pipeline untuk re-publish.
// Diisi oleh forwarder saat pipeline diinisialisasi.
var retryPipelineClients = make(map[int]mqtt.Client)
var retryClientRegister = make(chan retryClientEntry, 10)

type retryClientEntry struct {
	PipelineID int
	Client     mqtt.Client
}

// RegisterRetryClient mendaftarkan MQTT client pipeline agar bisa dipakai RetryWorker.
// Dipanggil oleh forwarder saat pipeline baru dimulai.
func RegisterRetryClient(pipelineID int, client mqtt.Client) {
	retryClientRegister <- retryClientEntry{PipelineID: pipelineID, Client: client}
}

// StartRetryWorker memulai goroutine yang periodik mencoba re-publish payload yang gagal.
// Interval: setiap 2 menit.
func StartRetryWorker() {
	go func() {
		// Proses registrasi client yang masuk
		go func() {
			for entry := range retryClientRegister {
				retryPipelineClients[entry.PipelineID] = entry.Client
			}
		}()

		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		logger.Lg("✅ RetryWorker dimulai: akan coba re-publish failed batches setiap 2 menit.")

		for range ticker.C {
			runRetry()
		}
	}()
}

// runRetry mencoba re-publish semua batch yang tersimpan di tabel failed_batch.
func runRetry() {
	batches, err := LoadFailedBatches()
	if err != nil {
		logger.HndlErr("RetryWorker.LoadFailedBatches", err)
		return
	}

	if len(batches) == 0 {
		return
	}

	logger.Lg("🔄 RetryWorker: ditemukan %d batch gagal, mencoba re-publish...", len(batches))

	for _, b := range batches {
		client, ok := retryPipelineClients[b.PipelineID]
		if !ok || client == nil {
			logger.Lg("⚠️ RetryWorker: tidak ada client untuk pipeline %d, skip.", b.PipelineID)
			IncrementRetryCount(b.ID)
			continue
		}

		if !client.IsConnected() {
			logger.Lg("⚠️ RetryWorker: client pipeline %d tidak terhubung, skip.", b.PipelineID)
			IncrementRetryCount(b.ID)
			continue
		}

		token := client.Publish(b.DestTopic, 1, false, b.Payload)
		if token.WaitTimeout(10*time.Second) && token.Error() != nil {
			logger.HndlErr(fmt.Sprintf("RetryWorker.Publish pipeline %d", b.PipelineID), token.Error())
			IncrementRetryCount(b.ID)
		} else {
			logger.Lg("✅ RetryWorker: batch %d (pipeline %d) berhasil di-re-publish. Menghapus dari DB.", b.ID, b.PipelineID)
			DeleteFailedBatch(b.ID)
		}
	}
}
