#include <Arduino.h>
#include "Config.h"
#include "BoardConfig.h"
#include "GamepadState.h"
#include "ButtonManager.h"
#include "AxisManager.h"
#include "LedController.h"
#include "GamepadOutput.h"

static ButtonManager buttonManager;
static AxisManager axisManager;
static LedController ledController;
static GamepadOutput gamepadOutput;

static uint32_t lastScanMs = 0;

void setup() {
    Serial.begin(115200); // útil para debug; inofensivo se você não abrir o monitor

    const BoardConfig& board = getBoardConfig();

    ledController.begin(board.leds);
    buttonManager.begin(board.matrix);
    axisManager.begin(board.axis, board.axisInvert, board.axisRead);
    gamepadOutput.begin();
}

void loop() {
    uint32_t now = millis();
    if (now - lastScanMs < SCAN_INTERVAL_MS) {
        return;
    }
    lastScanMs = now;

    buttonManager.update();
    axisManager.update();

    // --------------------------------------------------------------------
    // Exemplo de LED de status TOTALMENTE independente do report HID.
    // Para os seus casos reais (porta aberta, alerta, etc.), leia o
    // sinal/sensor relevante e chame o setter nomeado correspondente
    // (setDoorOpen, setAlert, ...) — nada disso passa pelo protocolo do
    // gamepad.
    // --------------------------------------------------------------------
    ledController.setStatus(gamepadOutput.isConnected());

    GamepadState state;
    state.buttons = buttonManager.getButtonBitmask();
    state.axis    = axisManager.getValues();

    gamepadOutput.update(state);
}