#include "mqtt.h"

// ── Statics ─────────────────────────────────────────────────────
WiFiClient     MqttNet::s_wifiClient;
PubSubClient   MqttNet::s_client(s_wifiClient);
const char*    MqttNet::s_broker = nullptr;
uint16_t       MqttNet::s_port = 1883;
const char*    MqttNet::s_user = nullptr;
const char*    MqttNet::s_pass = nullptr;
const char*    MqttNet::s_clientId = nullptr;
MqttNet::MessageCallback MqttNet::s_callback = nullptr;
uint32_t       MqttNet::s_publishCount = 0;
uint32_t       MqttNet::s_reconnectCount = 0;
unsigned long  MqttNet::s_lastReconnect = 0;

void MqttNet::begin(const char* broker, uint16_t port,
                    const char* user, const char* pass,
                    const char* clientId) {
    s_broker = broker;
    s_port = port;
    s_user = user;
    s_pass = pass;
    s_clientId = clientId;

    s_client.setServer(s_broker, s_port);
    s_client.setCallback(_onMessage);
    s_client.setBufferSize(2048);
}

void MqttNet::loop() {
    if (!s_client.connected()) {
        unsigned long now = millis();
        if (now - s_lastReconnect > 5000) {
            s_lastReconnect = now;
            reconnect();
        }
    } else {
        s_client.loop();
    }
}

bool MqttNet::reconnect() {
    if (WiFi.status() != WL_CONNECTED) return false;

    // LWT: offline status on nodes/{id}/status
    String willTopic = "iot/node/mdcw/";
    willTopic += s_clientId;
    willTopic += "/heartbeat"; // will be overwritten by retained heartbeat

    bool ok = s_client.connect(s_clientId, s_user, s_pass);
    if (ok) {
        s_reconnectCount++;
        // Application handles subscription on reconnect via system/connected
        if (s_callback) {
            // Notify application of reconnect
            String fake = "";
            s_callback("system/connected", (uint8_t*)fake.c_str(), 0);
        }
    }
    return ok;
}

bool MqttNet::isConnected() {
    return s_client.connected();
}

bool MqttNet::publish(const char* topic, const uint8_t* data, size_t len, bool retained) {
    if (!s_client.connected()) return false;
    bool ok = s_client.publish(topic, data, len, retained);
    if (ok) s_publishCount++;
    return ok;
}

bool MqttNet::publishStr(const char* topic, const char* str, bool retained) {
    return publish(topic, (const uint8_t*)str, strlen(str), retained);
}

bool MqttNet::subscribe(const char* topic) {
    if (!s_client.connected()) return false;
    return s_client.subscribe(topic);
}

void MqttNet::onMessage(MessageCallback cb) {
    s_callback = cb;
}

void MqttNet::_onMessage(char* topic, byte* payload, unsigned int length) {
    if (s_callback) {
        s_callback(String(topic), payload, length);
    }
}
