#include <Arduino.h>
#include "Config.h"
#include "ButtonManager.h"
#include "AxisManager.h"
#include "LedController.h"
#include "BleManager.h"
#include "HidGamepad.h"

static ButtonManager buttonManager;
static AxisManager axisManager;
static LedController ledController;
static BleManager bleManager;

static uint32_t lastScanMs = 0;

void setup() {
    Serial.begin(115200);

    ledController.begin();
    buttonManager.begin();
    axisManager.begin();
    bleManager.begin();

    Serial.println("ButtonBox pronto. Aguardando conexao BLE...");
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
    // Exemplo de LED de status TOTALMENTE independente do report HID:
    // aceso enquanto houver uma conexão BLE ativa.
    //
    // Para os seus casos (porta aberta, alerta, etc.), a ideia é a mesma:
    // leia o sinal/sensor relevante (pode até ser um botão da matriz que
    // você decida tratar como sensor, não como botão de jogo) e chame
    // ledController.set(...)/blink(...) com a condição que quiser aqui.
    // --------------------------------------------------------------------
    ledController.set(LED_BLE_STATUS, bleManager.isConnected());

    GamepadReport report;
    report.buttons = buttonManager.getButtonBitmask();

    uint8_t axisBytes[AXIS_COUNT];
    axisManager.fillReportBytes(axisBytes);
    report.axisX  = axisBytes[0];
    report.axisY  = axisBytes[1];
    report.axisZ  = axisBytes[2];
    report.axisRz = axisBytes[3];

    bleManager.sendReport(report);
}
