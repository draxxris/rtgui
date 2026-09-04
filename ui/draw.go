package ui

import (
	"rtgui/core"
	"rtgui/skin"
	"rtgui/widgets"
)

// Draw renders every registered widget in insertion order through the owned
// theme. It stays headless-safe: with no window ready the theme still appends
// to its draw log and skips GL. Scissor clipping stays app-side, matching the
// gallery and render contracts. Draw errors are ignored to keep the frame loop
// alive; use Theme().DrawLog() in tests to assert output.
func (u *UI) Draw() {
	if u == nil || u.theme == nil {
		return
	}
	for _, name := range u.order {
		u.drawOne(u.widgets[name])
	}
}

// drawOne renders a single widget: background/text/value plus border and
// dropdown-arrow parts where the gallery reference draws them.
func (u *UI) drawOne(w *widgets.Widget) {
	if w == nil {
		return
	}
	if !w.Enabled {
		w.State = core.StateDisabled
	}
	info := w.Info()
	info.HasCapture = u.capture.IsCaptured() && u.capture.ID() == w.ID
	_ = u.theme.DrawWidget(info, widgetText(w), w.Value, w.Checked)
	if needsBorder(w.Kind) {
		_ = u.theme.DrawWidgetPart(w.Kind, skin.PartBorder, w.Bounds, w.State)
	}
	if w.Kind == core.WidgetDropdown {
		u.drawDropdownArrow(w)
	}
}

// drawDropdownArrow renders the popup arrow at the right edge of a dropdown.
func (u *UI) drawDropdownArrow(w *widgets.Widget) {
	arrow := core.Rect{X: w.Bounds.X + w.Bounds.W - 34, Y: w.Bounds.Y + 7, W: 28, H: 28}
	_ = u.theme.DrawWidgetPart(w.Kind, skin.PartArrow, arrow, w.State)
}

// widgetText resolves the display string: textbox buffer, dropdown selection,
// or plain text. Nil-safe for tests driving bare widgets.
func widgetText(w *widgets.Widget) string {
	if w == nil {
		return ""
	}
	if w.TextBuf != nil {
		return w.TextBuf.String()
	}
	if w.Kind == core.WidgetDropdown && w.DropdownIndex >= 0 && w.DropdownIndex < len(w.DropdownItems) {
		return w.DropdownItems[w.DropdownIndex]
	}
	return w.Text
}

// needsBorder mirrors the gallery reference: these kinds get a border part.
func needsBorder(kind core.WidgetKind) bool {
	switch kind {
	case core.WidgetButton,
		core.WidgetLabel,
		core.WidgetTextbox,
		core.WidgetScrollPanel,
		core.WidgetDropdown,
		core.WidgetFrame:
		return true
	default:
		return false
	}
}
