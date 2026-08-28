#include "GamepadOutput.h"
#include "UsbGamepad.h"

// ============================================================================
//  Implementação do GamepadOutput para ATmega32U4 (Leonardo / Pro Micro) —
//  USB HID nativo (com fio).
//  Este arquivo só é compilado nos ambientes [env:leonardo] e
//  [env:promicro16] do platformio.ini.
// ============================================================================

static UsbGamepad usbGamepad;

void GamepadOutput::begin() {
    usbGamepad.begin();
}

void GamepadOutput::update(const GamepadState& state) {
    usbGamepad.update(state);
}

bool GamepadOutput::isConnected() const {
    return true; // USB cabeado: sempre ativo enquanto a placa estiver ligada
}
