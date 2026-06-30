package mdcw

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type cloudForwarder struct {
	client  mqtt.Client
	topic   string
	enabled bool
	mu      sync.Mutex
}

var cldFwd *cloudForwarder

func InitCloudForwarder() {
	uri := os.Getenv("CLOUD_MQTT_BROKER_URI")
	if uri == "" {
		log.Println("[CloudFwd] CLOUD_MQTT_BROKER_URI not set. Cloud forwarding INACTIVE.")
		cldFwd = &cloudForwarder{enabled: false}
		return
	}

	tpc := os.Getenv("CLOUD_MQTT_TOPIC_FORMING")
	if tpc == "" {
		tpc = "prod/mdcw"
	}

	usr := os.Getenv("CLOUD_MQTT_USERNAME")
	pwd := os.Getenv("CLOUD_MQTT_PASSWORD")

	opt := mqtt.NewClientOptions()
	opt.AddBroker(uri)
	opt.SetClientID(fmt.Sprintf("forming-cloud-fwd-%d", time.Now().UnixNano()))
	opt.SetAutoReconnect(true)
	opt.SetKeepAlive(60 * time.Second)
	opt.SetConnectTimeout(10 * time.Second)
	opt.SetProtocolVersion(4)

	if usr != "" {
		opt.SetUsername(usr)
		opt.SetPassword(pwd)
		log.Printf("[CloudFwd] Using cloud MQTT credentials: %s", usr)
	}

	opt.OnConnect = func(cln mqtt.Client) {
		log.Printf("[CloudFwd] Connected to Cloud MQTT Broker: %s", uri)
	}
	opt.OnConnectionLost = func(cln mqtt.Client, err error) {
		log.Printf("[CloudFwd] Connection lost: %v", err)
	}

	cln := mqtt.NewClient(opt)
	if tkn := cln.Connect(); tkn.Wait() && tkn.Error() != nil {
		log.Printf("[CloudFwd] Connect error: %v", tkn.Error())
		log.Println("[CloudFwd] Cloud forwarding active — will retry on publish.")
	}

	cldFwd = &cloudForwarder{
		client:  cln,
		topic:   tpc,
		enabled: true,
	}

	log.Printf("[CloudFwd] Cloud forwarder active -> broker: %s, topic: %s", uri, tpc)
}

var prefixToMachineID = map[string]int{
	"MDCW1 (UK)":          1,
	"MDCW2 (Siomay)":      2,
	"MDCW3 (Pentol)":      3,
	"MDCW4 (AP)":          4,
	"MDCW5 (ACIN)":        5,
	"MDCW6 (Lumpia)":      6,
	"MDCW7 (Kulit/Kerupuk)": 7,
	"MDCW8 (Mie)":         8,
	"MDCW9 (Mie)":         9,
	"MDCW1":               1,
	"MDCW2":               2,
	"MDCW3":               3,
	"MDCW4":               4,
	"MDCW5":               5,
	"MDCW6":               6,
	"MDCW7":               7,
	"MDCW8":               8,
	"MDCW9":               9,
}

func currentShift() int {
	hr := time.Now().Hour()
	switch {
	case hr >= 7 && hr < 15:
		return 1
	case hr >= 15 && hr < 23:
		return 2
	default:
		return 3
	}
}

func reg5IsOk(reg5 int) bool {
	return reg5 == 41 || reg5 == 521 || reg5 == 553
}

func reg5IsMetal(reg5 int) bool {
	return reg5 == 8201
}

func reg5IsUnder(reg5 int) bool {
	return reg5 == 25
}

func reg5IsOver(reg5 int) bool {
	return reg5 == 73
}

func ForwardToCloud(psn Payload, pfx string, reg5 int, reg114 int) {
	if cldFwd == nil || !cldFwd.enabled {
		return
	}

	macId, ok := prefixToMachineID[pfx]
	if !ok {
		log.Printf("[CloudFwd] Prefix '%s' not found in machine_id mapping. Data not forwarded.", pfx)
		return
	}

	var (
		out     = 1
		nce     = 0
		rjct    = 0
		rjctMtl = 0
		rjctOvr = 0
		rjctUnd = 0
		rjctOth = 0
		pwr     = 0.0
		eff     = 0.0
		sft     = currentShift()
	)

	switch {
	case reg5IsOk(reg5):
		nce = 1
	case reg5IsMetal(reg5):
		rjct = 1
		rjctMtl = 1
	case reg5IsUnder(reg5):
		rjct = 1
		rjctUnd = 1
	case reg5IsOver(reg5):
		rjct = 1
		rjctOvr = 1
	default:
		return
	}

	if nce > 0 {
		eff = 100.0
	} else {
		eff = 0.0
	}

	wkt := time.Now().Format(time.RFC3339)
	if tsRaw, ok := psn.Ts.(string); ok && tsRaw != "" {
		wkt = tsRaw
	}

	csv := fmt.Sprintf(
		"CSV,%d,%d,%d,%d,%d,%d,%d,%d,%d,%.3f,%.2f,%s",
		macId,
		sft,
		out,
		nce,
		rjct,
		rjctMtl,
		rjctOvr,
		rjctUnd,
		rjctOth,
		pwr,
		eff,
		wkt,
	)

	cldFwd.mu.Lock()
	defer cldFwd.mu.Unlock()

	if !cldFwd.client.IsConnected() {
		log.Println("[CloudFwd] Client not connected, reconnecting before publish...")
		tkn := cldFwd.client.Connect()
		tkn.Wait()
	}

	tkn := cldFwd.client.Publish(cldFwd.topic, 1, false, csv)
	tkn.Wait()
	if err := tkn.Error(); err != nil {
		log.Printf("[CloudFwd] Publish error: %v", err)
	} else {
		log.Printf("[CloudFwd] Forwarded to cloud [%s] machine=%d shift=%d ok:%d metal:%d over:%d under:%d",
			cldFwd.topic, macId, sft, nce, rjctMtl, rjctOvr, rjctUnd)
	}
}
