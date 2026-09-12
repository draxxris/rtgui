package render

import (
	"image/color"
	"math"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/widgets"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	// ScrollTrackWidth is the scrollbar track width reserved on overflowing
	// scroll containers. Lists and scroll panels share one geometry so hover,
	// press, and drawn thumbs always agree.
	ScrollTrackWidth = float32(16)
	// ScrollThumbMinHeight bounds scrollbar thumb retention.
	ScrollThumbMinHeight = float32(24)
	// ListSelectedAccent is the gold selection tick drawn on selected rows.
	ListSelectedAccent = float32(3)
)

// ListContent returns the skin-aware list viewport used for row drawing,
// hit testing, and scroll bounds. Callers must use this — never raw list
// bounds — so hovered, pressed, and drawn rows always agree. It is safe on
// a nil theme, where it returns bounds.
func (t *Theme) ListContent(bounds core.Rect, state core.WidgetState, class ...string) core.Rect {
	var background, border skin.SkinDescriptor
	if t != nil {
		background, _ = t.resolveDescriptor(core.WidgetList, skin.PartBackground, state, class...)
		border, _ = t.resolveDescriptor(core.WidgetList, skin.PartBorder, state, class...)
	}
	return t.snap(ContentRect(bounds, background, border))
}

// ListRowRect returns one fixed-height row rect inside content, translated
// by the scroll offset. Rows tile content exactly starting at content.Y.
func ListRowRect(content core.Rect, count int, rowH, scrollY float32, index int) (core.Rect, bool) {
	if count <= 0 || index < 0 || index >= count || rowH <= 0 || content.W <= 0 {
		return core.Rect{}, false
	}
	return core.Rect{X: content.X, Y: content.Y + float32(index)*rowH - scrollY, W: content.W, H: rowH}, true
}

// ListRowAt returns the visible row index under pos, or -1 outside content.
// It inverts ListRowRect exactly; scrolled-off rows never hit.
func ListRowAt(content core.Rect, count int, rowH, scrollY float32, pos core.Vec2) int {
	if count <= 0 || rowH <= 0 || content.W <= 0 || content.H <= 0 {
		return -1
	}
	if pos.X < content.X || pos.X > content.X+content.W || pos.Y < content.Y || pos.Y > content.Y+content.H {
		return -1
	}
	index := int((pos.Y - content.Y + scrollY) / rowH)
	if index < 0 || index >= count {
		return -1
	}
	return index
}

// ListVisibleRange returns the first and last row indices intersecting the
// viewport, clamped to [0, count). An empty range reports last < first.
func ListVisibleRange(contentH float32, count int, rowH, scrollY float32) (int, int) {
	if count <= 0 || rowH <= 0 || contentH <= 0 {
		return 0, -1
	}
	first := int(scrollY / rowH)
	if first < 0 {
		first = 0
	}
	last := int((scrollY + contentH - 1) / rowH)
	if last >= count {
		last = count - 1
	}
	return first, last
}

// ScrollTrackRect returns the vertical scrollbar track along the content
// right edge. It reports false when content cannot host a track.
func ScrollTrackRect(content core.Rect, maxScroll float32) (core.Rect, bool) {
	if maxScroll <= 0 || content.W < ScrollTrackWidth || content.H <= 0 {
		return core.Rect{}, false
	}
	return core.Rect{X: content.X + content.W - ScrollTrackWidth, Y: content.Y, W: ScrollTrackWidth, H: content.H}, true
}

// ScrollThumbRect returns the scrollbar thumb proportional to scroll offset.
func ScrollThumbRect(track core.Rect, scrollY, maxScroll float32) core.Rect {
	if maxScroll <= 0 || track.H <= 0 || track.W <= 0 {
		return core.Rect{}
	}
	total := track.H + maxScroll
	thumbH := track.H * (track.H / total)
	if thumbH < ScrollThumbMinHeight {
		thumbH = ScrollThumbMinHeight
	}
	if thumbH > track.H {
		thumbH = track.H
	}
	progress := scrollY / maxScroll
	if progress < 0 {
		progress = 0
	} else if progress > 1 {
		progress = 1
	}
	return core.Rect{X: track.X, Y: track.Y + progress*(track.H-thumbH), W: track.W, H: thumbH}
}

