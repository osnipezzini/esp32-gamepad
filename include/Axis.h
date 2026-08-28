#pragma once
#include <stdint.h>
#include "BoardConfig.h"

// Um eixo analógico individual: lê o ADC, aplica deadzone + filtro EMA para
// reduzir ruído, e converte para 0-255 (o que o report HID espera).
// Todas as configurações (resolução do ADC, deadzone, suavização) chegam via
// AxisReadSettings — a classe não depende de nenhuma constante global.
class Axis {
public:
    void begin(uint8_t pin, bool invert, const AxisReadSettings& settings);
    void update();
    uint8_t getValue() const;

private:
    uint8_t _pin = 0;
    bool _invert = false;
    AxisReadSettings _settings{};
    // Valor inicial é só um placeholder — begin() sobrescreve com a
    // primeira leitura real do ADC antes de qualquer uso.
    float _filtered = 0.0f;
};
