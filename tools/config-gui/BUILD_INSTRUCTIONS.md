# Build Instructions for ButtonBox Config GUI

## ✅ Linux Build (Working)

```bash
cd tools/config-gui
go build -o buttonbox-config-gui .
```

Binary criado: `buttonbox-config-gui` (29MB)

## ⚠️ Windows Build Issue

O build para Windows está falhando porque o Fyne requer OpenGL e dependências nativas que não podem ser cross-compiled facilmente do Linux.

### Soluções para Build Windows:

#### Opção 1: Build Direto no Windows (Recomendado)
No seu Windows, execute:

```powershell
cd tools\config-gui
go build -ldflags="-H windowsgui" -o buttonbox-config-gui.exe .
```

**Pré-requisitos no Windows:**
- GCC (MinGW ou TDM-GCC) instalado
- Variável de ambiente `CGO_ENABLED=1`

#### Opção 2: Usar Docker para Cross-compile

```bash
docker run --rm -it -v ${PWD}:/app -w /app golang:1.21 bash
apt-get update && apt-get install -y gcc-mingw-w64-x86-64
export CGO_ENABLED=1
export CC=x86_64-w64-mingw32-gcc
go build -ldflags="-H windowsgui" -o buttonbox-config-gui.exe .
```

#### Opção 3: GitHub Actions (CI/CD)

Criar um workflow que compila para ambas plataformas automaticamente.

## 🎯 Código Corrigido

Os seguintes erros foram corrigidos no `main.go`:

1. ❌ `TextSize` field removed from `widget.Label` (Fyne 2.x)
   - ✅ Removido das labels `descLabel` e `idealLabel`
   
2. ❌ `widget.NewMarkdown` não existe em Fyne 2.x
   - ✅ Substituído por `widget.NewLabel` no dialog de help

## 📝 Notas Importantes

- O código fonte agora é compatível com Fyne 2.x
- Build Linux testado e funcionando ✅
- Para Windows: compile diretamente no Windows com GCC instalado
- Joystick monitoring funciona nativamente no Linux (`/dev/input/js*`)
- No Windows, o joystick usa fallback (demo mode) até integrar SDL2

## 🚀 Próximos Passos

1. Teste o binário Linux: `./buttonbox-config-gui`
2. Para Windows, compile no próprio Windows
3. Opcional: Adicionar SDL2 para melhor suporte a joystick no Windows
