// Package widgets provides stateful, renderer-independent UI widgets.
package widgets

import (
	"unicode/utf8"

	"rtgui/core"
	"rtgui/input"
	"rtgui/text"
)

type Widget struct {
	Name          string
	ID            uint32
	Kind          core.WidgetKind
	Bounds        core.Rect
	State         core.WidgetState
	Enabled       bool
	Text          string
	Value         float32
	Checked       bool
	Scroll        core.Vec2
	DropdownIndex int
	DropdownItems []string
	Focused       bool
	TextBuf       *text.Buffer
}

// hashName derives the internal numeric ID from the external string name.
// It mirrors layout.hashID (FNV-1a with offset 2166136261 and prime 16777619,
// with 0 remapped to 1 since Capture zero means no capture). Duplicated here
// intentionally: widgets must not import layout just for the hash.
func hashName(s string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	if h == 0 {
		h = 1
	}
	return h
}

func NewButton(id string, bounds core.Rect, label string) *Widget {
	return &Widget{Name: id, ID: hashName(id), Kind: core.WidgetButton, Bounds: bounds, State: core.StateNormal, Enabled: true, Text: label}
}

func NewLabel(id string, bounds core.Rect, label string) *Widget {
	return &Widget{Name: id, ID: hashName(id), Kind: core.WidgetLabel, Bounds: bounds, Text: label, Enabled: true}
}

func NewCheckbox(id string, bounds core.Rect, checked bool) *Widget {
	return &Widget{Name: id, ID: hashName(id), Kind: core.WidgetCheckbox, Bounds: bounds, Checked: checked, Enabled: true, State: core.StateNormal}
}

func NewTextbox(id string, bounds core.Rect, capacity int) *Widget {
	return &Widget{Name: id, ID: hashName(id), Kind: core.WidgetTextbox, Bounds: bounds, TextBuf: text.NewBuffer(capacity, ""), Enabled: true, State: core.StateNormal}
}

func NewSlider(id string, bounds core.Rect, value float32) *Widget {
	return &Widget{Name: id, ID: hashName(id), Kind: core.WidgetSlider, Bounds: bounds, Value: value, Enabled: true, State: core.StateNormal}
}

func NewProgressBar(id string, bounds core.Rect, value float32) *Widget {
	return &Widget{Name: id, ID: hashName(id), Kind: core.WidgetProgressBar, Bounds: bounds, Value: value, Enabled: true, State: core.StateNormal}
}

func NewScrollPanel(id string, bounds core.Rect) *Widget {
	return &Widget{Name: id, ID: hashName(id), Kind: core.WidgetScrollPanel, Bounds: bounds, Enabled: true, State: core.StateNormal}
}

func NewDropdown(id string, bounds core.Rect, items []string, index int) *Widget {
	return &Widget{Name: id, ID: hashName(id), Kind: core.WidgetDropdown, Bounds: bounds, DropdownItems: items, DropdownIndex: index, Enabled: true, State: core.StateNormal}
}

func NewFrame(id string, bounds core.Rect) *Widget {
	return &Widget{Name: id, ID: hashName(id), Kind: core.WidgetFrame, Bounds: bounds, Enabled: true, State: core.StateNormal}
}

// HitTest expects a point in logical coordinates. Window-to-logical mapping is
// the responsibility of transform.Transform.
func (w *Widget) HitTest(pos core.Vec2) bool { return w != nil && w.Bounds.Contains(pos) }

func (w *Widget) UpdateHover(pos core.Vec2) {
	if w == nil {
		return
	}
	if !w.Enabled {
		w.State = core.StateDisabled
		return
	}
	if w.HitTest(pos) {
		w.State = core.StateHovered
	} else {
		w.State = core.StateNormal
	}
}

func (w *Widget) Press(pos core.Vec2, capture *input.Capture) bool {
	if w == nil || !w.Enabled || !w.HitTest(pos) {
		return false
	}
	w.State = core.StatePressed
	if capture != nil {
		_ = capture.Set(w.ID)
	}
	return true
}

func (w *Widget) Release(pos core.Vec2, capture *input.Capture) bool {
	if w == nil {
		return false
	}
	if capture != nil {
		capture.Release()
	}
	if w.State == core.StatePressed && w.HitTest(pos) {
		w.State = core.StateHovered
		if w.Kind == core.WidgetCheckbox {
			w.Checked = !w.Checked
		}
		return true
	}
	if w.HitTest(pos) {
		w.State = core.StateHovered
	} else {
		w.State = core.StateNormal
	}
	return false
}

func (w *Widget) Focus() {
	if w == nil {
		return
	}
	w.Focused = true
	w.State = core.StateFocused
}

func (w *Widget) Blur() {
	if w == nil {
		return
	}
	w.Focused = false
	if w.State == core.StateFocused {
		w.State = core.StateNormal
	}
}

func (w *Widget) TypeChar(ch rune) {
	if w == nil || w.Kind != core.WidgetTextbox || w.TextBuf == nil {
		return
	}
	w.TextBuf.Set(w.TextBuf.String() + string(ch))
}

func (w *Widget) Backspace() {
	if w == nil || w.Kind != core.WidgetTextbox || w.TextBuf == nil {
		return
	}
	s := w.TextBuf.String()
	if s == "" {
		return
	}
	_, size := utf8.DecodeLastRuneInString(s)
	if size <= 0 || size > len(s) {
		s = ""
	} else {
		s = s[:len(s)-size]
	}
	w.TextBuf.Set(s)
}

func (w *Widget) ScrollBy(dx, dy float32) {
	if w == nil || w.Kind != core.WidgetScrollPanel {
		return
	}
	w.Scroll.X += dx
	w.Scroll.Y += dy
}

func (w *Widget) SetSlider(value float32) {
	if w == nil || w.Kind != core.WidgetSlider {
		return
	}
	if value < 0 {
		value = 0
	}
	if value > 1 {
		value = 1
	}
	w.Value = value
}

func (w *Widget) Info() core.WidgetInfo {
	if w == nil {
		return core.WidgetInfo{}
	}
	return core.WidgetInfo{ID: w.ID, Name: w.Name, Bounds: w.Bounds, Kind: w.Kind, State: w.State}
}
