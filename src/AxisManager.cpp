#include "AxisManager.h"

void AxisManager::begin() {
    for (uint8_t i = 0; i < AXIS_COUNT; i++) {
        _axes[i].begin(AXIS_PINS[i], AXIS_INVERT[i]);
    }
}

void AxisManager::update() {
    for (uint8_t i = 0; i < AXIS_COUNT; i++) {
        _axes[i].update();
    }
}

void AxisManager::fillReportBytes(uint8_t* out) const {
    for (uint8_t i = 0; i < AXIS_COUNT; i++) {
        out[i] = _axes[i].getValue();
    }
}
