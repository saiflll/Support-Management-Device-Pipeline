#include "config.h"

AppConfig    Config::cfg;
Preferences  Config::_prefs;
bool         Config::_loaded = false;

void Config::init() {
    _prefs.begin("servfor", false);
    if (!_prefs.isKey("initialized")) {
        // First boot — set defaults
        reset();
        _prefs.putBool("initialized", true);
    }
    load();
}

void Config::load() {
    if (_loaded) return;
    cfg.interval  = _prefs.getUShort("interval", 5000);
    cfg.area      = _prefs.getInt("area", 20);
    cfg.prefix    = _prefs.getString("prefix", "MDCW1 (UK)");
    cfg.ssid      = _prefs.getString("ssid", "");
    cfg.pass      = _prefs.getString("pass", "");
    cfg.mqttBroker= _prefs.getString("mqttBroker", "10.201.40.1");
    cfg.mqttPort  = _prefs.getUShort("mqttPort", 1883);
    cfg.mqttUser  = _prefs.getString("mqttUser", "apps");
    cfg.mqttPass  = _prefs.getString("mqttPass", "apps");
    _loaded = true;
}

void Config::save() {
    _prefs.putUShort("interval", cfg.interval);
    _prefs.putInt("area", cfg.area);
    _prefs.putString("prefix", cfg.prefix);
    _prefs.putString("ssid", cfg.ssid);
    _prefs.putString("pass", cfg.pass);
    _prefs.putString("mqttBroker", cfg.mqttBroker);
    _prefs.putUShort("mqttPort", cfg.mqttPort);
    _prefs.putString("mqttUser", cfg.mqttUser);
    _prefs.putString("mqttPass", cfg.mqttPass);
}

void Config::reset() {
    cfg.interval   = 5000;
    cfg.area       = 20;
    cfg.prefix     = "MDCW1 (UK)";
    cfg.ssid       = "";
    cfg.pass       = "";
    cfg.mqttBroker = "10.201.40.1";
    cfg.mqttPort   = 1883;
    cfg.mqttUser   = "apps";
    cfg.mqttPass   = "apps";
    save();
}
