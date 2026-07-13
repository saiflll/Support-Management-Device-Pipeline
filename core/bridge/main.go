package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/redis/go-redis/v9"

	"github.com/rennn/servfor/bridge/internal/codec"
)

var (
	mqttBrokerURI  = env("MQTT_BROKER_URI", "tcp://emqx:1883")
	mqttUsername   = env("MQTT_USERNAME", "apps")
	mqttPassword   = env("MQTT_PASSWORD", "apps")
	redisURL       = os.Getenv("REDIS_URL")
	logLevel       = env("LOG_LEVEL", "info")
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ipString formats a uint32 as a dotted IPv4 string.
func ipString(ip uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d", byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip))
}

// extractPrefix extracts the prefix from a nodeId by taking everything before the last '-'.
// Example: "MDCW1_(UK)-A1B2C3D4" -> "MDCW1_(UK)"
func extractPrefix(nodeID string) string {
	if idx := strings.LastIndex(nodeID, "-"); idx != -1 {
		return nodeID[:idx]
	}
	return nodeID
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("Starting bridge service (log_level=%s)", logLevel)

	// --- Redis (optional) ---
	var rdb *redis.Client
	if redisURL != "" {
		opts, err := redis.ParseURL(redisURL)
		if err != nil {
			log.Printf("WARN: invalid REDIS_URL %q, disabling Redis: %v", redisURL, err)
		} else {
			rdb = redis.NewClient(opts)
			ctx, cancel := context.WithTimeout(context.Background(), 5e9)
			if err := rdb.Ping(ctx).Err(); err != nil {
				log.Printf("WARN: Redis unreachable (%v), continuing without Redis", err)
				rdb.Close()
				rdb = nil
			}
			cancel()
		}
	}

	// --- MQTT ---
	opts := mqtt.NewClientOptions()
	opts.AddBroker(mqttBrokerURI)
	opts.SetUsername(mqttUsername)
	opts.SetPassword(mqttPassword)
	opts.SetClientID(fmt.Sprintf("bridge-%d", os.Getpid()))
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		log.Printf("MQTT connected to %s", mqttBrokerURI)

		// Subscribe to telemetry
		if token := c.Subscribe("iot/data/mdcw/+/telemetry", 1, telemetryHandler(rdb)); token.Wait() && token.Error() != nil {
			log.Printf("ERROR subscribing to telemetry: %v", token.Error())
		} else {
			log.Printf("Subscribed to iot/data/mdcw/+/telemetry")
		}

		// Subscribe to heartbeat
		if token := c.Subscribe("iot/node/mdcw/+/heartbeat", 1, heartbeatHandler(rdb)); token.Wait() && token.Error() != nil {
			log.Printf("ERROR subscribing to heartbeat: %v", token.Error())
		} else {
			log.Printf("Subscribed to iot/node/mdcw/+/heartbeat")
		}
	})
	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		log.Printf("MQTT connection lost: %v", err)
	})

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}
	log.Printf("Bridge started, waiting for messages...")

	// --- Graceful shutdown ---
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Printf("Shutting down...")

	if token := client.Unsubscribe("iot/data/mdcw/+/telemetry", "iot/node/mdcw/+/heartbeat"); token.Wait() && token.Error() != nil {
		log.Printf("WARN: unsubscribe error: %v", token.Error())
	}
	client.Disconnect(250)
	if rdb != nil {
		rdb.Close()
	}
	log.Printf("Bridge stopped")
}

// telemetryHandler returns the MQTT message handler for telemetry frames.
func telemetryHandler(rdb *redis.Client) mqtt.MessageHandler {
	return func(c mqtt.Client, msg mqtt.Message) {
		topic := msg.Topic()
		payload := msg.Payload()
		log.Printf("RECV telemetry topic=%s len=%d", topic, len(payload))

		// Parse topic: iot/data/mdcw/{nodeId}/telemetry
		parts := strings.Split(topic, "/")
		if len(parts) < 5 {
			log.Printf("ERROR invalid telemetry topic: %s", topic)
			return
		}
		nodeID := parts[3]

		frame, err := codec.DecodeTelemetry(payload)
		if err != nil {
			log.Printf("ERROR decode telemetry node=%s: %v", nodeID, err)
			return
		}

		prefix := extractPrefix(nodeID)

		// Build the JSON payload
		type telemetryMsg struct {
			Ts         int64  `json:"ts"`
			Reg2       int16  `json:"reg2"`
			Reg5       int16  `json:"reg5"`
			Reg114     int16  `json:"reg114"`
			Total      int16  `json:"total"`
			Code       int16  `json:"code"`
			Weight     int16  `json:"weight"`
			Prefix     string `json:"prefix"`
			NodePrefix string `json:"node_prefix"`
			Ck         int16  `json:"ck"`
			Area       int16  `json:"area"`
		}

		msgJSON := telemetryMsg{
			Ts:         int64(frame.Ts),
			Reg2:       frame.Total,
			Reg5:       frame.Code,
			Reg114:     frame.Weight,
			Total:      frame.Total,
			Code:       frame.Code,
			Weight:     frame.Weight,
			Prefix:     prefix,
			NodePrefix: prefix,
			Ck:         frame.Ck,
			Area:       frame.Area,
		}

		jsonBytes, err := json.Marshal(msgJSON)
		if err != nil {
			log.Printf("ERROR marshal telemetry node=%s: %v", nodeID, err)
			return
		}

		targetTopic := "production/mdcw"
		if token := c.Publish(targetTopic, 1, false, jsonBytes); token.Wait() && token.Error() != nil {
			log.Printf("ERROR publish telemetry to %s: %v", targetTopic, token.Error())
		} else {
			log.Printf("SENT telemetry node=%s -> %s: %s", nodeID, targetTopic, string(jsonBytes))
		}

		// Redis publish
		if rdb != nil {
			ctx := context.Background()
			if err := rdb.Publish(ctx, "sensor:data:live", string(jsonBytes)).Err(); err != nil {
				log.Printf("WARN redis publish sensor:data:live: %v", err)
			}
		}
	}
}

