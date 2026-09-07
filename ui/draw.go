package ui

import (
	"fmt"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/render"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/widgets"
	rl "github.com/gen2brain/raylib-go/raylib"
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
// Textboxes render through the caret path so selection and caret draw with
// the same content area used for click mapping.
func (u *UI) drawOne(widget widgets.Widget) {
	if isNilWidget(widget) {
		return
	}
	switch w := widget.(type) {
	case *widgets.Canvas:
		if fn := w.CanvasDraw(); fn != nil {
			fn(w.Bounds())
		}
		return
	case *widgets.ScrollPanel:
		state := u.visualState(w)
		u.drawScrollPanel(w, state)
		return
	case *widgets.TabBar:
		state := u.visualState(w)
		u.drawTabBar(w, state)
		return
	case *widgets.RichText:
		state := u.visualState(w)
		u.drawRichText(w, state)
		return
	case *widgets.Textbox:
		state := u.visualState(w)
		u.drawTextbox(w, state)
		return
	}
	state := u.visualState(widget)
	var val float32
	var chk bool
	if s, ok := widget.(*widgets.Slider); ok {
		val = s.Value()
	} else if p, ok := widget.(*widgets.ProgressBar); ok {
		val = p.Value()
	} else if c, ok := widget.(*widgets.Checkbox); ok {
		chk = c.Checked()
	}
	info := widget.Snapshot(state)
	u.theme.DrawWidget(info, widgetText(widget), val, chk)
	if needsBorder(widget.Kind()) {
		u.theme.DrawWidgetPart(widget.Kind(), skin.PartBorder, widget.Bounds(), state)
	}
	if dd, ok := widget.(*widgets.Dropdown); ok {
		u.drawDropdownArrow(dd, state)
	}
}

// drawScrollPanel renders a scroll panel background, border, and scissored contents.
func (u *UI) drawScrollPanel(widget *widgets.ScrollPanel, state core.WidgetState) {
	info := widget.Snapshot(state)
	u.theme.DrawWidget(info, "", 0, false)
	drawer := widget.ScrollContentDrawer()
	if drawer != nil {
		content := u.scrollContentRect(widget)
		scissorW := content.W
		if widget.MaxScroll().Y > 0 {
			track := u.scrollTrackRect(widget)
			scissorW = track.X - content.X
			if scissorW < 0 {
				scissorW = 0
			}
		}
		scissorH := content.H
		if scissorH < 0 {
			scissorH = 0
		}
		sx, sy := u.Scale()
		if rl.IsWindowReady() {
			rl.BeginScissorMode(int32(content.X*sx), int32(content.Y*sy), int32(scissorW*sx), int32(scissorH*sy))
		}
		drawer(widget.Bounds(), widget.Scroll())
		if rl.IsWindowReady() {
			rl.EndScissorMode()
		}
	}
	u.drawScrollbar(widget)
	if needsBorder(widget.Kind()) {
		u.theme.DrawWidgetPart(widget.Kind(), skin.PartBorder, widget.Bounds(), state)
	}
}

// drawTextbox renders one textbox with its selection and focus caret. The
// caret shows only while focused and enabled; the selection shows whenever
// the buffer holds one. Background comes from DrawTextbox and the border
// draws here so textbox borders match every other widget.
func (u *UI) drawTextbox(widget *widgets.Textbox, state core.WidgetState) {
	info := widget.Snapshot(state)
	selStart, selEnd := -1, -1
	if widget.HasSelection() {
		selStart, selEnd = widget.Selection()
	}
	showCaret := u.focused == widget && widget.Enabled()
	u.theme.DrawTextbox(info, widget.Text(), widget.Caret(), selStart, selEnd, showCaret)
	if needsBorder(widget.Kind()) {
		u.theme.DrawWidgetPart(widget.Kind(), skin.PartBorder, widget.Bounds(), state)
	}
}

// visualState derives one state from widget availability and UI owner priority.
// Container focus shares the focused rank so an active frame glows while a
// textbox inside it keeps the caret; pressed and disabled still win outright.
func (u *UI) visualState(widget widgets.Widget) core.WidgetState {
	if widget == nil || !widget.Enabled() {
		return core.StateDisabled
	}
	if u.pressed == widget {
		return core.StatePressed
	}
	if u.focused == widget || u.activeFrame == widget {
		return core.StateFocused
	}
	if u.hovered == widget {
		return core.StateHovered
	}
	return core.StateNormal
}

// drawDropdownArrow renders the popup arrow at the dropdown's right edge.
func (u *UI) drawDropdownArrow(widget *widgets.Dropdown, state core.WidgetState) {
	bounds := widget.Bounds()
	arrow := core.Rect{X: bounds.X + bounds.W - 34, Y: bounds.Y + 7, W: 28, H: 28}
	u.theme.DrawWidgetPart(widget.Kind(), skin.PartArrow, arrow, state)
}

// dropdownPopupIndex resolves one skin-aware popup row under pos. The
// content area comes from the theme so hit testing always matches the
// drawn rows, including popup border and padding insets.
func (u *UI) dropdownPopupIndex(dropdown *widgets.Dropdown, pos core.Vec2) int {
	if dropdown == nil {
		return -1
	}
	content := u.theme.DropdownPopupContent(dropdown.DropdownPopupBounds(), core.StatePressed)
	return render.DropdownPopupIndex(content, dropdown.DropdownItemCount(), pos)
}

// drawDropdownPopup renders a copied item snapshot above all registered widgets.
func (u *UI) drawDropdownPopup(widget *widgets.Dropdown) {
	popup := widget.DropdownPopupBounds()
	if popup.H <= 0 {
		return
	}
	info := widget.Snapshot(core.StatePressed)
	info.Bounds = popup
	u.theme.DrawDropdownPopup(info, widget.DropdownItems(), u.dropdownPopupIndex(widget, u.pointer))
}

// widgetText resolves plain, textbox, or selected dropdown display text.
func widgetText(widget widgets.Widget) string {
	if isNilWidget(widget) {
		return ""
	}
	if dd, ok := widget.(*widgets.Dropdown); ok {
		if value, ok := dd.DropdownSelection(); ok {
			return value
		}
		return ""
	}
	if s, ok := widget.(*widgets.Slider); ok && s.Format() != "" {
		return fmt.Sprintf(s.Format(), s.Value()*100)
	}
	if p, ok := widget.(*widgets.ProgressBar); ok && p.Format() != "" {
		return fmt.Sprintf(p.Format(), p.Value()*100)
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
		core.WidgetTabBar,
		core.WidgetRichText:
		return true
	default:
		return false
	}
}

// drawRichText renders one message with hover-aware link highlighting.
func (u *UI) drawRichText(message *widgets.RichText, state core.WidgetState) {
	info := message.Snapshot(state)
	segments := message.RichSegments()
	if len(segments) == 0 {
		u.theme.DrawWidget(info, "", 0, false)
		if needsBorder(message.Kind()) {
			u.theme.DrawWidgetPart(message.Kind(), skin.PartBorder, message.Bounds(), state)
		}
		return
	}
	u.theme.DrawRichText(info, segments, u.richHoverSeg(message))
	if needsBorder(message.Kind()) {
		u.theme.DrawWidgetPart(message.Kind(), skin.PartBorder, message.Bounds(), state)
	}
}

// richHoverSeg resolves the highlighted link segment for drawing, or -1.
// The lookup is gated on hover-derived tip state so highlight, press arm,
// and tooltip always agree on the same segment.
func (u *UI) richHoverSeg(message *widgets.RichText) int {
	if message == nil || u.tipWidget != message || u.tipSeg < 0 {
		return -1
	}
	return u.tipSeg
}

// drawTabBar renders one tab strip with pointer-aware cell states.
func (u *UI) drawTabBar(bar *widgets.TabBar, state core.WidgetState) {
	info := bar.Snapshot(state)
	labels := bar.TabLabels()
	if len(labels) == 0 {
		u.theme.DrawWidget(info, "", 0, false)
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
func (u *UI) tabHoverIndex(bar *widgets.TabBar) int {
	if bar == nil || u.hovered != bar {
		return -1
	}
	return u.tabIndexAt(bar, u.pointer)
}

// tabPressedIndex resolves the armed tab cell for drawing, or -1.
func (u *UI) tabPressedIndex(bar *widgets.TabBar) int {
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
