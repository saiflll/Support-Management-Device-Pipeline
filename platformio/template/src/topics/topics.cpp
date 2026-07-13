#include "topics.h"

String Topics::s_telemetry;
String Topics::s_heartbeat;
String Topics::s_command;
String Topics::s_status;
String Topics::s_monitor;
bool   Topics::s_configured = false;

void Topics::configure(const String& nodeId) {
    s_telemetry = "iot/data/mdcw/";
    s_telemetry += nodeId;
    s_telemetry += "/telemetry";

    s_heartbeat = "iot/node/mdcw/";
    s_heartbeat += nodeId;
    s_heartbeat += "/heartbeat";

    s_command = "nodes/";
    s_command += nodeId;
    s_command += "/command";

    s_status = "nodes/";
    s_status += nodeId;
    s_status += "/status";

    s_monitor = "nodes/";
    s_monitor += nodeId;
    s_monitor += "/monitor";

    s_configured = true;
}

const String& Topics::telemetry()  { return s_telemetry; }
const String& Topics::heartbeat()  { return s_heartbeat; }
const String& Topics::command()    { return s_command; }
const String& Topics::status()     { return s_status; }
const String& Topics::monitor()    { return s_monitor; }
