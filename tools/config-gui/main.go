package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"
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
	hid "github.com/karalabe/hid"
	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
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
		Key:         "mode_two_shot",
		Name:        "TWO-SHOT",
		Description: "Fires once on press and again on release (two pulses). Useful for gear shifts, blinkers, or any action that needs press + release.",
		ShortDesc:   "Dispara ao pressionar e ao soltar",
		IdealFor:    "Marcha, seta, ações press+release",
		Color:       color.NRGBA{R: 155, G: 125, B: 232, A: 255},
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

var hidPathByDisplay = map[string]string{}
var hidDisplayByPath = map[string]string{}
var hidSerialByDisplay = map[string]string{}

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
	// refs para refresh incremental sem rebuild (evita tremor) — agora barra horizontal
	gridBars []*canvas.Rectangle
}

func normalizeMode(raw string) string {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "NORMAL", "ONE_SHOT", "TWO_SHOT", "TWO_WAY_SHOT", "TWO_WAY", "DUAL_SHOT", "TOGGLE", "LONG_PRESS":
		m := strings.ToUpper(strings.TrimSpace(raw))
		// canonicaliza aliases para TWO_SHOT
		if m == "TWO_WAY_SHOT" || m == "TWO_WAY" || m == "DUAL_SHOT" {
			return "TWO_SHOT"
		}
		return m
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
	// sem cache local — sempre inicia com defaults, leitura real vem do dispositivo via Read
	return buildDefaultConfig()
}

func persistLocalConfig(cfg []ButtonConfig) error {
	// sem cache local — mantém em memória e no dispositivo via Apply
	_ = cfg
	return nil
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
		// Windows: usa winmm joyGetDevCaps (sem hidapi) — mostra nome real e ID
		if lst, m := winmmListJoysticks(); len(lst) > 0 {
			// winmmListJoysticks já preenche hidPathByDisplay
			_ = m
			return lst
		}
		// fallback HID se winmm não achar (ex: ButtonBox BLE que não é winmm)
		hidPathByDisplay = map[string]string{}
		hidDisplayByPath = map[string]string{}
		seen := map[string]bool{}
		addWin := func(d hid.DeviceInfo) {
			name := d.Product
			if name == "" {
				name = d.Manufacturer
				if name == "" {
					name = "HID Joystick"
				}
			}
			if name == "Arduino Leonardo" {
				name = "ButtonBox"
			}
			display := fmt.Sprintf("%s (%04x:%04x)", name, d.VendorID, d.ProductID)
			if seen[display] {
				return
			}
			seen[display] = true
			hidPathByDisplay[display] = d.Path
			hidDisplayByPath[d.Path] = display
			joysticks = append(joysticks, display)
		}
		for _, d := range hid.Enumerate(0, 0) {
			if d.UsagePage == 0x01 && (d.Usage == 0x04 || d.Usage == 0x05 || d.Usage == 0x08) {
				addWin(d)
			}
		}
		if len(joysticks) == 0 {
			for _, d := range hid.Enumerate(0, 0) {
				lowerProd := strings.ToLower(d.Product)
				lowerMan := strings.ToLower(d.Manufacturer)
				if strings.Contains(lowerProd, "joystick") || strings.Contains(lowerProd, "gamepad") || strings.Contains(lowerProd, "buttonbox") || strings.Contains(lowerProd, "controller") ||
					strings.Contains(lowerMan, "buttonbox") || strings.Contains(lowerProd, "leonardo") || strings.Contains(lowerProd, "pro micro") {
					addWin(d)
				}
			}
		}
		sort.Strings(joysticks)
		return joysticks
	}
	// Linux: lista /dev/input/js* com nome e vendor:product reais
	entries, err := os.ReadDir("/dev/input")
	if err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if !strings.HasPrefix(name, "js") {
				continue
			}
			devPath := filepath.Join("/dev/input", name)
			display := devPath
			sysBase := filepath.Join("/sys/class/input", name, "device")
			if data, err := os.ReadFile(filepath.Join(sysBase, "name")); err == nil {
				devName := strings.TrimSpace(string(data))
				if devName != "" {
					vendor, _ := os.ReadFile(filepath.Join(sysBase, "id", "vendor"))
					product, _ := os.ReadFile(filepath.Join(sysBase, "id", "product"))
					v := strings.TrimSpace(string(vendor))
					p := strings.TrimSpace(string(product))
					if v != "" && p != "" {
						display = fmt.Sprintf("%s — %s (%s:%s)", devPath, devName, v, p)
					} else {
						display = fmt.Sprintf("%s — %s", devPath, devName)
					}
				}
			}
			joysticks = append(joysticks, display)
		}
	}
	sort.Strings(joysticks)
	return joysticks
}

