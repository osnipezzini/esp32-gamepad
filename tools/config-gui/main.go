package main

import (
	"encoding/json"
	"fmt"
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
	"image/color"
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
	config            []ButtonConfig
	buttonStates      []bool // true = pressed, false = released
	ports             []string
	selectedPort      string
	joysticks         []string
	selectedJoystick  string
	status            binding.String
	exportText        binding.String
	lastReadMs        int64
	jsFile            *os.File
	jsStopChan        chan bool
	jsUpdateCallback  func([]bool)
	jsMonitoring      bool
	jsMutex           sync.Mutex
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

func loadStoredConfig() []ButtonConfig {
	path := filepath.Join(".", "buttonbox-config.json")
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
	return os.WriteFile(filepath.Join(".", "buttonbox-config.json"), data, 0o644)
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
	candidates := map[string]struct{}{}
	if runtime.GOOS == "windows" {
		for drive := byte('A'); drive <= byte('Z'); drive++ {
			path := fmt.Sprintf("%c:", drive)
			if _, err := os.Stat(path); err != nil {
				continue
			}
			for i := 1; i <= 256; i++ {
				name := fmt.Sprintf("COM%d", i)
				if _, err := os.Stat(name); err == nil {
					candidates[name] = struct{}{}
				}
			}
		}
	} else {
		entries, err := os.ReadDir("/dev")
		if err == nil {
			for _, entry := range entries {
				name := entry.Name()
				if strings.Contains(name, "ttyUSB") || strings.Contains(name, "ttyACM") || strings.Contains(name, "cu.usb") || strings.Contains(name, "tty.usb") {
					candidates[filepath.Join("/dev", name)] = struct{}{}
				}
			}
		}
	}
	ports := make([]string, 0, len(candidates))
	for port := range candidates {
		ports = append(ports, port)
	}
	sort.Strings(ports)
	return ports
}

func listJoysticks() []string {
	joysticks := []string{}
	if runtime.GOOS == "linux" {
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
	file, err := os.Open(joystickPath)
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
		config:        loadStoredConfig(),
		buttonStates:  make([]bool, totalButtons),
		ports:         listSerialPorts(),
		joysticks:     listJoysticks(),
		status:        binding.NewString(),
		exportText:    binding.NewString(),
		jsStopChan:    make(chan bool, 1),
		jsMonitoring:  false,
		jsMutex:       sync.Mutex{},
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

func getModeColor(mode string) color.Color {
	switch normalizeMode(mode) {
	case "NORMAL":
		return color.NRGBA{R: 100, G: 150, B: 200, A: 255}
	case "ONE_SHOT":
		return color.NRGBA{R: 200, G: 150, B: 100, A: 255}
	case "TOGGLE":
		return color.NRGBA{R: 150, G: 200, B: 100, A: 255}
	case "LONG_PRESS":
		return color.NRGBA{R: 200, G: 100, B: 150, A: 255}
	default:
		return color.NRGBA{R: 100, G: 100, B: 100, A: 255}
	}
}

func buildGrid(state *uiState) *fyne.Container {
	cards := make([]fyne.CanvasObject, 0, totalButtons)
	for i, item := range state.config {
		index := i
		isPressed := false
		if i < len(state.buttonStates) {
			isPressed = state.buttonStates[i]
		}

		// LED circle for button press state (smaller: 24x24)
		ledColor := color.NRGBA{R: 100, G: 200, B: 100, A: 255}
		if isPressed {
			ledColor = color.NRGBA{R: 255, G: 50, B: 50, A: 255}
		}
		ledCircle := canvas.NewCircle(ledColor)
		ledCircle.StrokeColor = color.NRGBA{R: 50, G: 50, B: 50, A: 255}
		ledCircle.StrokeWidth = 1

		// Button number (small)
		numLabel := widget.NewLabel(fmt.Sprintf("%d", item.Index))
		numLabel.Alignment = fyne.TextAlignCenter
		numLabelContainer := container.NewCenter(numLabel)

		// LED with number overlaid
		ledContainer := container.NewStack(
			ledCircle,
			numLabelContainer,
		)
		ledContainer.Resize(fyne.NewSize(24, 24))

		// Mode selector with tooltip
		modeSelector := widget.NewSelect([]string{"NORMAL", "ONE_SHOT", "TOGGLE", "LONG_PRESS"}, func(mode string) {
			state.config[index].Mode = normalizeMode(mode)
			if err := persistLocalConfig(state.config); err != nil {
				setStatus(state, t("config_save_failed")+err.Error())
				return
			}
			setExport(state)
			setStatus(state, t("config_saved"))
		})
		modeSelector.SetSelected(item.Mode)
		modeSelector.PlaceHolder = "Select mode"

		// Mode color indicator bar (smaller)
		modeIndicator := canvas.NewRectangle(getModeColor(item.Mode))

		// Card layout: LED + mode selector + color bar
		card := container.NewVBox(
			container.NewCenter(ledContainer),
			modeSelector,
			modeIndicator,
		)
		
		// Wrap in box with padding
		cards = append(cards, container.NewPadded(card))
	}
	return container.NewGridWithColumns(8, cards...)
}

func buildLegend() *fyne.Container {
	ledLegend := container.NewVBox(
		widget.NewLabel(t("led_state")),
		container.NewHBox(
			container.NewStack(
				canvas.NewCircle(color.NRGBA{R: 100, G: 200, B: 100, A: 255}),
			),
			widget.NewLabel(t("not_pressed")),
		),
		container.NewHBox(
			container.NewStack(
				canvas.NewCircle(color.NRGBA{R: 255, G: 50, B: 50, A: 255}),
			),
			widget.NewLabel(t("pressed")),
		),
	)

	modeColorItems := []struct {
		mode  string
		descKey string
	}{
		{"NORMAL", "mode_normal"},
		{"ONE_SHOT", "mode_one_shot"},
		{"TOGGLE", "mode_toggle"},
		{"LONG_PRESS", "mode_long_press"},
	}

	legendCards := make([]fyne.CanvasObject, 0, len(modeColorItems))
	for _, item := range modeColorItems {
		colorRect := canvas.NewRectangle(getModeColor(item.mode))
		colorRect.SetMinSize(fyne.NewSize(12, 12))
		label := widget.NewLabel(t(item.descKey))
		card := container.NewHBox(colorRect, label)
		legendCards = append(legendCards, card)
	}

	return container.NewVBox(
		widget.NewRichTextFromMarkdown("### "+t("legend")),
		ledLegend,
		widget.NewRichTextFromMarkdown("**"+t("modes")+"**"),
		container.NewGridWithColumns(2, legendCards...),
	)
}

func buildUI(state *uiState, win fyne.Window) fyne.CanvasObject {
	ports := state.ports
	if len(ports) == 0 {
		ports = []string{t("no_ports_detected")}
	}
	portSelect := widget.NewSelect(ports, func(selected string) {
		state.selectedPort = selected
	})
	if state.selectedPort != "" {
		portSelect.SetSelected(state.selectedPort)
	} else if len(ports) > 0 {
		portSelect.SetSelected(ports[0])
	}

	joysticks := state.joysticks
	if len(joysticks) == 0 {
		joysticks = []string{t("no_joysticks_detected")}
	}
	joystickSelect := widget.NewSelect(joysticks, func(selected string) {
		state.selectedJoystick = selected
	})
	if state.selectedJoystick != "" {
		joystickSelect.SetSelected(state.selectedJoystick)
	} else if len(joysticks) > 0 {
		joystickSelect.SetSelected(joysticks[0])
	}

	statusLabel := widget.NewLabelWithData(state.status)
	
	exportEntry := widget.NewEntryWithData(state.exportText)
	exportEntry.MultiLine = true
	exportEntry.Disable()

	readBtn := widget.NewButton(t("read_from_board"), func() {
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
		if err := persistLocalConfig(state.config); err != nil {
			setStatus(state, t("read_ok_save_fail")+" "+err.Error())
			return
		}
		setExport(state)
		setStatus(state, t("config_read_board"))
		win.SetContent(buildUI(state, win))
	})

	applyBtn := widget.NewButton(t("apply_to_board"), func() {
		if state.selectedPort == "" || state.selectedPort == t("no_ports_detected") {
			setStatus(state, t("select_serial_port"))
			return
		}
		if err := applyButtonConfigToSerial(state.selectedPort, state.config); err != nil {
			setStatus(state, t("error_applying_board")+" "+err.Error())
			return
		}
		if err := persistLocalConfig(state.config); err != nil {
			setStatus(state, t("applied_save_fail")+" "+err.Error())
			return
		}
		setStatus(state, t("config_applied")+" "+state.selectedPort+".")
		setExport(state)
	})

	resetBtn := widget.NewButton(t("reset"), func() {
		state.config = buildDefaultConfig()
		if err := persistLocalConfig(state.config); err != nil {
			setStatus(state, t("error_resetting")+" "+err.Error())
			return
		}
		setExport(state)
		setStatus(state, t("config_restored"))
		win.SetContent(buildUI(state, win))
	})

	var monitorBtn *widget.Button
	monitorBtn = widget.NewButton(t("monitor_joystick"), func() {
		state.jsMutex.Lock()
		isMonitoring := state.jsMonitoring
		state.jsMutex.Unlock()
		
		if isMonitoring {
			state.jsMutex.Lock()
			state.jsMonitoring = false
			state.jsMutex.Unlock()
			
			if state.jsFile != nil {
				state.jsFile.Close()
				state.jsFile = nil
			}
			setStatus(state, t("monitoring_stopped"))
			monitorBtn.SetText(t("monitor_joystick"))
			return
		}
		
		if state.selectedJoystick == "" || state.selectedJoystick == t("no_joysticks_detected") {
			setStatus(state, t("select_joystick"))
			return
		}
		
		go readJoystickEvents(state, state.selectedJoystick, func() {
			win.SetContent(buildUI(state, win))
		})
		monitorBtn.SetText(t("stop_monitoring"))
	})

	// Language selector
	langSelect := widget.NewSelect([]string{"English", "Português (BR)"}, func(lang string) {
		if lang == "English" {
			currentLang = LangEnglish
		} else {
			currentLang = LangPtBr
		}
		win.SetContent(buildUI(state, win))
	})
	if currentLang == LangEnglish {
		langSelect.SetSelected("English")
	} else {
		langSelect.SetSelected("Português (BR)")
	}

	topBar := container.NewHBox(
		widget.NewLabel(t("serial_port")+" "),
		portSelect,
		readBtn,
		applyBtn,
		resetBtn,
		widget.NewLabel("  |  Lang: "),
		langSelect,
	)

	joystickBar := container.NewHBox(
		widget.NewLabel(t("joystick")+" "),
		joystickSelect,
		monitorBtn,
	)

	// Organize content in sections
	gridScroll := container.NewScroll(buildGrid(state))
	
	exportSection := container.NewVBox(
		widget.NewRichTextFromMarkdown("**C++ Export:**"),
		exportEntry,
	)

	return container.NewBorder(
		topBar,           // top
		buildLegend(),    // bottom
		nil,              // left
		nil,              // right
		container.NewVBox(
			joystickBar,
			gridScroll,
			exportSection,
			statusLabel,
		),
	)
}

func main() {
	if err := loadTranslations(filepath.Join(".", "locales")); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load translations: %v\n", err)
		os.Exit(1)
	}
	if _, ok := translations[LangEnglish]; !ok {
		fmt.Fprintln(os.Stderr, "missing English locale")
		os.Exit(1)
	}
	
	a := app.New()
	state := newUIState()
	w := a.NewWindow(t("title"))
	w.Resize(fyne.NewSize(1000, 800))
	w.SetContent(buildUI(state, w))
	w.ShowAndRun()
}
