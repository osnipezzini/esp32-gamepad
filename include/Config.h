#pragma once
#include <stdint.h>
#include <string.h>
#include "InputTypes.h"

// ============================================================================
//  Configuração de COMPORTAMENTO — independe de qual placa física está
//  rodando o firmware. A pinagem de cada placa vive em BoardConfig.h e nos
//  arquivos BoardConfig<Plataforma>.cpp, não aqui.
// ============================================================================

static const uint8_t MATRIX_ROWS = 4;
static const uint8_t MATRIX_COLS = 4;
static const uint8_t BUTTON_COUNT = MATRIX_ROWS * MATRIX_COLS; // 16

static const uint8_t AXIS_COUNT = 4;
static const uint8_t LED_COUNT  = 4;

#if defined(ESP32)
#include <Preferences.h>
// Identidade anunciada pelo dispositivo BLE HID no ESP32.
static const char* BLE_DEVICE_NAME = "ButtonBox";
static const char* BLE_MANUFACTURER = "SODevs";
#define BUTTONBOX_STORAGE_NAMESPACE "buttonbox"
#else
#include <EEPROM.h>
#define BUTTONBOX_STORAGE_NAMESPACE "buttonbox"
#endif

// Comportamento de cada botão. Índice = row * MATRIX_COLS + col.
// A estrutura pode ser alterada em runtime pela GUI/serial sem mexer em
// arquivos de hardware; o armazenamento persistente fica aqui.
static ButtonMode BUTTON_MODES[BUTTON_COUNT] = {
    /* 0*/ ButtonMode::NORMAL,
    /* 1*/ ButtonMode::NORMAL,
    /* 2*/ ButtonMode::ONE_SHOT,     // segurar = apenas 1 pulso curto (press+release)
    /* 3*/ ButtonMode::NORMAL,
    /* 4*/ ButtonMode::TOGGLE,       // 1º clique liga, 2º clique desliga
    /* 5*/ ButtonMode::NORMAL,
    /* 6*/ ButtonMode::NORMAL,
    /* 7*/ ButtonMode::LONG_PRESS,   // só ativa se segurar além do limiar
    /* 8*/ ButtonMode::NORMAL,
    /* 9*/ ButtonMode::NORMAL,
    /*10*/ ButtonMode::NORMAL,
    /*11*/ ButtonMode::NORMAL,
    /*12*/ ButtonMode::NORMAL,
    /*13*/ ButtonMode::NORMAL,
    /*14*/ ButtonMode::NORMAL,
    /*15*/ ButtonMode::NORMAL,
};

inline uint8_t encodeButtonMode(ButtonMode mode) {
    return static_cast<uint8_t>(mode);
}

inline ButtonMode decodeButtonMode(uint8_t raw) {
    switch (raw) {
        case static_cast<uint8_t>(ButtonMode::ONE_SHOT): return ButtonMode::ONE_SHOT;
        case static_cast<uint8_t>(ButtonMode::TOGGLE): return ButtonMode::TOGGLE;
        case static_cast<uint8_t>(ButtonMode::LONG_PRESS): return ButtonMode::LONG_PRESS;
        case static_cast<uint8_t>(ButtonMode::NORMAL):
        default: return ButtonMode::NORMAL;
    }
}

inline void resetButtonModesToDefault() {
    BUTTON_MODES[0] = ButtonMode::NORMAL;
    BUTTON_MODES[1] = ButtonMode::NORMAL;
    BUTTON_MODES[2] = ButtonMode::ONE_SHOT;
    BUTTON_MODES[3] = ButtonMode::NORMAL;
    BUTTON_MODES[4] = ButtonMode::TOGGLE;
    BUTTON_MODES[5] = ButtonMode::NORMAL;
    BUTTON_MODES[6] = ButtonMode::NORMAL;
    BUTTON_MODES[7] = ButtonMode::LONG_PRESS;
    BUTTON_MODES[8] = ButtonMode::NORMAL;
    BUTTON_MODES[9] = ButtonMode::NORMAL;
    BUTTON_MODES[10] = ButtonMode::NORMAL;
    BUTTON_MODES[11] = ButtonMode::NORMAL;
    BUTTON_MODES[12] = ButtonMode::NORMAL;
    BUTTON_MODES[13] = ButtonMode::NORMAL;
    BUTTON_MODES[14] = ButtonMode::NORMAL;
    BUTTON_MODES[15] = ButtonMode::NORMAL;
}

