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

// SelectTab changes a named tab bar selection through the UI's normal
// mutation and callback path without manufacturing pointer state.
func (s *Stage) SelectTab(name string, index int) bool {
	return s != nil && s.ui != nil && s.ui.SelectTab(name, index)
}