func parseVIDPID(display string) (uint16, uint16) {
	// display like "ButtonBox (2341:8036) [ID 0]" -> extrai 2341:8036
	start := strings.LastIndex(display, "(")
	end := strings.LastIndex(display, ")")
	if start >= 0 && end > start {
		inner := display[start+1 : end]
		parts := strings.Split(inner, ":")
		if len(parts) == 2 {
			var v, p uint64
			fmt.Sscanf(parts[0], "%04x", &v)
			fmt.Sscanf(parts[1], "%04x", &p)
			// fallback decimal
			if v == 0 {
				fmt.Sscanf(parts[0], "%d", &v)
			}
			if p == 0 {
				fmt.Sscanf(parts[1], "%d", &p)
			}
			return uint16(v), uint16(p)
		}
	}
	return 0, 0
}

var cachedSerialPort string
var cachedSerialPortTime time.Time
var serialMu sync.Mutex

func isButtonBoxPort(port string) bool {
	serialMu.Lock()
	defer serialMu.Unlock()
	if p, err := serial.Open(port, &serial.Mode{BaudRate: 115200}); err == nil {
		defer p.Close()
		_ = p.SetDTR(false)
		time.Sleep(200 * time.Millisecond)
		_ = p.ResetInputBuffer()
		p.SetReadTimeout(200 * time.Millisecond)
		drain := make([]byte, 512)
		for i := 0; i < 2; i++ {
			n, _ := p.Read(drain)
			if n > 0 {
				log.Printf("isButtonBoxPort %s dreno %d %q", port, n, string(drain[:n]))
			}
			if n == 0 {
				break
			}
		}
		if _, err := p.Write([]byte("GET\n")); err != nil {
			log.Printf("isButtonBoxPort %s write FAIL %v", port, err)
			return false
		}
		log.Printf("isButtonBoxPort %s GET enviado", port)
		var acc string
		buf := make([]byte, 512)
		deadline := time.Now().Add(1200 * time.Millisecond)
		for time.Now().Before(deadline) {
			n, _ := p.Read(buf)
			if n > 0 {
				chunk := string(buf[:n])
				acc += chunk
				log.Printf("isButtonBoxPort %s chunk %d %q acc=%q", port, n, chunk, acc)
				if strings.Count(acc, "=") >= 1 {
					break
				}
			} else {
				time.Sleep(50 * time.Millisecond)
			}
		}
		s := acc
		ok := strings.Count(s, "=") >= 1
		log.Printf("isButtonBoxPort %s final %q ok=%v", port, s, ok)
		return ok
	}
	log.Printf("isButtonBoxPort %s open FAIL", port)
	return false
}

