#pragma once
#include <Joystick.h>
#include "Config.h"
#include "GamepadState.h"

// Usa o USB nativo do ATmega32U4 para se apresentar como um gamepad HID
// genérico — mesma filosofia da versão BLE: sem perfil de vendor (XInput,
// Xbox, etc), reconhecido nativamente como joystick tanto no Windows
// (DirectInput) quanto no Linux (hid-generic + joydev), sem driver.
class UsbGamepad {
public:
    void begin();
    void update(const GamepadState& state);

private:
    Joystick_* _joystick = nullptr;
};