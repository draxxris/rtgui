package ui

import "github.com/draxxris/rtgui/core"

// SetRichTooltip copies rich hover content for a registered widget.
// Empty content removes it. Images remain borrowed until the tooltip is removed.
func (u *UI) SetRichTooltip(name string, data core.RichTooltip) {
	if u.Lookup(name) == nil {
		return
	}
	if !data.HasContent() {
		delete(u.richTips, name)
		return
	}
	if u.richTips == nil {
		u.richTips = make(map[string]core.RichTooltip)
	}
	data.Segments = append([]core.RichSegment(nil), data.Segments...)
	u.richTips[name] = data
}

// ShowRichTooltip copies an explicitly anchored tooltip until HideTooltip.
// Menus and active drags suppress all tooltips without stealing pointer input.
func (u *UI) ShowRichTooltip(data core.RichTooltip, at core.Vec2) {
	if u == nil || u.HasOpenMenu() {
		return
	}
	u.HideTooltip()
	data.Segments = append([]core.RichSegment(nil), data.Segments...)
	u.explicitRichTip = data
	u.tooltipAnchor = at
}

// drawRichTooltipPopup selects explicit or widget content without snapshots.
func (u *UI) drawRichTooltipPopup() bool {
	if u.HasOpenMenu() || u.pressed != nil || u.dragSource != nil {
		return false
	}
	data, anchor := u.explicitRichTip, u.tooltipAnchor
	if !data.HasContent() {
		if u.tooltipText != "" || u.tipText != "" || !u.available(u.hovered) {
			return false
		}
		data, anchor = u.richTips[u.hovered.Name()], u.pointer
	}
	if !data.HasContent() {
		return false
	}
	u.richTipCache.Update(u.theme, data, anchor, u.logicalSize(), u.tooltipPadding())
	u.theme.DrawRichTooltip(core.WidgetInfo{Name: "tooltip", Kind: core.WidgetTooltip}, &u.richTipCache)
	return true
}
