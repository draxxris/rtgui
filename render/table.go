package render

import (
	"image/color"
	"math"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/widgets"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// TableLayoutCache retains relative column slots for one table. Its key
// includes the source columns, shell bounds, scrollbar presence, skin
// revision, CSS class, and pixel-snap mode so draw and hit-test paths cannot
// reuse slots produced for a different track-excluded geometry.
type TableLayoutCache struct {
	slots        []core.Rect
	columns      []widgets.TableColumn
	contentW     float32
	bounds       core.Rect
	hasScrollbar bool
	skinRevision uint64
	class        string
	pixelSnap    bool
	ready        bool
}

// Slots returns the cached relative column slots without allocating. The
// returned slice aliases cache storage and is read-only until the next Update.
func (c *TableLayoutCache) Slots() []core.Rect {
	if c == nil {
		return nil
	}
	return c.slots
}

// Update refreshes relative slots when any layout-affecting key changes.
// Columns are copied only for a rebuild, and the returned slot storage stays
// reusable for warmed draw and hit-test paths. It reports whether rebuilding
// occurred.
func (c *TableLayoutCache) Update(contentW float32, columns []widgets.TableColumn, bounds core.Rect, hasScrollbar bool, skinRevision uint64, class string, pixelSnap bool) bool {
	if c == nil {
		return false
	}
	if c.ready && c.contentW == contentW && c.bounds == bounds && c.hasScrollbar == hasScrollbar && c.skinRevision == skinRevision && c.class == class && c.pixelSnap == pixelSnap && equalRenderTableColumns(c.columns, columns) {
		return false
	}
	c.slots = TableLayoutColumns(contentW, columns)
	c.columns = copyRenderTableColumns(c.columns, columns)
	c.contentW = contentW
	c.bounds = bounds
	c.hasScrollbar = hasScrollbar
	c.skinRevision = skinRevision
	c.class = class
	c.pixelSnap = pixelSnap
	c.ready = true
	return true
}

// TableLayoutColumns resolves relative column weights into contiguous slots.
// Slot X coordinates are relative to the supplied content origin and slot W
// values are logical pixels. MinWidth and MaxWidth are applied through a
// bounded redistribution pass; zero or negative weights share space equally
// only when every column has a zero weight. Excess minimum widths are clipped
// from the right so the result never introduces horizontal scrolling.
func TableLayoutColumns(contentW float32, columns []widgets.TableColumn) []core.Rect {
	if len(columns) == 0 {
		return nil
	}
	width := finiteNonNegative(contentW)
	widths := make([]float64, len(columns))
	fixed := make([]bool, len(columns))
	weights, totalWeight := tableColumnWeights(columns)
	remainingWidth := float64(width)
	remainingWeight := totalWeight
	resolveTableColumnClamps(columns, weights, widths, fixed, &remainingWidth, &remainingWeight)
	assignTableColumnRemainder(columns, weights, widths, fixed, remainingWidth, remainingWeight)
	fitTableColumnWidths(columns, widths, float64(width))
	return tableColumnRects(widths, width)
}

// TableRowsContent returns the track-excluded width shared by table headers
// and rows. The input is the raw shell content; when maxScroll is positive,
// the common scrollbar track is removed from its right edge. Fitting tables
// retain the full content rectangle, and degenerate widths clamp to zero.
func TableRowsContent(content core.Rect, maxScroll float32) core.Rect {
	return fixedRowsViewport(content, maxScroll)
}

// tableColumnWeights sanitizes relative weights and supplies equal fallback
// weights when no positive weight was authored.
func tableColumnWeights(columns []widgets.TableColumn) ([]float64, float64) {
	weights := make([]float64, len(columns))
	total := float64(0)
	for index, column := range columns {
		if column.Width > 0 && finiteFloat32(column.Width) {
			weights[index] = float64(column.Width)
			total += weights[index]
		}
	}
	if total > 0 {
		return weights, total
	}
	for index := range weights {
		weights[index] = 1
	}
	return weights, float64(len(weights))
}

// resolveTableColumnClamps fixes columns whose weighted proposal violates a
// bound, then leaves the remaining space and weight for later columns.
func resolveTableColumnClamps(columns []widgets.TableColumn, weights, widths []float64, fixed []bool, remainingWidth, remainingWeight *float64) {
	for pass := 0; pass <= len(columns); pass++ {
		changed := false
		for index, column := range columns {
			if fixed[index] {
				continue
			}
			proposal := weightedColumnProposal(weights[index], *remainingWidth, *remainingWeight)
			minimum, maximum := tableColumnBounds(column)
			if proposal < minimum {
				widths[index] = minimum
				fixed[index] = true
				*remainingWidth -= minimum
				*remainingWeight -= weights[index]
				changed = true
			} else if proposal > maximum {
				widths[index] = maximum
				fixed[index] = true
				*remainingWidth -= maximum
				*remainingWeight -= weights[index]
				changed = true
			}
		}
		if !changed {
			return
		}
		if *remainingWidth < 0 {
			*remainingWidth = 0
		}
		if *remainingWeight < 0 {
			*remainingWeight = 0
		}
	}
}

// weightedColumnProposal computes one active column's proportional share.
func weightedColumnProposal(weight, remainingWidth, remainingWeight float64) float64 {
	if remainingWeight <= 0 || weight <= 0 {
		return 0
	}
	return remainingWidth * weight / remainingWeight
}

// assignTableColumnRemainder distributes the remaining active width by weight.
func assignTableColumnRemainder(columns []widgets.TableColumn, weights, widths []float64, fixed []bool, remainingWidth, remainingWeight float64) {
	for index, column := range columns {
		if fixed[index] {
			continue
		}
		proposal := weightedColumnProposal(weights[index], remainingWidth, remainingWeight)
		minimum, maximum := tableColumnBounds(column)
		if proposal < minimum {
			proposal = minimum
		}
		if proposal > maximum {
			proposal = maximum
		}
		widths[index] = proposal
	}
}

// fitTableColumnWidths corrects floating remainders and clips overflow from
// the right, preserving earlier columns when minimums cannot all fit.
func fitTableColumnWidths(columns []widgets.TableColumn, widths []float64, total float64) {
	used := sumTableColumnWidths(widths)
	if used > total {
		reduceTableColumnWidths(widths, used-total)
		return
	}
	distributeTableColumnSpace(columns, widths, total-used)
}

// sumTableColumnWidths returns the finite-width total before slot placement.
func sumTableColumnWidths(widths []float64) float64 {
	total := float64(0)
	for _, width := range widths {
		if width > 0 && !math.IsInf(width, 0) && !math.IsNaN(width) {
			total += width
		}
	}
	return total
}

// reduceTableColumnWidths truncates the rightmost widths when bounds overflow.
func reduceTableColumnWidths(widths []float64, overflow float64) {
	for index := len(widths) - 1; index >= 0 && overflow > 0; index-- {
		reduction := widths[index]
		if reduction > overflow {
			reduction = overflow
		}
		widths[index] -= reduction
		overflow -= reduction
	}
}

// distributeTableColumnSpace uses available upper-bound capacity from right
// to left so rounding or clamp gaps do not create avoidable blank columns.
func distributeTableColumnSpace(columns []widgets.TableColumn, widths []float64, remainder float64) {
	if remainder <= 0 {
		return
	}
	for index := len(widths) - 1; index >= 0 && remainder > 0; index-- {
		_, maximum := tableColumnBounds(columns[index])
		capacity := maximum - widths[index]
		if math.IsInf(capacity, 1) {
			capacity = remainder
		}
		if capacity <= 0 {
			continue
		}
		if capacity > remainder {
			capacity = remainder
		}
		widths[index] += capacity
		remainder -= capacity
	}
}

// tableColumnRects turns relative widths into contiguous zero-height slots.
func tableColumnRects(widths []float64, total float32) []core.Rect {
	slots := make([]core.Rect, len(widths))
	position := float64(0)
	for index, width := range widths {
		width = finiteNonNegative64(width)
		available := float64(total) - position
		if available <= 0 {
			width = 0
		} else if width > available {
			width = available
		}
		slots[index] = core.Rect{X: float32(position), W: float32(width)}
		position += width
	}
	return slots
}

// tableColumnBounds returns sanitized lower and upper pixel bounds.
func tableColumnBounds(column widgets.TableColumn) (float64, float64) {
	minimum := float64(0)
	if column.MinWidth > 0 && finiteFloat32(column.MinWidth) {
		minimum = float64(column.MinWidth)
	}
	maximum := math.Inf(1)
	if column.MaxWidth > 0 && finiteFloat32(column.MaxWidth) {
		maximum = float64(column.MaxWidth)
	}
	if maximum < minimum {
		maximum = minimum
	}
	return minimum, maximum
}

// TableHeaderCellRect returns one header slot translated into content space.
// It clips the slot to content so a track-excluded width remains authoritative
// at the right edge. The rectangle uses half-open hit-test coordinates and
// reports false for empty, invalid, or out-of-range slots.
func TableHeaderCellRect(content core.Rect, columnSlots []core.Rect, columnIndex int, headerHeight float32) (core.Rect, bool) {
	if !validTableContent(content) || columnIndex < 0 || columnIndex >= len(columnSlots) || !validTableHeight(headerHeight) {
		return core.Rect{}, false
	}
	height := minTableFloat(headerHeight, content.H)
	if height <= 0 || !finiteFloat32(columnSlots[columnIndex].X) || !finiteFloat32(columnSlots[columnIndex].W) || columnSlots[columnIndex].W <= 0 {
		return core.Rect{}, false
	}
	slot := columnSlots[columnIndex]
	left := maxTableFloat(content.X, content.X+slot.X)
	right := minTableFloat(content.X+content.W, content.X+slot.X+slot.W)
	if right <= left {
		return core.Rect{}, false
	}
	return core.Rect{X: left, Y: content.Y, W: right - left, H: height}, true
}

// TableRowRect returns one raw data-row rectangle below the fixed header.
// Rows may extend past the bottom and are clipped by callers to content; an
// invalid metric, empty rows viewport, track-excluded zero width, or index
// outside count reports false.
func TableRowRect(content core.Rect, count int, rowHeight, headerHeight, scrollY float32, index int) (core.Rect, bool) {
	if !validTableContent(content) || count <= 0 || index < 0 || index >= count || !validTableHeight(rowHeight) || !validTableHeight(headerHeight) || content.H <= headerHeight {
		return core.Rect{}, false
	}
	scrollY = normalizedTableScroll(scrollY)
	y := float64(content.Y) + float64(headerHeight) + float64(index)*float64(rowHeight) - float64(scrollY)
	if math.IsNaN(y) || math.IsInf(y, 0) {
		return core.Rect{}, false
	}
	return core.Rect{X: content.X, Y: float32(y), W: content.W, H: rowHeight}, true
}

// TableRowAt returns a sorted-view row index under pos and whether the point
// is in the fixed header. It uses half-open content/header/row edges, so the
// right edge reserved for a scrollbar never becomes a row hit; callers can
// test the track first and therefore give the track precedence.
func TableRowAt(content core.Rect, count int, rowHeight, headerHeight, scrollY float32, pos core.Vec2) (int, bool) {
	if !validTableContent(content) || !validTableHeight(headerHeight) || !validTableHeight(rowHeight) || !tableContainsHalfOpen(content, pos) {
		return -1, false
	}
	headerBottom := minTableFloat(content.Y+content.H, content.Y+headerHeight)
	if pos.Y < headerBottom {
		return -1, true
	}
	if pos.Y >= content.Y+content.H || headerBottom >= content.Y+content.H || count <= 0 {
		return -1, false
	}
	scrollY = normalizedTableScroll(scrollY)
	rowOffset := (float64(pos.Y) - float64(content.Y) - float64(headerHeight) + float64(scrollY)) / float64(rowHeight)
	if rowOffset < 0 || math.IsNaN(rowOffset) || math.IsInf(rowOffset, 0) {
		return -1, false
	}
	index := int(math.Floor(rowOffset))
	if index < 0 || index >= count {
		return -1, false
	}
	return index, false
}

// TableCellAt returns the column index under a row point, or -1 outside the
// track-excluded content, row, or any positive-width column slot. Adjacent
// slots use half-open edges, so a shared boundary belongs to the next column.
func TableCellAt(content core.Rect, columnSlots []core.Rect, row core.Rect, pos core.Vec2) int {
	if !validTableContent(content) || !tableContainsHalfOpen(row, pos) || !tableContainsHalfOpen(content, pos) {
		return -1
	}
	for index, slot := range columnSlots {
		if !finiteFloat32(slot.X) || !finiteFloat32(slot.W) || slot.W <= 0 {
			continue
		}
		left := maxTableFloat(content.X, content.X+slot.X)
		right := minTableFloat(content.X+content.W, content.X+slot.X+slot.W)
		if right > left && pos.X >= left && pos.X < right {
			return index
		}
	}
	return -1
}

// TableVisibleRange returns the first and last data rows intersecting the
// rows viewport. contentH includes the fixed header, so the explicit rows
// viewport is contentH-headerHeight; half-open boundaries make fractional
// scroll and exact bottom edges deterministic.
func TableVisibleRange(contentH float32, count int, rowHeight, headerHeight, scrollY float32) (int, int) {
	if count <= 0 || !validTableHeight(rowHeight) || !validTableHeight(headerHeight) || !finiteFloat32(contentH) || contentH <= headerHeight {
		return 0, -1
	}
	scrollY = normalizedTableScroll(scrollY)
	rowsHeight := float64(contentH - headerHeight)
	firstValue := math.Floor(float64(scrollY) / float64(rowHeight))
	lastExclusive := math.Ceil((float64(scrollY) + rowsHeight) / float64(rowHeight))
	if firstValue < 0 {
		firstValue = 0
	}
	if lastExclusive <= 0 || firstValue >= float64(count) {
		return 0, -1
	}
	lastValue := lastExclusive - 1
	if lastValue >= float64(count) {
		lastValue = float64(count - 1)
	}
	if lastValue < firstValue {
		return 0, -1
	}
	return int(firstValue), int(lastValue)
}

// validTableContent rejects non-finite or empty content dimensions.
func validTableContent(content core.Rect) bool {
	return finiteFloat32(content.X) && finiteFloat32(content.Y) && content.W > 0 && content.H > 0 && finiteFloat32(content.W) && finiteFloat32(content.H)
}

// tableContainsHalfOpen applies the shared [left,right) and [top,bottom)
// convention used by all table geometry and track-excluded hit testing.
func tableContainsHalfOpen(rect core.Rect, pos core.Vec2) bool {
	return validTableContent(rect) && finiteFloat32(pos.X) && finiteFloat32(pos.Y) && pos.X >= rect.X && pos.X < rect.X+rect.W && pos.Y >= rect.Y && pos.Y < rect.Y+rect.H
}

// validTableHeight reports whether a fixed table metric is drawable.
func validTableHeight(value float32) bool {
	return value > 0 && finiteFloat32(value)
}

// normalizedTableScroll turns invalid or negative scroll into the origin.
func normalizedTableScroll(value float32) float32 {
	if value <= 0 || !finiteFloat32(value) {
		return 0
	}
	return value
}

// finiteFloat32 reports whether a float32 is neither NaN nor infinite.
func finiteFloat32(value float32) bool {
	return !math.IsNaN(float64(value)) && !math.IsInf(float64(value), 0)
}

// finiteNonNegative clamps invalid or negative layout width to zero.
func finiteNonNegative(value float32) float32 {
	if value <= 0 || !finiteFloat32(value) {
		return 0
	}
	return value
}

// finiteNonNegative64 clamps invalid or negative layout width to zero.
func finiteNonNegative64(value float64) float64 {
	if value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return value
}

// minTableFloat returns the smaller logical coordinate.
func minTableFloat(left, right float32) float32 {
	if left < right {
		return left
	}
	return right
}

// maxTableFloat returns the larger logical coordinate.
func maxTableFloat(left, right float32) float32 {
	if left > right {
		return left
	}
	return right
}

// equalRenderTableColumns compares cached column keys without allocation.
func equalRenderTableColumns(left, right []widgets.TableColumn) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

// copyRenderTableColumns copies a changed column key into reusable storage.
func copyRenderTableColumns(dst []widgets.TableColumn, source []widgets.TableColumn) []widgets.TableColumn {
	oldLen := len(dst)
	if cap(dst) < len(source) {
		dst = make([]widgets.TableColumn, len(source))
	} else {
		for index := len(source); index < oldLen; index++ {
			dst[index] = widgets.TableColumn{}
		}
		dst = dst[:len(source)]
	}
	copy(dst, source)
	return dst
}

// TableContent returns the skin-aware table shell viewport. The same content
// rectangle is used by UI hit testing and by DrawTableWithState, so border and padding
// never make a header or row appear clickable outside the painted area.
func (t *Theme) TableContent(bounds core.Rect, state core.WidgetState, class ...string) core.Rect {
	return t.fixedRowsContent(core.WidgetTable, bounds, state, class...)
}

// TableHeaderRect returns the fixed header viewport at the top of content.
// Its width is inherited unchanged from the track-excluded content rectangle.
func TableHeaderRect(content core.Rect, headerHeight float32) core.Rect {
	if !validTableContent(content) || !validTableHeight(headerHeight) {
		return core.Rect{}
	}
	if headerHeight > content.H {
		headerHeight = content.H
	}
	return core.Rect{X: content.X, Y: content.Y, W: content.W, H: headerHeight}
}

// TableRowsRect returns the data-row viewport below the fixed header. It
// retains the exact width of content, including the scrollbar exclusion.
func TableRowsRect(content core.Rect, headerHeight float32) core.Rect {
	if !validTableContent(content) || !validTableHeight(headerHeight) {
		return core.Rect{}
	}
	if headerHeight > content.H {
		headerHeight = content.H
	}
	return core.Rect{X: content.X, Y: content.Y + headerHeight, W: content.W, H: content.H - headerHeight}
}

// TableCellRect returns the positive-width intersection of one row and one
// relative column slot. It is the draw-side counterpart to TableCellAt.
func TableCellRect(content core.Rect, columnSlots []core.Rect, row core.Rect, columnIndex int) (core.Rect, bool) {
	if !validTableContent(content) || columnIndex < 0 || columnIndex >= len(columnSlots) || row.W <= 0 || row.H <= 0 || !finiteFloat32(row.X) || !finiteFloat32(row.Y) || !finiteFloat32(row.W) || !finiteFloat32(row.H) {
		return core.Rect{}, false
	}
	slot := columnSlots[columnIndex]
	if !finiteFloat32(slot.X) || !finiteFloat32(slot.W) || slot.W <= 0 {
		return core.Rect{}, false
	}
	left := maxTableFloat(content.X, maxTableFloat(row.X, content.X+slot.X))
	right := minTableFloat(content.X+content.W, minTableFloat(row.X+row.W, content.X+slot.X+slot.W))
	top := maxTableFloat(content.Y, row.Y)
	bottom := minTableFloat(content.Y+content.H, row.Y+row.H)
	if right <= left || bottom <= top {
		return core.Rect{}, false
	}
	return core.Rect{X: left, Y: top, W: right - left, H: bottom - top}, true
}

// TableDrawState carries pointer-owned stable row and column IDs into a draw.
// IDs remain valid when sorting or data refreshes changes view indexes.
type TableDrawState struct {
	HoveredRow    string
	PressedRow    string
	HoveredColumn string
	PressedColumn string
	// The explicit ID spellings are accepted as aliases for callers that make
	// the stable identity contract visible at the call site.
	HoveredRowID    string
	PressedRowID    string
	HoveredColumnID string
	PressedColumnID string
	ThumbState      core.WidgetState
	// ColumnSlots are relative to the track-excluded content rectangle. UI
	// supplies its owned cache here; a direct draw may leave them empty and
	// the theme will resolve a one-shot fallback layout.
	ColumnSlots []core.Rect
}

// DrawTableWithState is the typed stable-ID table draw entry point. The state
// is owned by UI; table data remains read-only while the frame draws.
func (t *Theme) DrawTableWithState(info core.WidgetInfo, table *widgets.Table, drawState TableDrawState) {
	if t == nil || table == nil {
		return
	}
	normalizeTableDrawIDs(&drawState)
	t.drawPart(info.Kind, skin.PartBackground, info.Bounds, info.State, info.Class)
	if t.recorder != nil {
		t.recorder.setLastWidgetInfo(info)
	}
	content := t.TableContent(info.Bounds, info.State, info.Class)
	// UI reconciles this skin-aware viewport before drawing. The renderer keeps
	// the widget's current limit untouched so direct draws remain read-only.
	maxScroll := table.MaxScroll()
	rowsContent := TableRowsContent(content, maxScroll)
	slots := drawTableSlots(table, rowsContent.W, drawState.ColumnSlots)
	t.drawTableHeader(info, table, rowsContent, slots, drawState)
	t.drawTableRows(info, table, rowsContent, slots, drawState)
	t.drawListScrollbar(info, content, table.ScrollOffset(), maxScroll, drawState.ThumbState)
}

// normalizeTableDrawIDs folds explicit stable-ID aliases into canonical fields.
func normalizeTableDrawIDs(state *TableDrawState) {
	if state == nil {
		return
	}
	if state.HoveredRow == "" {
		state.HoveredRow = state.HoveredRowID
	}
	if state.PressedRow == "" {
		state.PressedRow = state.PressedRowID
	}
	if state.HoveredColumn == "" {
		state.HoveredColumn = state.HoveredColumnID
	}
	if state.PressedColumn == "" {
		state.PressedColumn = state.PressedColumnID
	}
}

// drawTableSlots accepts UI-owned slots and resolves a safe direct-draw
// fallback when no matching slots were supplied.
func drawTableSlots(table *widgets.Table, contentW float32, slots []core.Rect) []core.Rect {
	if table == nil {
		return nil
	}
	if len(slots) == table.ColumnCount() {
		return slots
	}
	return TableLayoutColumns(contentW, table.Columns())
}

// drawTableHeader draws the fixed header under its own clip, including titles
// and the active geometry-only sort arrow.
func (t *Theme) drawTableHeader(info core.WidgetInfo, table *widgets.Table, content core.Rect, slots []core.Rect, drawState TableDrawState) {
	header := TableHeaderRect(content, table.HeaderHeight())
	if header.W <= 0 || header.H <= 0 {
		return
	}
	t.PushClip(header)
	t.drawTableHeaderBackground(info, header)
	for index := 0; index < table.ColumnCount(); index++ {
		column, ok := table.ColumnAt(index)
		if !ok {
			continue
		}
		cell, ok := TableHeaderCellRect(content, slots, index, table.HeaderHeight())
		if !ok {
			continue
		}
		cellState := tableCellState(info.State, column.ID, drawState.HoveredColumn, drawState.PressedColumn)
		sortColumn, sortDir := table.SortColumn()
		t.PushClip(cell)
		t.drawTableHeaderText(info, cell, column, cellState, sortColumn, sortDir)
		t.PopClip()
	}
	t.PopClip()
}

// drawTableHeaderBackground paints the authored header rule or a restrained
// fallback fill. The fallback remains visible only as pixels; the recorder
// still receives the PartHeader operation either way.
func (t *Theme) drawTableHeaderBackground(info core.WidgetInfo, header core.Rect) {
	descriptor, fallback := t.resolveDescriptor(info.Kind, skin.PartHeader, info.State, info.Class)
	tint := effectiveTint(descriptor, fallback)
	destination := t.snap(header)
	t.logDrawCall(info.Kind, skin.PartHeader, info.State, header, destination, descriptor, tint, fallback)
	if !rl.IsWindowReady() {
		return
	}
	if !fallback && hasVisualBackground(descriptor) {
		drawDescriptorBackground(descriptor, destination, tint)
		return
	}
	if fallback {
		rl.DrawRectangleRec(toRaylibRect(destination), color.RGBA{R: 34, G: 44, B: 61, A: 255})
	}
}

// drawTableHeaderText draws one title and reserves arrow space for sortable
// columns. Header styling is made explicit on the info snapshot so the shared
// drawTextInContent helper applies exactly one content inset.
func (t *Theme) drawTableHeaderText(info core.WidgetInfo, cell core.Rect, column widgets.TableColumn, state core.WidgetState, sortColumn string, sortDir core.SortDir) {
	headerInfo := info
	headerInfo.Bounds = cell
	headerInfo.State = state
	header, _ := t.resolveDescriptor(info.Kind, skin.PartHeader, state, info.Class)
	if header.HasTextColor && !headerInfo.HasTextColor {
		headerInfo.TextColor = header.TextColor
		headerInfo.HasTextColor = true
	}
	if header.HasFontSize && headerInfo.FontSize <= 0 {
		headerInfo.FontSize = header.FontSize
	}
	headerInfo.Align = tableDefaultAlign(info.Align, column.Align)
	if column.Numeric {
		headerInfo.Align = core.AlignRight
	}
	text := cell
	if column.Sortable {
		text.W -= TableSortArrowWidth
		if text.W < 0 {
			text.W = 0
		}
	}
	t.drawTextInContent(headerInfo, column.Title, text, state)
	if sortColumn == column.ID && (sortDir == core.SortAsc || sortDir == core.SortDesc) {
		t.drawTableSortArrow(info, cell, state, sortDir)
	}
}

// drawTableSortArrow draws a tint-driven triangle and records it as PartArrow.
// CSS cannot supply an image for this part; geometry remains deterministic.
func (t *Theme) drawTableSortArrow(info core.WidgetInfo, cell core.Rect, state core.WidgetState, direction core.SortDir) {
	width := TableSortArrowWidth
	if width > cell.W {
		width = cell.W
	}
	zone := core.Rect{X: cell.X + cell.W - width, Y: cell.Y, W: width, H: cell.H}
	descriptor, fallback := t.resolveDescriptor(info.Kind, skin.PartArrow, state, info.Class)
	tint := tableArrowTint(descriptor, fallback)
	t.logDrawCall(info.Kind, skin.PartArrow, state, zone, t.snap(zone), descriptor, tint, fallback)
	if !rl.IsWindowReady() || zone.W <= 0 || zone.H <= 0 {
		return
	}
	size := float32(6)
	if size > zone.W*0.45 {
		size = zone.W * 0.45
	}
	if size > zone.H*0.35 {
		size = zone.H * 0.35
	}
	if size <= 0 {
		return
	}
	cx := zone.X + zone.W/2
	cy := zone.Y + zone.H/2
	var first, second, third rl.Vector2
	if direction == core.SortAsc {
		first = rl.NewVector2(cx-size/2, cy+size/2)
		second = rl.NewVector2(cx+size/2, cy+size/2)
		third = rl.NewVector2(cx, cy-size/2)
	} else {
		first = rl.NewVector2(cx-size/2, cy-size/2)
		second = rl.NewVector2(cx+size/2, cy-size/2)
		third = rl.NewVector2(cx, cy+size/2)
	}
	rl.DrawTriangle(first, second, third, tint)
}

// tableArrowTint lets both background-image-tint and color style an arrow.
func tableArrowTint(descriptor skin.SkinDescriptor, fallback bool) color.RGBA {
	if fallback {
		return color.RGBA{R: 201, G: 184, B: 150, A: 255}
	}
	if descriptor.HasBackgroundColor {
		return descriptor.BackgroundColor.RGBA()
	}
	if descriptor.HasTextColor {
		return descriptor.TextColor.RGBA()
	}
	return descriptor.Tint.RGBA()
}

// drawTableRows renders only the sorted-view window intersecting the rows
// viewport. Row backgrounds precede cell content and every cell gets a
// nested clip so long text or icons cannot cross a column boundary.
func (t *Theme) drawTableRows(info core.WidgetInfo, table *widgets.Table, content core.Rect, slots []core.Rect, drawState TableDrawState) {
	rows := TableRowsRect(content, table.HeaderHeight())
	if rows.W <= 0 || rows.H <= 0 || table.ViewRowCount() == 0 {
		return
	}
	first, last := TableVisibleRange(content.H, table.ViewRowCount(), table.RowHeight(), table.HeaderHeight(), table.ScrollOffset())
	if last < first {
		return
	}
	selected, _ := table.Selected()
	t.PushClip(rows)
	for index := first; index <= last; index++ {
		rowID, ok := table.ViewRowIDAt(index)
		if !ok {
			continue
		}
		row, ok := TableRowRect(content, table.ViewRowCount(), table.RowHeight(), table.HeaderHeight(), table.ScrollOffset(), index)
		if !ok {
			continue
		}
		state := tableRowState(info.State, rowID == selected, rowID == drawState.HoveredRow, rowID == drawState.PressedRow)
		if index%2 == 1 {
			t.drawTableStripe(info, row)
		}
		t.drawFixedRowHighlight(info, row, state)
		if rowID == selected {
			t.drawFixedRowAccent(info, row, TableSelectedAccent)
		}
		t.drawFixedRowSeparator(info, row)
		for columnIndex := 0; columnIndex < table.ColumnCount(); columnIndex++ {
			column, ok := table.ColumnAt(columnIndex)
			if !ok {
				continue
			}
			cell, ok := table.ViewCellAt(index, columnIndex)
			if !ok {
				continue
			}
			t.drawTableCell(info, rows, slots, row, column, cell, columnIndex, state)
		}
	}
	t.PopClip()
}

// tableRowState derives a row state from table ownership and stable IDs.
func tableRowState(tableState core.WidgetState, selected, hovered, pressed bool) core.WidgetState {
	if tableState == core.StateDisabled {
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

// tableCellState derives a header state from stable column ownership.
func tableCellState(tableState core.WidgetState, id, hovered, pressed string) core.WidgetState {
	if tableState == core.StateDisabled {
		return core.StateDisabled
	}
	if id != "" && id == pressed {
		return core.StatePressed
	}
	if id != "" && id == hovered {
		return core.StateHovered
	}
	return core.StateNormal
}

// drawTableStripe paints an optional authored odd-row stripe. An absent rule
// deliberately does nothing, preserving a flat unskinned table.
func (t *Theme) drawTableStripe(info core.WidgetInfo, row core.Rect) {
	descriptor, ok := t.Lookup(info.Kind, skin.PartStripe, core.StateNormal, info.Class)
	if !ok {
		return
	}
	destination := t.snap(row)
	tint := effectiveTint(descriptor, false)
	t.logDrawCall(info.Kind, skin.PartStripe, core.StateNormal, row, destination, descriptor, tint, false)
	if rl.IsWindowReady() && hasVisualBackground(descriptor) {
		drawDescriptorBackground(descriptor, destination, tint)
	}
}

// tableDefaultAlign applies the table-wide alignment when a column keeps the
// zero-value left alignment; non-left column values remain explicit overrides.
func tableDefaultAlign(tableAlign, columnAlign core.TextAlign) core.TextAlign {
	if columnAlign == core.AlignLeft && tableAlign != core.AlignLeft {
		return tableAlign
	}
	return columnAlign
}

// drawTableCell draws icon and text inside one column clip. Numeric columns
// force right alignment; all other columns retain their authored alignment.
func (t *Theme) drawTableCell(info core.WidgetInfo, content core.Rect, slots []core.Rect, row core.Rect, column widgets.TableColumn, cell widgets.TableCell, columnIndex int, state core.WidgetState) {
	cellRect, ok := TableCellRect(content, slots, row, columnIndex)
	if !ok {
		return
	}
	t.PushClip(cellRect)
	cellInfo := info
	cellInfo.Bounds = cellRect
	cellInfo.State = state
	cellInfo.Align = tableDefaultAlign(info.Align, column.Align)
	if column.Numeric {
		cellInfo.Align = core.AlignRight
	}
	if cell.HasColor {
		cellInfo.TextColor = cell.Color
		cellInfo.HasTextColor = true
	}
	if cell.Invalid {
		cellInfo.TextColor = core.ToColor(t.invalidTableColor(cellInfo, state))
		cellInfo.HasTextColor = true
	}
	textRect := cellRect
	if cell.Icon != "" {
		iconWidth := float32(RichIconSize)
		iconAvailable := textRect.W - 6
		if iconAvailable < 0 {
			iconAvailable = 0
		}
		if iconWidth > iconAvailable {
			iconWidth = iconAvailable
		}
		iconRect := core.Rect{X: textRect.X + 6, Y: textRect.Y, W: iconWidth, H: textRect.H}
		t.drawRichIconBox(cellInfo, cell.Icon, iconRect)
		textRect.X = iconRect.X + iconWidth + 4
		textRect.W = cellRect.X + cellRect.W - textRect.X
		if textRect.W < 0 {
			textRect.W = 0
		}
	}
	t.drawTextInContent(cellInfo, cell.Text, textRect, state)
	t.PopClip()
}

// invalidTableColor applies a stable muted-red invalid tint to the effective
// text color, preserving explicit per-cell or table colors as the base.
func (t *Theme) invalidTableColor(info core.WidgetInfo, state core.WidgetState) color.RGBA {
	base := defaultWidgetTextColor(info, state)
	if !info.HasTextColor {
		if descriptor, ok := t.Lookup(info.Kind, skin.PartBackground, state, info.Class); ok && descriptor.HasTextColor {
			base = descriptor.TextColor.RGBA()
		}
	}
	return color.RGBA{R: maxByte(base.R, 170), G: base.G / 2, B: base.B / 2, A: base.A}
}

// maxByte returns the larger channel without converting through integers.
func maxByte(left, right uint8) uint8 {
	if left > right {
		return left
	}
	return right
}

const (
	// TableSortArrowWidth reserves a right-edge header zone for the arrow.
	TableSortArrowWidth = float32(18)
)
