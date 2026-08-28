#pragma once
#include <stdint.h>
#include "BoardConfig.h"

// Controla LEDs que NÃO fazem parte do report HID do gamepad — são
// indicadores próprios seus (porta aberta, alerta, status de conexão, etc.).
// Cada papel tem seu próprio setter nomeado (setDoorOpen, setAlert, ...) em
// vez de um índice genérico — evita acender o LED errado por engano de índice.
class LedController {
public:
    void begin(const LedPins& pins);

    void setDoorOpen(bool on);
    void setAlert(bool on);
    void setStatus(bool on);
    void setSpare(bool on);

    // Pisca o LED de alerta no período informado. Chame a cada loop.
    void blinkAlert(uint32_t periodMs);

private:
    LedPins _pins{};

    uint32_t _alertBlinkLastMs = 0;
    bool _alertBlinkState = false;

    static void write(uint8_t pin, bool on);
};