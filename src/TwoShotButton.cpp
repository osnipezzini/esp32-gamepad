#include "TwoShotButton.h"
#include "Config.h"

void TwoShotButton::update(bool physicalState, uint32_t nowMs) {
    const bool rising  = physicalState && !_lastPhysical;
    const bool falling = !physicalState && _lastPhysical;
    const bool edge    = rising || falling;
    _lastPhysical = physicalState;

    if (edge) {
        if (_phase == Phase::IDLE) {
            _phase = Phase::PULSE;
            _phaseStart = nowMs;
        } else {
            // Já está em PULSE ou GAP — enfileira o próximo pulso.
            // Apenas 1 pendente é suficiente (press+release = 2 pulsos no total).
            _queued = true;
        }
    }

    if (_phase == Phase::PULSE && (nowMs - _phaseStart >= TWO_SHOT_PULSE_MS)) {
        if (_queued) {
            _phase = Phase::GAP;
            _phaseStart = nowMs;
            // mantém _queued = true até o GAP terminar e o próximo PULSE começar
        } else {
            _phase = Phase::IDLE;
        }
    } else if (_phase == Phase::GAP && (nowMs - _phaseStart >= TWO_SHOT_GAP_MS)) {
        _phase = Phase::PULSE;
        _phaseStart = nowMs;
        _queued = false;
    }
}

bool TwoShotButton::getOutputState() const {
    return _phase == Phase::PULSE;
}
