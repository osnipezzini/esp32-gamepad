#pragma once
#include <NimBLEDevice.h>
#include "HidGamepad.h"

// Sobe o dispositivo BLE como um HID over GATT genérico (serviço 0x1812)
// usando o report descriptor definido em HidGamepad.h. Não usa nenhum
// perfil de vendor — é um gamepad HID puro.
class BleManager {
public:
    void begin();
    void sendReport(const GamepadReport& report);
    bool isConnected() const;

private:
    NimBLEHIDDevice* _hid = nullptr;
    NimBLECharacteristic* _input = nullptr;
    bool _connected = false;

    class ServerCallbacks : public NimBLEServerCallbacks {
    public:
        explicit ServerCallbacks(BleManager* owner) : _owner(owner) {}
        void onConnect(NimBLEServer* pServer) override;
        void onDisconnect(NimBLEServer* pServer) override;

    private:
        BleManager* _owner;
    };
};
