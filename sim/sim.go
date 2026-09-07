package sim

import (
	"errors"

	"github.com/draxxris/rtgui/ui"
)

// ErrNilUI reports construction without the UI instance the stage adapts.
var ErrNilUI = errors.New("sim: nil UI")

// Stage is a semantic test adapter containing only an existing UI.
type Stage struct {
	ui *ui.UI
}

// NewStage attaches semantic simulation to u and rejects a nil UI.
func NewStage(u *ui.UI) (*Stage, error) {
	if u == nil {
		return nil, ErrNilUI
	}
	return &Stage{ui: u}, nil
}

// Click activates a named widget through the UI's normal mutation and callback path.
func (s *Stage) Click(name string) bool {
	return s != nil && s.ui != nil && s.ui.Activate(name)
}

// Type focuses and types into a named textbox through the UI's normal edit path.
func (s *Stage) Type(name, value string) bool {
	return s != nil && s.ui != nil && s.ui.TypeText(name, value)
}

// Focus focuses a named focusable widget through the UI's semantic path.
func (s *Stage) Focus(name string) bool {
	return s != nil && s.ui != nil && s.ui.Focus(name)
}

// FocusFrame gives container focus to a named frame through the UI's normal
// focus path without manufacturing pointer state. It mirrors clicking the
// frame background: keyboard focus clears and scoped hotkeys arm.
func (s *Stage) FocusFrame(name string) bool {
	return s != nil && s.ui != nil && s.ui.FocusFrame(name)
}

// PressHotkey routes one bare hotkey press edge through the UI's normal key
// dispatch and reports consumption. It mirrors the gallery's single KeyEvent
// build: text wins while editing, scoped registrations fire only when their
// frame holds container focus, and the rest passes to the host game.
func (s *Stage) PressHotkey(key rune) bool {
	return s != nil && s.ui != nil && s.ui.HandleKey(ui.KeyEvent{Hotkeys: []rune{key}})
}

// ClickLink activates the nth link of a named rich-text widget in segment
// order through the UI's normal mutation and callback path.
func (s *Stage) ClickLink(name string, index int) bool {
	return s != nil && s.ui != nil && s.ui.ActivateLink(name, index)
}

// SelectTab changes a named tab bar selection through the UI's normal
// mutation and callback path without manufacturing pointer state.
func (s *Stage) SelectTab(name string, index int) bool {
	return s != nil && s.ui != nil && s.ui.SelectTab(name, index)
}
