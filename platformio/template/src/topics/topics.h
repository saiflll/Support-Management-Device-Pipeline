#ifndef TOPICS_H
#define TOPICS_H

#include <Arduino.h>

/// Build MQTT topic strings for MDCW device.
/// Format:
///   telemetry: iot/data/mdcw/{nodeId}/telemetry            (binary 20B)
///   heartbeat: iot/node/mdcw/{nodeId}/heartbeat            (binary 35B, retained)
///   command:   nodes/{nodeId}/command                      (subscribe)
///   status:    nodes/{nodeId}/status                       (publish, retained)
///   monitor:   nodes/{nodeId}/monitor                      (publish)

class Topics {
public:
    static void configure(const String& nodeId);

    static const String& telemetry();
    static const String& heartbeat();
    static const String& command();
    static const String& status();
    static const String& monitor();

private:
    static String s_telemetry;
    static String s_heartbeat;
    static String s_command;
    static String s_status;
    static String s_monitor;
    static bool   s_configured;
};

#endif // TOPICS_H
