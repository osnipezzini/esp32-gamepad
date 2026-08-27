#include "NormalButton.h"

void NormalButton::update(bool physicalState, uint32_t /*nowMs*/) {
    _state = physicalState;
}

bool NormalButton::getOutputState() const {
    return _state;
}
