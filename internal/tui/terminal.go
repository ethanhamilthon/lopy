package tui

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Bubble Tea v1.3.10 exposes unknown CSI sequences only through Stringer;
// it has no native Kitty keyboard protocol API. Keep compatibility here.
func normalizeTerminalKey(msg tea.Msg) tea.Msg {
	if sequence, ok := msg.(fmt.Stringer); ok {
		if paste, release := pasteSequence(sequence.String()); paste {
			if release {
				return nil
			}
			return tea.KeyMsg{Type: tea.KeyCtrlV}
		}
		if key, ok := kittyKey(sequence.String()); ok {
			return key
		}
		switch sequence.String() {
		case fmt.Sprintf("?CSI%+v?", []byte("13;2u")), fmt.Sprintf("?CSI%+v?", []byte("27;2;13~")), fmt.Sprintf("?CSI%+v?", []byte("13;2~")):
			return tea.KeyMsg{Type: tea.KeyEnter, Alt: true}
		}
	}
	return msg
}

func csiU(sequence string) ([]string, []string, bool) {
	if !strings.HasPrefix(sequence, "?CSI[") || !strings.HasSuffix(sequence, "]?") {
		return nil, nil, false
	}
	var payload strings.Builder
	for _, number := range strings.Fields(strings.TrimSuffix(strings.TrimPrefix(sequence, "?CSI["), "]?")) {
		value, err := strconv.Atoi(number)
		if err != nil || value < 0 || value > 127 {
			return nil, nil, false
		}
		payload.WriteByte(byte(value))
	}
	text, ok := strings.CutSuffix(payload.String(), "u")
	parts := strings.Split(text, ";")
	if !ok || len(parts) > 2 {
		return nil, nil, false
	}
	var modifiers []string
	if len(parts) == 2 {
		modifiers = strings.Split(parts[1], ":")
	}
	return strings.Split(parts[0], ":"), modifiers, true
}

// kittyKey decodes the keys that the Kitty keyboard protocol's disambiguate
// mode reports as CSI sequences instead of legacy bytes: Esc arrives as
// CSI 27 u, which Bubble Tea does not recognise.
func kittyKey(sequence string) (tea.Msg, bool) {
	codes, modifiers, ok := csiU(sequence)
	if !ok || codes[0] != "27" {
		return nil, false
	}
	if len(modifiers) > 0 {
		if value, err := strconv.Atoi(modifiers[0]); err != nil || value != 1 {
			return nil, false
		}
		if len(modifiers) > 1 && modifiers[1] == "3" {
			return nil, true // key release
		}
	}
	return tea.KeyMsg{Type: tea.KeyEsc}, true
}

// Kitty reports modifiers and optional base-layout keycodes even on Cyrillic layouts.
func pasteSequence(sequence string) (bool, bool) {
	codes, modifiers, ok := csiU(sequence)
	if !ok || len(modifiers) == 0 {
		return false, false
	}
	value, err := strconv.Atoi(modifiers[0])
	if err != nil || value < 1 || (value-1)&(4|8|32) == 0 {
		return false, false
	}
	for _, code := range codes {
		if code == "118" || code == "86" || code == "1084" || code == "1052" {
			return true, len(modifiers) > 1 && modifiers[1] != "1"
		}
	}
	return false, false
}

type keyboardProtocol struct{ enabled bool }

func (k *keyboardProtocol) close() {
	if k.enabled {
		fmt.Fprint(os.Stdout, "\x1b[<u")
		k.enabled = false
	}
}
func (k *keyboardProtocol) filter(_ tea.Model, msg tea.Msg) tea.Msg {
	if _, quit := msg.(tea.QuitMsg); quit {
		k.close()
	} else if !k.enabled {
		fmt.Fprint(os.Stdout, "\x1b[>5u")
		k.enabled = true
	}
	return msg
}
