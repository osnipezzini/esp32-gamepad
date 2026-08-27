# ButtonBox — ESP32-WROOM-32 + BLE HID genérico

Button box sem fio (Bluetooth Low Energy) para Windows e Linux, reconhecida
como um **gamepad HID genérico** — não é XInput, não emula Xbox/PlayStation
nem nenhum outro perfil de vendor. Aparece no sistema como um joystick HID
comum (DirectInput no Windows, `hid-generic` + `joydev` no Linux), sem driver
nenhum.

## Hardware

- **Placa:** ESP32-WROOM-32 (DevKit V1, 38 pinos)
- **Ambiente:** PlatformIO
- **Stack BLE:** [NimBLE-Arduino](https://github.com/h2zero/NimBLE-Arduino) `1.4.x`
  (a API de callbacks do servidor BLE muda entre a série 1.x e 2.x da lib —
  o código deste projeto foi escrito contra a 1.4.x, fixada no `platformio.ini`)

## Pinagem necessária

### Matriz de botões (4×4 = 16 botões, 8 GPIOs)

| Sinal | GPIO | Observação |
|---|---|---|
| Linha 0 (ROW) | GPIO13 | saída, driven LOW durante o scan |
| Linha 1 (ROW) | GPIO12 | ⚠️ pino de strapping (MTDI) — ver nota abaixo |
| Linha 2 (ROW) | GPIO14 | saída |
| Linha 3 (ROW) | GPIO27 | saída |
| Coluna 0 (COL) | GPIO26 | entrada, pull-up interno |
| Coluna 1 (COL) | GPIO25 | entrada, pull-up interno |
| Coluna 2 (COL) | GPIO33 | entrada, pull-up interno |
| Coluna 3 (COL) | GPIO32 | entrada, pull-up interno |

Cada botão físico liga a interseção de uma linha com uma coluna (mais o
diodo, se quiser suportar múltiplas teclas pressionadas ao mesmo tempo sem
"ghosting" — recomendado para 16 botões).

Índice do botão no software = `linha × 4 + coluna` (0 a 15), usado em
`BUTTON_MODES[]` no `Config.h`.

### Eixos analógicos (4 eixos, ADC1 apenas)

| Eixo | GPIO | Observação |
|---|---|---|
| X | GPIO34 | entrada analógica, **input only** |
| Y | GPIO35 | entrada analógica, **input only** |
| Z | GPIO36 | entrada analógica, **input only** (também chamado VP) |
| Rz | GPIO39 | entrada analógica, **input only** (também chamado VN) |

Importante: use **apenas pinos ADC1** (GPIO32-39). O ADC2 (GPIO0, 2, 4,
12-15, 25-27) tem conflito conhecido com o rádio Wi-Fi/BLE do ESP32 e não
deve ser usado para leitura analógica enquanto o BLE está ativo.

### LEDs de status (4 LEDs, independentes do report HID)

| LED | GPIO | Uso sugerido |
|---|---|---|
| `LED_DOOR_OPEN` | GPIO2 | ⚠️ pino de strapping — também é o LED onboard da maioria dos DevKits |
| `LED_ALERT` | GPIO4 | livre |
| `LED_BLE_STATUS` | GPIO5 | já usado no `main.cpp` como indicador de conexão BLE |
| `LED_SPARE` | GPIO18 | livre para uso futuro |

Esses LEDs **não fazem parte do protocolo HID** — são pinos digitais comuns,
controlados 100% pela sua lógica em `main.cpp`/`LedController`. Use para
qualquer indicador que não seja um botão de jogo: porta aberta, alerta,
status de algum sistema externo, etc.

## ⚠️ Notas sobre pinos de strapping

GPIO0, GPIO2, GPIO5, GPIO12 e GPIO15 são pinos de *strapping* — o ESP32 lê o
nível deles no momento do boot para decidir o modo de inicialização. Usá-los
como GPIO comum depois do boot (como fazemos aqui) é uma prática comum e
segura, mas:

- Evite conectar resistores de pull-down fortes ou fontes externas que
  possam forçar nível LOW nesses pinos durante o power-on/reset.
- Um LED com resistor em série (uso normal) não costuma causar problema.
- Se notar boot instável, mova a linha/LED correspondente para outro GPIO
  livre (por exemplo GPIO15, GPIO19 ou GPIO23).

## Resumo de todos os GPIOs usados

```
Linhas (ROW):     13, 12, 14, 27
Colunas (COL):    26, 25, 33, 32
Eixos (ADC1):     34, 35, 36, 39
LEDs:             2, 4, 5, 18
```

Total: 20 GPIOs, todos dentro da faixa livre de uso geral do ESP32-WROOM-32
DevKit (evita GPIO6-11, reservados para a flash SPI interna, e GPIO1/GPIO3,
usados pela UART0/Serial de programação).

## Arquitetura do código

```
buttonbox/
├── platformio.ini
├── include/
│   ├── Config.h            <- único arquivo que você deveria precisar editar
│   ├── InputTypes.h        <- enum ButtonMode
│   ├── IButton.h           <- interface abstrata de comportamento de botão
│   ├── NormalButton.h
│   ├── OneShotButton.h     <- segurar = apenas 1 pulso (press+release)
│   ├── ToggleButton.h
│   ├── LongPressButton.h
│   ├── ButtonManager.h     <- scan da matriz + debounce + delega comportamento
│   ├── Axis.h              <- leitura/filtro de 1 eixo analógico
│   ├── AxisManager.h
│   ├── LedController.h     <- LEDs de status, independentes do HID
│   ├── HidGamepad.h        <- report descriptor HID (gamepad genérico)
│   └── BleManager.h        <- serviço BLE HID over GATT (NimBLE)
└── src/
    ├── main.cpp            <- só orquestra: setup() e loop()
    └── *.cpp               <- implementação de cada header acima
```

### Como adicionar/mudar o comportamento de um botão

Edite apenas `BUTTON_MODES[]` em `include/Config.h`:

```cpp
static const ButtonMode BUTTON_MODES[BUTTON_COUNT] = {
    /* 0*/ ButtonMode::NORMAL,
    /* 2*/ ButtonMode::ONE_SHOT,   // <- é isso que você pediu: hold vira pulso único
    ...
};
```

Nenhuma outra parte do código precisa ser tocada — o `ButtonManager`
instancia a classe certa (`OneShotButton`, `ToggleButton`, etc.) sozinho.

### Comportamentos disponíveis

| Modo | O que faz |
|---|---|
| `NORMAL` | Report = estado físico, 1:1 |
| `ONE_SHOT` | Qualquer press vira um pulso único de `ONE_SHOT_PULSE_MS` |
| `TOGGLE` | Cada press alterna entre ligado/desligado |
| `LONG_PRESS` | Só ativa se segurar por `LONG_PRESS_THRESHOLD_MS` |

Para criar um novo comportamento: implemente `IButton`, adicione o caso no
`switch` de `ButtonManager::createButton()`, e use o novo modo no `Config.h`.

## Build

```bash
pio run                 # compila
pio run -t upload       # grava no ESP32
pio device monitor      # monitor serial (115200 baud)
```

## Formato do report HID

6 bytes por report (mais 1 byte de Report ID gerenciado pela camada BLE):

| Campo | Tamanho | Conteúdo |
|---|---|---|
| `buttons` | 2 bytes | bitmap, bit N = botão N pressionado (bits 0-15) |
| `axisX` | 1 byte | 0-255 |
| `axisY` | 1 byte | 0-255 |
| `axisZ` | 1 byte | 0-255 |
| `axisRz` | 1 byte | 0-255 |

## Limitação conhecida

Alguns jogos (principalmente simuladores no Windows) só aceitam **XInput**
nativamente e não reconhecem gamepad HID genérico automaticamente. Nesses
casos específicos, ferramentas como **x360ce** ou **JoyToKey** convertem o
HID genérico em um XInput virtual do lado do PC — não é algo que o firmware
precise resolver.
