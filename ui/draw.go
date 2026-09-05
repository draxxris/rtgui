package ui

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/widgets"
)

// Draw reconciles disabled owners, starts one optional recorder frame, and
// renders registered widgets plus the library-owned dropdown popup.
func (u *UI) Draw() {
	if u == nil || u.theme == nil {
		return
	}
	u.reconcileInteraction()
	u.theme.BeginFrame()
	for _, name := range u.order {
		u.drawOne(u.widgets[name])
	}
	if dropdown := u.openDropdown(); dropdown != nil {
		u.drawDropdownPopup(dropdown)
	}
}

// drawOne creates the sole renderer-facing widget snapshot with UI state.
func (u *UI) drawOne(widget *widgets.Widget) {
	if widget == nil {
		return
	}
	state := u.visualState(widget)
	info := widget.Snapshot(state)
	u.theme.DrawWidget(info, widgetText(widget), widget.Value(), widget.Checked())
	if needsBorder(widget.Kind()) {
		u.theme.DrawWidgetPart(widget.Kind(), skin.PartBorder, widget.Bounds(), state)
	}
	if widget.Kind() == core.WidgetDropdown {
		u.drawDropdownArrow(widget, state)
	}
}

// visualState derives one state from widget availability and UI owner priority.
func (u *UI) visualState(widget *widgets.Widget) core.WidgetState {
	if widget == nil || !widget.Enabled() {
		return core.StateDisabled
	}
	if u.pressed == widget {
		return core.StatePressed
	}
	if u.focused == widget {
		return core.StateFocused
	}
	if u.hovered == widget {
		return core.StateHovered
	}
	return core.StateNormal
}

// drawDropdownArrow renders the popup arrow at the dropdown's right edge.
func (u *UI) drawDropdownArrow(widget *widgets.Widget, state core.WidgetState) {
	bounds := widget.Bounds()
	arrow := core.Rect{X: bounds.X + bounds.W - 34, Y: bounds.Y + 7, W: 28, H: 28}
	u.theme.DrawWidgetPart(widget.Kind(), skin.PartArrow, arrow, state)
}

// drawDropdownPopup renders a copied item snapshot above all registered widgets.
func (u *UI) drawDropdownPopup(widget *widgets.Widget) {
	popup := widget.DropdownPopupBounds()
	if popup.H <= 0 {
		return
	}
	info := widget.Snapshot(core.StatePressed)
	info.Bounds = popup
	u.theme.DrawDropdownPopup(info, widget.DropdownItems(), widget.DropdownIndexAt(u.pointer))
}

// widgetText resolves plain, textbox, or selected dropdown display text.
func widgetText(widget *widgets.Widget) string {
	if widget == nil {
		return ""
	}
	if widget.Kind() == core.WidgetDropdown {
		if value, ok := widget.DropdownSelection(); ok {
			return value
		}
		return ""
	}
	return widget.Text()
}

// needsBorder reports the widget kinds with a separate border part.
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
