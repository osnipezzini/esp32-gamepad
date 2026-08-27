#include "OneShotButton.h"
#include "Config.h"

void OneShotButton::update(bool physicalState, uint32_t nowMs) {
    // Detecta a borda de subida: estava solto, agora está pressionado.
    if (physicalState && !_lastPhysical) {
        _pulseActive = true;
        _pulseStart = nowMs;
    }
    _lastPhysical = physicalState;

    // O pulso se encerra sozinho após ONE_SHOT_PULSE_MS, independente de o
    // usuário ainda estar segurando o botão físico ou não.
    if (_pulseActive && (nowMs - _pulseStart >= ONE_SHOT_PULSE_MS)) {
        _pulseActive = false;
    }
}

bool OneShotButton::getOutputState() const {
    return _pulseActive;
}
