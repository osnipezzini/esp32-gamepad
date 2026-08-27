#include "Axis.h"
#include <Arduino.h>
#include "Config.h"

void Axis::begin(uint8_t pin, bool invert) {
    _pin = pin;
    _invert = invert;
    pinMode(_pin, INPUT);
    _filtered = analogRead(_pin);
}

void Axis::update() {
    int raw = analogRead(_pin); // 0-4095

    // Deadzone: ignora variações pequenas (ruído do potenciômetro/ADC).
    if (abs(raw - (int)_filtered) < AXIS_DEADZONE) {
        return;
    }

    // Média móvel exponencial: suaviza o restante do ruído sem introduzir
    // muito atraso na resposta.
    _filtered = (AXIS_SMOOTHING * raw) + ((1.0f - AXIS_SMOOTHING) * _filtered);
}

uint8_t Axis::getValue() const {
    int value = (int)_filtered;
    if (_invert) {
        value = 4095 - value;
    }
    return (uint8_t)constrain(map(value, 0, 4095, 0, 255), 0, 255);
}
