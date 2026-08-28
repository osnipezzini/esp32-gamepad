#include "UsbGamepad.h"

// A ArduinoJoystickLibrary exige uma instância global (ela reprograma o
// descriptor HID do 32U4 internamente). BUTTON_COUNT vem do Config.h, então
// mudar a quantidade de botões lá também ajusta o descriptor automaticamente.
Joystick_ Joystick(
    JOYSTICK_DEFAULT_REPORT_ID,
    JOYSTICK_TYPE_GAMEPAD,
    BUTTON_COUNT,   // quantidade de botões
    0,              // sem hat switch (D-pad)
    true,           // eixo X
    true,           // eixo Y
    true,           // eixo Z
    false,          // sem Rx
    false,          // sem Ry
    true,           // eixo Rz
    false,          // sem rudder
    false,          // sem throttle
    false,          // sem accelerator
    false,          // sem brake
    false           // sem steering
);

void UsbGamepad::begin() {
    _joystick = &Joystick;
    _joystick->begin(false); // false = só envia report quando chamarmos sendState()

    _joystick->setXAxisRange(0, 255);
    _joystick->setYAxisRange(0, 255);
    _joystick->setZAxisRange(0, 255);
    _joystick->setRzAxisRange(0, 255);
}

void UsbGamepad::update(const GamepadState& state) {
    for (uint8_t i = 0; i < BUTTON_COUNT; i++) {
        _joystick->setButton(i, (state.buttons >> i) & 0x01);
    }

    _joystick->setXAxis(state.axis.x);
    _joystick->setYAxis(state.axis.y);
    _joystick->setZAxis(state.axis.z);
    _joystick->setRzAxis(state.axis.rz);

    _joystick->sendState();
}