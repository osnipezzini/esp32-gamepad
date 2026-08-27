#pragma once
#include "IButton.h"

// Comportamento padrão de um botão: o report reflete exatamente o
// estado físico (pressionado = 1, solto = 0), sem transformação nenhuma.
class NormalButton : public IButton {
public:
    void update(bool physicalState, uint32_t nowMs) override;
    bool getOutputState() const override;

private:
    bool _state = false;
};