func resolveSerialPort(state *uiState) string {
	log.Printf("resolveSerialPort: ports=%v selectedPort=%q selectedJoystick=%q", state.ports, state.selectedPort, state.selectedJoystick)
	// inverte cache se não for mais ButtonBox (ex: COM1 fantasma)
	if cachedSerialPort != "" && time.Since(cachedSerialPortTime) < 5*time.Second {
		for _, p := range state.ports {
			if p == cachedSerialPort {
				if isButtonBoxPort(p) {
					log.Printf("resolveSerialPort: usando cache %s", p)
					return p
				}
				log.Printf("resolveSerialPort: cache %s não é ButtonBox, invalidando", p)
				cachedSerialPort = ""
				break
			}
		}
	}
	// atualiza lista de portas (pode ter plugado/desplugado)
	state.ports = listSerialPorts()
	log.Printf("resolveSerialPort: lista atualizada %v", state.ports)
	if len(state.ports) == 0 {
		log.Printf("resolveSerialPort: nenhuma porta")
		return ""
	}
	// 1. se usuário selecionou manualmente e ainda existe E é ButtonBox, usa
	if state.selectedPort != "" {
		for _, p := range state.ports {
			if p == state.selectedPort && isButtonBoxPort(p) {
				cachedSerialPort = p
				cachedSerialPortTime = time.Now()
				log.Printf("resolveSerialPort: usando selectedPort %s", p)
				return p
			}
		}
		log.Printf("resolveSerialPort: selectedPort %q não é ButtonBox, ignorando", state.selectedPort)
	}
	// 2. tenta match direto por VID:PID do joystick selecionado (não adivinha)
	if state.selectedJoystick != "" {
		vid, pid := parseVIDPID(state.selectedJoystick)
		if vid != 0 || pid != 0 {
			if details, err := enumerator.GetDetailedPortsList(); err == nil {
				for _, d := range details {
					if strings.EqualFold(d.VID, fmt.Sprintf("%04x", vid)) && strings.EqualFold(d.PID, fmt.Sprintf("%04x", pid)) {
						log.Printf("resolveSerialPort: match VID:PID %04x:%04x -> %s", vid, pid, d.Name)
						cachedSerialPort = d.Name
						cachedSerialPortTime = time.Now()
						return d.Name
					}
				}
				// fallback case-insensitive sem zero pad
				for _, d := range details {
					var dv, dp uint64
					fmt.Sscanf(d.VID, "%x", &dv)
					fmt.Sscanf(d.PID, "%x", &dp)
					if uint16(dv) == vid && uint16(dp) == pid {
						log.Printf("resolveSerialPort: match VID:PID %04x:%04x -> %s (2)", vid, pid, d.Name)
						cachedSerialPort = d.Name
						cachedSerialPortTime = time.Now()
						return d.Name
					}
				}
			}
		}
	}
	// 3. brute-force só se VID:PID não achou
	for _, port := range state.ports {
		log.Printf("resolveSerialPort: testando %s", port)
		if isButtonBoxPort(port) {
			log.Printf("resolveSerialPort: %s parece ButtonBox", port)
			cachedSerialPort = port
			cachedSerialPortTime = time.Now()
			return port
		}
		log.Printf("resolveSerialPort: %s sem resposta ButtonBox", port)
	}
	// 3. fallback: se só há uma porta, usa ela
	if len(state.ports) == 1 {
		cachedSerialPort = state.ports[0]
		cachedSerialPortTime = time.Now()
		return state.ports[0]
	}
	// 4. último fallback: primeira porta
	cachedSerialPort = state.ports[0]
	cachedSerialPortTime = time.Now()
	return state.ports[0]
}

func readJoystickEvents(state *uiState, joystickPath string, winUpdateFunc func()) {
	// Windows display agora é "ButtonBox (2341:8036)" — resolve para Path real via mapa
	hidPath := hidPathByDisplay[joystickPath]
	if hidPath == "" {
		// compat: formato antigo "PATH — Nome (vid:pid)" ou Linux "/dev/input/js0 — Nome"
		tmp := joystickPath
		if idx := strings.Index(joystickPath, " —"); idx > 0 {
			tmp = strings.TrimSpace(joystickPath[:idx])
		}
		if alt, ok := hidPathByDisplay[tmp]; ok {
			hidPath = alt
		} else {
			hidPath = tmp
		}
	}
	realPath := hidPath
	if realPath == "" {
		realPath = joystickPath
	}

	if runtime.GOOS == "windows" {
		// Windows: usa winmm joyGetPosEx (não hidapi cru) — leitura correta sem offset
		joyID := winmmParseJoyID(joystickPath)
		state.jsMutex.Lock()
		state.jsMonitoring = true
		state.jsMutex.Unlock()
		setStatus(state, t("monitoring")+" "+joystickPath)
		for {
			state.jsMutex.Lock()
			if !state.jsMonitoring {
				state.jsMutex.Unlock()
				return
			}
			state.jsMutex.Unlock()
			buttons, err := winmmReadButtons(joyID)
			if err != nil {
				// tenta HID como fallback se winmm falhar (ex: BLE que não é winmm)
				// fallback para HID já foi enumerado, mas winmm deve cobrir 99%
				setStatus(state, t("error_reading_joystick")+": "+err.Error())
				// não sai, tenta novamente em 100ms
				time.Sleep(100 * time.Millisecond)
				continue
			}
			for i := 0; i < totalButtons; i++ {
				pressed := (buttons>>uint(i))&1 == 1
				state.jsMutex.Lock()
				state.buttonStates[i] = pressed
				state.jsMutex.Unlock()
				if i < len(state.gridBars) {
					if pressed {
						state.gridBars[i].FillColor = colLedOn
					} else {
						state.gridBars[i].FillColor = colLedOff
					}
					state.gridBars[i].Refresh()
				}
			}
			time.Sleep(16 * time.Millisecond)
		}
	}
	var file *os.File
	var err error

	// Linux: leitura direta do dispositivo
	file, err = os.Open(realPath)
	if err != nil {
		setStatus(state, t("error_opening_joystick")+" "+realPath+": "+err.Error())
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
				pressed := buf[4] != 0
				state.jsMutex.Lock()
				state.buttonStates[jsNumber] = pressed
				state.jsMutex.Unlock()

				// barra horizontal — sem círculo, evita tremor
				if int(jsNumber) < len(state.gridBars) {
					if pressed {
						state.gridBars[jsNumber].FillColor = colLedOn
					} else {
						state.gridBars[jsNumber].FillColor = colLedOff
					}
					state.gridBars[jsNumber].Refresh()
				}
			}
		}
	}
}

