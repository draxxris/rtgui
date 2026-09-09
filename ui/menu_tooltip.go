package ui

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/render"
)

// MenuItem is one context-menu row. It aliases the renderer-shared shape so
// callers configure menus without importing render.
type MenuItem = core.MenuItem

// ShowContextMenu opens an ephemeral menu anchored at pos with copied items.
// It replaces any open dropdown focus and any existing menu. Empty lists and
// nil receivers are no-ops. The callback runs synchronously on row commit.
// Opening clears container focus; modal dismissal needs a fresh frame click.
func (u *UI) ShowContextMenu(items []MenuItem, pos core.Vec2, onSelect func(string)) {
	if u == nil || len(items) == 0 {
		return
	}
	u.cancelDrag()
	u.scrollThumbDragging = nil
	u.scrollThumbHovered = nil
	u.mouseCaptured = false
	u.clearFocus()
	u.clearActiveFrame()
	u.pressed = nil
	u.setHovered(nil)
	u.menuItems = append([]core.MenuItem(nil), items...)
	u.menuBounds = render.MenuOuterBounds(pos, u.logicalSize(), len(items))
	u.menuOnSelect = onSelect
	u.menuArmed = -1
	u.menuDown = false
	u.tooltipText = ""
	u.disarmLink()
	u.clearLinkTip()
}

// CloseMenu dismisses an open menu without selecting and reports a dismiss.
func (u *UI) CloseMenu() bool {
	if !u.HasOpenMenu() {
		return false
	}
	u.closeMenuState()
	return true
}

// HasOpenMenu reports whether a context menu is currently visible. Openness
// derives from menu items: opening rejects empty input and closing nils them.
func (u *UI) HasOpenMenu() bool { return u != nil && len(u.menuItems) != 0 }

// MenuBounds returns the clamped outer menu bounds while a menu is open.
func (u *UI) MenuBounds() (core.Rect, bool) {
	if !u.HasOpenMenu() {
		return core.Rect{}, false
	}
	return u.menuBounds, true
}

// SetTooltip maps plain hover text to a registered widget. Empty text clears it.
// Widget removal disposes the mapping.
func (u *UI) SetTooltip(name, text string) {
	if w := u.Lookup(name); w != nil {
		w.SetTooltipText(text)
	}
}

// TooltipText returns the mapped tooltip for name, or false when unmapped.
func (u *UI) TooltipText(name string) (string, bool) {
	if u == nil {
		return "", false
	}
	if w := u.widgets[name]; w != nil && w.Tooltip() != "" {
		return w.Tooltip(), true
	}
	return "", false
}

// ShowTooltip shows an explicitly anchored plain-text tooltip at a logical
// point. The tooltip shell sizes itself from wrapped text; the anchor only
// positions it. It is hidden while a menu is open and cleared on press,
// wheel, or Escape.
func (u *UI) ShowTooltip(text string, at core.Vec2) {
	if u == nil || text == "" {
		return
	}
	if u.HasOpenMenu() {
		return
	}
	u.explicitRichTip = core.RichTooltip{}
	u.tooltipText = text
	u.explicitAnchor = at
}

// HideTooltip clears an explicitly shown tooltip and reports a hide.
// It also restarts the hover dwell for the current hover owner so a
// dismissed tooltip does not instantly reappear while the pointer stays put.
func (u *UI) HideTooltip() bool {
	if u == nil {
		return false
	}
	visible := u.tooltipText != "" || u.explicitRichTip.HasContent()
	u.tooltipText = ""
	u.explicitRichTip = core.RichTooltip{}
	u.richTipCache.Invalidate()
	u.resetHoverDwell()
	return visible
}

// logicalSize returns the fixed logical design size for popup clamping.
func (u *UI) logicalSize() core.Vec2 {
	if u == nil || u.transform == nil {
		return core.Vec2{}
	}
	return u.transform.Viewport.LogicalSize
}

// closeMenuState clears ephemeral menu ownership without reporting.
func (u *UI) closeMenuState() {
	u.menuItems = nil
	u.menuBounds = core.Rect{}
	u.menuOnSelect = nil
	u.menuArmed = -1
	u.menuDown = false
}

// menuRowAt resolves the menu row under pos using skin-aware content.
func (u *UI) menuRowAt(pos core.Vec2) int {
	if !u.HasOpenMenu() || u.theme == nil {
		return -1
	}
	content := u.theme.MenuContent(u.menuBounds, core.StateNormal)
	return render.MenuIndexAt(content, len(u.menuItems), pos)
}

// menuSelectable reports whether row is committable (not separator/disabled).
func (u *UI) menuSelectable(index int) bool {
	if u == nil || index < 0 || index >= len(u.menuItems) {
		return false
	}
	return !u.menuItems[index].Separator && !u.menuItems[index].Disabled
}

// commitMenuRow closes the menu before firing the selection callback.
func (u *UI) commitMenuRow(index int) {
	if !u.menuSelectable(index) {
		return
	}
	item := u.menuItems[index]
	callback := u.menuOnSelect
	u.closeMenuState()
	if callback != nil {
		callback(item.ID)
	}
}

// derivedTooltip returns the currently visible tooltip text and anchor point.
// Precedence is explicit tooltip, hovered link tip, then widget mapping.
// Hover-derived tips wait out the effective dwell and resolve through the
// effective anchor; explicit tips draw immediately at their own point.
func (u *UI) derivedTooltip() (string, core.Vec2, bool) {
	if u == nil || u.HasOpenMenu() || u.pressed != nil || u.dragSource != nil {
		return "", core.Vec2{}, false
	}
	if u.tooltipText != "" {
		return u.tooltipText, u.explicitAnchor, true
	}
	if !u.hoverDwellElapsed() {
		return "", core.Vec2{}, false
	}
	anchor := u.resolveHoverAnchor()
	if u.tipWidget != nil && u.tipWidget == u.hovered && u.tipText != "" {
		return u.tipText, anchor, true
	}
	if u.available(u.hovered) {
		if text, ok := u.TooltipText(u.hovered.Name()); ok {
			return text, anchor, true
		}
	}
	return "", core.Vec2{}, false
}
