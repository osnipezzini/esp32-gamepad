#include "GamepadOutput.h"
#include "BleManager.h"
#include "HidGamepad.h"

// ============================================================================
//  Implementação do GamepadOutput para ESP32 — BLE HID (sem fio).
//  Este arquivo só é compilado no ambiente [env:esp32dev] do platformio.ini.
// ============================================================================

static BleManager bleManager;

void GamepadOutput::begin() {
    bleManager.begin();
}

void GamepadOutput::update(const GamepadState& state) {
    // GamepadState é a representação lógica (objetos tipados); GamepadReport
    // é o formato de bytes exato que vai no ar via BLE (struct empacotada).
    GamepadReport report;
    report.buttons = state.buttons;
    report.axisX   = state.axis.x;
    report.axisY   = state.axis.y;
    report.axisZ   = state.axis.z;
    report.axisRz  = state.axis.rz;

    bleManager.sendReport(report);
}

bool GamepadOutput::isConnected() const {
    return bleManager.isConnected();
}
