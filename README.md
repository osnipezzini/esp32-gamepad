# ButtonBox — ESP32 / Arduino Leonardo / SparkFun Pro Micro

Projeto PlatformIO com suporte a **três placas**, escolhidas pelo ambiente
de build. Em todas, o dispositivo é reconhecido como um **gamepad HID
genérico** — sem XInput, sem perfil de vendor.

| Ambiente | Placa | Transporte | Fio? |
|---|---|---|---|
| `esp32dev` | ESP32-WROOM-32 | BLE HID genérico | **Sem fio** |
| `leonardo` | Arduino Leonardo (ATmega32U4) | USB HID nativo | Com fio |
| `promicro16` | SparkFun Pro Micro 5V/16MHz (ATmega32U4) | USB HID nativo | Com fio |

## Build

```bash
pio run -e esp32dev    -t upload   # ESP32 (BLE)
pio run -e leonardo    -t upload   # Arduino Leonardo (USB)
pio run -e promicro16  -t upload   # SparkFun Pro Micro (USB)
pio device monitor                 # monitor serial, 115200 baud
```

## Como a separação por plataforma funciona

**Nenhum `#if defined(ESP32)` misturado dentro da lógica.** Cada coisa
específica de placa (pinagem física e transporte HID) mora no seu próprio
arquivo `.cpp`, e o `platformio.ini` decide, por ambiente, quais entram no
build via `build_src_filter`:

| Arquivo | O que faz | Compilado em |
|---|---|---|
| `BoardConfigEsp32.cpp` | pinagem física do ESP32 | só `esp32dev` |
| `BoardConfigAvr.cpp` | pinagem física do Leonardo/Pro Micro | `leonardo` e `promicro16` |
| `GamepadOutputEsp32.cpp` | transporte BLE HID | só `esp32dev` |
| `GamepadOutputAvr.cpp` | transporte USB HID nativo | `leonardo` e `promicro16` |
| `BleManager.cpp` / `HidGamepad.cpp` | serviço BLE + report descriptor | só `esp32dev` |
| `UsbGamepad.cpp` | wrapper da ArduinoJoystickLibrary | `leonardo` e `promicro16` |

Todo o resto (`ButtonManager`, `Axis*`, `LedController`, os 4 comportamentos
de botão, `main.cpp`) é **um único arquivo, sem duplicação**, porque nunca
dependeu de nada específico de placa.

## Dados trafegam como objetos tipados, não como arrays/parâmetros soltos

| Antes (v1) | Agora (v2) |
|---|---|
| `uint8_t ROW_PINS[4]`, `uint8_t COL_PINS[4]` soltos | `struct MatrixPins { rows; cols; }` |
| `uint8_t AXIS_PINS[4]` indexado por posição | `struct AxisPins { x, y, z, rz; }` |
| `uint16_t ADC_MAX_VALUE` / `AXIS_DEADZONE` globais | `struct AxisReadSettings { adcMaxValue; deadzone; smoothing; }` |
| `uint8_t LED_PINS[4]` + `enum LedIndex` genérico | `struct LedPins { doorOpen; alert; status; spare; }` + setters nomeados |
| `update(uint16_t buttons, uint8_t x, uint8_t y, uint8_t z, uint8_t rz)` | `update(const GamepadState& state)` |

Isso significa, por exemplo, que **não tem mais como trocar X com Y por
engano** ao chamar uma função — o compilador exige o campo certo pelo nome,
não pela posição no array. E setar um LED errado (`setDoorOpen` vs
`setAlert`) também não é mais possível por engano de índice numérico.

### Os objetos centrais

```cpp
// BoardConfig.h - descreve o HARDWARE FÍSICO da placa
struct MatrixPins     { std::array<uint8_t,4> rows, cols; };
struct AxisPins        { uint8_t x, y, z, rz; };
struct AxisInvertFlags { bool x, y, z, rz; };
struct AxisReadSettings{ uint16_t adcMaxValue, deadzone; float smoothing; };
struct LedPins         { uint8_t doorOpen, alert, status, spare; };
struct BoardConfig     { MatrixPins matrix; AxisPins axis; AxisInvertFlags axisInvert;
                          AxisReadSettings axisRead; LedPins leds; };

// GamepadState.h - descreve o ESTADO LÓGICO do gamepad, pronto para
// qualquer transporte (BLE ou USB) consumir
struct AxisValues   { uint8_t x, y, z, rz; };
struct GamepadState { uint16_t buttons; AxisValues axis; };
```

