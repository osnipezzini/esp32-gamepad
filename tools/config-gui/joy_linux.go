//go:build !windows

package main

import (
	"fmt"
	"strings"
)

// Stubs for non-Windows builds so main.go compiles on Linux
func winmmListJoysticks() ([]string, map[string]string) {
	return nil, nil
}

func winmmReadButtons(joyID int) (uint32, error) {
	return 0, fmt.Errorf("winmm not available on this OS")
}

func winmmParseJoyID(display string) int {
	// fallback: try to parse "[ID n]" or last number
	var id int
	if _, err := fmt.Sscanf(display, "%*s (%*s) [ID %d]", &id); err == nil {
		return id
	}
	// search for "[ID "
	if idx := strings.Index(display, "[ID "); idx >= 0 {
		fmt.Sscanf(display[idx:], "[ID %d]", &id)
		return id
	}
	return 0
}
