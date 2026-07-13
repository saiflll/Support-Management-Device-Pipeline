#ifndef CONFIG_H
#define CONFIG_H

#include <Arduino.h>
#include <Preferences.h>

#define FW_MAJOR 1
#define FW_MINOR 0

struct AppConfig {
    uint16_t interval;     // publish interval (ms)
    int      area;         // area ID
    String   prefix;       // device prefix (e.g. "MDCW1 (UK)")
    String   ssid;
    String   pass;
    String   mqttBroker;
    uint16_t mqttPort;
    String   mqttUser;
    String   mqttPass;
};

class Config {
public:
    static void init();
    static void load();
    static void save();
    static void reset();

    static AppConfig cfg;

private:
    static Preferences _prefs;
    static bool _loaded;
};

#endif // CONFIG_H
