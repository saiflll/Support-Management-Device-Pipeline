package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// === STATE & KONFIGURASI CLOUD ===

// cloudForwarder mengelola koneksi ke Cloud MQTT Broker untuk forward data forming
type cloudForwarder struct {
	client  mqtt.Client
	topic   string
	enabled bool
	mu      sync.Mutex
}

var cldFwd *cloudForwarder

// initCloudForwarder menginisialisasi koneksi ke Cloud MQTT Broker.
// Konfigurasi diambil dari environment variable dengan prefix CLOUD_.
// Jika CLOUD_MQTT_BROKER_URI tidak di-set, forwarder tidak akan aktif.
func initCloudForwarder() {
	uri := os.Getenv("CLOUD_MQTT_BROKER_URI")
	if uri == "" {
		lg("[CloudFwd] CLOUD_MQTT_BROKER_URI tidak diatur. Cloud forwarding TIDAK aktif.")
		cldFwd = &cloudForwarder{enabled: false}
		return
	}

	tpc := os.Getenv("CLOUD_MQTT_TOPIC_FORMING")
	if tpc == "" {
		tpc = "prod/mdcw" // default topic yang dipahami backend cloud
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
		lg("[CloudFwd] Menggunakan kredensial MQTT cloud: %s", usr)
	}

	opt.OnConnect = func(cln mqtt.Client) {
		lg("[CloudFwd] ✅ Terhubung ke Cloud MQTT Broker: %s", uri)
	}
	opt.OnConnectionLost = func(cln mqtt.Client, err error) {
		hndlErr("CloudFwdConnectionLost", err)
	}

	cln := mqtt.NewClient(opt)
	if tkn := cln.Connect(); tkn.Wait() && tkn.Error() != nil {
		hndlErr("CloudFwdConnect", tkn.Error())
		lg("[CloudFwd] Cloud forwarding tetap aktif — akan mencoba reconnect saat publish.")
	}

	cldFwd = &cloudForwarder{
		client:  cln,
		topic:   tpc,
		enabled: true,
	}

	lg("[CloudFwd] Cloud forwarder aktif → broker: %s, topik: %s", uri, tpc)
}

// prefixToMachineID memetakan prefix MDCW lokal ke machine_id di database cloud.
var prefixToMachineID = map[string]int{
	"MDCW1 (UK)":      1,
	"MDCW2 (Siomay)":  2,
	"MDCW3 (Pentol)":  3,
	"MDCW4 (AP)":      4,
	"MDCW5 (ACIN)":    5,
	"MDCW6 (Lumpia)":  6,
	"MDCW1":           1,
	"MDCW2":           2,
	"MDCW3":           3,
	"MDCW4":           4,
	"MDCW5":           5,
	"MDCW6":           6,
}

// === UTRED (UTILITIES & DETECTOR) ===

// reg5ToShift memetakan status code reg5 ke nomor shift berdasarkan waktu saat ini.
// Shift 1: 07:00-15:00, Shift 2: 15:00-23:00, Shift 3: 23:00-07:00
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

// reg5IsOk mengembalikan true jika status code menandakan OK/passed
func reg5IsOk(reg5 int) bool {
	return reg5 == 41 || reg5 == 521 || reg5 == 553
}

// reg5IsMetal mengembalikan true jika status code menandakan metal detected
func reg5IsMetal(reg5 int) bool {
	return reg5 == 8201
}

// reg5IsUnder mengembalikan true jika status code menandakan underweight
func reg5IsUnder(reg5 int) bool {
	return reg5 == 25
}

// reg5IsOver mengembalikan true jika status code menandakan overweight
func reg5IsOver(reg5 int) bool {
	return reg5 == 73
}

// === FORWARD DATA ===

// ForwardToCloud mengirim satu record MDCW ke Cloud MQTT dalam format CSV
// yang dipahami backend cloud (prod/mdcw).
func ForwardToCloud(psn Payload, pfx string, reg5 int, reg114 int) {
	if cldFwd == nil || !cldFwd.enabled {
		return
	}

	// tentukan machine_id dari prefix
	macId, ok := prefixToMachineID[pfx]
	if !ok {
		lg("[CloudFwd] Prefix '%s' tidak ditemukan di mapping machine_id. Data tidak di-forward.", pfx)
		return
	}

	// status reject berdasarkan reg5
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
		// status lain (idle, mati, dll) — tidak hitung sebagai output/reject
		return
	}

	if nce > 0 {
		eff = 100.0
	} else {
		eff = 0.0
	}

	// timestamp dalam format RFC3339 (UTC)
	wkt := time.Now().Format(time.RFC3339)
	if tsRaw, ok := psn.Ts.(string); ok && tsRaw != "" {
		wkt = tsRaw
	}

	// format CSV untuk cloud backend
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
		lg("[CloudFwd] Client tidak terkoneksi, mencoba reconnect sebelum publish...")
		tkn := cldFwd.client.Connect()
		tkn.Wait()
	}

	tkn := cldFwd.client.Publish(cldFwd.topic, 1, false, csv)
	tkn.Wait()
	if err := tkn.Error(); err != nil {
		hndlErr("CloudFwdPublish", err)
	} else {
		lg("[CloudFwd] ✅ Forward ke cloud [%s] machine=%d shift=%d status=ok:%d metal:%d over:%d under:%d",
			cldFwd.topic, macId, sft, nce, rjctMtl, rjctOvr, rjctUnd)
	}
}