func applyButtonConfigToSerial(portName string, cfg []ButtonConfig) error {
	if strings.TrimSpace(portName) == "" {
		return fmt.Errorf("serial port not informed")
	}
	serialMu.Lock()
	log.Printf("applyButtonConfigToSerial: %s com %d modos", portName, len(cfg))
	p, err := serial.Open(portName, &serial.Mode{BaudRate: 115200})
	if err != nil {
		serialMu.Unlock()
		return err
	}
	_ = p.SetDTR(false)
	time.Sleep(300 * time.Millisecond)
	_ = p.ResetInputBuffer()
	_ = p.ResetOutputBuffer()
	p.SetReadTimeout(100 * time.Millisecond)
	drain := make([]byte, 512)
	for i := 0; i < 2; i++ {
		n, _ := p.Read(drain)
		if n == 0 {
			break
		}
		log.Printf("apply: dreno %d bytes: %q", n, string(drain[:n]))
	}
	for _, item := range cfg {
		line := fmt.Sprintf("%d=%s\n", item.Index, normalizeMode(item.Mode))
		log.Printf("apply: -> %q", line)
		if _, err := p.Write([]byte(line)); err != nil {
			p.Close()
			serialMu.Unlock()
			return err
		}
		p.SetReadTimeout(200 * time.Millisecond)
		n, _ := p.Read(drain)
		if n > 0 {
			log.Printf("apply: <- %q", string(drain[:n]))
		}
		time.Sleep(30 * time.Millisecond)
	}
	p.Close()
	serialMu.Unlock()
	log.Printf("apply: concluído")
	return nil
}

func readButtonConfigFromSerial(portName string) ([]ButtonConfig, error) {
	if strings.TrimSpace(portName) == "" {
		return nil, fmt.Errorf("serial port not informed")
	}
	serialMu.Lock()
	log.Printf("readButtonConfigFromSerial: %s", portName)
	p, err := serial.Open(portName, &serial.Mode{BaudRate: 115200})
	if err != nil {
		serialMu.Unlock()
		return nil, err
	}
	_ = p.SetDTR(false)
	time.Sleep(300 * time.Millisecond)
	_ = p.ResetInputBuffer()
	_ = p.ResetOutputBuffer()
	p.SetReadTimeout(100 * time.Millisecond)
	drain := make([]byte, 512)
	for i := 0; i < 2; i++ {
		n, _ := p.Read(drain)
		if n > 0 {
			log.Printf("read: dreno %d %q", n, string(drain[:n]))
		}
		if n == 0 {
			break
		}
	}
	if _, err := p.Write([]byte("GET\n")); err != nil {
		p.Close()
		serialMu.Unlock()
		return nil, err
	}
	log.Printf("read: GET enviado")
	p.SetReadTimeout(500 * time.Millisecond)
	result := make([]ButtonConfig, 0, totalButtons)
	for len(result) < totalButtons {
		buf := make([]byte, 256)
		n, err := p.Read(buf)
		if err != nil {
			if strings.Contains(err.Error(), "timeout") {
				break
			}
			p.Close()
			serialMu.Unlock()
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
		p.Close()
		serialMu.Unlock()
		return nil, fmt.Errorf("device returned incomplete configuration")
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Index < result[j].Index })
	p.Close()
	serialMu.Unlock()
	log.Printf("read: %d modos lidos", len(result))
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
	// não pré-seleciona COM1 (pode ser porta da placa-mãe) — deixa vazio para auto-detecção via HID
	// selectedPort ficará vazio e resolveSerialPort fará brute-force
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
	colTwoShot    = color.NRGBA{R: 155, G: 125, B: 232, A: 255}
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
	case "TWO_SHOT":
		return colTwoShot
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
	case "TWO_SHOT":
		return "TWO-SHOT"
	case "TOGGLE":
		return "TOGGLE"
	case "LONG_PRESS":
		return "LONG-PRESS"
	default:
		return mode
	}
}

