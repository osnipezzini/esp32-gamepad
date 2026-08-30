#include "BoardConfig.h"
#include <Arduino.h> // A0-A3
#include <EEPROM.h>  // EEPROM para salvar configurações dos botões

// ============================================================================
//  Pinagem física do ATmega32U4 (Arduino Leonardo / SparkFun Pro Micro).
//  As duas placas compartilham a mesma numeração lógica de pino no core
//  Arduino, então este único arquivo serve para os dois ambientes
//  [env:leonardo] e [env:promicro16] do platformio.ini.
// ============================================================================

// Inicializa EEPROM uma vez no startup para AVR
static bool eepromInitialized = false;
static void initEEPROM() {
    if (!eepromInitialized) {
        EEPROM.begin(17);  // 1 byte magic + 16 bytes modos
        eepromInitialized = true;
    }
}

const BoardConfig& getBoardConfig() {
    // Garante que EEPROM está inicializada antes de qualquer uso
    initEEPROM();
    
    static const BoardConfig board = []() {
        BoardConfig cfg{};

        // Matriz de botões: linhas = saída (driven LOW no scan), colunas = entrada com pull-up.
        cfg.matrix.rows[0] = 2;
        cfg.matrix.rows[1] = 3;
        cfg.matrix.rows[2] = 4;
        cfg.matrix.rows[3] = 5;
        cfg.matrix.cols[0] = 6;
        cfg.matrix.cols[1] = 7;
        cfg.matrix.cols[2] = 8;
        cfg.matrix.cols[3] = 9;

        // Eixos analógicos.
        cfg.axis.x  = A0;
        cfg.axis.y  = A1;
        cfg.axis.z  = A2;
        cfg.axis.rz = A3;

        cfg.axisInvert.x  = false;
        cfg.axisInvert.y  = false;
        cfg.axisInvert.z  = false;
        cfg.axisInvert.rz = false;

        cfg.axisRead.adcMaxValue = 1023; // ATmega32U4: ADC de 10 bits
        cfg.axisRead.deadzone    = 5;
        cfg.axisRead.smoothing   = 0.25f;

        // LEDs de status — independentes do report HID.
        // Nota: pinos 14/15/16 ficam no header ICSP no Leonardo (não no
        // header principal); no Pro Micro já vêm soldados na borda. Veja o README.
        cfg.leds.doorOpen = 10;
        cfg.leds.alert    = 14;
        cfg.leds.status   = 15; // sempre aceso enquanto o USB estiver ligado
        cfg.leds.spare    = 16;

        return cfg;
    }();

    return board;
}
