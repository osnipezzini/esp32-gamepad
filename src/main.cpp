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

static void printCurrentModes() {
    for (uint8_t i = 0; i < BUTTON_COUNT; i++) {
        Serial.print(i);
        Serial.print("=");
        Serial.println(buttonModeToString(getButtonMode(i)));
    }
}

static void printButtonStates() {
    uint16_t bitmask = buttonManager.getButtonBitmask();
    for (uint8_t i = 0; i < BUTTON_COUNT; i++) {
        Serial.print(i);
        Serial.print(":");
        Serial.println((bitmask & (1u << i)) ? "1" : "0");
    }
}

static void handleSerialConfig() {
    if (Serial.available() <= 0) {
        return;
    }

    static String inputBuffer;
    while (Serial.available() > 0) {
        char ch = Serial.read();
        if (ch == '\n' || ch == '\r') {
            if (inputBuffer.length() == 0) {
                continue;
            }

            String cmd = inputBuffer;
            inputBuffer = "";
            cmd.trim();

            if (cmd == "GET") {
                printCurrentModes();
                continue;
            }

            if (cmd == "GETSTATES") {
                printButtonStates();
                continue;
            }

            if (cmd == "RESET") {
                resetButtonModesToDefault();
                saveButtonModesToStorage();
                printCurrentModes();
                continue;
            }

            int idx = cmd.indexOf('=');
            if (idx > 0) {
                String key = cmd.substring(0, idx);
                String value = cmd.substring(idx + 1);
                int buttonIndex = key.toInt();
                if (buttonIndex >= 0 && buttonIndex < BUTTON_COUNT) {
                    setButtonMode(buttonIndex, parseButtonModeString(value.c_str()));
                    saveButtonModesToStorage();
                    Serial.print("OK");
                    Serial.print(buttonIndex);
                    Serial.print("=");
                    Serial.println(buttonModeToString(getButtonMode(buttonIndex)));
                }
            }
            continue;
        }
        inputBuffer += ch;
    }
}

void setup() {
    Serial.begin(115200); // útil para debug; inofensivo se você não abrir o monitor
    loadButtonModesFromStorage();

    const BoardConfig& board = getBoardConfig();

    ledController.begin(board.leds);
    buttonManager.begin(board.matrix);
    axisManager.begin(board.axis, board.axisInvert, board.axisRead);
    gamepadOutput.begin();

    Serial.println("ButtonBox ready");
    Serial.println("Use <index>=<NORMAL|ONE_SHOT|TOGGLE|LONG_PRESS>");
}

void loop() {
    handleSerialConfig();

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