package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var (
	lastPayloads   = make(map[string]Payload)
	lastPayloadsMu sync.Mutex
)

// initMQTT configures and connects to our MQTT broker
func initMQTT() mqtt.Client {
	mqttHost := getEnv("MQTT_HOST", "emqx")
	mqttPort := getEnv("MQTT_PORT", "1883")
	mqttUser := getEnv("MQTT_USER", "apps")
	mqttPass := getEnv("MQTT_PASSWORD", "apps")
	brokerUrl := fmt.Sprintf("tcp://%s:%s", mqttHost, mqttPort)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerUrl)
	opts.SetClientID("forming-app-subscriber-" + fmt.Sprintf("%d", time.Now().Unix()))
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetConnectTimeout(10 * time.Second)
	opts.SetProtocolVersion(4) 

	if mqttUser != "" {
		opts.SetUsername(mqttUser)
		log.Printf("MQTT Username: %s", mqttUser)
	}
	if mqttPass != "" {
		opts.SetPassword(mqttPass)
		log.Println("MQTT Password: ***")
	}

	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		log.Printf("MQTT Connection lost: %v - Will auto-reconnect", err)
	}
	opts.OnConnect = func(c mqtt.Client) {
		log.Println("MQTT Connected successfully!")
		subscribe(c)
	}

	log.Printf("Connecting to MQTT broker: %s", brokerUrl)
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Printf("Warning: Could not connect to MQTT: %v", token.Error())
		log.Println("App will continue running. MQTT will auto-reconnect when available.")
	}

	return client
}

func subscribe(client mqtt.Client) {
	topic := "production/mdcw"
	if token := client.Subscribe(topic, 1, nil); token.Wait() && token.Error() != nil {
		log.Printf("Error subscribing to topic %s: %v", topic, token.Error())
	} else {
		log.Printf("Subscribed to topic: %s", topic)
	}
}

func messagePubHandler(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())

	var p Payload
	if err := json.Unmarshal(msg.Payload(), &p); err != nil {
		log.Println("Error parsing JSON:", err)
		return
	}

	// fallbacks for varying FW payloads
	if p.Prefix == "" && p.NodePrefix != "" { p.Prefix = p.NodePrefix }
	
	if p.Reg2 == 0 && p.Data.Reg2 != 0 { p.Reg2 = p.Data.Reg2 }
	if p.Reg5 == 0 && p.Data.Reg5 != 0 { p.Reg5 = p.Data.Reg5 }
	if p.Reg114 == 0 && p.Data.Reg114 != 0 { p.Reg114 = p.Data.Reg114 }

	if p.Reg2 == 0 && p.Total != 0 { p.Reg2 = p.Total }
	if p.Reg5 == 0 && p.Code != 0 { p.Reg5 = p.Code }
	if p.Reg114 == 0 && p.Weight != 0 { p.Reg114 = p.Weight }

	nowStr := time.Now().Format("2006-01-02 15:04:05")
	if p.Ts == nil {
		p.Ts = nowStr
	} else if _, ok := p.Ts.(string); !ok {
		p.Ts = nowStr
	}

	go insertData(p)
}
