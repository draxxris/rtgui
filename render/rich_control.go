package render

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
)

// controlContentRect resolves the text area for a control kind, sharing one
// definition between DrawWidget and the rich control paths. Checkboxes use
// full bounds by design so missing box art stays invisible; other kinds
// draw the background part and inset by the border descriptor.
func (t *Theme) controlContentRect(kind core.WidgetKind, bounds core.Rect, state core.WidgetState, class ...string) core.Rect {
	if kind == core.WidgetCheckbox {
		return t.snap(bounds)
	}
	className := ""
	if len(class) > 0 {
		className = class[0]
	}
	background, _ := t.drawPart(kind, skin.PartBackground, bounds, state, className)
	border, _ := t.resolveDescriptor(kind, skin.PartBorder, state, className)
	return t.snap(ContentRect(bounds, background, border))
}

// checkboxLayout splits control content into the box icon and label areas,
// sharing one definition between drawCheckbox and drawControlCheckbox.
func checkboxLayout(content core.Rect) (iconRect, textRect core.Rect) {
	boxSize := float32(20)
	if boxSize > content.H {
		boxSize = content.H
	}
	iconRect = core.Rect{
		X: content.X + 4,
		Y: content.Y + (content.H-boxSize)/2,
		W: boxSize,
		H: boxSize,
	}
	textRect = core.Rect{
		X: content.X + boxSize + 10,
		Y: content.Y,
		W: content.W - (boxSize + 10),
		H: content.H,
	}
	return iconRect, textRect
}

// DrawRichDropdownPopup renders popup rows with per-row rich runs when set.
// Rows without rich runs fall back to plain items with popup styling.
func (t *Theme) DrawRichDropdownPopup(info core.WidgetInfo, items []string, richItems [][]core.RichSegment, hovered int) {
	if t == nil || len(items) == 0 || info.Bounds.H <= 0 {
		return
	}
	t.drawPopupShell(info)
	content := t.DropdownPopupContent(info.Bounds, info.State)
	for index, item := range items {
		row, ok := DropdownPopupRow(content, len(items), index)
		if !ok {
			continue
		}
		if index == hovered {
			t.drawPopupRowHighlight(info, row)
		}
		if index < len(richItems) && len(richItems[index]) > 0 {
			t.drawRichPopupRow(info, row, richItems[index])
			continue
		}
		t.drawDropdownText(info, row, item)
	}
}

// drawRichPopupRow draws one rich popup row inset like plain popup text.
func (t *Theme) drawRichPopupRow(info core.WidgetInfo, row core.Rect, segments []core.RichSegment) {
	content := core.Rect{X: row.X + 16, Y: row.Y, W: row.W - 16, H: row.H}
	if content.W < 0 {
		content.W = 0
	}
	if t.recorder != nil {
		t.recorder.setLastWidgetInfo(info)
	}
	t.DrawRichSingleLine(info, content, segments, core.AlignLeft)
}
