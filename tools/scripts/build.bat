@echo off
REM ButtonBox — build local Windows (reproduz CI)
REM Uso:
REM   tools\scripts\build.bat            REM tudo
REM   tools\scripts\build.bat --firmware REM só firmwares
REM   tools\scripts\build.bat --gui      REM só GUI
REM   tools\scripts\build.bat --clean    REM limpa dist\
setlocal enabledelayedexpansion

set "ROOT=%~dp0..\.."
pushd "%ROOT%"

REM version from git
for /f "delims=" %%v in ('git describe --tags --always --dirty 2^>nul') do set "VER=%%v"
if "%VER%"=="" set "VER=v0.0.0-dev"
echo %VER% | findstr /b "v" >nul || set "VER=v%VER%"

set DO_FW=0
set DO_GUI=0
set DO_CLEAN=0
if "%~1"=="" set DO_FW=1 & set DO_GUI=1

:parse
if "%~1"=="" goto run
if "%~1"=="--firmware" set DO_FW=1
if "%~1"=="--gui" set DO_GUI=1
if "%~1"=="--all" set DO_FW=1 & set DO_GUI=1
if "%~1"=="--clean" set DO_CLEAN=1
if "%~1"=="-h" goto help
if "%~1"=="--help" goto help
shift
goto parse

:help
echo Uso: %~nx0 [--firmware] [--gui] [--all] [--clean]
exit /b 0

:run
if %DO_CLEAN%==1 (
  echo ==^> Limpando dist\ ...
  if exist dist rmdir /s /q dist
  if exist tools\config-gui\pkg rmdir /s /q tools\config-gui\pkg
  del /q tools\config-gui\buttonbox-config-gui*.exe 2>nul
  del /q tools\config-gui\buttonbox-config-gui-v* 2>nul
  where pio >nul 2>nul && pio run --target clean
  echo Limpeza ok.
  if %DO_FW%==0 if %DO_GUI%==0 goto end
)

if %DO_FW%==1 (
  echo ==^> Firmware — version %VER%
  where pio >nul 2>nul
  if errorlevel 1 (
    where platformio >nul 2>nul
    if errorlevel 1 (
      echo ERRO: PlatformIO nao encontrado. Instale: pip install platformio
      exit /b 1
    ) else set "PIO=platformio"
  ) else set "PIO=pio"
  call %PIO% run
  if errorlevel 1 exit /b 1

  if not exist dist\firmware mkdir dist\firmware
  if exist .pio\build\esp32dev\firmware.bin copy /y .pio\build\esp32dev\firmware.bin dist\firmware\firmware-esp32dev-%VER%.bin >nul
  if exist .pio\build\esp32dev\firmware.elf copy /y .pio\build\esp32dev\firmware.elf dist\firmware\firmware-esp32dev-%VER%.elf >nul
  if exist .pio\build\esp32dev\bootloader.bin copy /y .pio\build\esp32dev\bootloader.bin dist\firmware\bootloader-esp32dev-%VER%.bin >nul
  if exist .pio\build\esp32dev\partitions.bin copy /y .pio\build\esp32dev\partitions.bin dist\firmware\partitions-esp32dev-%VER%.bin >nul
  if exist .pio\build\leonardo\firmware.hex copy /y .pio\build\leonardo\firmware.hex dist\firmware\firmware-leonardo-%VER%.hex >nul
  if exist .pio\build\leonardo\firmware.elf copy /y .pio\build\leonardo\firmware.elf dist\firmware\firmware-leonardo-%VER%.elf >nul
  if exist .pio\build\promicro16\firmware.hex copy /y .pio\build\promicro16\firmware.hex dist\firmware\firmware-promicro16-%VER%.hex >nul
  if exist .pio\build\promicro16\firmware.elf copy /y .pio\build\promicro16\firmware.elf dist\firmware\firmware-promicro16-%VER%.elf >nul

  echo Firmwares em dist\firmware\:
  dir /b dist\firmware
  REM checksums via certutil
  pushd dist\firmware
  if exist SHA256SUMS.txt del SHA256SUMS.txt
  for %%f in (*.bin *.hex *.elf) do (
    certutil -hashfile "%%f" SHA256 | findstr /v "hash CertUtil" >> SHA256SUMS.txt
  )
  popd
)

if %DO_GUI%==1 (
  echo ==^> GUI — version %VER%
  where go >nul 2>nul
  if errorlevel 1 (
    echo ERRO: Go nao encontrado. Instale https://go.dev/dl/
    exit /b 1
  )
  pushd tools\config-gui
  set CGO_ENABLED=1
  echo   go vet...
  go vet ./...
  if errorlevel 1 exit /b 1

  echo   go build...
  go build -trimpath -ldflags="-s -w -H windowsgui -X main.version=%VER% -X main.buildVersion=%VER%" -o buttonbox-config-gui-%VER%.exe ./...
  if errorlevel 1 (
    echo ERRO: build falhou -- verifique se GCC ^(mingw^) esta instalado.
    echo Dica: instale via https://winlibs.com/ ou 'choco install mingw'
    exit /b 1
  )
  dir buttonbox-config-gui-%VER%.exe

  mkdir pkg 2>nul
  set "PKG_DIR=pkg\buttonbox-config-gui-local-%VER%"
  if exist "%PKG_DIR%" rmdir /s /q "%PKG_DIR%"
  mkdir "%PKG_DIR%"
  copy /y buttonbox-config-gui-%VER%.exe "%PKG_DIR%\buttonbox-config-gui.exe" >nul
  if exist locales xcopy /e /i /y locales "%PKG_DIR%\locales" >nul
  if exist buttonbox-config.json copy /y buttonbox-config.json "%PKG_DIR%\buttonbox-config.example.json" >nul
  echo ButtonBox Config GUI %VER% > "%PKG_DIR%\README.txt"
  echo Uso: execute o .exe ao lado da pasta locales\ >> "%PKG_DIR%\README.txt"

  if not exist ..\..\dist\gui mkdir ..\..\dist\gui
  REM tenta zip via powershell
  powershell -Command "Compress-Archive -Path '%PKG_DIR%' -DestinationPath '..\\..\\dist\\gui\\buttonbox-config-gui-windows-amd64-%VER%.zip' -Force" 2>nul
  if exist ..\..\dist\gui\buttonbox-config-gui-windows-amd64-%VER%.zip (
    echo Pacote: dist\gui\buttonbox-config-gui-windows-amd64-%VER%.zip
  )
  REM também deixa o exe solto em dist\gui
  copy /y buttonbox-config-gui-%VER%.exe ..\..\dist\gui\ >nul
  dir ..\..\dist\gui
  popd
)

echo.
echo ==^> Build concluido — versao %VER%
echo     Artefatos em dist\
dir /s /b dist 2>nul | more

:end
popd
endlocal
