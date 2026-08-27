#pragma once
#include <cstdint>

// Um eixo analógico individual: lê o ADC, aplica deadzone + filtro EMA
// para reduzir ruído, e converte para 0-255 (o que o report HID espera).
class Axis {
public:
    void begin(uint8_t pin, bool invert);
    void update();
    uint8_t getValue() const;

private:
    uint8_t _pin = 0;
    bool _invert = false;
    float _filtered = 2048.0f; // ADC do ESP32 é 12-bit (0-4095); começa centralizado
};
