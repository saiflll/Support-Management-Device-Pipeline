#ifndef HARDWARE_CONFIG_H
#define HARDWARE_CONFIG_H

// ══════════════════════════════════════════════════════════════
//  Hardware Pin Configuration — MDCW Device
//  Override via build flags di platformio.ini / platformio_local.ini
// ══════════════════════════════════════════════════════════════

// ── HX711 (Scale) ──
#ifndef PIN_SCALE_DOUT
#define PIN_SCALE_DOUT  19
#endif
#ifndef PIN_SCALE_SCK
#define PIN_SCALE_SCK   23
#endif

// ── Proximity Sensors (MDCW V2) ──
#ifndef PIN_PROX_1
#define PIN_PROX_1      17
#endif
#ifndef PIN_PROX_2
#define PIN_PROX_2      5
#endif
#ifndef PIN_PROX_3
#define PIN_PROX_3      18
#endif

// ── Relay ──
#ifndef PIN_RELAY
#define PIN_RELAY       16
#endif

// ── Serial Scanner (TROLI) ──
#ifndef SCANNER_RX
#define SCANNER_RX      16
#endif
#ifndef SCANNER_TX
#define SCANNER_TX      -1   // -1 = TX not used (RX only)
#endif
#ifndef SCANNER_BAUD
#define SCANNER_BAUD    9600
#endif

// ── LED ──
#ifndef PIN_LED
#define PIN_LED         2     // Built-in LED
#endif

#endif // HARDWARE_CONFIG_H