`const BoardConfig& getBoardConfig()` é a única função que muda de
implementação por placa (`BoardConfigEsp32.cpp` vs `BoardConfigAvr.cpp`) —
todo o resto do firmware só enxerga o tipo `BoardConfig`, nunca sabe qual
arquivo o preencheu.

## Pinagem por placa

### Matriz de botões (4×4 = 16 botões, 8 GPIOs)

| Sinal | ESP32 | Leonardo / Pro Micro |
|---|---|---|
| Linha 0 (ROW) | GPIO13 | D2 |
| Linha 1 (ROW) | GPIO12 ⚠️ strapping | D3 |
| Linha 2 (ROW) | GPIO14 | D4 |
| Linha 3 (ROW) | GPIO27 | D5 |
| Coluna 0 (COL) | GPIO26 | D6 |
| Coluna 1 (COL) | GPIO25 | D7 |
| Coluna 2 (COL) | GPIO33 | D8 |
| Coluna 3 (COL) | GPIO32 | D9 |

Índice do botão no software = `linha × 4 + coluna` (0 a 15), configurado em
`BUTTON_MODES[]` no `Config.h` — igual nas três placas.

### Eixos analógicos (4 eixos)

| Eixo | ESP32 (ADC1, 12 bits) | Leonardo / Pro Micro (10 bits) |
|---|---|---|
| X | GPIO34 | A0 |
| Y | GPIO35 | A1 |
| Z | GPIO36 | A2 |
| Rz | GPIO39 | A3 |

### LEDs de status (4 LEDs, independentes do report HID)

| LED (campo em `LedPins`) | ESP32 | Leonardo / Pro Micro |
|---|---|---|
| `doorOpen` | GPIO2 ⚠️ strapping | D10 |
| `alert` | GPIO4 | D14 ⚠️ ver nota |
| `status` | GPIO5 | D15 ⚠️ ver nota |
| `spare` | GPIO18 | D16 ⚠️ ver nota |

`status` já é usado no `main.cpp`: no ESP32 acende com conexão BLE ativa; no
Leonardo/Pro Micro fica sempre aceso (USB cabeado).

## ⚠️ Notas de pinagem

**ESP32 — pinos de strapping (GPIO2, GPIO12):** o ESP32 lê o nível desses
pinos no boot para decidir o modo de inicialização. Usá-los como GPIO comum
depois do boot é prática normal, mas evite pull-downs fortes neles durante o
power-on/reset. Se notar boot instável, troque por GPIO15, 19 ou 23 em
`BoardConfigEsp32.cpp`.

**Leonardo — pinos 14/15/16:** no Pro Micro esses pinos já vêm soldados como
pinos digitais normais na borda da placa. No **Leonardo**, os mesmos números
correspondem ao **header ICSP** (MISO=14, SCK=15, MOSI=16), separado do
header principal — funcionam normalmente por código, só exigem soldar no
header ICSP (6 pinos, no meio da placa) em vez do header lateral. Se
preferir evitar isso no Leonardo, troque os valores de `leds` em
`BoardConfigAvr.cpp` por pinos livres do header principal, como `{10, 11,
12, 13}` (isso afeta as duas placas, já que compartilham o mesmo arquivo —
se quiser mapas diferentes entre Leonardo e Pro Micro, basta duplicar
`BoardConfigAvr.cpp` e usar `#ifdef ARDUINO_AVR_LEONARDO` só nesse arquivo).

**ESP32 — ADC1 apenas:** use só pinos ADC1 (GPIO32-39) para os eixos. O ADC2
tem conflito conhecido com o rádio Wi-Fi/BLE e não deve ser usado enquanto o
BLE está ativo.

## Arquitetura do código

