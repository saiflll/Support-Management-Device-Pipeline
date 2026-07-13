#ifndef OTA_H
#define OTA_H

#include <Arduino.h>

#define OTA_TIMEOUT     15000   // ms
#define OTA_CHUNK_SIZE  1024    // bytes

class OTA {
public:
    /// Perform OTA update from HTTP URL.
    /// @param url: HTTP URL to firmware binary.
    /// @param expectedSha256: 64-char hex SHA256 (empty to skip verify).
    static void perform(const String& url, const String& expectedSha256);

    /// True if OTA is in progress.
    static bool isUpdating() { return s_updating; }

    /// Get last error string.
    static const String& lastError() { return s_lastError; }

private:
    static bool s_updating;
    static String s_lastError;
};

#endif // OTA_H
