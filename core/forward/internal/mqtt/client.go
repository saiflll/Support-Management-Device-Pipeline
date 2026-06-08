package mqtt

import (
	"IoTT/internal/forwarder"
	"IoTT/internal/logger"
	"IoTT/internal/models"
	"IoTT/internal/processor"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var client mqtt.Client

// SensorDataTopic akan diisi dari environment variable saat startup.
var SensorDataTopic string

var messageHandler mqtt.MessageHandler = func(cln mqtt.Client, psn mqtt.Message) {
	logger.Lg("📥 Pesan MQTT diterima dari topik: %s", psn.Topic())
	psnStr := string(psn.Payload())

	var dt []models.AreaData

	if strings.HasPrefix(psnStr, "CSV,") {
		// Konversi CSV Token Stream ke JSON struct
		logger.Lg("Info: Payload dideteksi berbentuk format CSV.")
		ad := parseCSVtoAreaData(psnStr)
		if ad != nil {
			dt = append(dt, *ad)
		}
	} else {
		// Coba unmarshal sebagai array dulu (JSON Mode)
		err := json.Unmarshal(psn.Payload(), &dt)
		if err != nil {
			// Jika gagal, coba unmarshal sebagai objek tunggal
			logger.Lg("Info: Gagal unmarshal sebagai array, mencoba sebagai objek tunggal. Error: %v", err)
			var singleData models.AreaData
			if err2 := json.Unmarshal(psn.Payload(), &singleData); err2 != nil {
				logger.HndlErr("UnmarshalPayload", err2)
				return
			}
			// Jika berhasil, bungkus dalam slice
			dt = []models.AreaData{singleData}
		}
	}

	// Inject topic metadata
	for i := range dt {
		dt[i].Topic = psn.Topic()
	}

	logger.Lg("Debug: Data setelah unmarshal: %+v", dt)

	if len(dt) == 0 {
		logger.Lg("Peringatan: Menerima payload MQTT kosong.")
		return
	}

	// Kirim data ke forwarder untuk agregasi
	forwarder.AddToBufferAndAggregate(dt)

	// Langsung panggil ProcessSensorData tanpa transaksi
	_, err := processor.ProcessSensorData(dt)
	if err != nil {
		logger.HndlErr("ProcessSensorDataFromMQTT", err)
	}
}

func parseCSVtoAreaData(payload string) *models.AreaData {
	// Format contoh: CSV,CK,1,AREA,10,TS,2024-05-07T09:00:00Z,T,0,28.5,M,3,25.1,60.5,D,1001,1
	parts := strings.Split(payload, ",")
	var ad models.AreaData
	var currentTS string

	for i := 0; i < len(parts); {
		switch parts[i] {
		case "CK":
			if i+1 < len(parts) {
				if val, err := strconv.Atoi(parts[i+1]); err == nil { ad.CK = val }
				i += 2
			} else { i++ }
		case "AREA":
			if i+1 < len(parts) {
				if val, err := strconv.Atoi(parts[i+1]); err == nil { ad.Area = val }
				i += 2
			} else { i++ }
		case "TS":
			if i+1 < len(parts) {
				currentTS = parts[i+1]
				i += 2
			} else { i++ }
		case "T":
			if i+2 < len(parts) {
				no, err1 := strconv.Atoi(parts[i+1])
				temp, err2 := strconv.ParseFloat(parts[i+2], 64)
				if err1 == nil && err2 == nil { ad.Temp = append(ad.Temp, models.TempData{No: no, Ts: currentTS, Temp: temp}) }
				i += 3
			} else { i++ }
		case "M":
			if i+3 < len(parts) {
				no, err1 := strconv.Atoi(parts[i+1])
				temp, err2 := strconv.ParseFloat(parts[i+2], 64)
				rh, err3 := strconv.ParseFloat(parts[i+3], 64)
				if err1 == nil && err2 == nil && err3 == nil { ad.Temp = append(ad.Temp, models.TempData{No: no, Ts: currentTS, Temp: temp, RH: &rh}) }
				i += 4
			} else { i++ }
		case "D":
			if i+2 < len(parts) {
				id, err1 := strconv.Atoi(parts[i+1])
				val, err2 := strconv.Atoi(parts[i+2])
				if err1 == nil && err2 == nil { ad.Door = append(ad.Door, models.DoorData{DoorID: id, Value: val}) }
				i += 3
			} else { i++ }
		default:
			i++
		}
	}
	return &ad
}

var connectHandler mqtt.OnConnectHandler = func(cln mqtt.Client) {
	logger.Lg("✅ Berhasil terhubung ke MQTT Broker.")
	// Berlangganan ke topik setelah koneksi berhasil
	token := cln.Subscribe(SensorDataTopic, 1, messageHandler)
	token.Wait()
	logger.Lg("✔️ Berlangganan ke topik: %s", SensorDataTopic)
}

var connectionLostHandler mqtt.ConnectionLostHandler = func(cln mqtt.Client, err error) {
	logger.HndlErr("MQTTConnectionLost", err)
}

func StartClient() {
	brokerURI := os.Getenv("MQTT_BROKER_URI")
	if brokerURI == "" {
		logger.Lg("Peringatan: MQTT_BROKER_URI tidak diatur. MQTT client tidak akan dimulai.")
		return
	}

	SensorDataTopic = os.Getenv("MQTT_TOPIC")
	if SensorDataTopic == "" {
		SensorDataTopic = "sensor/data/ingest"
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURI)
	opts.SetClientID(fmt.Sprintf("servfi-backend-%d", time.Now().UnixNano()))
	opts.SetDefaultPublishHandler(messageHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectionLostHandler

	// Tambahkan kredensial jika tersedia di environment
	username := os.Getenv("MQTT_USERNAME")
	if username == "" {
		username = "apps"
	}
	password := os.Getenv("MQTT_PASSWORD")
	if password == "" {
		password = "apps"
	}

	if username != "" {
		opts.SetUsername(username)
		opts.SetPassword(password)
		logger.Lg("Menggunakan kredensial MQTT untuk koneksi lokal.")
	}

	client = mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		logger.Ftl("❌ Gagal terhubung ke MQTT Broker: %v", token.Error())
	}
}
