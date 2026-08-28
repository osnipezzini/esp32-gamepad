#pragma once
#include <stdint.h>
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
// Identidade anunciada pelo dispositivo BLE HID no ESP32.
static const char* BLE_DEVICE_NAME = "ButtonBox";
static const char* BLE_MANUFACTURER = "SODevs";
#endif

// Comportamento de cada botão. Índice = row * MATRIX_COLS + col.
// Para mudar o comportamento de um botão físico, só troque o modo aqui —
// nenhuma outra parte do código precisa ser tocada.
static const ButtonMode BUTTON_MODES[BUTTON_COUNT] = {
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

static const uint32_t DEBOUNCE_MS             = 12;
static const uint32_t ONE_SHOT_PULSE_MS       = 80;
static const uint32_t LONG_PRESS_THRESHOLD_MS = 500;

// Loop principal: intervalo entre leituras/envios de report (125 Hz aprox.)
static const uint32_t SCAN_INTERVAL_MS = 8;
