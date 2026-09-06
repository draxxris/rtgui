package ui

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/render"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/widgets"
)

// Draw reconciles disabled owners, starts one optional recorder frame, and
// renders registered widgets plus library-owned popups (dropdown, menu,
// tooltip). Apps that draw their own layers after Draw keep popups covered;
// use DrawWidgets, then app layers, then DrawPopup to keep popups on top.
func (u *UI) Draw() {
	if u == nil || u.theme == nil {
		return
	}
	u.reconcileInteraction()
	u.theme.BeginFrame()
	u.drawWidgets()
	u.drawPopup()
}

// DrawWidgets reconciles disabled owners, starts one optional recorder
// frame, and renders registered widgets without library-owned popups.
// Follow app-specific layers with DrawPopup so popups stay on top.
func (u *UI) DrawWidgets() {
	if u == nil || u.theme == nil {
		return
	}
	u.reconcileInteraction()
	u.theme.BeginFrame()
	u.drawWidgets()
}

// DrawPopup renders library-owned popups (dropdown, menu, tooltip) into the
// current recorder frame without starting a new one, so widget and app-layer
// calls drawn since DrawWidgets are retained underneath the popups.
func (u *UI) DrawPopup() {
	if u == nil || u.theme == nil {
		return
	}
	u.drawPopup()
}

// drawWidgets renders every registered widget in order without the popup.
func (u *UI) drawWidgets() {
	for _, name := range u.order {
		u.drawOne(u.widgets[name])
	}
}

// drawPopup renders open popups above all registered widgets and app
// layers: dropdown first, then menu, then tooltip on top.
func (u *UI) drawPopup() {
	if dropdown := u.openDropdown(); dropdown != nil {
		u.drawDropdownPopup(dropdown)
	}
	u.drawMenuPopup()
	u.drawTooltipPopup()
}

// drawOne creates the sole renderer-facing widget snapshot with UI state.
// Tab bars render through the dedicated tab path so per-cell skins apply.
func (u *UI) drawOne(widget *widgets.Widget) {
	if widget == nil {
		return
	}
	state := u.visualState(widget)
	if widget.Kind() == core.WidgetTabBar {
		u.drawTabBar(widget, state)
		return
	}
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

// dropdownPopupIndex resolves one skin-aware popup row under pos. The
// content area comes from the theme so hit testing always matches the
// drawn rows, including popup border and padding insets.
func (u *UI) dropdownPopupIndex(dropdown *widgets.Widget, pos core.Vec2) int {
	if dropdown == nil {
		return -1
	}
	content := u.theme.DropdownPopupContent(dropdown.DropdownPopupBounds(), core.StatePressed)
	return render.DropdownPopupIndex(content, dropdown.DropdownItemCount(), pos)
}

// drawDropdownPopup renders a copied item snapshot above all registered widgets.
func (u *UI) drawDropdownPopup(widget *widgets.Widget) {
	popup := widget.DropdownPopupBounds()
	if popup.H <= 0 {
		return
	}
	info := widget.Snapshot(core.StatePressed)
	info.Bounds = popup
	u.theme.DrawDropdownPopup(info, widget.DropdownItems(), u.dropdownPopupIndex(widget, u.pointer))
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
		core.WidgetProgressBar,
		core.WidgetFrame,
		core.WidgetTabBar:
		return true
	default:
		return false
	}
}

// drawTabBar renders one tab strip with pointer-aware cell states.
func (u *UI) drawTabBar(bar *widgets.Widget, state core.WidgetState) {
	info := bar.Snapshot(state)
	labels := bar.TabLabels()
	if len(labels) == 0 {
		u.theme.DrawWidget(info, "", bar.Value(), bar.Checked())
		if needsBorder(bar.Kind()) {
			u.theme.DrawWidgetPart(bar.Kind(), skin.PartBorder, bar.Bounds(), state)
		}
		return
	}
	u.theme.DrawTabBar(info, labels, bar.SelectedTab(), u.tabHoverIndex(bar), u.tabPressedIndex(bar))
	if needsBorder(bar.Kind()) {
		u.theme.DrawWidgetPart(bar.Kind(), skin.PartBorder, bar.Bounds(), state)
	}
}

// tabHoverIndex resolves the pointer tab cell for drawing, or -1. The cell
// lookup is gated on the single hover owner so an overlapped bar never
// renders hover; the pointer only identifies the cell within the owner.
func (u *UI) tabHoverIndex(bar *widgets.Widget) int {
	if bar == nil || u.hovered != bar {
		return -1
	}
	return u.tabIndexAt(bar, u.pointer)
}

// tabPressedIndex resolves the armed tab cell for drawing, or -1.
func (u *UI) tabPressedIndex(bar *widgets.Widget) int {
	if bar == nil || u.pressed != bar {
		return -1
	}
	return u.tabIndexAt(bar, u.pointer)
}

// drawMenuPopup renders the open context menu above widgets and app layers.
func (u *UI) drawMenuPopup() {
	if !u.HasOpenMenu() {
		return
	}
	info := core.WidgetInfo{Name: "menu", Bounds: u.menuBounds, Kind: core.WidgetMenu, State: core.StateNormal}
	u.theme.DrawMenu(info, u.menuItems, u.menuRowAt(u.pointer))
}

// drawTooltipPopup renders the derived hover or explicit tooltip on top.
func (u *UI) drawTooltipPopup() {
	text, anchor, ok := u.derivedTooltip()
	if !ok {
		return
	}
	bounds := render.TooltipOuterBounds(text, anchor, u.logicalSize(), u.tooltipPadding())
	if bounds.W <= 0 || bounds.H <= 0 {
		return
	}
	info := core.WidgetInfo{Name: "tooltip", Bounds: bounds, Kind: core.WidgetTooltip, State: core.StateNormal}
	u.theme.DrawTooltip(info, text)
}

// tooltipPadding returns the maximum authored tooltip content inset so
// outer bounds match the drawn shell. Unskinned tooltips use zero and let
// the renderer apply its fallback inset.
func (u *UI) tooltipPadding() float32 {
	if u == nil || u.theme == nil {
		return 0
	}
	padding := float32(0)
	if background, ok := u.theme.Lookup(core.WidgetTooltip, skin.PartBackground, core.StateNormal); ok {
		padding = maxTooltipInset(padding, background)
	}
	if border, ok := u.theme.Lookup(core.WidgetTooltip, skin.PartBorder, core.StateNormal); ok {
		padding = maxTooltipInset(padding, border)
	}
	return padding
}

// maxTooltipInset folds one descriptor's content insets into the maximum.
func maxTooltipInset(current float32, descriptor skin.SkinDescriptor) float32 {
	for _, inset := range []float32{descriptor.PaddingLeft, descriptor.PaddingTop, descriptor.PaddingRight, descriptor.PaddingBottom} {
		if inset > current {
			current = inset
		}
	}
	if descriptor.HasNinePatch {
		for _, inset := range []float32{float32(descriptor.NinePatch.Left), float32(descriptor.NinePatch.Top), float32(descriptor.NinePatch.Right), float32(descriptor.NinePatch.Bottom)} {
			if inset > current {
				current = inset
			}
		}
	}
	return current
}
