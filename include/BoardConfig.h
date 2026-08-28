#pragma once
#include <stdint.h>
#include "Config.h" // MATRIX_ROWS / MATRIX_COLS

// ============================================================================
//  Tipos que descrevem o HARDWARE FÍSICO da placa (quais pinos fazem o quê).
//
//  Em vez de arrays soltos de uint8_t indexados por posição (onde é fácil
//  trocar "linha 2" com "coluna 2" sem o compilador reclamar), cada grupo de
//  pinos vira um objeto com campos nomeados. Isso também deixa explícito,
//  no protótipo de cada função, exatamente que dado ela espera receber.
// ============================================================================

// Pinos da matriz de botões: uma linha por posição, uma coluna por posição.
struct MatrixPins {
    uint8_t rows[MATRIX_ROWS];
    uint8_t cols[MATRIX_COLS];
};

// Um pino por eixo, com nome em vez de índice de array.
struct AxisPins {
    uint8_t x;
    uint8_t y;
    uint8_t z;
    uint8_t rz;
};

// Sinaliza se cada eixo deve inverter a leitura (0 vira 255 e vice-versa).
struct AxisInvertFlags {
    bool x = false;
    bool y = false;
    bool z = false;
    bool rz = false;
};

// Parâmetros de leitura do ADC — variam por chip (resolução do conversor),
// não por placa específica, mas moram aqui porque descrevem como interpretar
// o que vem dos pinos de eixo.
struct AxisReadSettings {
    uint16_t adcMaxValue; // valor máximo bruto do ADC (4095 no ESP32, 1023 no AVR)
    uint16_t deadzone;    // contagens ADC ignoradas como ruído
    float smoothing;      // fator do filtro EMA (0=muito suave/lento, 1=sem filtro)
};

// Um pino por LED de status, com papel nomeado em vez de índice genérico.
struct LedPins {
    uint8_t doorOpen;
    uint8_t alert;
    uint8_t status;
    uint8_t spare;
};

// Agrega tudo que descreve fisicamente a placa de destino.
struct BoardConfig {
    MatrixPins matrix;
    AxisPins axis;
    AxisInvertFlags axisInvert;
    AxisReadSettings axisRead;
    LedPins leds;
};

// Implementado separadamente por plataforma — nenhum #if defined aqui:
//   src/BoardConfigEsp32.cpp  (compilado apenas no ambiente esp32dev)
//   src/BoardConfigAvr.cpp    (compilado nos ambientes leonardo e promicro16)
const BoardConfig& getBoardConfig();
