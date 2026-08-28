#pragma once
#include <stdint.h>

// ============================================================================
//  Representação do ESTADO LÓGICO do gamepad — o que foi lido dos botões e
//  eixos, já pronto para ser enviado por qualquer transporte (BLE ou USB).
//
//  Em vez de passar 4 bytes de eixo soltos por parâmetro (onde trocar a
//  ordem X/Y/Z/Rz na chamada não gera nenhum erro de compilação), agrupamos
//  em objetos com campos nomeados.
// ============================================================================

struct AxisValues {
    uint8_t x  = 0;
    uint8_t y  = 0;
    uint8_t z  = 0;
    uint8_t rz = 0;
};

struct GamepadState {
    uint16_t buttons = 0;   // bit N = botão N pressionado
    AxisValues axis;
};
