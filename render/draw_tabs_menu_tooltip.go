package render

import (
	"image/color"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// DrawTabBar renders a tab strip: one bar background plus one PartTab cell
// per label with per-cell state. selected is the active tab index, hovered
// is the pointer tab index, pressed is the armed tab index (or -1).
func (t *Theme) DrawTabBar(info core.WidgetInfo, labels []string, selected, hovered, pressed int) {
	if t == nil || len(labels) == 0 {
		return
	}
	t.drawPart(info.Kind, skin.PartBackground, info.Bounds, info.State, info.Class)
	if t.recorder != nil {
		t.recorder.setLastWidgetInfo(info)
	}
	content := t.TabContent(info.Bounds, info.State, info.Class)
	for index, label := range labels {
		cell, ok := TabTabRect(content, len(labels), index)
		if !ok {
			continue
		}
		state := tabCellState(info.State, index, selected, hovered, pressed)
		t.drawPart(info.Kind, skin.PartTab, cell, state, info.Class)
		t.drawTextInContent(info, label, cell, state)
	}
}

// tabCellState derives one visual state per tab cell from bar availability
// and pointer ownership. Disabled wins, then pressed, selected, hovered.
func tabCellState(bar core.WidgetState, index, selected, hovered, pressed int) core.WidgetState {
	if bar == core.StateDisabled {
		return core.StateDisabled
	}
	if index == pressed && pressed >= 0 {
		return core.StatePressed
	}
	if index == selected {
		return core.StateSelected
	}
	if index == hovered {
		return core.StateHovered
	}
	return core.StateNormal
}

// DrawMenu renders an ephemeral context menu with popup shell parts and one
// highlightable row per item. hovered is the pointer row index, or -1.
func (t *Theme) DrawMenu(info core.WidgetInfo, items []core.MenuItem, hovered int) {
	if t == nil || len(items) == 0 || info.Bounds.H <= 0 {
		return
	}
	t.drawPopupShell(info)
	content := t.MenuContent(info.Bounds, info.State)
	for index, item := range items {
		row, ok := MenuRowRect(content, len(items), index)
		if !ok {
			continue
		}
		if item.Separator {
			t.drawMenuSeparator(info, row)
			continue
		}
		if index == hovered && !item.Disabled {
			t.drawPopupRowHighlight(info, row)
		}
		t.drawMenuText(info, row, item)
	}
}

// drawMenuSeparator records and draws one horizontal rule centered in row.
func (t *Theme) drawMenuSeparator(info core.WidgetInfo, row core.Rect) {
	destination := t.snap(core.Rect{X: row.X + 8, Y: row.Y + (row.H-1)/2, W: row.W - 16, H: 1})
	tint := color.RGBA{R: 120, G: 140, B: 165, A: 140}
	t.logDrawCall(info.Kind, skin.PartOverlay, core.StateNormal, row, destination, skin.SkinDescriptor{}, tint, false)
	if rl.IsWindowReady() {
		rl.DrawRectangleRec(toRaylibRect(destination), tint)
	}
}

// drawMenuText records and draws one menu label with disabled dimming.
func (t *Theme) drawMenuText(info core.WidgetInfo, row core.Rect, item core.MenuItem) {
	tint := color.RGBA{R: 230, G: 240, B: 255, A: 255}
	if item.Disabled {
		tint = color.RGBA{R: 130, G: 140, B: 155, A: 255}
	}
	t.logDrawCall(info.Kind, skin.PartText, info.State, row, row, skin.SkinDescriptor{}, tint, false)
	if !rl.IsWindowReady() {
		return
	}
	if t.HasFont() {
		rl.DrawTextEx(t.FontForSize(22), item.Label, rl.NewVector2(row.X+16, row.Y+6), 22, 2.2, tint)
		return
	}
	rl.DrawText(item.Label, int32(row.X+16), int32(row.Y+6), 22, tint)
}

// DrawTooltip renders a plain-text tooltip shell with wrapped lines.
// info.Bounds is the complete outer bounds computed by TooltipOuterBounds.
func (t *Theme) DrawTooltip(info core.WidgetInfo, text string) {
	if t == nil || text == "" || info.Bounds.W <= 0 || info.Bounds.H <= 0 {
		return
	}
	background, _ := t.drawPart(info.Kind, skin.PartBackground, info.Bounds, info.State)
	border, _ := t.resolveDescriptor(info.Kind, skin.PartBorder, info.State)
	t.DrawWidgetPart(info.Kind, skin.PartBorder, info.Bounds, info.State)
	if t.recorder != nil {
		t.recorder.setLastWidgetInfo(info)
	}
	content := t.snap(ContentRect(info.Bounds, background, border))
	lines := TooltipLines(text, TooltipMaxWidth, TooltipFontSize)
	tint := color.RGBA{R: 230, G: 240, B: 255, A: 255}
	for index, line := range lines {
		row := core.Rect{X: content.X, Y: content.Y + float32(index)*TooltipLineHeight, W: content.W, H: TooltipLineHeight}
		t.logDrawCall(info.Kind, skin.PartText, info.State, row, row, skin.SkinDescriptor{}, tint, false)
		if !rl.IsWindowReady() {
			continue
		}
		if t.HasFont() {
			rl.DrawTextEx(t.FontForSize(TooltipFontSize), line, rl.NewVector2(row.X, row.Y), TooltipFontSize, 2.0, tint)
			continue
		}
		rl.DrawText(line, int32(row.X), int32(row.Y), TooltipFontSize, tint)
	}
}
