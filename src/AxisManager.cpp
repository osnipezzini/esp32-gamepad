#include "AxisManager.h"

void AxisManager::begin(const AxisPins& pins, const AxisInvertFlags& invert, const AxisReadSettings& settings) {
    _x.begin(pins.x, invert.x, settings);
    _y.begin(pins.y, invert.y, settings);
    _z.begin(pins.z, invert.z, settings);
    _rz.begin(pins.rz, invert.rz, settings);
}

void AxisManager::update() {
    _x.update();
    _y.update();
    _z.update();
    _rz.update();
}

AxisValues AxisManager::getValues() const {
    AxisValues values;
    values.x  = _x.getValue();
    values.y  = _y.getValue();
    values.z  = _z.getValue();
    values.rz = _rz.getValue();
    return values;
}