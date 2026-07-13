#include "ota.h"
#include "../system/watchdog.h"
#include "../system/logger.h"
#include "../diagnostic/diagnostic.h"

#include <HTTPClient.h>
#include <Update.h>
#include <esp_partition.h>
#include <esp_ota_ops.h>
#include <mbedtls/sha256.h>

bool   OTA::s_updating = false;
String OTA::s_lastError = "";

void OTA::perform(const String& url, const String& expectedSha256) {
    if (s_updating) {
        Logger::warn("OTA already in progress");
        return;
    }
    if (url.length() == 0) {
        Logger::error("OTA: empty URL");
        s_lastError = "Empty URL";
        return;
    }

    s_updating = true;
    s_lastError = "";
    Logger::info("OTA: starting update from %s", url.c_str());

    // ── 1. HTTP GET ──
    HTTPClient http;
    WiFiClient tcp;
    http.setTimeout(OTA_TIMEOUT);
    http.setFollowRedirects(HTTPC_STRICT_FOLLOW_REDIRECTS);

    if (!http.begin(tcp, url)) {
        s_lastError = "HTTP begin failed";
        Logger::error("OTA: %s", s_lastError.c_str());
        s_updating = false;
        return;
    }

    int httpCode = http.GET();
    if (httpCode != HTTP_CODE_OK) {
        s_lastError = "HTTP ";
        s_lastError += httpCode;
        Logger::error("OTA: HTTP %d", httpCode);
        http.end();
        s_updating = false;
        return;
    }

    int contentLength = http.getSize();
    if (contentLength <= 0) {
        s_lastError = "Invalid content length";
        Logger::error("OTA: %s", s_lastError.c_str());
        http.end();
        s_updating = false;
        return;
    }

    Logger::info("OTA: size=%d bytes, verifying partition...", contentLength);

    // ── 2. Prepare partition ──
    if (!Update.begin(contentLength)) {
        s_lastError = Update.errorString();
        Logger::error("OTA: Update.begin: %s", s_lastError.c_str());
        http.end();
        s_updating = false;
        return;
    }

    // ── 3. Stream download → flash ──
    WiFiClient* stream = http.getStreamPtr();
    uint8_t buf[OTA_CHUNK_SIZE];
    size_t total = 0;
    bool failed = false;

    while (total < (size_t)contentLength) {
        int avail = stream->available();
        if (avail == 0) {
            Watchdog::feed();
            delay(10);
            continue;
        }

        size_t toRead = min((size_t)avail, OTA_CHUNK_SIZE);
        size_t len = stream->readBytes(buf, toRead);

        size_t written = Update.write(buf, len);
        if (written != len) {
            s_lastError = "Flash write mismatch";
            Logger::error("OTA: write error at byte %u", total);
            failed = true;
            break;
        }

        total += written;
        Watchdog::feed();
    }

    if (failed) {
        Update.abort();
        http.end();
        s_updating = false;
        return;
    }

    // ── 4. Finalize ──
    if (!Update.end(true)) {
        s_lastError = Update.errorString();
        Logger::error("OTA: Update.end: %s", s_lastError.c_str());
        http.end();
        s_updating = false;
        return;
    }

    http.end();

    if (!Update.isFinished()) {
        s_lastError = "Update not finished";
        Logger::error("OTA: %s", s_lastError.c_str());
        s_updating = false;
        return;
    }

    Logger::info("OTA: flashed %u bytes OK", total);

    // ── 5. Optional SHA256 verification ──
    if (expectedSha256.length() == 64) {
        const esp_partition_t* running = esp_ota_get_running_partition();
        if (running) {
            uint8_t hash[32];
            char hexHash[65];
            esp_partition_get_sha256(running, hash);
            for (int i = 0; i < 32; i++) {
                sprintf(hexHash + i * 2, "%02x", hash[i]);
            }
            hexHash[64] = '\0';

            if (strcasecmp(hexHash, expectedSha256.c_str()) != 0) {
                s_lastError = "SHA256 mismatch";
                Logger::error("OTA: %s", s_lastError.c_str());
                // JANGAN reboot — wait for backend retry
                s_updating = false;
                return;
            }
            Logger::info("OTA: SHA256 verified OK");
        }
    }

    // ── 6. Reboot ──
    Logger::info("OTA: rebooting in 500ms...");
    s_updating = false;
    delay(500);
    ESP.restart();
}
