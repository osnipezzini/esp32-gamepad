#pragma once
#include "BoardConfig.h"
#include "GamepadState.h"
#include "Axis.h"

// Dono dos 4 eixos nomeados (X, Y, Z, Rz). Recebe a configuração da placa
// como objetos tipados e devolve os valores lidos como um AxisValues —
// nunca como array solto por índice.
class AxisManager {
public:
    void begin(const AxisPins& pins, const AxisInvertFlags& invert, const AxisReadSettings& settings);
    void update();
    AxisValues getValues() const;

private:
    Axis _x;
    Axis _y;
    Axis _z;
    Axis _rz;
};