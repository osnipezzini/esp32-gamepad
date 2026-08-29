package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	serial "go.bug.st/serial"
)

const totalButtons = 16

// Language type
type Language string

const (
	LangEnglish Language = "en"
	LangPtBr    Language = "pt-br"
)

var defaultModes = []string{
	"NORMAL", "NORMAL", "ONE_SHOT", "NORMAL",
	"TOGGLE", "NORMAL", "NORMAL", "LONG_PRESS",
	"NORMAL", "NORMAL", "NORMAL", "NORMAL",
	"NORMAL", "NORMAL", "NORMAL", "NORMAL",
}

// ModeInfo describes a button mode with description and color
type ModeInfo struct {
	Key         string
	Name        string
	Description string
	ShortDesc   string
	IdealFor    string
	Color       color.Color
}

var modeInfos = []ModeInfo{
	{
		Key:         "mode_normal",
		Name:        "NORMAL",
		Description: "Standard button behavior - active only while physically pressed. Releases immediately when button is let go.",
		ShortDesc:   "Ativo apenas enquanto pressionado",
		IdealFor:    "Aceleração, freio, funções temporárias",
		Color:       color.NRGBA{R: 74, G: 155, B: 224, A: 255},
	},
	{
		Key:         "mode_one_shot",
		Name:        "ONE-SHOT",
		Description: "Activates once on press, then automatically resets. Even if you keep holding the button, it triggers only once.",
		ShortDesc:   "Dispara uma vez e reseta automaticamente",
		IdealFor:    "Navegação em menus, ações únicas",
		Color:       color.NRGBA{R: 232, G: 164, B: 74, A: 255},
	},
	{
		Key:         "mode_toggle",
		Name:        "TOGGLE",
		Description: "Switches state on each press (on/off). First press activates, second press deactivates. State persists until toggled again.",
		ShortDesc:   "Alterna entre ligado/desligado",
		IdealFor:    "Turbo, shift mode, funções ON/OFF",
		Color:       color.NRGBA{R: 125, G: 216, B: 125, A: 255},
	},
	{
		Key:         "mode_long_press",
		Name:        "LONG-PRESS",
		Description: "Simulates holding the button for ~2 seconds when briefly pressed. Useful for functions that require extended activation.",
		ShortDesc:   "Simula pressão de ~2 segundos",
		IdealFor:    "Funções especiais, reset, pairing",
		Color:       color.NRGBA{R: 232, G: 90, B: 122, A: 255},
	},
}

// Global language (default English, can be changed)
var currentLang Language = LangEnglish
var translations = map[Language]map[string]string{}

func loadTranslations(baseDir string) error {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return err
	}
	translations = map[Language]map[string]string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(baseDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		langMap := map[string]string{}
		if err := json.Unmarshal(data, &langMap); err != nil {
			return err
		}
		langCode := Language(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())))
		translations[langCode] = langMap
	}
	if len(translations) == 0 {
		return fmt.Errorf("no locale files found in %s", baseDir)
	}
	return nil
}

func t(key string) string {
	if words, ok := translations[currentLang]; ok {
		if value, ok := words[key]; ok {
			return value
		}
	}
	if words, ok := translations[LangEnglish]; ok {
		if value, ok := words[key]; ok {
			return value
		}
	}
	return key
}

type ButtonConfig struct {
	Index int    `json:"index"`
	Mode  string `json:"mode"`
}

type uiState struct {
	config           []ButtonConfig
	buttonStates     []bool // true = pressed, false = released
	ports            []string
	selectedPort     string
	joysticks        []string
	selectedJoystick string
	status           binding.String
	exportText       binding.String
	lastReadMs       int64
	jsFile           *os.File
	jsStopChan       chan bool
	jsUpdateCallback func([]bool)
	jsMonitoring     bool
	jsMutex          sync.Mutex
}

func normalizeMode(raw string) string {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "NORMAL", "ONE_SHOT", "TOGGLE", "LONG_PRESS":
		return strings.ToUpper(strings.TrimSpace(raw))
	default:
		return "NORMAL"
	}
}

func buildDefaultConfig() []ButtonConfig {
	cfg := make([]ButtonConfig, totalButtons)
	for i, mode := range defaultModes {
		cfg[i] = ButtonConfig{Index: i, Mode: normalizeMode(mode)}
	}
	return cfg
}

