#ifndef CMD_CONFIG_H
#define CMD_CONFIG_H

#include <ArduinoJson.h>

void handleSetConfig(JsonDocument& doc);
void handleGetConfig();
void handleReboot();

#endif // CMD_CONFIG_H
