#include "ToggleButton.h"

void ToggleButton::update(bool physicalState, uint32_t /*nowMs*/) {
    if (physicalState && !_lastPhysical) {
        _state = !_state; // flip apenas na borda de subida, nunca ao soltar
    }
    _lastPhysical = physicalState;
}

bool ToggleButton::getOutputState() const {
    return _state;
}
