#pragma once
#include <Arduino.h>
#include "Config.h"
#include "IButton.h"

// Dono de todos os objetos IButton*, um por botão físico da matriz.
// Responsável por: escanear a matriz, aplicar debounce, alimentar cada
// IButton com o estado já limpo, e montar o bitmask final para o report HID.
class ButtonManager {
public:
    void begin();
    void update();
    uint16_t getButtonBitmask() const;

    ~ButtonManager();

private:
    IButton* _buttons[BUTTON_COUNT] = {nullptr};

    bool _rawState[BUTTON_COUNT]        = {false};
    bool _lastRawState[BUTTON_COUNT]    = {false};
    bool _debouncedState[BUTTON_COUNT]  = {false};
    uint32_t _lastRawChangeMs[BUTTON_COUNT] = {0};

    void scanMatrix();
    static IButton* createButton(ButtonMode mode);
};
