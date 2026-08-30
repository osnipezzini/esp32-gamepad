# ButtonBox GUI - Melhorias Implementadas

## 📋 Resumo das Mudanças

### 1. **Multi-idioma (i18n) - EN/PT-BR**
- Sistema de tradução completo implementado
- **Seletor de idioma** na barra superior (English / Português BR)
- Todas as mensagens, títulos e labels em estrutura de tradução
- Troca de idioma em tempo real (sem reiniciar)

**Traduções incluídas:**
- Títulos e labels da UI
- Mensagens de status
- Descrições dos modos de botão
- Mensagens de erro

---

### 2. **Layout Reorganizado e Compacto**
- **Grid reduzido de 4 colunas → 8 colunas**: Permite visualizar os 16 botões em 2 linhas
- **LEDs menores**: 24x24 pixels (antes: 40x40)
- **Botões menos gigantes**: Removido label redundante de modo
- **Grid com scroll**: Permite navegação se necessário

**Estrutura visual:**
```
┌─────────────────────────────────────────────────────────┐
│  Porta Serial: [selector] [Read] [Apply] [Reset] | Lang: [EN/PT-BR] │
├─────────────────────────────────────────────────────────┤
│ Joystick: [selector] [Monitor Joystick / Stop]         │
├─────────────────────────────────────────────────────────┤
│  [Button 0]  [Button 1]  [Button 2] ... [Button 15]    │
│   • LED                   • LED                          │
│   • Mode selector         • Mode selector                │
│   • Color bar             • Color bar                    │
├─────────────────────────────────────────────────────────┤
│ C++ Export: [text field]                                │
├─────────────────────────────────────────────────────────┤
│ Status: [mensagens de status]                           │
├─────────────────────────────────────────────────────────┤
│ LEGENDA                                                  │
│ LED State:                                              │
│  • Green: Not Pressed                                   │
│  • Red: Pressed                                         │
│                                                          │
│ Button Modes:                                           │
│  • NORMAL: Simple button press                          │
│  • ONE-SHOT: Activates once, then resets                │
│  • TOGGLE: Toggles on/off with each press              │
│  • LONG_PRESS: Emulates held press                      │
└─────────────────────────────────────────────────────────┘
```

---

### 3. **Tooltips e Descrições dos Modos**
Cada modo de botão agora tem descrição clara:
- **NORMAL**: "Simple button press"
- **ONE_SHOT**: "Activates once, then resets"
- **TOGGLE**: "Toggles on/off with each press"
- **LONG_PRESS**: "Emulates held press"

---

### 4. **Joystick Monitoring Corrigido**
Implementados **fixes críticos** para o monitoramento em tempo real:

#### Problema anterior:
- Eventos não causavam atualização da UI
- Sem sincronização entre goroutine e UI
- Estado de monitoramento não era rastreado corretamente

#### Soluções implementadas:
- ✅ **Mutex sincronizado** (`jsMutex`) para thread-safety
- ✅ **Flag `jsMonitoring`** para rastrear estado
- ✅ **Callback de atualização UI** passado ao `readJoystickEvents()`
- ✅ **Toggle do botão**: "Monitor Joystick" ↔ "Stop Monitoring"
- ✅ **Detecção de múltiplos joysticks** (`/dev/input/js*`)
- ✅ **Tratamento de erros** melhorado com mensagens i18n

---

### 5. **Interface Mais Profissional**
- Melhor espaçamento e organização
- Usando `widget.RichTextFromMarkdown()` para títulos formatados
- Cores consistentes para os modos
- Status bar com mensagens claras
- Suporte a múltiplas seleções (porta serial + joystick independentes)

---

## 🔧 Mudanças Técnicas

### Estrutura de Estado Atualizada
```go
type uiState struct {
    // ... campos anteriores ...
    jsMonitoring   bool         // ✨ Novo: rastreia se está monitorando
    jsMutex        sync.Mutex   // ✨ Novo: sincroniza acesso concurrent
}
```

### Função de Leitura de Eventos Melhorada
```go
func readJoystickEvents(state *uiState, joystickPath string, winUpdateFunc func()) {
    // ✨ Agora recebe callback para atualizar UI
    // ✨ Usa mutex para sincronização thread-safe
    // ✨ Respeita flag jsMonitoring para parar limpo
}
```

### Sistema de Tradução (i18n)
```go
var translations = map[Language]map[string]string{
    LangEnglish: { "title": "ButtonBox – Button Mode Configuration", ... },
    LangPtBr:    { "title": "ButtonBox – Configuração de Modos", ... },
}

func t(key string) string { /* retorna tradução */ }
```

---

## 🚀 Como Usar

### 1. **Compilar**
```bash
cd /home/osni/projetos/sodevs/buttonbox/tools/config-gui
go build ./...
./buttonbox-config-gui
```

### 2. **Operação**
1. **Selecione a porta serial** onde o ButtonBox está conectado
2. Clique **"Read from Board"** para ler a configuração atual
3. Selecione o **modo de botão** para cada posição (dropdown)
4. Clique **"Apply to Board"** para salvar na placa
5. Selecione o **joystick** em `/dev/input/jsX`
6. Clique **"Monitor Joystick"** para ver LEDs em tempo real
7. Pressione botões físicos → LEDs devem mudar instantaneamente
8. Mude o idioma com o seletor **"Lang"** no topo

---

## ✅ Testes Recomendados

- [ ] Compilação sem erros
- [ ] GUI abre com layout correto
- [ ] Seletor de idioma funciona (EN ↔ PT-BR)
- [ ] Botões de porta serial funcionam
- [ ] Monitoramento de joystick detecta múltiplos devices
- [ ] Pressionar botões físicos → LEDs atualizam em tempo real
- [ ] Parar/iniciar monitoramento limpa recursos corretamente
- [ ] Modo "NORMAL"/"ONE_SHOT"/"TOGGLE"/"LONG_PRESS" exibidos com cores corretas

---

## 🐛 Correções de Erros Implementadas

1. **Problema**: LEDs eram 40x40, ocupando muito espaço
   **Solução**: Reduzido para 24x24 pixels

2. **Problema**: Grid com 4 colunas mostrava 4 botões por linha (gigantes)
   **Solução**: Expandido para 8 colunas (2 linhas de 8 botões)

3. **Problema**: Eventos de joystick não atualizavam UI
   **Solução**: Adicionar callback e sincronização com mutex

4. **Problema**: Monitoramento não podia ser parado
   **Solução**: Flag `jsMonitoring` + botão toggle

5. **Problema**: Texto português fixo (sem suporte a EN)
   **Solução**: Sistema i18n completo

---

## 📦 Dependências
- `fyne.io/fyne/v2` v2.5.3
- `go.bug.st/serial` v1.6.2
- Stdlib: `sync`, `os`, `io`, `encoding/json`, etc.

---

## 🔮 Melhorias Futuras (Opcional)

- [ ] Gráfico de histórico de pressões por botão
- [ ] Perfis salvos/carregados por nome
- [ ] Teste de latência botão→LED
- [ ] Suporte para mais idiomas
- [ ] Dark mode
- [ ] Exportar/importar configurações em JSON

