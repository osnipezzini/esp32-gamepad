#pragma once
#include "Config.h"
#include "Axis.h"

class AxisManager {
public:
    void begin();
    void update();

    // Escreve AXIS_COUNT bytes (0-255 cada) prontos para o report HID.
    void fillReportBytes(uint8_t* out) const;

private:
    Axis _axes[AXIS_COUNT];
};
