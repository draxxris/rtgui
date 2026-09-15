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
		w := u.widgets[name]
		if w != nil && u.parentWidget(w) == nil {
			u.drawNode(w.Frame(), core.Rect{}, false)
		}
	}
}

// drawPopup renders open popups above all registered widgets and app
// layers: dropdown first, then menu, then tooltip on top.
func (u *UI) drawPopup() {
	if u.drag != nil && u.drag.IsDragging() && u.dragGhost != nil {
		u.dragGhost(u.drag.Ghost())
	}
	if dropdown := u.openDropdown(); dropdown != nil {
		u.drawDropdownPopup(dropdown)
	}
	u.drawMenuPopup()
	u.drawTooltipPopup()
}

// drawOne creates the sole renderer-facing widget snapshot with UI state.
// Rich segments render single-line rich runs for buttons, labels, checks,
// and dropdowns; links stay inert there and activate only in RichText.
// Tab bars render through the dedicated tab path so per-cell skins apply;
// tables use their virtual header/row path for the same reason. Textboxes
// render through the caret path so selection and caret draw with the same
// content area used for click mapping.
// Frames draw only their background here; drawNode paints the frame border
// after children so overlapping body content never covers the ring.
func (u *UI) drawOne(widget widgets.Widget) {
	if isNilWidget(widget) {
		return
	}
	if u.drawOneSpecial(widget) {
		return
	}
	if widget.Kind() == core.WidgetFrame {
		u.drawFrameBackground(widget)
		return
	}
	state := u.visualState(widget)
	val, chk := drawOneValue(widget)
	info := widget.Snapshot(state)
	u.theme.DrawControl(info, widgetText(widget), u.controlSegments(widget), val, chk)
	if needsBorder(widget.Kind()) {
		u.theme.DrawWidgetPart(widget.Kind(), skin.PartBorder, widget.Bounds(), state, widget.Class())
	}
	if dd, ok := widget.(*widgets.Dropdown); ok {
		u.drawDropdownArrow(dd, state)
	}
}

// drawFrameBackground renders a frame's background and text without its border.
func (u *UI) drawFrameBackground(widget widgets.Widget) {
	if isNilWidget(widget) {
		return
	}
	state := u.visualState(widget)
	val, chk := drawOneValue(widget)
	info := widget.Snapshot(state)
	u.theme.DrawControl(info, widgetText(widget), u.controlSegments(widget), val, chk)
}

// drawFrameBorder renders a frame's border ring on top of its children.
func (u *UI) drawFrameBorder(widget widgets.Widget) {
	if isNilWidget(widget) {
		return
	}
	if !needsBorder(widget.Kind()) {
		return
	}
	state := u.visualState(widget)
	u.theme.DrawWidgetPart(widget.Kind(), skin.PartBorder, widget.Bounds(), state, widget.Class())
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
		u.theme.PushClip(core.Rect{X: content.X, Y: content.Y, W: scissorW, H: scissorH})
		drawer(widget.Bounds(), widget.Scroll())
		u.theme.PopClip()
	}
	u.drawScrollbar(widget)
	if needsBorder(widget.Kind()) {
		u.theme.DrawWidgetPart(widget.Kind(), skin.PartBorder, widget.Bounds(), state, widget.Class())
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
		u.theme.DrawWidgetPart(widget.Kind(), skin.PartBorder, widget.Bounds(), state, widget.Class())
	}
}

