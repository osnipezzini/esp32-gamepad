#pragma once
#include <cstdint>
#include "Config.h"

// Controla LEDs que NÃO fazem parte do report HID do gamepad — são
// indicadores próprios seus (porta aberta, alerta, status de conexão, etc.).
// Use como quiser a partir do main.cpp: ligar/desligar/piscar conforme
// qualquer condição da sua aplicação (inclusive lendo botões que você
// decida tratar como "sensores" em vez de botões de jogo).
class LedController {
public:
    void begin();
    void set(uint8_t index, bool on);
    void toggle(uint8_t index);

    // Chame a cada loop; o LED pisca sozinho no período informado.
    void blink(uint8_t index, uint32_t periodMs);

private:
    uint32_t _lastBlinkMs[LED_COUNT] = {0};
    bool _blinkState[LED_COUNT] = {false};
};
