#pragma once
#include <Arduino.h>
#include "InputTypes.h"

// ============================================================================
//  Este é o ÚNICO arquivo que você deveria precisar editar para adaptar o
//  projeto ao seu hardware: pinos, quantidade de botões/eixos/LEDs e o
//  comportamento de cada botão.
// ============================================================================

// ----------------------------------------------------------------------------
//  MATRIZ DE BOTÕES (4x4 = 16 botões usando apenas 8 GPIOs)
// ----------------------------------------------------------------------------
static const uint8_t MATRIX_ROWS = 4;
static const uint8_t MATRIX_COLS = 4;
static const uint8_t BUTTON_COUNT = MATRIX_ROWS * MATRIX_COLS; // 16

// Linhas: saídas, ficam em HIGH e são levadas a LOW uma de cada vez durante o scan.
static const uint8_t ROW_PINS[MATRIX_ROWS] = {13, 12, 14, 27};

// Colunas: entradas com pull-up interno. Botão fecha a coluna para a linha ativa (nível LOW = pressionado).
static const uint8_t COL_PINS[MATRIX_COLS] = {26, 25, 33, 32};

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

static const uint32_t DEBOUNCE_MS              = 12;   // tempo de estabilização do sinal físico
static const uint32_t ONE_SHOT_PULSE_MS        = 80;   // duração do pulso gerado pelo modo ONE_SHOT
static const uint32_t LONG_PRESS_THRESHOLD_MS  = 500;  // tempo mínimo segurando para o modo LONG_PRESS ativar

// ----------------------------------------------------------------------------
//  EIXOS ANALÓGICOS (potenciômetros, manoplas, etc.)
//  Usar apenas pinos ADC1 — ADC2 tem conflito conhecido com o rádio Wi-Fi/BLE.
// ----------------------------------------------------------------------------
static const uint8_t AXIS_COUNT = 4;
static const uint8_t AXIS_PINS[AXIS_COUNT] = {34, 35, 36, 39}; // X, Y, Z, Rz
static const bool AXIS_INVERT[AXIS_COUNT]  = {false, false, false, false};

static const float AXIS_SMOOTHING   = 0.25f; // filtro EMA: 0=muito suave/lento, 1=sem filtro
static const uint16_t AXIS_DEADZONE = 20;    // contagens ADC (faixa 0-4095) ignoradas como ruído

// ----------------------------------------------------------------------------
//  LEDS DE STATUS (independentes do report HID — lógica 100% sua)
// ----------------------------------------------------------------------------
static const uint8_t LED_COUNT = 4;
static const uint8_t LED_PINS[LED_COUNT] = {2, 4, 5, 18};

enum LedIndex : uint8_t {
    LED_DOOR_OPEN  = 0,  // exemplo: indicador de porta aberta
    LED_ALERT      = 1,  // exemplo: indicador de alerta genérico
    LED_BLE_STATUS = 2,  // aceso quando há uma conexão BLE ativa
    LED_SPARE      = 3,  // livre para uso futuro
};

// ----------------------------------------------------------------------------
//  IDENTIDADE DO DISPOSITIVO BLE
// ----------------------------------------------------------------------------
static const char* BLE_DEVICE_NAME = "ButtonBox";
static const char* BLE_MANUFACTURER = "DIY";

// Loop principal: intervalo entre leituras/envios de report (125 Hz aprox.)
static const uint32_t SCAN_INTERVAL_MS = 8;