func appBaseDir() string {
	// tenta diretório do executável primeiro (funciona quando exe está em tools/config-gui ou distribuído)
	// fallback para diretório atual (funciona com `go run .`)
	if exe, err := os.Executable(); err == nil {
		if dir := filepath.Dir(exe); dir != "" && dir != "." {
			// verifica se o dir contém locales ou é utilizável
			if _, err := os.Stat(dir); err == nil {
				return dir
			}
		}
	}
	return "."
}

func configFilePath() string {
	// Para `go run` o executável é temporário em %TEMP%\go-build* — não usar esse dir
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		isTemp := strings.Contains(strings.ToLower(dir), "temp") || strings.Contains(dir, "go-build")
		if !isTemp && dir != "" && dir != "." {
			p := filepath.Join(dir, "buttonbox-config.json")
			// se já existe ao lado do exe, usa; se não, usa exe dir apenas se for o diretório
			// real do projeto (contém locales/go.mod), senão fallback para cwd
			if _, err := os.Stat(p); err == nil {
				return p
			}
			if _, err := os.Stat(filepath.Join(dir, "locales")); err == nil {
				return p
			}
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return p
			}
		}
	}
	return filepath.Join(".", "buttonbox-config.json")
}

func localesDir() string {
	candidates := []string{
		filepath.Join(".", "locales"),
		filepath.Join(appBaseDir(), "locales"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	// fallback para o primeiro (erro será reportado)
	return candidates[0]
}

func loadStoredConfig() []ButtonConfig {
	path := configFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return buildDefaultConfig()
	}

	var payload map[string][]ButtonConfig
	if err := json.Unmarshal(data, &payload); err != nil {
		return buildDefaultConfig()
	}

	cfg, ok := payload["buttons"]
	if !ok || len(cfg) != totalButtons {
		return buildDefaultConfig()
	}
	for i := range cfg {
		cfg[i].Index = i
		cfg[i].Mode = normalizeMode(cfg[i].Mode)
	}
	return cfg
}

func persistLocalConfig(cfg []ButtonConfig) error {
	payload := map[string][]ButtonConfig{"buttons": cfg}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFilePath(), data, 0o644)
}

func exportCPP(cfg []ButtonConfig) string {
	lines := []string{"static const ButtonMode BUTTON_MODES[BUTTON_COUNT] = {"}
	for i, item := range cfg {
		comma := ","
		if i == len(cfg)-1 {
			comma = ""
		}
		lines = append(lines, fmt.Sprintf("    /* %2d */ ButtonMode::%s%s", i, item.Mode, comma))
	}
	lines = append(lines, "};")
	return strings.Join(lines, "\n")
}

func listSerialPorts() []string {
	// go.bug.st/serial já implementa enumeração nativa cross-platform:
	// Windows via SetupAPI, Linux via /dev, macOS via IOKit.
	// É a única forma confiável — os.Stat("COMx") nunca funciona no Windows.
	ports, err := serial.GetPortsList()
	if err == nil {
		// GetPortsList pode retornar lista vazia sem erro quando não há portas
		dedup := make(map[string]struct{}, len(ports))
		uniq := make([]string, 0, len(ports))
		for _, p := range ports {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if _, ok := dedup[p]; !ok {
				dedup[p] = struct{}{}
				uniq = append(uniq, p)
			}
		}
		sort.Strings(uniq)
		return uniq
	}
	// fallback apenas se a API falhar (ex: permissão) — mantém compatibilidade Linux antiga
	candidates := map[string]struct{}{}
	entries, err := os.ReadDir("/dev")
	if err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if strings.Contains(name, "ttyUSB") || strings.Contains(name, "ttyACM") || strings.Contains(name, "cu.usb") || strings.Contains(name, "tty.usb") {
				candidates[filepath.Join("/dev", name)] = struct{}{}
			}
		}
	}
	fallback := make([]string, 0, len(candidates))
	for port := range candidates {
		fallback = append(fallback, port)
	}
	sort.Strings(fallback)
	return fallback
}

