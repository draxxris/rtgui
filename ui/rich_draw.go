package ui

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/widgets"
)

// drawOneSpecial renders dedicated widget paths and reports handling.
func (u *UI) drawOneSpecial(widget widgets.Widget) bool {
	switch w := widget.(type) {
	case *widgets.Canvas:
		if fn := w.CanvasDraw(); fn != nil {
			fn(w.Bounds())
		}
		return true
	case *widgets.ScrollPanel:
		u.drawScrollPanel(w, u.visualState(w))
		return true
	case *widgets.TabBar:
		u.drawTabBar(w, u.visualState(w))
		return true
	case *widgets.RichText:
		u.drawRichText(w, u.visualState(w))
		return true
	case *widgets.Textbox:
		u.drawTextbox(w, u.visualState(w))
		return true
	case *widgets.LineGraph:
		u.drawLineGraph(w)
		return true
	default:
		return false
	}
}

// drawOneValue resolves slider, progress, and checkbox control values.
func drawOneValue(widget widgets.Widget) (float32, bool) {
	if s, ok := widget.(*widgets.Slider); ok {
		return s.Value(), false
	}
	if p, ok := widget.(*widgets.ProgressBar); ok {
		return p.Value(), false
	}
	if c, ok := widget.(*widgets.Checkbox); ok {
		return 0, c.Checked()
	}
	return 0, false
}

// controlSegments resolves a control's rich runs without allocating.
// Dropdowns contribute the selected row's runs by alias (read-only,
// consumed synchronously); buttons, labels, frames, and checkboxes copy
// base runs into reused scratch. Every other kind yields nil, and links
// stay inert outside RichText.
func (u *UI) controlSegments(widget widgets.Widget) []core.RichSegment {
	if dd, ok := widget.(*widgets.Dropdown); ok {
		rows := dd.RichDropdownRows()
		if index := dd.DropdownIndex(); index >= 0 && index < len(rows) && len(rows[index]) > 0 {
			return rows[index]
		}
		return nil
	}
	provider, ok := widget.(widgets.RichProvider)
	if !ok || !provider.HasRichText() {
		return nil
	}
	switch widget.Kind() {
	case core.WidgetButton, core.WidgetLabel, core.WidgetFrame, core.WidgetCheckbox:
		u.richSegScratch = provider.CopyRichSegmentsInto(u.richSegScratch[:0])
		return u.richSegScratch
	default:
		return nil
	}
}
