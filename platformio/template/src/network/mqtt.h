#ifndef MQTT_NET_H
#define MQTT_NET_H

#include <Arduino.h>
#include <WiFi.h>
#include <PubSubClient.h>
#include <functional>

class MqttNet {
public:
    using MessageCallback = std::function<void(const String& topic, const uint8_t* payload, size_t len)>;

    static void begin(const char* broker, uint16_t port,
                      const char* user, const char* pass,
                      const char* clientId);
    static void loop();
    static bool reconnect();
    static bool isConnected();
    static bool publish(const char* topic, const uint8_t* data, size_t len, bool retained);
    static bool publishStr(const char* topic, const char* str, bool retained);
    static bool subscribe(const char* topic);
    static void onMessage(MessageCallback cb);

    // Statistics
    static uint32_t publishCount() { return s_publishCount; }
    static uint32_t reconnectCount() { return s_reconnectCount; }

private:
    static WiFiClient     s_wifiClient;
    static PubSubClient   s_client;
    static const char*    s_broker;
    static uint16_t       s_port;
    static const char*    s_user;
    static const char*    s_pass;
    static const char*    s_clientId;
    static MessageCallback s_callback;
    static uint32_t       s_publishCount;
    static uint32_t       s_reconnectCount;
    static unsigned long  s_lastReconnect;

    static void _onMessage(char* topic, byte* payload, unsigned int length);
};

#endif // MQTT_NET_H
