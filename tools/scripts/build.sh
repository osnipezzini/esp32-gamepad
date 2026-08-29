#!/usr/bin/env bash
# ButtonBox — build local (reproduz CI)
# Uso:
#   ./tools/scripts/build.sh            # tudo (firmware + gui)
#   ./tools/scripts/build.sh --firmware # só firmwares
#   ./tools/scripts/build.sh --gui      # só GUI
#   ./tools/scripts/build.sh --clean    # limpa dist/ + .pio/build
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

VER="$(git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-dev")"
[[ "$VER" == v* ]] || VER="v$VER"

DO_FW=0; DO_GUI=0; DO_CLEAN=0
if [[ $# -eq 0 ]]; then DO_FW=1; DO_GUI=1; fi
for arg in "$@"; do
  case "$arg" in
    --firmware) DO_FW=1 ;;
    --gui)      DO_GUI=1 ;;
    --all)      DO_FW=1; DO_GUI=1 ;;
    --clean)    DO_CLEAN=1 ;;
    -h|--help)
      echo "Uso: $0 [--firmware] [--gui] [--all] [--clean]"
      exit 0 ;;
    *) echo "Opção desconhecida: $arg" >&2; exit 1 ;;
  esac
done

if [[ $DO_CLEAN -eq 1 ]]; then
  echo "==> Limpando dist/ e artefatos locais..."
  rm -rf dist/ tools/config-gui/buttonbox-config-gui* tools/config-gui/pkg/
  if command -v pio &>/dev/null; then pio run --target clean || true; fi
  echo "Limpeza ok."
  if [[ $DO_FW -eq 0 && $DO_GUI -eq 0 ]]; then exit 0; fi
fi

# ── Firmware ──────────────────────────────────────────────
if [[ $DO_FW -eq 1 ]]; then
  echo "==> Firmware — version $VER"
  if ! command -v pio &>/dev/null && ! command -v platformio &>/dev/null; then
    echo "ERRO: PlatformIO não encontrado. Instale: pip install platformio" >&2
    exit 1
  fi
  PIO="pio"
  command -v pio &>/dev/null || PIO="platformio"

  $PIO run

  OUT="dist/firmware"
  mkdir -p "$OUT"
  # copia com versão
  [[ -f .pio/build/esp32dev/firmware.bin ]]       && cp .pio/build/esp32dev/firmware.bin       "$OUT/firmware-esp32dev-${VER}.bin"
  [[ -f .pio/build/esp32dev/firmware.elf ]]       && cp .pio/build/esp32dev/firmware.elf       "$OUT/firmware-esp32dev-${VER}.elf" || true
  [[ -f .pio/build/esp32dev/bootloader.bin ]]     && cp .pio/build/esp32dev/bootloader.bin     "$OUT/bootloader-esp32dev-${VER}.bin" || true
  [[ -f .pio/build/esp32dev/partitions.bin ]]     && cp .pio/build/esp32dev/partitions.bin     "$OUT/partitions-esp32dev-${VER}.bin" || true
  [[ -f .pio/build/leonardo/firmware.hex ]]       && cp .pio/build/leonardo/firmware.hex       "$OUT/firmware-leonardo-${VER}.hex"
  [[ -f .pio/build/leonardo/firmware.elf ]]       && cp .pio/build/leonardo/firmware.elf       "$OUT/firmware-leonardo-${VER}.elf" || true
  [[ -f .pio/build/promicro16/firmware.hex ]]     && cp .pio/build/promicro16/firmware.hex     "$OUT/firmware-promicro16-${VER}.hex"
  [[ -f .pio/build/promicro16/firmware.elf ]]     && cp .pio/build/promicro16/firmware.elf     "$OUT/firmware-promicro16-${VER}.elf" || true

  (cd "$OUT" && sha256sum * > SHA256SUMS.txt 2>/dev/null || shasum -a 256 * > SHA256SUMS.txt)
  echo "Firmwares em $OUT/:"
  ls -lh "$OUT/"
fi

# ── GUI ───────────────────────────────────────────────────
if [[ $DO_GUI -eq 1 ]]; then
  echo "==> GUI — version $VER"
  if ! command -v go &>/dev/null; then
    echo "ERRO: Go não encontrado (https://go.dev/dl/)" >&2
    exit 1
  fi

  pushd tools/config-gui >/dev/null

  # deps de sistema (Linux) — só avisa se faltar
  if [[ "$(uname -s)" == "Linux" ]]; then
    for pkg in gcc pkg-config; do
      command -v "$pkg" &>/dev/null || echo "AVISO: $pkg não encontrado — build Fyne pode falhar. Instale libgl1-mesa-dev xorg-dev libasound2-dev etc." >&2
    done
  fi

  echo "  go vet..."
  go vet ./...

  echo "  go build..."
  CGO_ENABLED=1 go build -trimpath -ldflags="-s -w -X main.version=${VER} -X main.buildVersion=${VER}" -o "buttonbox-config-gui-${VER}" ./...
  ls -lh "buttonbox-config-gui-${VER}" 2>/dev/null || ls -lh buttonbox-config-gui* || true
  file "buttonbox-config-gui-${VER}" || true

  # pacote local (zip/tar.gz conforme OS)
  PKG_DIR="pkg/buttonbox-config-gui-local-${VER}"
  mkdir -p "$PKG_DIR"
  cp "buttonbox-config-gui-${VER}" "$PKG_DIR/buttonbox-config-gui" 2>/dev/null || cp buttonbox-config-gui* "$PKG_DIR/" 2>/dev/null || true
  [[ -d locales ]] && cp -r locales "$PKG_DIR/" || true
  [[ -f buttonbox-config.json ]] && cp buttonbox-config.json "$PKG_DIR/buttonbox-config.example.json" || true

  STAGE="../../dist/gui"
  mkdir -p "$STAGE"
  if [[ "$(uname -s)" == "Linux" || "$(uname -s)" == "Darwin" ]]; then
    tar -czf "$STAGE/buttonbox-config-gui-local-${VER}.tar.gz" -C pkg "$(basename "$PKG_DIR")"
    echo "Pacote: $STAGE/buttonbox-config-gui-local-${VER}.tar.gz"
  else
    # fallback
    tar -czf "$STAGE/buttonbox-config-gui-local-${VER}.tar.gz" -C pkg "$(basename "$PKG_DIR")" 2>/dev/null || true
  fi
  ls -lh "$STAGE/" || true

  popd >/dev/null
fi

echo ""
echo "==> Build concluído — versão $VER"
echo "    Artefatos em dist/"
ls -R dist 2>/dev/null | head -80 || true
