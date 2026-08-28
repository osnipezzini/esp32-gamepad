#pragma once
#include <stdint.h>

// Interface comum a todos os comportamentos de botão.
// O ButtonManager não sabe (nem precisa saber) qual implementação concreta
// está por trás de cada IButton* — só chama update()/getOutputState().
class IButton {
public:
    virtual ~IButton() = default;

    // Chamado a cada ciclo de scan com o estado físico JÁ tratado por debounce.
    virtual void update(bool physicalState, uint32_t nowMs) = 0;

    // O que deve ser reportado no HID (pode ser diferente do estado físico).
    virtual bool getOutputState() const = 0;
};