func buildGrid(state *uiState) *fyne.Container {
	// barra horizontal preenche toda a largura do card — sem círculo
	state.gridBars = make([]*canvas.Rectangle, 0, totalButtons)
	cards := make([]fyne.CanvasObject, 0, totalButtons)
	for i, item := range state.config {
		idx := i
		mode := normalizeMode(item.Mode)
		isPressed := i < len(state.buttonStates) && state.buttonStates[i]
		accent := getModeColor(mode)

		accentBar := canvas.NewRectangle(accent)
		accentBar.SetMinSize(fyne.NewSize(0, 3))
		accentBar.CornerRadius = 2

		numText := canvas.NewText(fmt.Sprintf("%02d", item.Index), colInk)
		numText.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
		numText.TextSize = 22
		numText.Alignment = fyne.TextAlignCenter

		// LED agora é barra horizontal que preenche o card (10px, cantos 4)
		ledBar := canvas.NewRectangle(colLedOff)
		if isPressed {
			ledBar.FillColor = colLedOn
		}
		ledBar.SetMinSize(fyne.NewSize(0, 10))
		ledBar.CornerRadius = 4
		ledBar.StrokeColor = colPanelEdge
		ledBar.StrokeWidth = 1
		state.gridBars = append(state.gridBars, ledBar)

		modeLabel := canvas.NewText(getModeLabel(mode), colSteel)
		modeLabel.TextSize = 9
		modeLabel.TextStyle = fyne.TextStyle{Bold: true}
		modeLabel.Alignment = fyne.TextAlignCenter

		sel := widget.NewSelect([]string{"NORMAL", "ONE_SHOT", "TWO_SHOT", "TOGGLE", "LONG_PRESS"}, func(m string) {
			state.config[idx].Mode = normalizeMode(m)
			_ = persistLocalConfig(state.config)
			setExport(state)
			setStatus(state, t("config_saved"))
		})
		sel.SetSelected(mode)
		sel.PlaceHolder = "Modo"

		bg := canvas.NewRectangle(colPanel)
		bg.StrokeColor = colPanelEdge
		bg.StrokeWidth = 1
		bg.CornerRadius = 10

		inner := container.NewVBox(
			accentBar,
			container.NewPadded(numText),
			ledBar,
			modeLabel,
			sel,
		)
		innerPad := container.NewPadded(inner)

		cardStack := container.NewStack(bg, innerPad)
		wrapper := container.NewPadded(cardStack)
		cardWithSize := container.NewStack(wrapper)
		cardWithSize.Resize(fyne.NewSize(160, 135))
		cards = append(cards, container.NewPadded(cardWithSize))
	}
	grid := container.NewGridWithColumns(4, cards...)
	return container.NewPadded(grid)
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

		idealLabel := widget.NewLabel("💡 " + t(mode.Key+"_ideal"))
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
	joysticks := state.joysticks
	if len(joysticks) == 0 {
		joysticks = []string{t("no_joysticks_detected")}
	}
	joySel := widget.NewSelect(joysticks, func(v string) {
		state.selectedJoystick = v
		if v == "" || v == t("no_joysticks_detected") {
			return
		}
		go func(sel string) {
			state.jsMutex.Lock()
			if state.jsMonitoring {
				state.jsMonitoring = false
				if state.jsFile != nil {
					state.jsFile.Close()
					state.jsFile = nil
				}
			}
			state.jsMutex.Unlock()
			time.Sleep(80 * time.Millisecond)
			go readJoystickEvents(state, sel, nil)
			// também auto-lê configuração da placa associada
			time.Sleep(300 * time.Millisecond)
			if port := resolveSerialPort(state); port != "" {
				if cfg, err := readButtonConfigFromSerial(port); err == nil {
					state.config = cfg
					setExport(state)
					setStatus(state, t("config_read_board"))
					win.SetContent(buildUI(state, win))
				}
			}
		}(v)
	})
	if state.selectedJoystick != "" {
		joySel.SetSelected(state.selectedJoystick)
	} else if len(joysticks) > 0 {
		joySel.SetSelected(joysticks[0])
		state.selectedJoystick = joysticks[0]
		// auto-monitoring no arranque
		go func(sel string) {
			time.Sleep(300 * time.Millisecond)
			go readJoystickEvents(state, sel, nil)
		}(joysticks[0])
	}

	// styled buttons — rodam em goroutine para não travar a UI
	readBtn := widget.NewButtonWithIcon(t("read_from_board"), theme.DownloadIcon(), func() {
		go func() {
			port := resolveSerialPort(state)
			if port == "" {
				setStatus(state, t("select_serial_port"))
				return
			}
			setStatus(state, t("monitoring")+" "+port+"...")
			cfg, err := readButtonConfigFromSerial(port)
			if err != nil {
				setStatus(state, t("error_reading_board")+" "+err.Error())
				return
			}
			state.config = cfg
			setExport(state)
			setStatus(state, t("config_read_board"))
			win.SetContent(buildUI(state, win))
		}()
	})
	applyBtn := widget.NewButtonWithIcon(t("apply_to_board"), theme.UploadIcon(), func() {
		go func() {
			port := resolveSerialPort(state)
			if port == "" {
				setStatus(state, t("select_serial_port"))
				return
			}
			setStatus(state, "Aplicando em "+port+"...")
			if err := applyButtonConfigToSerial(port, state.config); err != nil {
				setStatus(state, t("error_applying_board")+" "+err.Error())
				return
			}
			setStatus(state, t("config_applied")+" "+port+".")
			setExport(state)
		}()
	})
	applyBtn.Importance = widget.HighImportance
	resetBtn := widget.NewButtonWithIcon(t("reset"), theme.ViewRefreshIcon(), func() {
		// reseta local e tenta resetar na placa também
		go func() {
			if port := resolveSerialPort(state); port != "" {
				serialMu.Lock()
				p, err := serial.Open(port, &serial.Mode{BaudRate: 115200})
				if err == nil {
					time.Sleep(1500 * time.Millisecond)
					_, _ = p.Write([]byte("RESET\n"))
					time.Sleep(200 * time.Millisecond)
					_ = p.Close()
				}
				serialMu.Unlock()
			}
		}()
		state.config = buildDefaultConfig()
		setExport(state)
		setStatus(state, t("config_restored"))
		win.SetContent(buildUI(state, win))
	})

	// deck único — só joystick (monitoramento automático), serial é auto-detectada do mesmo dispositivo
	bg := canvas.NewRectangle(colPanel)
	bg.StrokeColor = colPanelEdge
	bg.StrokeWidth = 1
	bg.CornerRadius = 10
	joyTitle := canvas.NewText("DISPOSITIVO", colSteel)
	joyTitle.TextSize = 9
	joyTitle.TextStyle = fyne.TextStyle{Bold: true}
	deckInner := container.NewVBox(
		joyTitle,
		joySel,
		container.NewGridWithColumns(3, readBtn, applyBtn, resetBtn),
	)
	deck := container.NewStack(bg, container.NewPadded(deckInner))
	return deck
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

	// grid scroll with fixed min so it never collapses to 4px (screenshot bug)
	gridScroll := container.NewVScroll(grid)
	gridScroll.SetMinSize(fyne.NewSize(760, 640))

	// legend fixed width on the right
	legendWrap := container.NewPadded(legend)
	legendScroll := container.NewVScroll(legendWrap)
	legendScroll.SetMinSize(fyne.NewSize(280, 640))

	middle := container.NewBorder(nil, nil, nil, legendScroll, gridScroll)

	// outer bg
	outerBg := canvas.NewRectangle(colBg)
	content := container.NewVBox(header, deck, middle, exportCard)
	padded := container.NewPadded(content)
	scroll := container.NewVScroll(padded)
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
	// auto-leitura da placa ao abrir (sem precisar clicar Read)
	go func() {
		time.Sleep(700 * time.Millisecond)
		port := resolveSerialPort(state)
		if port == "" {
			log.Printf("auto-read: nenhuma porta ButtonBox")
			return
		}
		log.Printf("auto-read: lendo de %s", port)
		cfg, err := readButtonConfigFromSerial(port)
		if err != nil {
			log.Printf("auto-read falhou: %v", err)
			setStatus(state, t("error_reading_board")+" "+err.Error())
			return
		}
		state.config = cfg
		setExport(state)
		setStatus(state, t("config_read_board"))
		w.SetContent(buildUI(state, w))
		log.Printf("auto-read: ok")
	}()
	w.ShowAndRun()
}
