#ifndef HEARTBEAT_H
#define HEARTBEAT_H

#include <Arduino.h>

#define BEAT_INTERVAL 30000  // 30 detik

class Heartbeat {
public:
    static void init();
    static void publishTick(unsigned long nowMs);
    static void publishNow();

private:
    static unsigned long s_lastBeat;
    static void doPublish();
};

#endif // HEARTBEAT_H
