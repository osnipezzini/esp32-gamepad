#pragma once
#include "IButton.h"

// O output só vai para "pressionado" se o botão físico continuar segurado
// por pelo menos LONG_PRESS_THRESHOLD_MS. Pressionar rápido não gera nada,
// o que evita acionamentos acidentais em ações críticas/irreversíveis.
class LongPressButton : public IButton {
public:
    void update(bool physicalState, uint32_t nowMs) override;
    bool getOutputState() const override;

private:
    bool _wasPhysical = false;
    uint32_t _pressStart = 0;
    bool _active = false;
};
