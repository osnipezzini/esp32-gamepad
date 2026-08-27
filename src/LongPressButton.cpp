#include "LongPressButton.h"
#include "Config.h"

void LongPressButton::update(bool physicalState, uint32_t nowMs) {
    if (physicalState && !_wasPhysical) {
        _pressStart = nowMs; // começou a segurar agora, inicia a contagem
    }
    _wasPhysical = physicalState;

    if (physicalState) {
        _active = (nowMs - _pressStart) >= LONG_PRESS_THRESHOLD_MS;
    } else {
        _active = false; // soltou antes do tempo (ou depois) -> sempre desativa
    }
}

bool LongPressButton::getOutputState() const {
    return _active;
}