func listJoysticks() []string {
	joysticks := []string{}

	if runtime.GOOS == "windows" {
		// Windows: tenta detectar joysticks via existência de dispositivos HID
		// A abordagem mais simples é verificar se há dispositivos gamepad/joystick
		// O Fyne/Go não tem API nativa para isso sem bibliotecas externas
		// Então retornamos uma lista genérica que o usuário pode testar
		// Em produção, recomendo usar github.com/gamerathon/go-sdl2 ou similar
		candidates := []string{
			"Joystick 0 (Windows HID)",
			"Joystick 1 (Windows HID)",
			"Joystick 2 (Windows HID)",
			"Joystick 3 (Windows HID)",
		}
		// Verifica simplificada - em produção use biblioteca específica
		for _, c := range candidates {
			joysticks = append(joysticks, c)
		}
	} else {
		// Linux: /dev/input/js*
		entries, err := os.ReadDir("/dev/input")
		if err == nil {
			for _, entry := range entries {
				name := entry.Name()
				if strings.HasPrefix(name, "js") {
					path := filepath.Join("/dev/input", name)
					joysticks = append(joysticks, path)
				}
			}
		}
	}

	sort.Strings(joysticks)
	return joysticks
}

func readJoystickEvents(state *uiState, joystickPath string, winUpdateFunc func()) {
	var file *os.File
	var err error

	if runtime.GOOS == "windows" {
		// Windows: não suporta leitura direta via /dev/input
		// Em produção, use SDL2 ou similar para leitura nativa
		setStatus(state, t("windows_joystick_note"))
		state.jsMutex.Lock()
		state.jsMonitoring = true
		state.jsMutex.Unlock()
		
		// Simula estados aleatórios para demonstração da UI
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		
		for {
			state.jsMutex.Lock()
			if !state.jsMonitoring {
				state.jsMutex.Unlock()
				return
			}
			state.jsMutex.Unlock()
			
			select {
			case <-ticker.C:
				// Simula pressão aleatória de botões para demo
				state.jsMutex.Lock()
				for i := range state.buttonStates {
					state.buttonStates[i] = (i % 3) == (int(time.Now().Unix()%3))
				}
				state.jsMutex.Unlock()
				if winUpdateFunc != nil {
					winUpdateFunc()
				}
			default:
				time.Sleep(50 * time.Millisecond)
			}
		}
	}

	// Linux: leitura direta do dispositivo
	file, err = os.Open(joystickPath)
	if err != nil {
		setStatus(state, t("error_opening_joystick")+" "+joystickPath+": "+err.Error())
		state.jsMutex.Lock()
		state.jsMonitoring = false
		state.jsMutex.Unlock()
		return
	}
	defer file.Close()

	state.jsMutex.Lock()
	state.jsFile = file
	state.jsMonitoring = true
	state.jsMutex.Unlock()

	setStatus(state, t("monitoring")+" "+joystickPath)

	buf := make([]byte, 8)
	for {
		state.jsMutex.Lock()
		if !state.jsMonitoring {
			state.jsMutex.Unlock()
			return
		}
		state.jsMutex.Unlock()

		n, err := file.Read(buf)
		if err != nil {
			setStatus(state, t("error_reading_joystick")+": "+err.Error())
			state.jsMutex.Lock()
			state.jsMonitoring = false
			state.jsMutex.Unlock()
			return
		}
		if n < 8 {
			continue
		}

		// Parse joystick event: time(4) + value(2) + type(1) + number(1)
		jsType := buf[6]
		jsNumber := buf[7]

		// jsType & 0x80 = initialization, we ignore it
		// jsType & 0x01 = button event
		// jsType & 0x02 = axis event

		if (jsType & 0x01) != 0 { // Button event
			if jsNumber < totalButtons {
				state.jsMutex.Lock()
				state.buttonStates[jsNumber] = (buf[4] != 0)
				state.jsMutex.Unlock()

				// Trigger UI update
				if winUpdateFunc != nil {
					winUpdateFunc()
				}
			}
		}
	}
}

func applyButtonConfigToSerial(portName string, cfg []ButtonConfig) error {
	if strings.TrimSpace(portName) == "" {
		return fmt.Errorf("serial port not informed")
	}
	p, err := serial.Open(portName, &serial.Mode{BaudRate: 115200})
	if err != nil {
		return err
	}
	defer p.Close()
	for _, item := range cfg {
		line := fmt.Sprintf("%d=%s\n", item.Index, normalizeMode(item.Mode))
		if _, err := p.Write([]byte(line)); err != nil {
			return err
		}
		time.Sleep(30 * time.Millisecond)
	}
	return nil
}

