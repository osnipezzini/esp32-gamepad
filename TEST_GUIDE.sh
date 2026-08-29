#!/bin/bash
# ButtonBox GUI + Firmware - Testing Guide
# This script provides a complete testing procedure

set -e

echo "╔═══════════════════════════════════════════════════════════╗"
echo "║  ButtonBox - Complete Testing Guide                      ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test 1: Check firmware build
echo -e "${BLUE}▶ Test 1: Firmware Compilation${NC}"
cd /home/osni/projetos/sodevs/buttonbox
if pio run -e promicro16 -s 2>&1 | grep -q "SUCCESS"; then
    echo -e "${GREEN}✓ Firmware compiled successfully${NC}"
else
    echo -e "${RED}✗ Firmware compilation failed${NC}"
    exit 1
fi
echo ""

# Test 2: Check GUI build
echo -e "${BLUE}▶ Test 2: GUI Compilation${NC}"
cd /home/osni/projetos/sodevs/buttonbox/tools/config-gui
if go build ./... 2>&1; then
    echo -e "${GREEN}✓ GUI compiled successfully${NC}"
else
    echo -e "${RED}✗ GUI compilation failed${NC}"
    exit 1
fi
echo ""

# Test 3: Check serial ports
echo -e "${BLUE}▶ Test 3: Serial Port Detection${NC}"
PORTS=$(ls /dev/ttyUSB* /dev/ttyACM* 2>/dev/null || echo "NONE")
if [ "$PORTS" != "NONE" ]; then
    echo -e "${GREEN}✓ Serial ports found:${NC}"
    echo "$PORTS"
else
    echo -e "${YELLOW}⚠ No USB serial ports detected (expected if ButtonBox not connected)${NC}"
fi
echo ""

# Test 4: Check joystick devices
echo -e "${BLUE}▶ Test 4: Joystick Detection${NC}"
JOYSTICKS=$(ls /dev/input/js* 2>/dev/null || echo "NONE")
if [ "$JOYSTICKS" != "NONE" ]; then
    echo -e "${GREEN}✓ Joystick devices found:${NC}"
    echo "$JOYSTICKS"
else
    echo -e "${YELLOW}⚠ No joystick devices detected (expected if ButtonBox not connected)${NC}"
fi
echo ""

# Test 5: Show build artifacts
echo -e "${BLUE}▶ Test 5: Build Artifacts${NC}"
echo "Firmware size:"
ls -lh /home/osni/projetos/sodevs/buttonbox/.pio/build/promicro16/firmware.elf 2>/dev/null || echo "  (not built yet)"
echo ""
echo "GUI executable:"
ls -lh /home/osni/projetos/sodevs/buttonbox/tools/config-gui/buttonbox-config-gui || echo "  (not built)"
echo ""

# Test 6: Manual testing instructions
echo -e "${BLUE}▶ Test 6: Manual Testing Instructions${NC}"
echo ""
echo -e "${YELLOW}Step 1: Connect ButtonBox Hardware${NC}"
echo "  1. Connect the ButtonBox (Pro Micro) via USB to your computer"
echo "  2. Check serial port: ls /dev/tty{USB,ACM}*"
echo "  3. Add user to 'dialout' group if needed: sudo usermod -aG dialout \$USER"
echo "  4. Verify joystick detection: ls /dev/input/js*"
echo ""

echo -e "${YELLOW}Step 2: Upload Firmware${NC}"
echo "  1. Navigate to project folder:"
echo "     cd /home/osni/projetos/sodevs/buttonbox"
echo "  2. Upload firmware:"
echo "     pio run --target upload -e promicro16"
echo "  3. Wait for 'SUCCESS' message"
echo ""

echo -e "${YELLOW}Step 3: Run GUI Application${NC}"
echo "  1. Start the GUI:"
echo "     cd /home/osni/projetos/sodevs/buttonbox/tools/config-gui"
echo "     ./buttonbox-config-gui"
echo ""
echo "  2. GUI Layout:"
echo "     - Serial Port Selector: Choose COM port with ButtonBox"
echo "     - 'Read from Board': Load current button modes from device"
echo "     - 'Apply to Board': Save changes to device EEPROM"
echo "     - Mode Selector (per button): NORMAL, ONE_SHOT, TOGGLE, LONG_PRESS"
echo "     - Joystick Selector: Choose which /dev/input/jsX to monitor"
echo "     - 'Monitor Joystick': Start real-time LED updates"
echo "     - Language Selector: Switch between English and Português (BR)"
echo ""

