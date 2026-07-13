package forwarder

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"IoTT/internal/models"
	"IoTT/internal/redis"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Deadband state tracking
var (
	deltaMutex      = &sync.Mutex{}
	lastSensorValues = make(map[string]float64)
	lastDoorStatus   = make(map[string]int)
	lastSentTime     = make(map[string]time.Time)
)

// ProcessAndPublish applies deadband filtering, then publishes filtered data
// to local EMQX topic 'forwarder/outgoing' for the EMQX bridge to forward to cloud.
func ProcessAndPublish(data []models.AreaData, client mqtt.Client) {
	if client == nil || !client.IsConnected() {
		log.Println("⚠️ Forwarder: MQTT client not available, skipping publish")
		return
	}

	deltaMutex.Lock()
	defer deltaMutex.Unlock()

	var filteredData []models.AreaData

	for _, areaData := range data {
		fd := models.AreaData{
			CK:    areaData.CK,
			Area:  areaData.Area,
			Topic: "forwarder/outgoing",
			Temp:  []models.TempData{},
			Door:  []models.DoorData{},
		}

		// Deadband filter for temperature (delta > 2.0 or heartbeat > 10min)
		for _, t := range areaData.Temp {
			key := fmt.Sprintf("%d-%d-%d", areaData.CK, areaData.Area, t.No)
			lastVal, exists := lastSensorValues[key]
			lastTs := lastSentTime[key]
			delta := 0.0
			if t.Temp > lastVal {
				delta = t.Temp - lastVal
			} else {
				delta = lastVal - t.Temp
			}
			if !exists || delta >= 2.0 || time.Since(lastTs) > 10*time.Minute {
				fd.Temp = append(fd.Temp, t)
				lastSensorValues[key] = t.Temp
				lastSentTime[key] = time.Now()
			}
		}

		// Deadband filter for door (only on state change)
		for _, d := range areaData.Door {
			key := fmt.Sprintf("door-%d-%d-%d", areaData.CK, areaData.Area, d.DoorID)
			lastVal, exists := lastDoorStatus[key]
			if !exists || d.Value != lastVal {
				fd.Door = append(fd.Door, d)
				lastDoorStatus[key] = d.Value
			}
		}

		if len(fd.Temp) > 0 || len(fd.Door) > 0 {
			filteredData = append(filteredData, fd)
		}
	}

	if len(filteredData) == 0 {
		return
	}

	payload, err := json.Marshal(filteredData)
	if err != nil {
		log.Printf("❌ Failed to marshal filtered data: %v", err)
		return
	}

	token := client.Publish("forwarder/outgoing", 1, false, payload)
	token.WaitTimeout(5 * time.Second)
	log.Printf("📤 Published %d filtered data packets to forwarder/outgoing", len(filteredData))

	// Also publish to Redis pub/sub for real-time service distribution
	redis.PublishSensorData(filteredData)
}
