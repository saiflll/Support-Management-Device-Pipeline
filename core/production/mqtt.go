package main

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func initMQTT() mqtt.Client {
	hst := getEnv("MQTT_HOST", "emqx")
	prt := getEnv("MQTT_PORT", "1883")
	usr := getEnv("MQTT_USER", "apps")
	pwd := getEnv("MQTT_PASSWORD", "apps")
	url := fmt.Sprintf("tcp://%s:%s", hst, prt)

	opt := mqtt.NewClientOptions()
	opt.AddBroker(url)
	opt.SetClientID("forming-app-subscriber-" + fmt.Sprintf("%d", time.Now().Unix()))
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
	}

	lg("Connecting to MQTT broker: %s", url)
	cln := mqtt.NewClient(opt)
	if tkn := cln.Connect(); tkn.Wait() && tkn.Error() != nil {
		hndlErr("Warning: Could not connect to MQTT", tkn.Error())
		lg("App will continue running. MQTT will auto-reconnect when available.")
	}

	return cln
}