// ListRowsContent returns the row layout area: the viewport minus the
// scrollbar track when overflowing. Draw and hit-test paths must both use
// this — never raw content — so chevrons never slide under the track.
func ListRowsContent(content core.Rect, maxScroll float32) core.Rect {
	track, ok := ScrollTrackRect(content, maxScroll)
	if !ok {
		return content
	}
	content.W = track.X - content.X
	if content.W < 0 {
		content.W = 0
	}
	return content
}

// DrawList renders a collapsible list shell, visible rows, and scrollbar.
// Rows come from the widget cache without allocation; hovered and pressed
// are visible-row indices (or -1). The thumb state is UI-owned hover/press.
func (t *Theme) DrawList(info core.WidgetInfo, list *widgets.List, hovered, pressed int, thumbState core.WidgetState) {
	if t == nil || list == nil {
		return
	}
	t.drawPart(info.Kind, skin.PartBackground, info.Bounds, info.State, info.Class)
	if t.recorder != nil {
		t.recorder.setLastWidgetInfo(info)
	}
	content := t.ListContent(info.Bounds, info.State, info.Class)
	count := list.VisibleRowCount()
	scrollY := list.ScrollOffset()
	maxScroll := list.MaxScroll()
	rowsContent := ListRowsContent(content, maxScroll)
	if rowsContent.W > 0 && rowsContent.H > 0 && count > 0 && list.TotalHeight() > 0 {
		t.PushClip(rowsContent)
		t.drawListRows(info, list, rowsContent, hovered, pressed)
		t.PopClip()
	}
	t.drawListScrollbar(info, content, scrollY, maxScroll, thumbState)
}

// drawListRows renders the visible window of cached rows with per-row state.
func (t *Theme) drawListRows(info core.WidgetInfo, list *widgets.List, content core.Rect, hovered, pressed int) {
	first, last := list.VisibleRange(content.H)
	selected, _ := list.Selected()
	for index := first; index <= last; index++ {
		row, ok := list.VisibleRowAt(index)
		if !ok {
			continue
		}
		rect, ok := list.RowRect(content, index)
		if !ok {
			continue
		}
		state := listRowState(info.State, row.ID == selected, index == hovered, index == pressed)
		t.drawListRowHighlight(info, rect, state)
		if row.ID == selected {
			t.drawListAccent(info, rect)
		}
		t.drawListSeparator(info, rect)
		t.drawListRowContent(info, list, row, rect, state)
	}
}

// listRowState derives one visual state per row from list availability and
// pointer ownership. Disabled wins, then pressed, selected, hovered.
func listRowState(listState core.WidgetState, selected, hovered, pressed bool) core.WidgetState {
	if listState == core.StateDisabled {
		return core.StateDisabled
	}
	if pressed {
		return core.StatePressed
	}
	if selected {
		return core.StateSelected
	}
	if hovered {
		return core.StateHovered
	}
	return core.StateNormal
}

// drawListRowHighlight records and draws one row background. An authored
// ::highlight replaces the fixed fill; without one the fixed fill draws so
// unskinned lists keep selection, hover, and press feedback. The row state
// selects the descriptor directly so :active rules can match presses.
func (t *Theme) drawListRowHighlight(info core.WidgetInfo, row core.Rect, state core.WidgetState) {
	if state == core.StateNormal || state == core.StateDisabled {
		return
	}
	descriptor, fallback := t.resolveDescriptor(info.Kind, skin.PartOverlay, state, info.Class)
	if !fallback && hasVisualBackground(descriptor) {
		tint := effectiveTint(descriptor, false)
		t.logDrawCall(info.Kind, skin.PartOverlay, state, row, t.snap(row), descriptor, tint, false)
		if rl.IsWindowReady() {
			drawDescriptorBackground(descriptor, t.snap(row), tint)
		}
		return
	}
	tint := listFallbackTint(state)
	t.logDrawCall(info.Kind, skin.PartOverlay, state, row, t.snap(row), skin.SkinDescriptor{}, tint, false)
	if rl.IsWindowReady() {
		rl.DrawRectangleRec(toRaylibRect(t.snap(row)), tint)
	}
}

