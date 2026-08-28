#include "GamepadOutput.h"
#include "Config.h"

#if defined(ESP32)
// ============================================================================
//  ESP32 — BLE HID (sem fio)
// ============================================================================
#include "BleManager.h"
#include "HidGamepad.h"

static BleManager bleManager;

void GamepadOutput::begin() {
    bleManager.begin();
}

void GamepadOutput::update(uint16_t buttonMask, uint8_t axisX, uint8_t axisY, uint8_t axisZ, uint8_t axisRz) {
    GamepadReport report;
    report.buttons = buttonMask;
    report.axisX  = axisX;
    report.axisY  = axisY;
    report.axisZ  = axisZ;
    report.axisRz = axisRz;
    bleManager.sendReport(report);
}

bool GamepadOutput::isConnected() const {
    return bleManager.isConnected();
}

#else
// ============================================================================
//  Leonardo / Pro Micro (ATmega32U4) — USB HID nativo (com fio)
// ============================================================================
#include "UsbGamepad.h"

static UsbGamepad usbGamepad;

void GamepadOutput::begin() {
    usbGamepad.begin();
}

void GamepadOutput::update(uint16_t buttonMask, uint8_t axisX, uint8_t axisY, uint8_t axisZ, uint8_t axisRz) {
    usbGamepad.update(buttonMask, axisX, axisY, axisZ, axisRz);
}

bool GamepadOutput::isConnected() const {
    return true; // USB cabeado: sempre ativo enquanto a placa estiver ligada
}

#endif