inline const char* buttonModeToString(ButtonMode mode) {
    switch (mode) {
        case ButtonMode::ONE_SHOT: return "ONE_SHOT";
        case ButtonMode::TOGGLE: return "TOGGLE";
        case ButtonMode::LONG_PRESS: return "LONG_PRESS";
        case ButtonMode::NORMAL:
        default: return "NORMAL";
    }
}

inline ButtonMode parseButtonModeString(const char* value) {
    if (value == nullptr) {
        return ButtonMode::NORMAL;
    }

    const char* normalized = value;
    while (*normalized == ' ' || *normalized == '\t' || *normalized == '\r' || *normalized == '\n') {
        normalized++;
    }

    if (strcmp(normalized, "ONE_SHOT") == 0) return ButtonMode::ONE_SHOT;
    if (strcmp(normalized, "TOGGLE") == 0) return ButtonMode::TOGGLE;
    if (strcmp(normalized, "LONG_PRESS") == 0) return ButtonMode::LONG_PRESS;
    return ButtonMode::NORMAL;
}

inline void setButtonMode(uint8_t index, ButtonMode mode) {
    if (index < BUTTON_COUNT) {
        BUTTON_MODES[index] = mode;
    }
}

inline ButtonMode getButtonMode(uint8_t index) {
    if (index < BUTTON_COUNT) {
        return BUTTON_MODES[index];
    }
    return ButtonMode::NORMAL;
}

inline void saveButtonModesToStorage() {
#if defined(ESP32)
    Preferences prefs;
    prefs.begin(BUTTONBOX_STORAGE_NAMESPACE, false);
    uint8_t payload[BUTTON_COUNT];
    for (uint8_t i = 0; i < BUTTON_COUNT; i++) {
        payload[i] = encodeButtonMode(BUTTON_MODES[i]);
    }
    prefs.putBytes("modes", payload, sizeof(payload));
    prefs.putUChar("magic", 0xB1);
    prefs.end();
#else
    EEPROM.update(0, 0xB1);
    for (uint8_t i = 0; i < BUTTON_COUNT; i++) {
        EEPROM.update(1 + i, encodeButtonMode(BUTTON_MODES[i]));
    }
#endif
}

inline void loadButtonModesFromStorage() {
    resetButtonModesToDefault();

#if defined(ESP32)
    Preferences prefs;
    prefs.begin(BUTTONBOX_STORAGE_NAMESPACE, false);
    uint8_t magic = prefs.getUChar("magic", 0x00);
    if (magic == 0xB1) {
        uint8_t payload[BUTTON_COUNT];
        if (prefs.getBytes("modes", payload, sizeof(payload)) == sizeof(payload)) {
            for (uint8_t i = 0; i < BUTTON_COUNT; i++) {
                BUTTON_MODES[i] = decodeButtonMode(payload[i]);
            }
        }
    } else {
        saveButtonModesToStorage();
    }
    prefs.end();
#else
    if (EEPROM.read(0) != 0xB1) {
        saveButtonModesToStorage();
        return;
    }
    for (uint8_t i = 0; i < BUTTON_COUNT; i++) {
        BUTTON_MODES[i] = decodeButtonMode(EEPROM.read(1 + i));
    }
#endif
}

static const uint32_t DEBOUNCE_MS             = 12;
static const uint32_t ONE_SHOT_PULSE_MS       = 80;
static const uint32_t LONG_PRESS_THRESHOLD_MS = 500;

// Loop principal: intervalo entre leituras/envios de report (125 Hz aprox.)
static const uint32_t SCAN_INTERVAL_MS = 8;