// heartbeatHandler returns the MQTT message handler for heartbeat frames.
func heartbeatHandler(rdb *redis.Client) mqtt.MessageHandler {
	return func(c mqtt.Client, msg mqtt.Message) {
		topic := msg.Topic()
		payload := msg.Payload()
		log.Printf("RECV heartbeat topic=%s len=%d", topic, len(payload))

		// Parse topic: iot/node/mdcw/{nodeId}/heartbeat
		parts := strings.Split(topic, "/")
		if len(parts) < 5 {
			log.Printf("ERROR invalid heartbeat topic: %s", topic)
			return
		}
		nodeID := parts[3]

		frame, err := codec.DecodeHeartbeat(payload)
		if err != nil {
			log.Printf("ERROR decode heartbeat node=%s: %v", nodeID, err)
			return
		}

		ip := ipString(frame.IPAddr)
		fwVersion := fmt.Sprintf("%d.%d", frame.FwMajor, frame.FwMinor)
		cpuTemp := float64(frame.CPUTempX10) / 10.0

		// Status JSON
		type statusMsg struct {
			Status      string  `json:"status"`
			IP          string  `json:"ip"`
			Model       string  `json:"model"`
			Version     string  `json:"version"`
			Ck          int16   `json:"ck"`
			Area        int16   `json:"area"`
			RAMFreeBytes uint32 `json:"ram_free_bytes"`
			SdOK        bool    `json:"sd_ok"`
			Interval    uint16  `json:"interval"`
			Uptime      uint32  `json:"uptime"`
			RSSI        int8    `json:"rssi"`
			FwMajor     uint8   `json:"fw_major"`
			FwMinor     uint8   `json:"fw_minor"`
			CPUTemp     float64 `json:"cpu_temp"`
		}

		statusJSON := statusMsg{
			Status:       "online",
			IP:           ip,
			Model:        "MDCW",
			Version:      fwVersion,
			Ck:           frame.Ck,
			Area:         frame.Area,
			RAMFreeBytes: frame.Heap,
			SdOK:         true,
			Interval:     frame.Interval,
			Uptime:       frame.Uptime,
			RSSI:         frame.RSSI,
			FwMajor:      frame.FwMajor,
			FwMinor:      frame.FwMinor,
			CPUTemp:      cpuTemp,
		}

		statusBytes, err := json.Marshal(statusJSON)
		if err != nil {
			log.Printf("ERROR marshal heartbeat node=%s: %v", nodeID, err)
			return
		}

		// Publish status (retained)
		statusTopic := fmt.Sprintf("nodes/%s/status", nodeID)
		if token := c.Publish(statusTopic, 1, true, statusBytes); token.Wait() && token.Error() != nil {
			log.Printf("ERROR publish status to %s: %v", statusTopic, token.Error())
		} else {
			log.Printf("SENT status node=%s -> %s: %s", nodeID, statusTopic, string(statusBytes))
		}

		// Monitor JSON (lighter, not retained)
		type monitorMsg struct {
			RAMFree uint32 `json:"ram_free"`
			Status  string `json:"status"`
			Uptime  uint32 `json:"uptime"`
			RSSI    int8   `json:"rssi"`
			Ck      int16  `json:"ck"`
			Area    int16  `json:"area"`
		}

		monitorJSON := monitorMsg{
			RAMFree: frame.Heap,
			Status:  "online",
			Uptime:  frame.Uptime,
			RSSI:    frame.RSSI,
			Ck:      frame.Ck,
			Area:    frame.Area,
		}

		monitorBytes, err := json.Marshal(monitorJSON)
		if err != nil {
			log.Printf("ERROR marshal monitor node=%s: %v", nodeID, err)
			return
		}

		monitorTopic := fmt.Sprintf("nodes/%s/monitor", nodeID)
		if token := c.Publish(monitorTopic, 1, false, monitorBytes); token.Wait() && token.Error() != nil {
			log.Printf("ERROR publish monitor to %s: %v", monitorTopic, token.Error())
		} else {
			log.Printf("SENT monitor node=%s -> %s: %s", nodeID, monitorTopic, string(monitorBytes))
		}

		// Redis publish
		if rdb != nil {
			ctx := context.Background()
			if err := rdb.Publish(ctx, "sensor:status:change", string(statusBytes)).Err(); err != nil {
				log.Printf("WARN redis publish sensor:status:change: %v", err)
			}
		}
	}
}

