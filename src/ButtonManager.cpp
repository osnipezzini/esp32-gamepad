#include "ButtonManager.h"
#include "NormalButton.h"
#include "OneShotButton.h"
#include "ToggleButton.h"
#include "LongPressButton.h"
#include "TwoShotButton.h"

IButton* ButtonManager::createButton(ButtonMode mode) {
    switch (mode) {
        case ButtonMode::ONE_SHOT:   return new OneShotButton();
        case ButtonMode::TWO_SHOT:   return new TwoShotButton();
        case ButtonMode::TOGGLE:     return new ToggleButton();
        case ButtonMode::LONG_PRESS: return new LongPressButton();
        case ButtonMode::NORMAL:
        default:                     return new NormalButton();
    }
}

void ButtonManager::begin(const MatrixPins& pins) {
    _pins = pins;

    for (uint8_t r = 0; r < MATRIX_ROWS; r++) {
        pinMode(_pins.rows[r], OUTPUT);
        digitalWrite(_pins.rows[r], HIGH); // linhas ficam em repouso HIGH
    }
    for (uint8_t c = 0; c < MATRIX_COLS; c++) {
        pinMode(_pins.cols[c], INPUT_PULLUP);
    }
    for (uint8_t i = 0; i < BUTTON_COUNT; i++) {
        _buttons[i] = createButton(BUTTON_MODES[i]);
    }
}

void ButtonManager::scanMatrix() {
    for (uint8_t r = 0; r < MATRIX_ROWS; r++) {
        digitalWrite(_pins.rows[r], LOW);  // ativa só esta linha
        delayMicroseconds(20);             // tempo para a linha estabilizar

        for (uint8_t c = 0; c < MATRIX_COLS; c++) {
            uint8_t index = r * MATRIX_COLS + c;
            // Pull-up interno: nível LOW na coluna = botão fechado (pressionado).
            _rawState[index] = (digitalRead(_pins.cols[c]) == LOW);
        }

        digitalWrite(_pins.rows[r], HIGH); // desativa a linha antes de ir para a próxima
    }
}

void ButtonManager::update() {
    scanMatrix();
    uint32_t now = millis();

    for (uint8_t i = 0; i < BUTTON_COUNT; i++) {
        // Debounce simples: só aceita o novo valor depois que ele ficou
        // estável por DEBOUNCE_MS sem oscilar.
        if (_rawState[i] != _lastRawState[i]) {
            _lastRawChangeMs[i] = now;
            _lastRawState[i] = _rawState[i];
        }
        if ((now - _lastRawChangeMs[i]) >= DEBOUNCE_MS) {
            _debouncedState[i] = _rawState[i];
        }

        _buttons[i]->update(_debouncedState[i], now);
    }
}

uint16_t ButtonManager::getButtonBitmask() const {
    uint16_t mask = 0;
    for (uint8_t i = 0; i < BUTTON_COUNT; i++) {
        if (_buttons[i]->getOutputState()) {
            mask |= (1u << i);
        }
    }
    return mask;
}

ButtonManager::~ButtonManager() {
    for (uint8_t i = 0; i < BUTTON_COUNT; i++) {
        delete _buttons[i];
    }
}