```
buttonbox/
├── platformio.ini              <- 3 ambientes + build_src_filter por placa
├── include/
│   ├── Config.h                 <- comportamento (timings, modos de botão) - sem pinos
│   ├── BoardConfig.h             <- structs tipados de pinagem (MatrixPins, AxisPins, LedPins...)
│   ├── GamepadState.h            <- structs tipados de estado (AxisValues, GamepadState)
│   ├── InputTypes.h              <- enum ButtonMode
│   ├── IButton.h                 <- interface abstrata de comportamento
│   ├── NormalButton.h / OneShotButton.h / ToggleButton.h / LongPressButton.h
│   ├── ButtonManager.h           <- recebe MatrixPins tipado
│   ├── Axis.h                    <- recebe AxisReadSettings tipado
│   ├── AxisManager.h             <- recebe AxisPins/AxisInvertFlags/AxisReadSettings, devolve AxisValues
│   ├── LedController.h           <- recebe LedPins, expõe setDoorOpen/setAlert/setStatus/setSpare
│   ├── GamepadOutput.h            <- interface única (SEM implementação)
│   ├── HidGamepad.h                <- report descriptor BLE (só ESP32)
│   ├── BleManager.h                 <- serviço BLE HID via NimBLE (só ESP32)
│   └── UsbGamepad.h                  <- wrapper da ArduinoJoystickLibrary (só AVR)
└── src/
    ├── main.cpp                     <- idêntico nas 3 placas
    ├── BoardConfigEsp32.cpp          <- ⚡ implementação de pinagem, só ESP32
    ├── BoardConfigAvr.cpp             <- ⚡ implementação de pinagem, só AVR
    ├── GamepadOutputEsp32.cpp          <- ⚡ implementação de transporte, só ESP32
    ├── GamepadOutputAvr.cpp             <- ⚡ implementação de transporte, só AVR
    ├── HidGamepad.cpp / BleManager.cpp   <- excluídos do build em AVR
    ├── UsbGamepad.cpp                     <- excluído do build no ESP32
    ├── Axis.cpp / AxisManager.cpp / ButtonManager.cpp / LedController.cpp <- portáteis
    └── NormalButton.cpp / OneShotButton.cpp / ToggleButton.cpp / LongPressButton.cpp <- portáteis
```

### Como adicionar/mudar o comportamento de um botão

Continua igual — edite só `BUTTON_MODES[]` em `include/Config.h`:

```cpp
static const ButtonMode BUTTON_MODES[BUTTON_COUNT] = {
    /* 0*/ ButtonMode::NORMAL,
    /* 2*/ ButtonMode::ONE_SHOT,   // <- hold vira pulso único
    ...
};
```

| Modo | O que faz |
|---|---|
| `NORMAL` | Report = estado físico, 1:1 |
| `ONE_SHOT` | Qualquer press vira um pulso único de `ONE_SHOT_PULSE_MS` |
| `TOGGLE` | Cada press alterna entre ligado/desligado |
| `LONG_PRESS` | Só ativa se segurar por `LONG_PRESS_THRESHOLD_MS` |

### Como adicionar uma quarta placa no futuro

1. Crie `BoardConfig<Placa>.cpp` implementando `getBoardConfig()` com os
   pinos daquela placa (copie um dos dois existentes como base).
2. Crie `GamepadOutput<Placa>.cpp` implementando `begin()`/`update()`/
   `isConnected()` para o transporte daquela placa.
3. Adicione um `[env:<placa>]` no `platformio.ini`, excluindo via
   `build_src_filter` os arquivos das OUTRAS plataformas.

Nenhum arquivo existente precisa ser modificado.

## Escalando para mais botões (ex: 25)

Trocar `MATRIX_ROWS`/`MATRIX_COLS` de 4 para 5 em `Config.h` (matriz 5×5 =
25 botões, 10 pinos) funciona igual nas três placas — só é preciso também
atualizar os arrays `rows`/`cols` em `BoardConfigEsp32.cpp` e
`BoardConfigAvr.cpp` para terem 5 elementos cada. Nesse tamanho, use
**diodos em série com cada botão** para evitar leitura fantasma ("ghosting")
quando várias teclas forem pressionadas juntas.

## Limitação conhecida (todas as placas)

Alguns jogos (principalmente simuladores no Windows) só aceitam **XInput**
nativamente e não reconhecem gamepad HID genérico automaticamente. Nesses
casos, ferramentas como **x360ce** ou **JoyToKey** convertem o HID genérico
em um XInput virtual do lado do PC.

## Caminho futuro: Bluetooth no Leonardo/Pro Micro

Módulos como HC-05 (Bluetooth Classic/SPP) ou HM-10 (BLE-UART) não
implementam o perfil HID — só fazem ponte serial. Isso exigiria um programa
rodando no PC (Windows e Linux) para converter os bytes recebidos em um
joystick virtual (vJoy/ViGEmBus no Windows, `uinput` no Linux). O ESP32 já
resolve isso nativamente via BLE HID, sem precisar de nada disso.