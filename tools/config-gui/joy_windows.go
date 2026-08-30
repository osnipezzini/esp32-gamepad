//go:build windows

package main

import (
	"fmt"
	"sort"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	winmmDLL           = windows.NewLazySystemDLL("winmm.dll")
	procJoyGetNumDevs  = winmmDLL.NewProc("joyGetNumDevs")
	procJoyGetDevCapsW = winmmDLL.NewProc("joyGetDevCapsW")
	procJoyGetPosEx    = winmmDLL.NewProc("joyGetPosEx")
)

type joyCapsW struct {
	WMid         uint16
	WPid         uint16
	SzPname      [32]uint16
	WXmin, WXmax uint32
	WYmin, WYmax uint32
	WZmin, WZmax uint32
	WNumButtons  uint32
	WPeriodMin   uint32
	WPeriodMax   uint32
	WRgnCount    uint32
	WUmin, WUmax uint32
	WVmin, WVmax uint32
	WCaps        uint32
	WMaxAxes     uint32
	WNumAxes     uint32
	WMaxButtons  uint32
	SzRegKey     [32]uint16
	SzOEMVxD     [260]uint16
}

type joyInfoEx struct {
	DwSize         uint32
	DwFlags        uint32
	DwXpos         uint32
	DwYpos         uint32
	DwZpos         uint32
	DwRpos         uint32
	DwUpos         uint32
	DwVpos         uint32
	DwButtons      uint32
	DwButtonNumber uint32
	DwPOV          uint32
	DwReserved1    uint32
	DwReserved2    uint32
}

const joyReturnAll = 0x000000ff // JOY_RETURNX|Y|Z|R|U|V|POV|BUTTONS

func winmmListJoysticks() ([]string, map[string]string) {
	hidPathByDisplay = map[string]string{}
	hidDisplayByPath = map[string]string{}
	n, _, _ := procJoyGetNumDevs.Call()
	count := int(n)
	if count <= 0 || count > 16 {
		count = 16
	}
	var res []string
	seen := map[string]bool{}
	for i := 0; i < count; i++ {
		var caps joyCapsW
		caps.WMid = 0
		// joyGetDevCapsW expects size as third param
		r, _, _ := procJoyGetDevCapsW.Call(uintptr(i), uintptr(unsafe.Pointer(&caps)), uintptr(unsafe.Sizeof(caps)))
		if r != 0 {
			continue
		}
		if caps.WNumButtons == 0 {
			continue
		}
		name := windows.UTF16ToString(caps.SzPname[:])
		if strings.TrimSpace(name) == "" {
			name = fmt.Sprintf("Joystick %d", i)
		}
		display := fmt.Sprintf("%s (%04x:%04x) [ID %d]", name, caps.WMid, caps.WPid, i)
		if seen[display] {
			continue
		}
		seen[display] = true
		key := fmt.Sprintf("%d", i)
		hidPathByDisplay[display] = key
		hidDisplayByPath[key] = display
		res = append(res, display)
	}
	sort.Strings(res)
	return res, hidPathByDisplay
}

func winmmReadButtons(joyID int) (uint32, error) {
	var info joyInfoEx
	info.DwSize = uint32(unsafe.Sizeof(info))
	info.DwFlags = joyReturnAll
	r, _, _ := procJoyGetPosEx.Call(uintptr(joyID), uintptr(unsafe.Pointer(&info)))
	if r != 0 {
		// JOYERR_NOERROR = 0, JOYERR_UNPLUGGED = 167
		return 0, fmt.Errorf("joyGetPosEx id %d err %d", joyID, r)
	}
	return info.DwButtons, nil
}

func winmmParseJoyID(display string) int {
	if s, ok := hidPathByDisplay[display]; ok {
		var id int
		fmt.Sscanf(s, "%d", &id)
		return id
	}
	// fallback: parse "[ID 1]" suffix
	var id int
	if _, err := fmt.Sscanf(display, "%*s (%*s) [ID %d]", &id); err == nil {
		return id
	}
	// try to extract last number
	for i := len(display) - 1; i >= 0; i-- {
		if display[i] == ']' {
			// find [
			for j := i; j >= 0; j-- {
				if display[j] == '[' {
					// inside is "ID 1"
					var iid int
					fmt.Sscanf(display[j:], "[ID %d]", &iid)
					return iid
				}
			}
		}
	}
	return 0
}
