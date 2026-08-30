#pragma once
#include "IButton.h"

// Gera um pulso curto ao PRESSIONAR e outro ao SOLTAR.
// - Borda de subida (press)  -> pulso de TWO_SHOT_PULSE_MS
// - Borda de descida (release)-> pulso de TWO_SHOT_PULSE_MS
// Se a soltura acontecer ainda durante o primeiro pulso, o segundo
// pulso é enfileirado e disparado após um GAP curto, garantindo que
// o host veja dois eventos distintos (1 -> 0 -> 1).
// Útil para simular "toque duplo" físico (ex: troca de marcha, pisca).
class TwoShotButton : public IButton {
public:
    void update(bool physicalState, uint32_t nowMs) override;
    bool getOutputState() const override;

private:
    enum class Phase : uint8_t { IDLE, PULSE, GAP };

    bool _lastPhysical = false;
    Phase _phase = Phase::IDLE;
    uint32_t _phaseStart = 0;
    bool _queued = false; // há um segundo pulso pendente?
};
