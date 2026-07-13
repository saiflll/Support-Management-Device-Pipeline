#ifndef LOGGER_H
#define LOGGER_H

#include <Arduino.h>

/// Minimal serial logger.
/// Use -D SERIAL_DEBUG build flag to enable.
class Logger {
public:
    static void info(const char* fmt, ...) {
#ifdef SERIAL_DEBUG
        va_list args;
        va_start(args, fmt);
        Serial.print("[I] ");
        Serial.printf(fmt, args);
        Serial.println();
        va_end(args);
#endif
    }

    static void warn(const char* fmt, ...) {
#ifdef SERIAL_DEBUG
        va_list args;
        va_start(args, fmt);
        Serial.print("[W] ");
        Serial.printf(fmt, args);
        Serial.println();
        va_end(args);
#endif
    }

    static void error(const char* fmt, ...) {
#ifdef SERIAL_DEBUG
        va_list args;
        va_start(args, fmt);
        Serial.print("[E] ");
        Serial.printf(fmt, args);
        Serial.println();
        va_end(args);
#endif
    }
};

#endif // LOGGER_H
