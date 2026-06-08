package dev

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"iot-ota-server/core"
)

// MockMQTTClient is a mock of the MQTT client
type MockMQTTClient struct {
	mock.Mock
	mqtt.Client
}

func (m *MockMQTTClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	args := m.Called(topic, qos, retained, payload)
	return args.Get(0).(mqtt.Token)
}

func (m *MockMQTTClient) Connect() mqtt.Token {
	return &MockToken{}
}

// MockToken is a mock of the MQTT token
type MockToken struct {
	mqtt.Token
}

func (m *MockToken) Wait() bool { return true }
func (m *MockToken) Error() error { return nil }

func TestMonitoringLogic(t *testing.T) {
	app := fiber.New()

	// Setup test data
	core.NodeMutex.Lock()
	core.NodeStatus["test-node-1234567890ab"] = &core.NodeInfo{
		Status:  "online",
		Updated: time.Now().Format("2006-01-02 15:04:05"),
		Area:    "10",
	}
	core.NodeMutex.Unlock()

	app.Get("/api/nodes", func(c *fiber.Ctx) error {
		core.NodeMutex.RLock()
		defer core.NodeMutex.RUnlock()
		return c.JSON(core.NodeStatus)
	})

	req := httptest.NewRequest("GET", "/api/nodes", nil)
	resp, _ := app.Test(req)

	assert.Equal(t, 200, resp.StatusCode)

	var nodes map[string]core.NodeInfo
	json.NewDecoder(resp.Body).Decode(&nodes)
	assert.Contains(t, nodes, "test-node-1234567890ab")
}

func TestSetConfigCommand(t *testing.T) {
	app := fiber.New()
	mockMQTT := new(MockMQTTClient)
	core.MqttClient = mockMQTT // Override global

	// Expectation: set_config with some values
	mockMQTT.On("Publish", "nodes/node-1/command", mock.Anything, mock.Anything, mock.MatchedBy(func(payload []byte) bool {
		return strings.Contains(string(payload), "cmd=set_config") && strings.Contains(string(payload), "node_prefix=TEST")
	})).Return(&MockToken{})

	app.Post("/config", func(c *fiber.Ctx) error {
		// Simulating the protected route handler logic
		var req map[string]interface{}
		c.BodyParser(&req)
		nodeID := req["node"].(string)

		payload := make(map[string]interface{})
		for k, v := range req {
			if k != "node" {
				payload[k] = v
			}
		}
		payload["cmd"] = "set_config"

		values := url.Values{}
		for k, v := range payload {
			values.Set(k, fmt.Sprintf("%v", v))
		}
		b := []byte(values.Encode())
		core.MqttClient.Publish(fmt.Sprintf("nodes/%s/command", nodeID), 0, false, b)

		return c.SendStatus(200)
	})

	body := `{"node": "node-1", "node_prefix": "TEST", "interval": 10000}`
	req := httptest.NewRequest("POST", "/config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	assert.Equal(t, 200, resp.StatusCode)
	mockMQTT.AssertExpectations(t)
}

func TestOTATrigger(t *testing.T) {
	app := fiber.New()
	mockMQTT := new(MockMQTTClient)
	core.MqttClient = mockMQTT

	otaURL := "http://example.com/firmware.bin"
	expectedPayload := fmt.Sprintf("cmd=ota&url=%s", url.QueryEscape(otaURL))

	mockMQTT.On("Publish", "nodes/node-1/command", mock.Anything, mock.Anything, []byte(expectedPayload)).Return(&MockToken{})

	app.Post("/ota", func(c *fiber.Ctx) error {
		type O struct {
			Node string `json:"node"`
			URL  string `json:"url"`
		}
		var o O
		c.BodyParser(&o)
		b := []byte(fmt.Sprintf("cmd=ota&url=%s", url.QueryEscape(o.URL)))
		core.MqttClient.Publish(fmt.Sprintf("nodes/%s/command", o.Node), 0, false, b)
		return c.SendStatus(200)
	})

	body := fmt.Sprintf(`{"node": "node-1", "url": "%s"}`, otaURL)
	req := httptest.NewRequest("POST", "/ota", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	assert.Equal(t, 200, resp.StatusCode)
	mockMQTT.AssertExpectations(t)
}