// listFallbackTint selects the fixed row fill when no ::highlight is authored.
func listFallbackTint(state core.WidgetState) color.RGBA {
	if state == core.StateSelected {
		return color.RGBA{R: 74, G: 58, B: 32, A: 255}
	}
	if state == core.StatePressed {
		return color.RGBA{R: 44, G: 34, B: 20, A: 255}
	}
	return color.RGBA{R: 44, G: 58, B: 78, A: 255}
}

// drawListAccent records and draws the gold selection tick on selected rows.
func (t *Theme) drawListAccent(info core.WidgetInfo, row core.Rect) {
	accent := core.Rect{X: row.X, Y: row.Y, W: ListSelectedAccent, H: row.H}
	tint := color.RGBA{R: 232, G: 200, B: 118, A: 255}
	t.logDrawCall(info.Kind, skin.PartOverlay, core.StateSelected, row, t.snap(accent), skin.SkinDescriptor{}, tint, false)
	if rl.IsWindowReady() {
		rl.DrawRectangleRec(toRaylibRect(t.snap(accent)), tint)
	}
}

// drawListSeparator records and draws one subtle row divider. The divider
// covers exactly one physical pixel (see listDividerRect) so it stays
// visible at every window scale instead of fading on bad subpixel phases.
func (t *Theme) drawListSeparator(info core.WidgetInfo, row core.Rect) {
	line := t.listDividerRect(row)
	tint := color.RGBA{R: 255, G: 255, B: 255, A: 16}
	t.logDrawCall(info.Kind, skin.PartOverlay, core.StateNormal, row, line, skin.SkinDescriptor{}, tint, false)
	if rl.IsWindowReady() {
		rl.DrawRectangleRec(toRaylibRect(line), tint)
	}
}

// listDividerRect quantizes a row-bottom divider to exactly one physical
// pixel anchored inside the row: at downscaled sizes a 1-logical-px line
// would straddle two physical rows and fade, with the worst phase dropping
// below visibility as the window resizes. Anchoring to the row bottom keeps
// the following row's opaque state backgrounds from ever covering it.
// Without a usable transform it falls back to the 1-logical-px line.
func (t *Theme) listDividerRect(row core.Rect) core.Rect {
	fallback := core.Rect{X: row.X, Y: row.Y + row.H - 1, W: row.W, H: 1}
	if t == nil || t.transform == nil {
		return fallback
	}
	_, sy := t.transform.Scale()
	if math.IsNaN(float64(sy)) || math.IsInf(float64(sy), 0) || sy <= 0 {
		return fallback
	}
	bottomPhys := float64(t.transform.ViewportToPhysical(core.Vec2{Y: row.Y + row.H}).Y)
	topPhys := math.Round(bottomPhys) - 1
	if topPhys < 0 {
		topPhys = 0
	}
	top := t.transform.PhysicalToViewport(core.Vec2{Y: float32(topPhys)}).Y
	x, w := row.X, row.W
	if t.GetPixelSnap() {
		x, w = t.transform.Snap(x), t.transform.Snap(w)
	}
	return core.Rect{X: x, Y: top, W: w, H: 1 / sy}
}

// drawListRowContent renders one row: a borrowed Content widget wins over
// the Label fast path, and the expander chevron always draws for categories.
// Row content is render-only; inner widgets never take input.
func (t *Theme) drawListRowContent(info core.WidgetInfo, list *widgets.List, row widgets.ListRow, rect core.Rect, state core.WidgetState) {
	if row.Content != nil {
		t.drawListWidgetRow(info, list, row, rect, state)
		return
	}
	t.drawListSingleLine(info, list, row, rect, state)
}

