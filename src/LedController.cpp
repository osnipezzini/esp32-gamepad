#include "LedController.h"
#include <Arduino.h>

void LedController::begin() {
    for (uint8_t i = 0; i < LED_COUNT; i++) {
        pinMode(LED_PINS[i], OUTPUT);
        digitalWrite(LED_PINS[i], LOW);
    }
}

void LedController::set(uint8_t index, bool on) {
    if (index >= LED_COUNT) return;
    digitalWrite(LED_PINS[index], on ? HIGH : LOW);
}

void LedController::toggle(uint8_t index) {
    if (index >= LED_COUNT) return;
    digitalWrite(LED_PINS[index], !digitalRead(LED_PINS[index]));
}

void LedController::blink(uint8_t index, uint32_t periodMs) {
    if (index >= LED_COUNT) return;
    uint32_t now = millis();
    if (now - _lastBlinkMs[index] >= periodMs) {
        _lastBlinkMs[index] = now;
        _blinkState[index] = !_blinkState[index];
        digitalWrite(LED_PINS[index], _blinkState[index] ? HIGH : LOW);
    }
}
