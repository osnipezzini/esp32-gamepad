#include "Axis.h"
#include <Arduino.h>

void Axis::begin(uint8_t pin, bool invert, const AxisReadSettings& settings) {
    _pin = pin;
    _invert = invert;
    _settings = settings;
    pinMode(_pin, INPUT);
    _filtered = analogRead(_pin);
}

void Axis::update() {
    int raw = analogRead(_pin); // 0 a _settings.adcMaxValue

    // Deadzone: ignora variações pequenas (ruído do potenciômetro/ADC).
    if (abs(raw - (int)_filtered) < _settings.deadzone) {
        return;
    }

    // Média móvel exponencial: suaviza o restante do ruído sem introduzir
    // muito atraso na resposta.
    _filtered = (_settings.smoothing * raw) + ((1.0f - _settings.smoothing) * _filtered);
}

uint8_t Axis::getValue() const {
    int value = (int)_filtered;
    if (_invert) {
        value = _settings.adcMaxValue - value;
    }
    return (uint8_t)constrain(map(value, 0, _settings.adcMaxValue, 0, 255), 0, 255);
}