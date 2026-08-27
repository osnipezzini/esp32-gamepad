#include "BleManager.h"
#include "Config.h"

// BLE GAP Appearance: "Gamepad" (categoria HID > Gamepad).
static const uint16_t BLE_APPEARANCE_GAMEPAD = 0x03C4;

void BleManager::ServerCallbacks::onConnect(NimBLEServer* /*pServer*/) {
    _owner->_connected = true;
}

void BleManager::ServerCallbacks::onDisconnect(NimBLEServer* /*pServer*/) {
    _owner->_connected = false;
    NimBLEDevice::startAdvertising(); // volta a anunciar para reconectar sozinho
}

void BleManager::begin() {
    NimBLEDevice::init(BLE_DEVICE_NAME);

    // Sem bonding/MITM obrigatório: só "just works", suficiente para um
    // periférico HID que não digita senhas nem lida com dado sensível.
    NimBLEDevice::setSecurityAuth(false, false, true);

    NimBLEServer* server = NimBLEDevice::createServer();
    server->setCallbacks(new ServerCallbacks(this));

    _hid = new NimBLEHIDDevice(server);
    _input = _hid->inputReport(1); // Report ID 1 - precisa bater com o descriptor

    _hid->manufacturer()->setValue(BLE_MANUFACTURER);
    _hid->pnp(0x02, 0xE502, 0xA111, 0x0210); // VID/PID arbitrários, não ligados a nenhum produto real
    _hid->hidInfo(0x00, 0x01);

    _hid->reportMap((uint8_t*)GAMEPAD_REPORT_DESCRIPTOR, GAMEPAD_REPORT_DESCRIPTOR_SIZE);
    _hid->startServices();
    _hid->setBatteryLevel(100); // opcional - remova se não tiver leitura de bateria real

    NimBLEAdvertising* advertising = server->getAdvertising();
    advertising->setAppearance(BLE_APPEARANCE_GAMEPAD);
    advertising->addServiceUUID(_hid->hidService()->getUUID());
    advertising->start();
}

void BleManager::sendReport(const GamepadReport& report) {
    if (!_connected || _input == nullptr) return;
    _input->setValue((uint8_t*)&report, sizeof(report));
    _input->notify();
}

bool BleManager::isConnected() const {
    return _connected;
}
