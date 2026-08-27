#pragma once
#include "IButton.h"

// Ao detectar a borda de subida (botão acabou de ser pressionado), gera um
// único pulso de duração fixa (ONE_SHOT_PULSE_MS) e depois solta sozinho,
// não importa por quanto tempo o usuário continue segurando o botão físico.
//
// Útil para ações de "toque único" em jogos que não devem repetir/segurar:
// troca de câmera, ignição, flash de farol, etc.
class OneShotButton : public IButton {
public:
    void update(bool physicalState, uint32_t nowMs) override;
    bool getOutputState() const override;

private:
    bool _lastPhysical = false;
    bool _pulseActive = false;
    uint32_t _pulseStart = 0;
};