func readButtonConfigFromSerial(portName string) ([]ButtonConfig, error) {
	if strings.TrimSpace(portName) == "" {
		return nil, fmt.Errorf("serial port not informed")
	}
	p, err := serial.Open(portName, &serial.Mode{BaudRate: 115200})
	if err != nil {
		return nil, err
	}
	defer p.Close()
	if _, err := p.Write([]byte("GET\n")); err != nil {
		return nil, err
	}
	p.SetReadTimeout(200 * time.Millisecond)
	result := make([]ButtonConfig, 0, totalButtons)
	for len(result) < totalButtons {
		buf := make([]byte, 128)
		n, err := p.Read(buf)
		if err != nil {
			if strings.Contains(err.Error(), "timeout") {
				break
			}
			return nil, err
		}
		if n <= 0 {
			continue
		}
		payload := string(buf[:n])
		for _, line := range strings.Split(payload, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			idx, err := strconv.Atoi(parts[0])
			if err != nil || idx < 0 || idx >= totalButtons {
				continue
			}
			result = append(result, ButtonConfig{Index: idx, Mode: normalizeMode(parts[1])})
		}
	}
	if len(result) != totalButtons {
		return nil, fmt.Errorf("device returned incomplete configuration")
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Index < result[j].Index })
	return result, nil
}

func readButtonStatesFromSerial(portName string) ([]bool, error) {
	if strings.TrimSpace(portName) == "" {
		return nil, fmt.Errorf("serial port not informed")
	}
	p, err := serial.Open(portName, &serial.Mode{BaudRate: 115200})
	if err != nil {
		return nil, err
	}
	defer p.Close()
	if _, err := p.Write([]byte("GETSTATES\n")); err != nil {
		return nil, err
	}
	p.SetReadTimeout(200 * time.Millisecond)
	result := make([]bool, totalButtons)
	for i := 0; i < totalButtons; i++ {
		result[i] = false
	}
	bytesRead := 0
	for bytesRead < totalButtons {
		buf := make([]byte, 128)
		n, err := p.Read(buf)
		if err != nil {
			if strings.Contains(err.Error(), "timeout") {
				break
			}
			return nil, err
		}
		if n <= 0 {
			continue
		}
		payload := string(buf[:n])
		for _, line := range strings.Split(payload, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			idx, err := strconv.Atoi(parts[0])
			if err != nil || idx < 0 || idx >= totalButtons {
				continue
			}
			result[idx] = parts[1] == "1"
			bytesRead++
		}
	}
	return result, nil
}

func setStatus(state *uiState, text string) {
	_ = state.status.Set(text)
}

func setExport(state *uiState) {
	_ = state.exportText.Set(exportCPP(state.config))
}

func newUIState() *uiState {
	state := &uiState{
		config:           loadStoredConfig(),
		buttonStates:     make([]bool, totalButtons),
		ports:            listSerialPorts(),
		joysticks:        listJoysticks(),
		status:           binding.NewString(),
		exportText:       binding.NewString(),
		jsStopChan:       make(chan bool, 1),
		jsMonitoring:     false,
		jsMutex:          sync.Mutex{},
		jsUpdateCallback: nil,
	}
	_ = state.status.Set(t("ready"))
	setExport(state)
	if len(state.ports) > 0 {
		state.selectedPort = state.ports[0]
	}
	if len(state.joysticks) > 0 {
		state.selectedJoystick = state.joysticks[0]
	}
	return state
}

// ---- Design Tokens: ButtonBox Cockpit ----
var (
	colBg         = color.NRGBA{R: 11, G: 14, B: 17, A: 255} // #0B0E11 cockpit
	colPanel      = color.NRGBA{R: 20, G: 26, B: 31, A: 255} // #141A1F
	colPanelEdge  = color.NRGBA{R: 35, G: 47, B: 55, A: 255} // #232F37
	colSteel      = color.NRGBA{R: 138, G: 150, B: 163, A: 255}
	colInk        = color.NRGBA{R: 232, G: 234, B: 227, A: 255}
	colInkMuted   = color.NRGBA{R: 138, G: 150, B: 163, A: 255}
	colCodeBg     = color.NRGBA{R: 14, G: 19, B: 23, A: 255}
	colAccent     = color.NRGBA{R: 255, G: 138, B: 24, A: 255} // amber signature
	colNormal     = color.NRGBA{R: 74, G: 155, B: 224, A: 255}
	colOneShot    = color.NRGBA{R: 232, G: 164, B: 74, A: 255}
	colToggle     = color.NRGBA{R: 125, G: 216, B: 125, A: 255}
	colLongPress  = color.NRGBA{R: 232, G: 90, B: 122, A: 255}
	colLedOff     = color.NRGBA{R: 45, G: 58, B: 68, A: 255}
	colLedOn      = color.NRGBA{R: 255, G: 77, B: 77, A: 255}
	colLedGlowOff = color.NRGBA{R: 45, G: 58, B: 68, A: 60}
	colLedGlowOn  = color.NRGBA{R: 255, G: 77, B: 77, A: 90}
)