// drawListSingleLine renders the legacy one-line row with icon and chevron.
func (t *Theme) drawListSingleLine(info core.WidgetInfo, list *widgets.List, row widgets.ListRow, rect core.Rect, state core.WidgetState) {
	x := rect.X + float32(row.Depth)*list.Indent() + 6
	availableW := rect.W
	if row.HasChildren {
		availableW -= widgets.ListChevronWidth
	}
	if row.Icon != "" {
		slot := core.Rect{X: x, Y: rect.Y, W: widgets.ListIconSlotWidth, H: rect.H}
		rowInfo := info
		rowInfo.State = state
		rowInfo.Bounds = rect
		t.drawRichIconBox(rowInfo, row.Icon, slot)
		x += widgets.ListIconSlotWidth + 4
	}
	textW := rect.X + availableW - x - 6
	if textW < 0 {
		textW = 0
	}
	rowInfo := info
	rowInfo.State = state
	rowInfo.Bounds = rect
	t.drawTextInContent(rowInfo, row.Label, core.Rect{X: x, Y: rect.Y, W: textW, H: rect.H}, state)
	if row.HasChildren {
		t.drawListChevron(info, rect, state, row.Expanded)
	}
}

// drawListWidgetRow positions a borrowed Content widget inside the row
// content area (indented, chevron-reserved) and draws its subtree clipped
// to the row. The list owns row chrome (highlight, accent, separator,
// chevron); the content owns everything inside its bounds. Positioning
// mutates the borrowed widget bounds on the owning goroutine, mirroring
// layout arrangement.
func (t *Theme) drawListWidgetRow(info core.WidgetInfo, list *widgets.List, row widgets.ListRow, rect core.Rect, state core.WidgetState) {
	x := rect.X + float32(row.Depth)*list.Indent() + 6
	availableW := rect.W - (x - rect.X)
	if row.HasChildren {
		availableW -= widgets.ListChevronWidth
	}
	contentW := availableW - 6
	if contentW < 0 {
		contentW = 0
	}
	content := core.Rect{X: x, Y: rect.Y, W: contentW, H: rect.H}
	row.Content.SetBounds(content)
	t.PushClip(rect)
	t.drawRowWidget(row.Content, content, state)
	t.PopClip()
	if row.HasChildren {
		t.drawListChevron(info, rect, state, row.Expanded)
	}
}

// drawRowWidget draws one borrowed content widget at its resolved bounds:
// children carry content-local bounds, so each frame level translates its
// children by its own resolved origin without mutating child bounds.
func (t *Theme) drawRowWidget(w widgets.Widget, bounds core.Rect, state core.WidgetState) {
	if t == nil || w == nil || !w.Visible() {
		return
	}
	if frame, ok := w.(*widgets.Frame); ok {
		t.drawRowFrame(frame, bounds, state)
		return
	}
	t.drawRowLeaf(w, bounds, state)
}

// drawRowFrame draws an attached frame background and its children
// translated by the frame origin.
func (t *Theme) drawRowFrame(frame *widgets.Frame, origin core.Rect, state core.WidgetState) {
	info := frame.Snapshot(state)
	info.Bounds = origin
	info.State = state
	t.DrawWidget(info, "", 0, false)
	children := frame.AppendChildren(t.listChildScratch[:0])
	for _, child := range children {
		if child == nil || !child.Visible() {
			continue
		}
		t.drawRowWidget(child, shiftRect(child.Bounds(), origin), state)
	}
	clear(children)
	t.listChildScratch = children[:0]
}

// shiftRect translates content-local bounds by a positioned parent origin.
func shiftRect(local, origin core.Rect) core.Rect {
	local.X += origin.X
	local.Y += origin.Y
	return local
}

