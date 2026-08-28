#pragma once
#include <stdint.h>

// Layout exato dos bytes enviados no report HID (sem o Report ID, que vai
// como prefixo separado pela camada BLE HID). Total: 6 bytes.
#pragma pack(push, 1)
struct GamepadReport {
    uint16_t buttons;  // bit N = botão N pressionado (usa os BUTTON_COUNT bits menos significativos)
    uint8_t axisX;
    uint8_t axisY;
    uint8_t axisZ;
    uint8_t axisRz;
};
#pragma pack(pop)

// Report Descriptor HID de um gamepad GENÉRICO:
//   16 botões (bitmap) + 4 eixos analógicos (X, Y, Z, Rz), 8 bits cada.
// Isto NÃO é XInput nem nenhum perfil de vendor (Xbox/PlayStation/etc) —
// é reconhecido pelo Windows e Linux como um HID gamepad comum (DirectInput
// no Windows, hid-generic + joydev no Linux), sem precisar de driver algum.
extern const uint8_t GAMEPAD_REPORT_DESCRIPTOR[];
extern const uint16_t GAMEPAD_REPORT_DESCRIPTOR_SIZE;
