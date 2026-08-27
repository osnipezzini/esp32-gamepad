#pragma once
#include "IButton.h"

// A primeira borda de subida liga o output, a próxima desliga, e assim por
// diante — como um interruptor. Soltar o botão físico não muda nada.
class ToggleButton : public IButton {
public:
    void update(bool physicalState, uint32_t nowMs) override;
    bool getOutputState() const override;

private:
    bool _lastPhysical = false;
    bool _state = false;
};
