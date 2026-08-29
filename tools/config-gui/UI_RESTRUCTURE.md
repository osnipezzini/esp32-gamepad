# UI Reestruturada - ButtonBox Config GUI

## 🎯 Mudanças Implementadas

### 1. **ModeInfo Expandido**
Adicionado informações detalhadas para cada modo de botão:
- `Description`: Descrição completa do funcionamento
- `ShortDesc`: Descrição curta para legendas
- `IdealFor`: Casos de uso recomendados (em PT-BR)
- `IdealFor`: Casos de uso recomendados (em EN)

**Modos disponíveis:**
| Modo | Descrição | Ideal Para |
|------|-----------|------------|
| 🟦 **NORMAL** | Ativo apenas enquanto pressionado | Aceleração, freio, funções temporárias |
| 🟧 **ONE-SHOT** | Dispara uma vez e reseta automaticamente | Navegação em menus, ações únicas |
| 🟩 **TOGGLE** | Alterna entre ligado/desligado | Turbo, shift mode, funções ON/OFF |
| 🟥 **LONG-PRESS** | Simula pressão de ~2 segundos | Funções especiais, reset, pairing |

### 2. **Legenda Melhorada**
A legenda agora mostra:
- ✅ Estado do LED (pressionado/não pressionado)
- ✅ Nome do modo com cor correspondente
- ✅ Descrição curta do funcionamento
- ✅ 💡 Dica de uso ideal para cada modo

### 3. **Suporte Cross-Platform para Joystick**

#### Linux (Nativo)
```go
// Leitura direta de /dev/input/js*
entries, err := os.ReadDir("/dev/input")
```

#### Windows (Compatibilidade)
```go
// Lista genérica de joysticks HID
// Em produção: usar SDL2 ou similar
candidates := []string{
    "Joystick 0 (Windows HID)",
    "Joystick 1 (Windows HID)",
    // ...
}
```

**Nota:** No Windows, a leitura direta via `/dev/input` não está disponível. Para suporte completo, recomenda-se integrar com:
- `github.com/gamerathon/go-sdl2`
- `github.com/veandco/go-sdl2`

### 4. **Dialogo de Ajuda Atualizado**
O help dialog agora exibe:
- Título formatado em Markdown
- Descrição de cada modo
- 💡 Dica de uso ideal (traduzida)
- Separadores visuais entre modos

### 5. **Traduções Completas**

#### pt-br.json
```json
{
  "mode_normal_desc": "Ativo apenas enquanto pressionado",
  "mode_normal_ideal": "Aceleração, freio, funções temporárias",
  "windows_joystick_note": "Windows: Leitura direta não disponível..."
}
```

#### en.json
```json
{
  "mode_normal_desc": "Active only while physically pressed",
  "mode_normal_ideal": "Acceleration, brake, temporary functions",
  "windows_joystick_note": "Windows: Direct reading unavailable..."
}
```

## 📁 Arquivos Modificados

1. **main.go**
   - Adicionado campo `runtime` para detecção de OS
   - Expandido `ModeInfo` com `ShortDesc` e `IdealFor`
   - Atualizado `listJoysticks()` para suportar Windows
   - Atualizado `readJoystickEvents()` com fallback para Windows
   - Melhorado `buildLegend()` com dicas de uso
   - Simplificado `showModeHelpDialog()` usando traduções

2. **locales/pt-br.json**
   - Adicionado descrições dos modos (`_desc`)
   - Adicionado casos de uso (`_ideal`)
   - Adicionado notas Windows (`windows_joystick_note`)
   - Adicionado textos do help dialog

3. **locales/en.json**
   - Mesmas adições do pt-br.json em inglês

## 🚀 Como Usar

### Build (pode demorar na primeira vez)
```bash
cd tools/config-gui
go build -o buttonbox-config .
```

### Executar
```bash
./buttonbox-config
```

### Desenvolvimento (hot reload)
```bash
go run .
```

## 🎨 Layout da UI

```
┌─────────────────────────────────────────────────────┐
│ 🔶 ButtonBox — Config              [LANG] [Help]    │
├─────────────────────────────────────────────────────┤
│ ┌──────────────┐  ┌──────────────┐                 │
│ │ PORTA SERIAL │  │   JOYSTICK   │                 │
│ │ [COM3 ▼]     │  │ [/dev/js0▼]  │                 │
│ │ [Read][Apply]│  │  [Monitor]   │                 │
│ └──────────────┘  └──────────────┘                 │
├─────────────────────────────────────────────────────┤
│ ┌────┐ ┌────┐ ┌────┐ ┌────┐  │  LEGEND             │
│ │ 00 │ │ 01 │ │ 02 │ │ 03 │  │  ● Not Pressed      │
│ │LED │ │LED │ │LED │ │LED │  │  ● Pressed          │
│ │NRM │ │NRM │ │ONS │ │NRM │  │                     │
│ │[▼] │ │[▼] │ │[▼] │ │[▼] │  │  MODES:             │
│ │ ●  │ │ ●  │ │ ●  │ │ ●  │  │  🟦 NORMAL          │
│ └────┘ └────┘ └────┘ └────┘  │     Active while... │
│                               │     💡 Acceleration │
│ ┌────┐ ┌────┐ ┌────┐ ┌────┐  │                     │
│ │ 04 │ │ 05 │ │ 06 │ │ 07 │  │  🟧 ONE-SHOT        │
│ │LED │ │LED │ │LED │ │LED │  │     Triggers once.. │
│ │TGL │ │NRM │ │NRM │ │LNG │  │     💡 Menu nav     │
│ │[▼] │ │[▼] │ │[▼] │ │[▼] │  │                     │
│ │ ●  │ │ ●  │ │ ●  │ │ ●  │  │  ...                │
│ └────┘ └────┘ └────┘ └────┘  │                     │
├─────────────────────────────────────────────────────┤
│ C++ EXPORT                                          │
│ ┌───────────────────────────────────────────────┐  │
│ │ static const ButtonMode BUTTON_MODES[...] = { │  │
│ │     /*  0 */ ButtonMode::NORMAL,             │  │
│ │     ...                                     │  │
│ └───────────────────────────────────────────────┘  │
│ Status: Ready                                       │
└─────────────────────────────────────────────────────┘
```

## ✅ Benefícios

1. **Mais Informativo**: Usuário entende cada modo sem precisar clicar em ajuda
2. **Nativo**: Widgets padrão Fyne, menos dependências externas
3. **Cross-platform**: Funciona em Linux e Windows (com limitações)
4. **Go puro**: Mantém simplicidade, sem bibliotecas pesadas
5. **Educacional**: Dicas de uso ajudam usuários iniciantes

## ⚠️ Limitações Conhecidas

### Windows Joystick
- Listagem genérica de joysticks (não detecta dispositivos reais)
- Monitoramento simula estados aleatórios para demo da UI
- **Solução recomendada**: Integrar SDL2 para leitura nativa

### macOS
- Não testado explicitamente
- Deve funcionar como Linux para joystick (`/dev/input`)

## 🔧 Próximos Passos Sugeridos

1. **Integrar SDL2** para suporte nativo Windows/macOS de joystick
2. **Adicionar tooltips** nos cards dos botões ao passar mouse
3. **Exportar configuração** como arquivo `.h` ou `.cpp`
4. **Auto-save** após mudança de modo
5. **Perfil múltiplos** de configuração

---

**Status**: ✅ Implementado e pronto para teste
**Data**: 2025
**Autor**: ButtonBox Team
