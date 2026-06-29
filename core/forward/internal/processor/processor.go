package processor

import (
	"IoTT/internal/config"
	"IoTT/internal/database"
	"IoTT/internal/models"
	"IoTT/internal/telegram"
	timed "IoTT/internal/time"
	"IoTT/internal/worker"
	"fmt"
	"log"
	"time"
)

// ProcessSensorData memvalidasi dan mengevaluasi data sensor masuk.
// Fungsi ini TIDAK lagi menulis ke database lokal — tugas itu dialihkan ke forwarder (on-failure).
// Tugas fungsi ini:
//  1. Memvalidasi apakah sensor terdaftar di config
//  2. Memperbarui status operasional sensor (offline tracking)
//  3. Mengirim alert Telegram jika nilai melampaui threshold
func ProcessSensorData(data []models.AreaData) (int, error) {
	processedItemCount := 0
	var combinedError error

	for _, areaData := range data {
		areaID := areaData.Area

		for _, tempData := range areaData.Temp {
			var parsedTS time.Time
			if tempData.Ts == "" || tempData.Ts == " " {
				parsedTS = timed.Now().In(config.Timezone)
			} else {
				var err error
				parsedTS, err = time.Parse(time.RFC3339Nano, tempData.Ts)
				if err != nil {
					log.Printf("invalid timestamp format for area %d: '%s', error: %v", areaID, tempData.Ts, err)
					if combinedError == nil {
						combinedError = fmt.Errorf("invalid timestamp format for area %d: '%s'", areaID, tempData.Ts)
					} else {
						combinedError = fmt.Errorf("%v; invalid timestamp format for area %d: '%s'", combinedError, areaID, tempData.Ts)
					}
					continue
				}
				parsedTS = parsedTS.In(config.Timezone)
			}

			// Validasi: Hanya proses sensor suhu yang terdaftar di config
			if !config.IsSensorRegistered("temp", areaID, tempData.No) {
				log.Printf("Peringatan: Menerima data untuk sensor suhu tidak terdaftar (Area: %d, No: %d). Data diabaikan.", areaID, tempData.No)
				continue
			}

			sensorKeyTemp := fmt.Sprintf("temp-%d-%d", areaID, tempData.No)
			models.RegisterOrUpdateSensorStatus(sensorKeyTemp, "temp", areaID, tempData.No, 0, parsedTS, tempData.Temp)
			processedItemCount++

			// Evaluasi threshold suhu → kirim alert Telegram jika perlu
			safetyStatus := worker.EvaluateTemp(areaID, tempData.No, tempData.Temp)
			if safetyStatus.IsAlert && safetyStatus.Message != "" {
				telegram.SendAlert("🔥 **Alert Temperature**\n" + safetyStatus.Message)
			}

			// Proses RH jika ada
			if tempData.RH != nil {
				if config.IsSensorRegistered("rh", areaID, tempData.No) {
					sensorKeyRH := fmt.Sprintf("rh-%d-%d", areaID, tempData.No)
					models.RegisterOrUpdateSensorStatus(sensorKeyRH, "rh", areaID, tempData.No, 0, parsedTS, *tempData.RH)
					processedItemCount++

					// Evaluasi threshold kelembaban → kirim alert Telegram jika perlu
					rhSafetyStatus := worker.EvaluateRh(areaID, tempData.No, *tempData.RH)
					if rhSafetyStatus.IsAlert && rhSafetyStatus.Message != "" {
						telegram.SendAlert("💧 **Alert Humidity**\n" + rhSafetyStatus.Message)
					}
				} else {
					log.Printf("Peringatan: Menerima data untuk sensor kelembaban tidak terdaftar (Area: %d, No: %d). Data diabaikan.", areaID, tempData.No)
				}
			}
		}

		for _, doorData := range areaData.Door {
			// Validasi: Hanya proses pintu yang terdaftar di database
			if !database.IsDoorRegistered(doorData.DoorID) {
				log.Printf("Peringatan: Menerima data untuk pintu tidak terdaftar (DoorID: %d). Data diabaikan.", doorData.DoorID)
				continue
			}

			ts := timed.Now().In(config.Timezone)
			if len(areaData.Temp) > 0 && areaData.Temp[0].Ts != "" {
				if parsedTS, err := time.Parse(time.RFC3339Nano, areaData.Temp[0].Ts); err == nil {
					ts = parsedTS.In(config.Timezone)
				}
			}

			sensorKeyProx := fmt.Sprintf("prox-%d", doorData.DoorID)
			models.RegisterOrUpdateSensorStatus(sensorKeyProx, "prox", areaID, 0, doorData.DoorID, ts, float64(doorData.Value))
			processedItemCount++
		}
	}

	return processedItemCount, combinedError
}