func getModeColor(mode string) color.Color {
	switch normalizeMode(mode) {
	case "NORMAL":
		return colNormal
	case "ONE_SHOT":
		return colOneShot
	case "TOGGLE":
		return colToggle
	case "LONG_PRESS":
		return colLongPress
	default:
		return colSteel
	}
}

func getModeLabel(mode string) string {
	switch normalizeMode(mode) {
	case "NORMAL":
		return "NORMAL"
	case "ONE_SHOT":
		return "ONE-SHOT"
	case "TOGGLE":
		return "TOGGLE"
	case "LONG_PRESS":
		return "LONG-PRESS"
	default:
		return mode
	}
}

func buildGrid(state *uiState) *fyne.Container {
	cards := make([]fyne.CanvasObject, 0, totalButtons)
	for i, item := range state.config {
		idx := i
		mode := normalizeMode(item.Mode)
		isPressed := i < len(state.buttonStates) && state.buttonStates[i]
		accent := getModeColor(mode)

		// hairline top signature
		accentBar := canvas.NewRectangle(accent)
		accentBar.SetMinSize(fyne.NewSize(0, 3))

		// number badge + LED ring
		numText := canvas.NewText(fmt.Sprintf("%02d", item.Index), colInk)
		numText.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
		numText.TextSize = 18
		numText.Alignment = fyne.TextAlignCenter

		ledOuter := canvas.NewCircle(colLedGlowOff)
		ledInner := canvas.NewCircle(colLedOff)
		ledInner.StrokeColor = colPanelEdge
		ledInner.StrokeWidth = 1.5
		if isPressed {
			ledOuter.FillColor = colLedGlowOn
			ledInner.FillColor = colLedOn
			ledInner.StrokeColor = color.NRGBA{R: 80, G: 20, B: 20, A: 255}
		}
		ledStack := container.NewStack(ledOuter, container.NewCenter(ledInner), container.NewCenter(numText))
		ledStack.Resize(fyne.NewSize(52, 52))
		ledBox := container.NewCenter(ledStack)
		ledBox.Resize(fyne.NewSize(56, 56))

		// Mode indicator with color bar
		modeLabel := canvas.NewText(getModeLabel(mode), colSteel)
		modeLabel.TextSize = 9
		modeLabel.TextStyle = fyne.TextStyle{Bold: true}
		modeLabel.Alignment = fyne.TextAlignCenter

		// Color bar under mode label
		modeColorBar := canvas.NewRectangle(accent)
		modeColorBar.SetMinSize(fyne.NewSize(0, 2))
		modeColorBar.CornerRadius = 1

		// Select — keep native but with placeholder fix
		sel := widget.NewSelect([]string{"NORMAL", "ONE_SHOT", "TOGGLE", "LONG_PRESS"}, func(m string) {
			state.config[idx].Mode = normalizeMode(m)
			_ = persistLocalConfig(state.config)
			setExport(state)
			setStatus(state, t("config_saved"))
		})
		sel.SetSelected(mode)
		sel.PlaceHolder = "Modo"

		// status dot under select
		dot := canvas.NewCircle(accent)
		dot.Resize(fyne.NewSize(8, 8))
		dotBox := container.NewCenter(dot)

		// card background + border
		bg := canvas.NewRectangle(colPanel)
		bg.StrokeColor = colPanelEdge
		bg.StrokeWidth = 1
		bg.CornerRadius = 10

		inner := container.NewVBox(
			accentBar,
			layout.NewSpacer(),
			ledBox,
			modeLabel,
			modeColorBar,
			sel,
			dotBox,
			layout.NewSpacer(),
		)
		innerPad := container.NewPadded(inner)

		cardStack := container.NewStack(bg, innerPad)
		cardStack.Resize(fyne.NewSize(0, 0))
		wrapper := container.NewPadded(cardStack)
		cardWithSize := container.NewStack(wrapper)
		cardWithSize.Resize(fyne.NewSize(160, 148))
		cards = append(cards, container.NewPadded(cardWithSize))
	}
	grid := container.NewGridWithColumns(4, cards...)
	scrollContent := container.NewPadded(grid)
	holder := container.NewWithoutLayout(scrollContent)
	return container.NewStack(holder)
}

