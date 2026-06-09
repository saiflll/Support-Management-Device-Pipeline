package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// === STATE GLOBAL ===

var (
	lastPayloads   = make(map[string]Payload)
	lastPayloadsMu sync.Mutex
)

// === INISIALISASI MQTT ===

// initMQTT configures and connects to our MQTT broker
func initMQTT() mqtt.Client {
	hst := getEnv("MQTT_HOST", "emqx")
	prt := getEnv("MQTT_PORT", "1883")
	usr := getEnv("MQTT_USER", "apps")
	pwd := getEnv("MQTT_PASSWORD", "apps")
	url := fmt.Sprintf("tcp://%s:%s", hst, prt)

	opt := mqtt.NewClientOptions()
	opt.AddBroker(url)
	opt.SetClientID("forming-app-subscriber-" + fmt.Sprintf("%d", time.Now().Unix()))
	opt.SetDefaultPublishHandler(messagePubHandler)
	opt.SetCleanSession(true)
	opt.SetAutoReconnect(true)
	opt.SetKeepAlive(60 * time.Second)
	opt.SetPingTimeout(10 * time.Second)
	opt.SetConnectTimeout(10 * time.Second)
	opt.SetProtocolVersion(4) 

	if usr != "" {
		opt.SetUsername(usr)
		lg("MQTT Username: %s", usr)
	}
	if pwd != "" {
		opt.SetPassword(pwd)
		lg("MQTT Password: ***")
	}

	opt.OnConnectionLost = func(cln mqtt.Client, err error) {
		hndlErr("MQTT Connection lost (auto-reconnect)", err)
	}
	opt.OnConnect = func(cln mqtt.Client) {
		lg("MQTT Connected successfully!")
		subscribe(cln)
	}

	lg("Connecting to MQTT broker: %s", url)
	cln := mqtt.NewClient(opt)
	if tkn := cln.Connect(); tkn.Wait() && tkn.Error() != nil {
		hndlErr("Warning: Could not connect to MQTT", tkn.Error())
		lg("App will continue running. MQTT will auto-reconnect when available.")
	}

	return cln
}

// subscribe handles subscribing to the required topics
func subscribe(cln mqtt.Client) {
	tpc := "production/mdcw"
	if tkn := cln.Subscribe(tpc, 1, nil); tkn.Wait() && tkn.Error() != nil {
		hndlErr(fmt.Sprintf("Error subscribing to topic %s", tpc), tkn.Error())
	} else {
		lg("Subscribed to topic: %s", tpc)
	}
}

// === HANDLER PESAN ===

func messagePubHandler(cln mqtt.Client, psn mqtt.Message) {
	lg("Received message: %s from topic: %s", string(psn.Payload()), psn.Topic())

	var pl Payload
	if err := json.Unmarshal(psn.Payload(), &pl); err != nil {
		hndlErr("Error parsing JSON", err)
		return
	}

	// fallbacks for varying FW payloads
	if pl.Prefix == "" && pl.NodePrefix != "" {
		pl.Prefix = pl.NodePrefix
	}

	if pl.Reg2 == 0 && pl.Data.Reg2 != 0 {
		pl.Reg2 = pl.Data.Reg2
	}
	if pl.Reg5 == 0 && pl.Data.Reg5 != 0 {
		pl.Reg5 = pl.Data.Reg5
	}
	if pl.Reg114 == 0 && pl.Data.Reg114 != 0 {
		pl.Reg114 = pl.Data.Reg114
	}

	if pl.Reg2 == 0 && pl.Total != 0 {
		pl.Reg2 = pl.Total
	}
	if pl.Reg5 == 0 && pl.Code != 0 {
		pl.Reg5 = pl.Code
	}
	if pl.Reg114 == 0 && pl.Weight != 0 {
		pl.Reg114 = pl.Weight
	}

	wkt := time.Now().Format("2006-01-02 15:04:05")
	if pl.Ts == nil {
		pl.Ts = wkt
	} else if _, ok := pl.Ts.(string); !ok {
		pl.Ts = wkt
	}

	go insertData(pl)
}