// visualState derives one state from widget availability and UI owner priority.
// Container focus shares the focused rank so an active frame glows while a
// textbox inside it keeps the caret; pressed and disabled still win outright.
func (u *UI) visualState(widget widgets.Widget) core.WidgetState {
	if !u.available(widget) {
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
	u.theme.DrawWidgetPart(widget.Kind(), skin.PartArrow, arrow, state, widget.Class())
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
// Rows with rich runs draw icons and colors without link activation.
func (u *UI) drawDropdownPopup(widget *widgets.Dropdown) {
	popup := widget.DropdownPopupBounds()
	if popup.H <= 0 {
		return
	}
	info := widget.Snapshot(core.StatePressed)
	info.Bounds = popup
	if widget.HasRichDropdownItems() {
		u.stringScratch = widget.AppendDropdownItems(u.stringScratch[:0])
		u.theme.DrawRichDropdownPopup(info, u.stringScratch, widget.RichDropdownRows(), u.dropdownPopupIndex(widget, u.pointer))
		clear(u.stringScratch)
		return
	}
	u.stringScratch = widget.AppendDropdownItems(u.stringScratch[:0])
	u.theme.DrawDropdownPopup(info, u.stringScratch, u.dropdownPopupIndex(widget, u.pointer))
	clear(u.stringScratch)
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
		core.WidgetRichText,
		core.WidgetList,
		core.WidgetChatLog,
		core.WidgetTable:
		return true
	default:
		return false
	}
}

// drawRichText renders one message with hover-aware link highlighting.
func (u *UI) drawRichText(message *widgets.RichText, state core.WidgetState) {
	info := message.Snapshot(state)
	cache := u.richCache(message)
	if cache.Len() == 0 {
		u.theme.DrawWidget(info, "", 0, false)
		if needsBorder(message.Kind()) {
			u.theme.DrawWidgetPart(message.Kind(), skin.PartBorder, message.Bounds(), state, message.Class())
		}
		return
	}
	u.theme.DrawRichTextLayout(info, cache, u.richHoverSeg(message))
	if needsBorder(message.Kind()) {
		u.theme.DrawWidgetPart(message.Kind(), skin.PartBorder, message.Bounds(), state, message.Class())
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
	u.stringScratch = bar.AppendTabLabels(u.stringScratch[:0])
	labels := u.stringScratch
	defer clear(labels)
	if len(labels) == 0 {
		u.theme.DrawWidget(info, "", 0, false)
		if needsBorder(bar.Kind()) {
			u.theme.DrawWidgetPart(bar.Kind(), skin.PartBorder, bar.Bounds(), state, bar.Class())
		}
		return
	}
	u.theme.DrawTabBar(info, labels, bar.SelectedTab(), u.tabHoverIndex(bar), u.tabPressedIndex(bar))
	if needsBorder(bar.Kind()) {
		u.theme.DrawWidgetPart(bar.Kind(), skin.PartBorder, bar.Bounds(), state, bar.Class())
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
	if u.drawRichTooltipPopup() {
		return
	}
	text, anchor, ok := u.derivedTooltip()
	if !ok {
		u.richTipCache.Invalidate()
		return
	}
	segments := [1]core.RichSegment{{Text: text}}
	u.richTipCache.Update(u.theme, core.RichTooltip{Segments: segments[:]}, anchor, u.logicalSize(), u.tooltipPadding(""))
	u.theme.DrawRichTooltip(core.WidgetInfo{Name: "tooltip", Kind: core.WidgetTooltip}, &u.richTipCache)
}

// drawLineGraph renders cached graph geometry and its optional skin border.
func (u *UI) drawLineGraph(graph *widgets.LineGraph) {
	state := u.visualState(graph)
	u.theme.DrawLineGraph(graph.Snapshot(state), graph)
	u.theme.DrawWidgetPart(graph.Kind(), skin.PartBorder, graph.Bounds(), state, graph.Class())
}

// tooltipPadding returns the maximum authored tooltip content inset so
// outer bounds match the drawn shell. The class variant selects
// Tooltip.<class> when set. Unskinned tooltips use zero and let the
// renderer apply its fallback inset.
func (u *UI) tooltipPadding(class string) float32 {
	if u == nil || u.theme == nil {
		return 0
	}
	padding := float32(0)
	if background, ok := u.theme.Lookup(core.WidgetTooltip, skin.PartBackground, core.StateNormal, class); ok {
		padding = maxTooltipInset(padding, background)
	}
	if border, ok := u.theme.Lookup(core.WidgetTooltip, skin.PartBorder, core.StateNormal, class); ok {
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
