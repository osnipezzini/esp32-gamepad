#include "BoardConfig.h"

// ============================================================================
//  Pinagem física do ESP32-WROOM-32 DevKit.
//  Este arquivo só é compilado no ambiente [env:esp32dev] do platformio.ini.
// ============================================================================

const BoardConfig& getBoardConfig() {
    static const BoardConfig board = []() {
        BoardConfig cfg{};

        // Matriz de botões: linhas = saída (driven LOW no scan), colunas = entrada com pull-up.
        cfg.matrix.rows[0] = 13;
        cfg.matrix.rows[1] = 12;
        cfg.matrix.rows[2] = 14;
        cfg.matrix.rows[3] = 27;
        cfg.matrix.cols[0] = 26;
        cfg.matrix.cols[1] = 25;
        cfg.matrix.cols[2] = 33;
        cfg.matrix.cols[3] = 32;

        // Eixos analógicos — usar apenas pinos ADC1 (GPIO32-39). O ADC2 tem
        // conflito conhecido com o rádio Wi-Fi/BLE do ESP32.
        cfg.axis.x  = 34;
        cfg.axis.y  = 35;
        cfg.axis.z  = 36;
        cfg.axis.rz = 39;

        cfg.axisInvert.x  = false;
        cfg.axisInvert.y  = false;
        cfg.axisInvert.z  = false;
        cfg.axisInvert.rz = false;

        cfg.axisRead.adcMaxValue = 4095; // ESP32: ADC de 12 bits
        cfg.axisRead.deadzone    = 20;
        cfg.axisRead.smoothing   = 0.25f;

        // LEDs de status — independentes do report HID.
        cfg.leds.doorOpen = 2;  // ⚠️ pino de strapping — ver README
        cfg.leds.alert    = 4;
        cfg.leds.status   = 5;  // usado como indicador de conexão BLE
        cfg.leds.spare    = 18;

        return cfg;
    }();

    return board;
}
