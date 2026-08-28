#pragma once
#include <stdint.h>

// Cada botão físico pode se comportar de um jeito diferente no report HID,
// independente do que fisicamente está acontecendo no pino.
enum class ButtonMode : uint8_t {
    NORMAL = 0,   // saída = estado físico, 1:1 (comportamento padrão de um botão)
    ONE_SHOT,     // qualquer pressionar gera um único pulso curto, ignora quanto tempo fica segurado
    TOGGLE,       // primeira borda de subida liga, a próxima desliga (como um interruptor)
    LONG_PRESS    // só fica "pressionado" no report se for segurado além de um tempo mínimo
};