// drawRowLeaf draws one content leaf: canvases run their draw function,
// rich text draws wrapped without link activation, and every other kind
// draws through the single-line control path with scratch segments.
func (t *Theme) drawRowLeaf(w widgets.Widget, bounds core.Rect, state core.WidgetState) {
	if bounds.W <= 0 || bounds.H <= 0 {
		return
	}
	if canvas, ok := w.(*widgets.Canvas); ok {
		if fn := canvas.CanvasDraw(); fn != nil {
			fn(bounds)
		}
		return
	}
	info := w.Snapshot(state)
	info.Bounds = bounds
	info.State = state
	if rich, ok := w.(*widgets.RichText); ok {
		t.listSegScratch = rich.CopyRichSegmentsInto(t.listSegScratch[:0])
		segments := t.listSegScratch
		t.DrawRichText(info, segments, -1)
		clear(segments)
		t.listSegScratch = segments[:0]
		return
	}
	value, segments := t.rowLeafText(w)
	t.DrawControl(info, value, segments, 0, false)
	if len(segments) > 0 {
		clear(segments)
		t.listSegScratch = segments[:0]
	}
}

// rowLeafText resolves a leaf's plain text and scratch rich segments.
// Segments stay borrowed until the caller releases the scratch.
func (t *Theme) rowLeafText(w widgets.Widget) (string, []core.RichSegment) {
	provider, ok := w.(widgets.RichProvider)
	if !ok || !provider.HasRichText() {
		return w.Text(), nil
	}
	switch w.Kind() {
	case core.WidgetButton, core.WidgetLabel, core.WidgetFrame, core.WidgetCheckbox:
		t.listSegScratch = provider.CopyRichSegmentsInto(t.listSegScratch[:0])
		return w.Text(), t.listSegScratch
	default:
		return w.Text(), nil
	}
}

// drawListChevron records and draws the category expander triangle.
// Chevrons are geometry-only like textbox carets: right means collapsed,
// down means expanded. The call logs under PartArrow for diagnostics.
func (t *Theme) drawListChevron(info core.WidgetInfo, row core.Rect, state core.WidgetState, expanded bool) {
	size := float32(6)
	cx := row.X + row.W - widgets.ListChevronWidth/2
	cy := row.Y + row.H/2
	var v1, v2, v3 rl.Vector2
	if expanded {
		v1 = rl.NewVector2(cx-size, cy-size/2)
		v2 = rl.NewVector2(cx+size, cy-size/2)
		v3 = rl.NewVector2(cx, cy+size)
	} else {
		v1 = rl.NewVector2(cx-size/2, cy-size)
		v2 = rl.NewVector2(cx-size/2, cy+size)
		v3 = rl.NewVector2(cx+size, cy)
	}
	tint := color.RGBA{R: 201, G: 184, B: 150, A: 255}
	if state == core.StateDisabled {
		tint = color.RGBA{R: 130, G: 130, B: 130, A: 255}
	}
	zone := core.Rect{X: row.X + row.W - widgets.ListChevronWidth, Y: row.Y, W: widgets.ListChevronWidth, H: row.H}
	t.logDrawCall(info.Kind, skin.PartArrow, state, zone, t.snap(zone), skin.SkinDescriptor{}, tint, false)
	if rl.IsWindowReady() {
		rl.DrawTriangle(v1, v2, v3, tint)
	}
}

// drawListScrollbar renders the track and thumb when overflowing. Lists and
// chat logs share this path; parts resolve from info.Kind.
func (t *Theme) drawListScrollbar(info core.WidgetInfo, content core.Rect, scrollY, maxScroll float32, thumbState core.WidgetState) {
	track, ok := ScrollTrackRect(content, maxScroll)
	if !ok {
		return
	}
	t.drawPart(info.Kind, skin.PartTrack, track, core.StateNormal, info.Class)
	thumb := ScrollThumbRect(track, scrollY, maxScroll)
	if thumb.H > 0 && thumb.W > 0 {
		t.drawPart(info.Kind, skin.PartThumb, thumb, thumbState, info.Class)
	}
}