func buildLegend() fyne.CanvasObject {
	// LED state row
	ledOffOuter := canvas.NewCircle(colLedGlowOff)
	ledOffInner := canvas.NewCircle(colLedOff)
	ledOff := container.NewStack(ledOffOuter, container.NewCenter(ledOffInner))
	ledOff.Resize(fyne.NewSize(14, 14))
	ledOnOuter := canvas.NewCircle(colLedGlowOn)
	ledOnInner := canvas.NewCircle(colLedOn)
	ledOn := container.NewStack(ledOnOuter, container.NewCenter(ledOnInner))
	ledOn.Resize(fyne.NewSize(14, 14))

	ledTitle := canvas.NewText(strings.ToUpper(t("led_state")), colSteel)
	ledTitle.TextSize = 10
	ledTitle.TextStyle = fyne.TextStyle{Bold: true}
	ledRow := container.NewVBox(
		ledTitle,
		container.NewHBox(ledOff, widget.NewLabel(t("not_pressed"))),
		container.NewHBox(ledOn, widget.NewLabel(t("pressed"))),
	)

	modesTitle := canvas.NewText(strings.ToUpper(t("modes")), colSteel)
	modesTitle.TextSize = 10
	modesTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Mode descriptions with detailed info - now includes ideal for
	modeDescs := container.NewVBox()
	for _, mode := range modeInfos {
		sw := canvas.NewRectangle(mode.Color)
		sw.SetMinSize(fyne.NewSize(16, 16))
		sw.CornerRadius = 3
		
		nameLabel := canvas.NewText(mode.Name, colInk)
		nameLabel.TextSize = 11
		nameLabel.TextStyle = fyne.TextStyle{Bold: true}
		
		descLabel := widget.NewLabel(t(mode.Key + "_desc"))
		descLabel.Wrapping = fyne.TextWrapWord
		
		idealLabel := widget.NewLabel("💡 " + t(mode.Key + "_ideal"))
		idealLabel.TextStyle = fyne.TextStyle{Italic: true}
		idealLabel.Wrapping = fyne.TextWrapWord
		
		modeCol := container.NewVBox(
			container.NewHBox(sw, nameLabel),
			descLabel,
			idealLabel,
		)
		modeDescs.Add(modeCol)
	}

	title := canvas.NewText("BUTTONBOX", colInkMuted)
	title.TextSize = 9
	title.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	title.Alignment = fyne.TextAlignLeading

	legendCardBg := canvas.NewRectangle(colPanel)
	legendCardBg.StrokeColor = colPanelEdge
	legendCardBg.StrokeWidth = 1
	legendCardBg.CornerRadius = 12

	infoBtn := widget.NewButtonWithIcon(t("mode_help"), theme.HelpIcon(), func() {
		showModeHelpDialog()
	})

	inner := container.NewVBox(
		title,
		canvas.NewRectangle(color.NRGBA{R: 35, G: 47, B: 55, A: 255}),
		container.NewHBox(
			widget.NewLabelWithStyle(t("legend"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			layout.NewSpacer(),
			infoBtn,
		),
		ledRow,
		layout.NewSpacer(),
		modesTitle,
		modeDescs,
	)
	pad := container.NewPadded(inner)
	return container.NewStack(legendCardBg, pad)
}

func showModeHelpDialog() {
	helpText := strings.Builder{}
	helpText.WriteString("# " + t("button_modes_title") + "\n\n")
	
	for _, mode := range modeInfos {
		helpText.WriteString("## **" + mode.Name + "**\n")
		helpText.WriteString(t(mode.Key+"_desc") + "\n\n")
		helpText.WriteString("💡 *" + t(mode.Key+"_ideal") + "*\n\n")
		helpText.WriteString("---\n\n")
	}
	
	helpText.WriteString("_" + t("config_restored") + "_")

	// Use simple label instead of markdown widget (not available in Fyne 2.x)
	content := widget.NewLabel(helpText.String())
	
	dialog.ShowCustomConfirm(
		t("button_modes_title"),
		t("close"),
		t("ok"),
		content,
		func(confirmed bool) {},
		fyne.CurrentApp().Driver().AllWindows()[0],
	)
}

func buildHeader(state *uiState, win fyne.Window) fyne.CanvasObject {
	bg := canvas.NewRectangle(color.NRGBA{R: 8, G: 11, B: 14, A: 255})
	bg.CornerRadius = 0

	dot := canvas.NewCircle(colAccent)
	dot.Resize(fyne.NewSize(8, 8))
	title := canvas.NewText("ButtonBox — Button Mode Configuration", colInk)
	title.TextSize = 13
	title.TextStyle = fyne.TextStyle{Bold: true}
	sub := canvas.NewText("16× MATRIX  •  ESP32 / LEONARDO / PRO MICRO", colSteel)
	sub.TextSize = 9
	sub.TextStyle = fyne.TextStyle{Monospace: true}
	titleBox := container.NewVBox(title, sub)
	left := container.NewHBox(dot, titleBox)

	// lang select — no recursion
	langSel := widget.NewSelect([]string{"English", "Português (BR)"}, nil)
	if currentLang == LangEnglish {
		langSel.SetSelected("English")
	} else {
		langSel.SetSelected("Português (BR)")
	}
	langSel.OnChanged = func(v string) {
		if v == "English" {
			currentLang = LangEnglish
		} else {
			currentLang = LangPtBr
		}
		win.SetContent(buildUI(state, win))
	}
	langLabel := canvas.NewText("LANG", colSteel)
	langLabel.TextSize = 9
	langLabel.TextStyle = fyne.TextStyle{Bold: true}
	right := container.NewHBox(langLabel, langSel)

	headerInner := container.NewHBox(left, layout.NewSpacer(), right)
	pad := container.NewPadded(headerInner)
	return container.NewStack(bg, pad)
}

func buildDeck(state *uiState, win fyne.Window) fyne.CanvasObject {
	ports := state.ports
	if len(ports) == 0 {
		ports = []string{t("no_ports_detected")}
	}
	portSel := widget.NewSelect(ports, func(v string) { state.selectedPort = v })
	if state.selectedPort != "" {
		portSel.SetSelected(state.selectedPort)
	} else if len(ports) > 0 {
		portSel.SetSelected(ports[0])
	}

	joysticks := state.joysticks
	if len(joysticks) == 0 {
		joysticks = []string{t("no_joysticks_detected")}
	}
	joySel := widget.NewSelect(joysticks, func(v string) { state.selectedJoystick = v })
	if state.selectedJoystick != "" {
		joySel.SetSelected(state.selectedJoystick)
	} else if len(joysticks) > 0 {
		joySel.SetSelected(joysticks[0])
	}

	// styled buttons
	readBtn := widget.NewButtonWithIcon(t("read_from_board"), theme.DownloadIcon(), func() {
		if state.selectedPort == "" || state.selectedPort == t("no_ports_detected") {
			setStatus(state, t("select_serial_port"))
			return
		}
		cfg, err := readButtonConfigFromSerial(state.selectedPort)
		if err != nil {
			setStatus(state, t("error_reading_board")+" "+err.Error())
			return
		}
		state.config = cfg
		_ = persistLocalConfig(state.config)
		setExport(state)
		setStatus(state, t("config_read_board"))
		win.SetContent(buildUI(state, win))
	})
	applyBtn := widget.NewButtonWithIcon(t("apply_to_board"), theme.UploadIcon(), func() {
		if state.selectedPort == "" || state.selectedPort == t("no_ports_detected") {
			setStatus(state, t("select_serial_port"))
			return
		}
		if err := applyButtonConfigToSerial(state.selectedPort, state.config); err != nil {
			setStatus(state, t("error_applying_board")+" "+err.Error())
			return
		}
		_ = persistLocalConfig(state.config)
		setStatus(state, t("config_applied")+" "+state.selectedPort+".")
		setExport(state)
	})
	applyBtn.Importance = widget.HighImportance
	resetBtn := widget.NewButtonWithIcon(t("reset"), theme.ViewRefreshIcon(), func() {
		state.config = buildDefaultConfig()
		_ = persistLocalConfig(state.config)
		setExport(state)
		setStatus(state, t("config_restored"))
		win.SetContent(buildUI(state, win))
	})

	var monBtn *widget.Button
	monBtn = widget.NewButtonWithIcon(t("monitor_joystick"), theme.MediaPlayIcon(), func() {
		state.jsMutex.Lock()
		isMon := state.jsMonitoring
		state.jsMutex.Unlock()
		if isMon {
			state.jsMutex.Lock()
			state.jsMonitoring = false
			state.jsMutex.Unlock()
			if state.jsFile != nil {
				state.jsFile.Close()
				state.jsFile = nil
			}
			setStatus(state, t("monitoring_stopped"))
			monBtn.SetText(t("monitor_joystick"))
			monBtn.SetIcon(theme.MediaPlayIcon())
			return
		}
		if state.selectedJoystick == "" || state.selectedJoystick == t("no_joysticks_detected") {
			setStatus(state, t("select_joystick"))
			return
		}
		go readJoystickEvents(state, state.selectedJoystick, func() { win.SetContent(buildUI(state, win)) })
		monBtn.SetText(t("stop_monitoring"))
		monBtn.SetIcon(theme.MediaStopIcon())
	})

	// deck cards
	bg1 := canvas.NewRectangle(colPanel)
	bg1.StrokeColor = colPanelEdge
	bg1.StrokeWidth = 1
	bg1.CornerRadius = 10
	serialTitle := canvas.NewText("PORTA SERIAL", colSteel)
	serialTitle.TextSize = 9
	serialTitle.TextStyle = fyne.TextStyle{Bold: true}
	deck1Inner := container.NewVBox(
		serialTitle,
		portSel,
		container.NewHBox(readBtn, applyBtn, resetBtn),
	)
	deck1 := container.NewStack(bg1, container.NewPadded(deck1Inner))

	bg2 := canvas.NewRectangle(colPanel)
	bg2.StrokeColor = colPanelEdge
	bg2.StrokeWidth = 1
	bg2.CornerRadius = 10
	joyTitle := canvas.NewText("JOYSTICK", colSteel)
	joyTitle.TextSize = 9
	joyTitle.TextStyle = fyne.TextStyle{Bold: true}
	deck2Inner := container.NewVBox(
		joyTitle,
		joySel,
		monBtn,
	)
	deck2 := container.NewStack(bg2, container.NewPadded(deck2Inner))

	return container.NewGridWithColumns(2, deck1, deck2)
}

func buildUI(state *uiState, win fyne.Window) fyne.CanvasObject {
	statusLabel := widget.NewLabelWithData(state.status)
	statusLabel.Wrapping = fyne.TextWrapWord
	statusLabel.TextStyle = fyne.TextStyle{Italic: true}

	exportEntry := widget.NewEntryWithData(state.exportText)
	exportEntry.MultiLine = true
	exportEntry.Wrapping = fyne.TextWrapOff
	exportEntry.Disable()
	// mono code bg
	codeBg := canvas.NewRectangle(colCodeBg)
	codeBg.StrokeColor = colPanelEdge
	codeBg.StrokeWidth = 1
	codeBg.CornerRadius = 8
	codeStack := container.NewStack(codeBg, container.NewPadded(exportEntry))
	codeStack.Resize(fyne.NewSize(0, 110))

	exportTitle := canvas.NewText("C++ EXPORT", colSteel)
	exportTitle.TextSize = 10
	exportTitle.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	copyBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		if txt, err := state.exportText.Get(); err == nil {
			win.Clipboard().SetContent(txt)
			setStatus(state, "Copiado")
		}
	})
	copyBtn.Importance = widget.LowImportance
	exportHeader := container.NewHBox(exportTitle, layout.NewSpacer(), copyBtn)
	exportCardBg := canvas.NewRectangle(colPanel)
	exportCardBg.StrokeColor = colPanelEdge
	exportCardBg.StrokeWidth = 1
	exportCardBg.CornerRadius = 10
	exportCard := container.NewStack(exportCardBg, container.NewPadded(container.NewVBox(exportHeader, codeStack, statusLabel)))

	grid := buildGrid(state)
	legend := buildLegend()
	deck := buildDeck(state, win)
	header := buildHeader(state, win)

	// main layout: header / deck / grid+legend / export
	center := container.NewVBox(deck, container.NewHBox(container.NewPadded(grid), container.NewPadded(legend)))

	// outer bg
	outerBg := canvas.NewRectangle(colBg)
	content := container.NewVBox(header, center, exportCard)
	padded := container.NewPadded(content)
	scroll := container.NewVScroll(container.NewStack(outerBg, padded))
	scroll.Direction = container.ScrollBoth
	return container.NewStack(outerBg, scroll)
}

func main() {
	if err := loadTranslations(localesDir()); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load translations: %v\n", err)
		os.Exit(1)
	}
	if _, ok := translations[LangEnglish]; !ok {
		fmt.Fprintln(os.Stderr, "missing English locale")
		os.Exit(1)
	}

	a := app.New()
	// dark cockpit theme
	a.Settings().SetTheme(theme.DarkTheme())
	state := newUIState()
	w := a.NewWindow("ButtonBox — Button Mode Configuration")
	w.Resize(fyne.NewSize(1180, 860))
	w.SetContent(buildUI(state, w))
	w.ShowAndRun()
}
