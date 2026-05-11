package mqtt

import (
	"IoTT/internal/forwarder"
	"IoTT/internal/models"
	"IoTT/internal/processor"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var client mqtt.Client

// SensorDataTopic akan diisi dari environment variable saat startup.
var SensorDataTopic string

var messageHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Printf("📥 Pesan MQTT diterima dari topik: %s", msg.Topic())
	payloadStr := string(msg.Payload())

	var receivedData []models.AreaData

	if strings.HasPrefix(payloadStr, "CSV,") {
		// Konversi CSV Token Stream ke JSON struct
		log.Printf("Info: Payload dideteksi berbentuk format CSV.")
		ad := parseCSVtoAreaData(payloadStr)
		if ad != nil {
			receivedData = append(receivedData, *ad)
		}
	} else {
		// Coba unmarshal sebagai array dulu (JSON Mode)
		err := json.Unmarshal(msg.Payload(), &receivedData)
		if err != nil {
			// Jika gagal, coba unmarshal sebagai objek tunggal
			log.Printf("Info: Gagal unmarshal sebagai array, mencoba sebagai objek tunggal. Error: %v", err)
			var singleData models.AreaData
			if err2 := json.Unmarshal(msg.Payload(), &singleData); err2 != nil {
				log.Printf("Error: Gagal unmarshal payload baik sebagai array/objek: %v", err2)
				return
			}
			// Jika berhasil, bungkus dalam slice
			receivedData = []models.AreaData{singleData}
		}
	}

	// Inject topic metadata
	for i := range receivedData {
		receivedData[i].Topic = msg.Topic()
	}

	log.Printf("Debug: Data setelah unmarshal: %+v", receivedData)

	if len(receivedData) == 0 {
		log.Println("Peringatan: Menerima payload MQTT kosong.")
		return
	}

	// Kirim data ke forwarder untuk agregasi
	forwarder.AddToBufferAndAggregate(receivedData)

	// Langsung panggil ProcessSensorData tanpa transaksi
	_, err := processor.ProcessSensorData(receivedData)
	if err != nil {
		log.Printf("Error selama pemrosesan data sensor dari MQTT: %v", err)
		// Error di sini kemungkinan besar adalah dari validasi atau parsing,
		// karena penyimpanan data sudah ditangani oleh worker.
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

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Println("✅ Berhasil terhubung ke MQTT Broker.")
	// Berlangganan ke topik setelah koneksi berhasil
	token := client.Subscribe(SensorDataTopic, 1, messageHandler)
	token.Wait()
	log.Printf("✔️ Berlangganan ke topik: %s", SensorDataTopic)
}

var connectionLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Printf("⚠️ Koneksi ke MQTT Broker terputus: %v", err)
}

func StartClient() {
	brokerURI := os.Getenv("MQTT_BROKER_URI")
	if brokerURI == "" {
		log.Println("Peringatan: MQTT_BROKER_URI tidak diatur. MQTT client tidak akan dimulai.")
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
		log.Println("Menggunakan kredensial MQTT untuk koneksi lokal.")
	}

	client = mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("❌ Gagal terhubung ke MQTT Broker: %v", token.Error())
	}
}
