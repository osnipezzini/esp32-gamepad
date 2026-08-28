#pragma once
#include "GamepadState.h"

// Interface única de saída do gamepad, usada pelo main.cpp sem ele saber
// qual placa está rodando por baixo.
//
// A implementação NÃO fica aqui nem misturada com #if defined — cada
// plataforma tem seu próprio arquivo .cpp, e o platformio.ini garante que
// só o arquivo certo entra no build de cada ambiente:
//   src/GamepadOutputEsp32.cpp -> BLE HID (compilado só em [env:esp32dev])
//   src/GamepadOutputAvr.cpp   -> USB HID nativo (compilado em [env:leonardo] e [env:promicro16])
class GamepadOutput {
public:
    void begin();
    void update(const GamepadState& state);

    // ESP32: true enquanto houver uma conexão BLE ativa.
    // Leonardo/Pro Micro: sempre true (USB cabeado = sempre "conectado" enquanto ligado).
    bool isConnected() const;
};