echo -e "${YELLOW}Step 4: Test Button Modes${NC}"
echo "  1. Select a button in the GUI (e.g., Button 0)"
echo "  2. Choose a mode (e.g., TOGGLE)"
echo "  3. Click 'Apply to Board'"
echo "  4. Press the physical button (Button 0 on matrix)"
echo "  5. Observe LED state in GUI changes to RED when pressed"
echo ""

echo -e "${YELLOW}Step 5: Verify Joystick Monitoring${NC}"
echo "  1. Select joystick from dropdown (e.g., /dev/input/js0)"
echo "  2. Click 'Monitor Joystick' button"
echo "  3. Status should show 'Monitoring joystick: /dev/input/js0'"
echo "  4. Press physical buttons"
echo "  5. LED circles should change:"
echo "     - GREEN when released"
echo "     - RED when pressed"
echo "  6. Changes should be instant (no delay)"
echo ""

echo -e "${YELLOW}Step 6: Test Multi-Joystick Support${NC}"
echo "  1. If you have multiple joysticks, try switching between them"
echo "  2. Dropdown should show all detected /dev/input/js* devices"
echo "  3. 'Monitor Joystick' should toggle monitoring on/off"
echo "  4. Button text changes from 'Monitor' to 'Stop Monitoring'"
echo ""

echo -e "${YELLOW}Step 7: Language Switching${NC}"
echo "  1. Click language selector (top right: 'Lang')"
echo "  2. Choose English or Português (BR)"
echo "  3. All UI text should update immediately"
echo ""

echo ""
echo -e "${GREEN}╔═══════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  Testing Checklist                                        ║${NC}"
echo -e "${GREEN}╚═══════════════════════════════════════════════════════════╝${NC}"
echo ""
echo "Run these checks after hardware is connected:"
echo ""
echo "Firmware:"
echo "  [ ] Firmware compiles without errors"
echo "  [ ] Flash size < 50% (shows as 48.7% in output)"
echo "  [ ] RAM usage < 50% (shows as 32.6% in output)"
echo ""
echo "Serial Communication:"
echo "  [ ] Serial port detected when ButtonBox connected"
echo "  [ ] 'Read from Board' successfully loads config"
echo "  [ ] 'Apply to Board' sends config without errors"
echo "  [ ] 'Reset' restores default configuration"
echo ""
echo "Joystick Monitoring:"
echo "  [ ] Joystick device (/dev/input/jsX) detected"
echo "  [ ] 'Monitor Joystick' button starts monitoring"
echo "  [ ] LED circles change color when buttons pressed"
echo "  [ ] Color changes are instant (no lag)"
echo "  [ ] Button numbers are visible (0-15)"
echo "  [ ] Correct button index updates when physical button pressed"
echo ""
echo "User Interface:"
echo "  [ ] Layout is compact and organized"
echo "  [ ] 16 buttons visible in 2x8 grid"
echo "  [ ] Mode selectors have 4 options"
echo "  [ ] Color codes match mode descriptions"
echo "  [ ] Language selector switches EN/PT-BR"
echo "  [ ] Status messages are clear and translated"
echo ""
echo "Multi-Joystick:"
echo "  [ ] Multiple /dev/input/js* devices shown in dropdown"
echo "  [ ] Can switch between devices while monitoring"
echo "  [ ] Monitoring stops cleanly when button clicked"
echo ""

echo ""
echo -e "${BLUE}Need help?${NC}"
echo "  - Check firmware logs: less /tmp/buttonbox_*"
echo "  - Check GUI logs: less /tmp/buttonbox-gui.log"
echo "  - Verify permissions: groups (should include 'input')"
echo "  - Test joystick directly: jstest /dev/input/js0"
echo ""
