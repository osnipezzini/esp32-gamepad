#include "LedController.h"
#include <Arduino.h>

void LedController::write(uint8_t pin, bool on) {
    digitalWrite(pin, on ? HIGH : LOW);
}

void LedController::begin(const LedPins& pins) {
    _pins = pins;

    pinMode(_pins.doorOpen, OUTPUT);
    pinMode(_pins.alert, OUTPUT);
    pinMode(_pins.status, OUTPUT);
    pinMode(_pins.spare, OUTPUT);

    write(_pins.doorOpen, false);
    write(_pins.alert, false);
    write(_pins.status, false);
    write(_pins.spare, false);
}

void LedController::setDoorOpen(bool on) { write(_pins.doorOpen, on); }
void LedController::setAlert(bool on)    { write(_pins.alert, on); }
void LedController::setStatus(bool on)   { write(_pins.status, on); }
void LedController::setSpare(bool on)    { write(_pins.spare, on); }

void LedController::blinkAlert(uint32_t periodMs) {
    uint32_t now = millis();
    if (now - _alertBlinkLastMs >= periodMs) {
        _alertBlinkLastMs = now;
        _alertBlinkState = !_alertBlinkState;
        write(_pins.alert, _alertBlinkState);
    }
}