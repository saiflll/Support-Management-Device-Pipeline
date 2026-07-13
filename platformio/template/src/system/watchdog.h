#ifndef WATCHDOG_H
#define WATCHDOG_H

#include <Arduino.h>
#include "esp_task_wdt.h"

#define WDT_TIMEOUT_SECONDS 15

class Watchdog {
public:
    /// Initialize hardware watchdog.
    /// Must be fed at least every WDT_TIMEOUT_SECONDS.
    static void init() {
        esp_task_wdt_init(WDT_TIMEOUT_SECONDS, true);  // panic on timeout
        esp_task_wdt_add(NULL);  // current task
    }

    /// Feed (reset) the watchdog timer.
    static void feed() {
        esp_task_wdt_reset();
    }
};

#endif // WATCHDOG_H
