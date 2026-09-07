// Package widgets provides stateful widget data without owning UI interaction state.
package widgets

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/layout"
	"github.com/draxxris/rtgui/text"
)

// Widget is a compact tagged widget whose identity and kind are fixed at construction.
// UI-owned hover, press, and focus state is deliberately not stored here.
type Widget struct {
	name                string
	kind                core.WidgetKind
	frame               *layout.Node
	enabled             bool
	text                string
	value               float32
	checked             bool
	scroll              core.Vec2
	dropdownIndex       int
	dropdownItems       []string
	tabSelected         int
	tabLabels           []string
	richSegments        []core.RichSegment
	textBuf             *text.Buffer
	onClick             func()
	onChange            func(float32)
	onText              func(string)
	onTabSelect         func(int)
	onLinkClick         func(core.Link)
	onLinkTooltip       func(core.Link) string
	tooltip             string
	canvasDraw          func(bounds core.Rect)
	scrollContentDrawer func(bounds core.Rect, scrollOffset core.Vec2)
	maxScroll           core.Vec2
	textColor           core.Color
	hasTextColor        bool
	format              string
	fontSize            float32
	italic              bool
	align               core.TextAlign
}

// NewButton returns an enabled button with immutable name and kind.
func NewButton(name string, bounds core.Rect, label string) *Widget {
	return &Widget{name: name, kind: core.WidgetButton, frame: layout.New(name, bounds), enabled: true, text: label}
}

// NewLabel returns an enabled label with immutable name and kind.
func NewLabel(name string, bounds core.Rect, label string) *Widget {
	return &Widget{name: name, kind: core.WidgetLabel, frame: layout.New(name, bounds), enabled: true, text: label}
}

// NewStyledLabel returns an enabled label with custom font size, italic style, and alignment.
func NewStyledLabel(name string, bounds core.Rect, label string, fontSize float32, italic bool, align core.TextAlign) *Widget {
	return &Widget{
		name:     name,
		kind:     core.WidgetLabel,
		frame:    layout.New(name, bounds),
		enabled:  true,
		text:     label,
		fontSize: fontSize,
		italic:   italic,
		align:    align,
	}
}

// NewCheckbox returns an enabled checkbox with immutable name and kind.
func NewCheckbox(name string, bounds core.Rect, checked bool) *Widget {
	return &Widget{name: name, kind: core.WidgetCheckbox, frame: layout.New(name, bounds), enabled: true, checked: checked}
}

// NewCheckboxWithLabel returns an enabled checkbox with a label and immutable name and kind.
func NewCheckboxWithLabel(name string, bounds core.Rect, label string, checked bool) *Widget {
	return &Widget{name: name, kind: core.WidgetCheckbox, frame: layout.New(name, bounds), enabled: true, text: label, checked: checked}
}

// NewCanvas returns an enabled custom drawing widget that executes drawFn during Draw.
func NewCanvas(name string, bounds core.Rect, drawFn func(bounds core.Rect)) *Widget {
	return &Widget{name: name, kind: core.WidgetCanvas, frame: layout.New(name, bounds), enabled: true, canvasDraw: drawFn}
}

// NewTextbox returns an enabled textbox with bounded private UTF-8 storage.
func NewTextbox(name string, bounds core.Rect, capacity int) *Widget {
	return &Widget{name: name, kind: core.WidgetTextbox, frame: layout.New(name, bounds), enabled: true, textBuf: text.NewBuffer(capacity, "")}
}

// NewSlider returns an enabled slider with a clamped initial value.
func NewSlider(name string, bounds core.Rect, value float32) *Widget {
	widget := &Widget{name: name, kind: core.WidgetSlider, frame: layout.New(name, bounds), enabled: true}
	widget.SetValue(value)
	return widget
}

// NewProgressBar returns an enabled progress bar with a clamped initial value.
func NewProgressBar(name string, bounds core.Rect, value float32) *Widget {
	widget := &Widget{name: name, kind: core.WidgetProgressBar, frame: layout.New(name, bounds), enabled: true}
	widget.SetValue(value)
	return widget
}

// NewScrollPanel returns an enabled scroll panel with zero scroll offset.
func NewScrollPanel(name string, bounds core.Rect) *Widget {
	return &Widget{name: name, kind: core.WidgetScrollPanel, frame: layout.New(name, bounds), enabled: true}
}

// NewDropdown returns an enabled dropdown and copies items so caller mutation
// cannot change widget configuration. An invalid selection becomes -1.
func NewDropdown(name string, bounds core.Rect, items []string, index int) *Widget {
	widget := &Widget{name: name, kind: core.WidgetDropdown, frame: layout.New(name, bounds), enabled: true, dropdownIndex: -1}
	widget.SetDropdownItems(items)
	widget.SetDropdownIndex(index)
	return widget
}

// NewFrame returns an enabled visual frame with immutable name and kind.
func NewFrame(name string, bounds core.Rect) *Widget {
	return &Widget{name: name, kind: core.WidgetFrame, frame: layout.New(name, bounds), enabled: true}
}

// NewTabBar returns an enabled tab bar and copies labels so caller mutation
// cannot change widget configuration. An invalid selection becomes -1.
func NewTabBar(name string, bounds core.Rect, labels []string, selected int) *Widget {
	widget := &Widget{name: name, kind: core.WidgetTabBar, frame: layout.New(name, bounds), enabled: true, tabSelected: -1}
	widget.SetTabLabels(labels)
	widget.SetSelectedTab(selected)
	return widget
}

// NewRichText returns an enabled rich-text message and copies segments so
// caller mutation cannot change widget configuration.
func NewRichText(name string, bounds core.Rect, segments []core.RichSegment) *Widget {
	widget := &Widget{name: name, kind: core.WidgetRichText, frame: layout.New(name, bounds), enabled: true}
	widget.SetRichSegments(segments)
	return widget
}

// Name returns the widget's immutable external registry identity.
func (w *Widget) Name() string {
	if w == nil {
		return ""
	}
	return w.name
}

// Kind returns the widget's immutable tagged kind.
func (w *Widget) Kind() core.WidgetKind {
	if w == nil {
		return core.WidgetKind(-1)
	}
	return w.kind
}

// Bounds returns resolved frame bounds after arrangement and constructor or
// authored bounds for widgets that are not managed by a layout tree.
func (w *Widget) Bounds() core.Rect {
	if w == nil || w.frame == nil {
		return core.Rect{}
	}
	return w.frame.Bounds()
}

// SetBounds replaces the widget frame's authored bounds.
func (w *Widget) SetBounds(bounds core.Rect) {
	if w != nil && w.frame != nil {
		w.frame.SetBounds(bounds)
	}
}

// Frame returns the widget's layout node for ownership-tree construction.
func (w *Widget) Frame() *layout.Node {
	if w == nil {
		return nil
	}
	return w.frame
}

// SetPoint adds or replaces one typed relation on the widget frame.
func (w *Widget) SetPoint(source layout.Anchor, target *layout.Node, targetPoint layout.Anchor, offset core.Vec2) error {
	if w == nil || w.frame == nil {
		return layout.ErrNilNode
	}
	return w.frame.SetPoint(source, target, targetPoint, offset)
}

// Enabled reports whether the widget accepts interaction.
func (w *Widget) Enabled() bool { return w != nil && w.enabled }

// SetEnabled changes whether the widget accepts interaction and reports a change.
func (w *Widget) SetEnabled(enabled bool) bool {
	if w == nil || w.enabled == enabled {
		return false
	}
	w.enabled = enabled
	return true
}

// Text returns the widget text without exposing textbox storage.
func (w *Widget) Text() string {
	if w == nil {
		return ""
	}
	if w.kind == core.WidgetTextbox && w.textBuf != nil {
		return w.textBuf.String()
	}
	return w.text
}

// SetText replaces plain widget text or textbox content and reports a change.
func (w *Widget) SetText(value string) bool {
	if w == nil {
		return false
	}
	if w.kind == core.WidgetTextbox && w.textBuf != nil {
		return w.textBuf.Set(value)
	}
	if w.text == value {
		return false
	}
	w.text = value
	return true
}

// Checked reports the checkbox value.
func (w *Widget) Checked() bool { return w != nil && w.checked }

// SetChecked changes a checkbox value and reports a real mutation.
func (w *Widget) SetChecked(checked bool) bool {
	if w == nil || w.kind != core.WidgetCheckbox || w.checked == checked {
		return false
	}
	w.checked = checked
	return true
}

// Value returns a slider or progress-bar value.
func (w *Widget) Value() float32 {
	if w == nil {
		return 0
	}
	return w.value
}

// SetValue clamps and stores a slider or progress-bar value and reports a change.
func (w *Widget) SetValue(value float32) bool {
	if w == nil || (w.kind != core.WidgetSlider && w.kind != core.WidgetProgressBar) {
		return false
	}
	if value < 0 {
		value = 0
	} else if value > 1 {
		value = 1
	}
	if w.value == value {
		return false
	}
	w.value = value
	return true
}

// Scroll returns the scroll-panel offset by value.
func (w *Widget) Scroll() core.Vec2 {
	if w == nil {
		return core.Vec2{}
	}
	return w.scroll
}

// SetScroll replaces a scroll-panel offset and reports a change.
func (w *Widget) SetScroll(offset core.Vec2) bool {
	if w == nil || w.kind != core.WidgetScrollPanel || w.scroll == offset {
		return false
	}
	w.scroll = offset
	return true
}

// ScrollBy changes a scroll-panel offset and reports a change.
func (w *Widget) ScrollBy(dx, dy float32) bool {
	if w == nil || w.kind != core.WidgetScrollPanel || (dx == 0 && dy == 0) {
		return false
	}
	newX := w.scroll.X + dx
	newY := w.scroll.Y + dy
	if w.maxScroll.X > 0 {
		if newX < 0 {
			newX = 0
		} else if newX > w.maxScroll.X {
			newX = w.maxScroll.X
		}
	}
	if w.maxScroll.Y > 0 {
		if newY < 0 {
			newY = 0
		} else if newY > w.maxScroll.Y {
			newY = w.maxScroll.Y
		}
	}
	return w.SetScroll(core.Vec2{X: newX, Y: newY})
}

// DropdownIndex returns the selected dropdown item index, or -1 when unset.
func (w *Widget) DropdownIndex() int {
	if w == nil || w.kind != core.WidgetDropdown {
		return -1
	}
	return w.dropdownIndex
}

// SetDropdownIndex selects a valid dropdown item and reports a real mutation.
func (w *Widget) SetDropdownIndex(index int) bool {
	if w == nil || w.kind != core.WidgetDropdown || index < 0 || index >= len(w.dropdownItems) || w.dropdownIndex == index {
		return false
	}
	w.dropdownIndex = index
	return true
}

// DropdownSelection returns the selected item without exposing item storage.
func (w *Widget) DropdownSelection() (string, bool) {
	if w == nil || w.kind != core.WidgetDropdown || w.dropdownIndex < 0 || w.dropdownIndex >= len(w.dropdownItems) {
		return "", false
	}
	return w.dropdownItems[w.dropdownIndex], true
}

// DropdownItems returns a snapshot that callers may mutate freely.
func (w *Widget) DropdownItems() []string {
	if w == nil || w.kind != core.WidgetDropdown {
		return nil
	}
	return append([]string(nil), w.dropdownItems...)
}

// SetDropdownItems copies dropdown items and keeps the current selection only
// when it remains valid. It reports whether item data or selection changed.
func (w *Widget) SetDropdownItems(items []string) bool {
	if w == nil || w.kind != core.WidgetDropdown {
		return false
	}
	changed := !equalStrings(w.dropdownItems, items)
	if changed {
		w.dropdownItems = append(w.dropdownItems[:0], items...)
	}
	if w.dropdownIndex >= len(w.dropdownItems) {
		w.dropdownIndex = -1
		changed = true
	}
	return changed
}

// HitTest reports whether a logical point lies in the widget's effective bounds.
func (w *Widget) HitTest(pos core.Vec2) bool { return w != nil && w.Bounds().Contains(pos) }

// DropdownPopupBounds returns the dropdown list bounds below its control.
func (w *Widget) DropdownPopupBounds() core.Rect {
	if w == nil || w.kind != core.WidgetDropdown {
		return core.Rect{}
	}
	bounds := w.Bounds()
	return core.Rect{
		X: bounds.X,
		Y: bounds.Y + bounds.H + 4,
		W: bounds.W,
		H: float32(len(w.dropdownItems) * 36),
	}
}

// DropdownItemCount returns the number of dropdown items without exposing
// item storage. Row hit testing is skin-aware and lives in render
// (Theme.DropdownPopupContent with DropdownPopupIndex), so this package
// keeps only the skin-free item count for input and draw call sites.
func (w *Widget) DropdownItemCount() int {
	if w == nil || w.kind != core.WidgetDropdown {
		return 0
	}
	return len(w.dropdownItems)
}

// TabCount returns the number of tab labels without exposing label storage.
func (w *Widget) TabCount() int {
	if w == nil || w.kind != core.WidgetTabBar {
		return 0
	}
	return len(w.tabLabels)
}

// TabLabels returns a snapshot that callers may mutate freely.
func (w *Widget) TabLabels() []string {
	if w == nil || w.kind != core.WidgetTabBar {
		return nil
	}
	return append([]string(nil), w.tabLabels...)
}

// SetTabLabels copies tab labels and keeps the current selection only when
// it remains valid. It reports whether label data or selection changed.
func (w *Widget) SetTabLabels(labels []string) bool {
	if w == nil || w.kind != core.WidgetTabBar {
		return false
	}
	changed := !equalStrings(w.tabLabels, labels)
	if changed {
		w.tabLabels = append(w.tabLabels[:0], labels...)
	}
	if w.tabSelected >= len(w.tabLabels) {
		w.tabSelected = -1
		changed = true
	}
	return changed
}

// SelectedTab returns the selected tab index, or -1 when unset.
func (w *Widget) SelectedTab() int {
	if w == nil || w.kind != core.WidgetTabBar {
		return -1
	}
	return w.tabSelected
}

// SetSelectedTab selects a valid tab and reports a real mutation.
func (w *Widget) SetSelectedTab(index int) bool {
	if w == nil || w.kind != core.WidgetTabBar || index < 0 || index >= len(w.tabLabels) || w.tabSelected == index {
		return false
	}
	w.tabSelected = index
	return true
}

// TabSelection returns the selected tab label without exposing storage.
func (w *Widget) TabSelection() (string, bool) {
	if w == nil || w.kind != core.WidgetTabBar || w.tabSelected < 0 || w.tabSelected >= len(w.tabLabels) {
		return "", false
	}
	return w.tabLabels[w.tabSelected], true
}

// RichSegments returns a snapshot that callers may mutate freely.
func (w *Widget) RichSegments() []core.RichSegment {
	if w == nil || w.kind != core.WidgetRichText {
		return nil
	}
	return append([]core.RichSegment(nil), w.richSegments...)
}

// SetRichSegments copies message segments and reports whether segment data
// changed. Wrapping and link geometry derive from this data in render.
func (w *Widget) SetRichSegments(segments []core.RichSegment) bool {
	if w == nil || w.kind != core.WidgetRichText {
		return false
	}
	if equalRichSegments(w.richSegments, segments) {
		return false
	}
	w.richSegments = append(w.richSegments[:0], segments...)
	return true
}

// RichPlainText concatenates segment text for search and copy support.
func (w *Widget) RichPlainText() string {
	if w == nil || w.kind != core.WidgetRichText {
		return ""
	}
	text := ""
	for _, segment := range w.richSegments {
		text += segment.Text
	}
	return text
}

// LinkCount returns the number of linked segments without exposing storage.
func (w *Widget) LinkCount() int {
	if w == nil || w.kind != core.WidgetRichText {
		return 0
	}
	count := 0
	for _, segment := range w.richSegments {
		if segment.Link.Kind != core.LinkNone {
			count++
		}
	}
	return count
}

// LinkAt returns the nth link in segment order, or false when out of range.
func (w *Widget) LinkAt(index int) (core.Link, bool) {
	if w == nil || w.kind != core.WidgetRichText || index < 0 {
		return core.Link{}, false
	}
	for _, segment := range w.richSegments {
		if segment.Link.Kind == core.LinkNone {
			continue
		}
		if index == 0 {
			return segment.Link, true
		}
		index--
	}
	return core.Link{}, false
}

// TypeChar inserts ch at the textbox caret, replacing any selection, and
// reports whether its text changed.
func (w *Widget) TypeChar(ch rune) bool {
	if w == nil || w.kind != core.WidgetTextbox || w.textBuf == nil {
		return false
	}
	return w.textBuf.AppendRune(ch)
}

// Backspace removes the textbox selection or the rune before the caret and
// reports whether its text changed.
func (w *Widget) Backspace() bool {
	if w == nil || w.kind != core.WidgetTextbox || w.textBuf == nil {
		return false
	}
	return w.textBuf.Backspace()
}

// Delete removes the textbox selection or the rune after the caret and
// reports whether its text changed.
func (w *Widget) Delete() bool {
	if w == nil || w.kind != core.WidgetTextbox || w.textBuf == nil {
		return false
	}
	return w.textBuf.Delete()
}

// DeleteSelection removes the selected textbox runes and reports whether
// its text changed.
func (w *Widget) DeleteSelection() bool {
	if w == nil || w.kind != core.WidgetTextbox || w.textBuf == nil {
		return false
	}
	return w.textBuf.DeleteSelection()
}

// InsertString inserts valid UTF-8 at the textbox caret, replacing any
// selection with as many leading runes as fit. It reports whether its text
// changed.
func (w *Widget) InsertString(value string) bool {
	if w == nil || w.kind != core.WidgetTextbox || w.textBuf == nil {
		return false
	}
	return w.textBuf.InsertString(value)
}

// Caret returns the textbox caret as a rune index from 0 to RuneCount.
func (w *Widget) Caret() int {
	if w == nil || w.kind != core.WidgetTextbox || w.textBuf == nil {
		return 0
	}
	return w.textBuf.Caret()
}

// SetCaret moves the textbox caret, clears any selection, and reports
// whether the caret or selection changed.
func (w *Widget) SetCaret(pos int) bool {
	if w == nil || w.kind != core.WidgetTextbox || w.textBuf == nil {
		return false
	}
	return w.textBuf.SetCaret(pos)
}

// MoveCaret moves the textbox caret by delta runes, extending the selection
// when extend is true, and reports whether caret or selection changed.
func (w *Widget) MoveCaret(delta int, extend bool) bool {
	if w == nil || w.kind != core.WidgetTextbox || w.textBuf == nil {
		return false
	}
	return w.textBuf.MoveCaret(delta, extend)
}

// MoveCaretTo moves the textbox caret to pos, extending the selection when
// extend is true, and reports whether caret or selection changed.
func (w *Widget) MoveCaretTo(pos int, extend bool) bool {
	if w == nil || w.kind != core.WidgetTextbox || w.textBuf == nil {
		return false
	}
	return w.textBuf.MoveCaretTo(pos, extend)
}

// SelectAll selects every textbox rune and reports whether selection changed.
func (w *Widget) SelectAll() bool {
	if w == nil || w.kind != core.WidgetTextbox || w.textBuf == nil {
		return false
	}
	return w.textBuf.SelectAll()
}

// ClearSelection forgets any textbox selection without moving the caret. It
// reports whether a selection was present.
func (w *Widget) ClearSelection() bool {
	if w == nil || w.kind != core.WidgetTextbox || w.textBuf == nil {
		return false
	}
	return w.textBuf.ClearSelection()
}

// HasSelection reports whether the textbox holds a non-collapsed selection.
func (w *Widget) HasSelection() bool {
	return w != nil && w.kind == core.WidgetTextbox && w.textBuf != nil && w.textBuf.HasSelection()
}

// Selection returns the sorted textbox selection bounds as rune indices.
func (w *Widget) Selection() (int, int) {
	if w == nil || w.kind != core.WidgetTextbox || w.textBuf == nil {
		return 0, 0
	}
	return w.textBuf.Selection()
}

// SelectedText returns the selected textbox substring, or "" when idle.
func (w *Widget) SelectedText() string {
	if w == nil || w.kind != core.WidgetTextbox || w.textBuf == nil {
		return ""
	}
	return w.textBuf.SelectedText()
}

// RuneCount returns the number of runes in the textbox.
func (w *Widget) RuneCount() int {
	if w == nil || w.kind != core.WidgetTextbox || w.textBuf == nil {
		return 0
	}
	return w.textBuf.RuneCount()
}

// Snapshot returns the one renderer-facing value, using UI-computed visual state.
func (w *Widget) Snapshot(state core.WidgetState) core.WidgetInfo {
	if w == nil {
		return core.WidgetInfo{}
	}
	info := core.WidgetInfo{
		Name:     w.name,
		Bounds:   w.Bounds(),
		Kind:     w.kind,
		State:    state,
		FontSize: w.fontSize,
		Italic:   w.italic,
		Align:    w.align,
	}
	if w.hasTextColor {
		info.TextColor = w.textColor
		info.HasTextColor = true
	}
	return info
}

// OnClick attaches a synchronous activation callback directly to the widget.
func (w *Widget) OnClick(fn func()) *Widget {
	if w != nil {
		w.onClick = fn
	}
	return w
}

// OnClickHandler returns the widget's direct activation callback.
func (w *Widget) OnClickHandler() func() {
	if w == nil {
		return nil
	}
	return w.onClick
}

// OnChange attaches a synchronous slider callback directly to the widget.
func (w *Widget) OnChange(fn func(float32)) *Widget {
	if w != nil {
		w.onChange = fn
	}
	return w
}

// OnChangeHandler returns the widget's direct value-change callback.
func (w *Widget) OnChangeHandler() func(float32) {
	if w == nil {
		return nil
	}
	return w.onChange
}

// OnText attaches a synchronous textbox edit callback directly to the widget.
func (w *Widget) OnText(fn func(string)) *Widget {
	if w != nil {
		w.onText = fn
	}
	return w
}

// OnTextHandler returns the widget's direct text-change callback.
func (w *Widget) OnTextHandler() func(string) {
	if w == nil {
		return nil
	}
	return w.onText
}

// OnTabSelect attaches a synchronous tab selection callback directly to the widget.
func (w *Widget) OnTabSelect(fn func(int)) *Widget {
	if w != nil {
		w.onTabSelect = fn
	}
	return w
}

// OnTabSelectHandler returns the widget's direct tab selection callback.
func (w *Widget) OnTabSelectHandler() func(int) {
	if w == nil {
		return nil
	}
	return w.onTabSelect
}

// OnLinkClick attaches a synchronous link activation callback directly to the widget.
func (w *Widget) OnLinkClick(fn func(core.Link)) *Widget {
	if w != nil {
		w.onLinkClick = fn
	}
	return w
}

// OnLinkClickHandler returns the widget's direct link click callback.
func (w *Widget) OnLinkClickHandler() func(core.Link) {
	if w == nil {
		return nil
	}
	return w.onLinkClick
}

// OnLinkTooltipRequested attaches a hover-text provider directly to the widget.
func (w *Widget) OnLinkTooltipRequested(fn func(core.Link) string) *Widget {
	if w != nil {
		w.onLinkTooltip = fn
	}
	return w
}

// OnLinkTooltipHandler returns the widget's direct link tooltip provider.
func (w *Widget) OnLinkTooltipHandler() func(core.Link) string {
	if w == nil {
		return nil
	}
	return w.onLinkTooltip
}

// SetTooltip attaches a hover tooltip string directly to the widget.
func (w *Widget) SetTooltip(text string) *Widget {
	if w != nil {
		w.tooltip = text
	}
	return w
}

// Tooltip returns the widget's direct hover tooltip string.
func (w *Widget) Tooltip() string {
	if w == nil {
		return ""
	}
	return w.tooltip
}

// CanvasDraw returns the widget's custom draw function, if any.
func (w *Widget) CanvasDraw() func(bounds core.Rect) {
	if w == nil {
		return nil
	}
	return w.canvasDraw
}

// SetCanvasDraw replaces the widget's custom draw function.
func (w *Widget) SetCanvasDraw(fn func(bounds core.Rect)) *Widget {
	if w != nil {
		w.canvasDraw = fn
	}
	return w
}

// SetScrollContentDrawer registers a custom drawer for the scroll panel's contents.
func (w *Widget) SetScrollContentDrawer(fn func(bounds core.Rect, scrollOffset core.Vec2)) *Widget {
	if w != nil {
		w.scrollContentDrawer = fn
	}
	return w
}

// ScrollContentDrawer returns the scroll panel's content drawer.
func (w *Widget) ScrollContentDrawer() func(bounds core.Rect, scrollOffset core.Vec2) {
	if w == nil {
		return nil
	}
	return w.scrollContentDrawer
}

// SetMaxScroll configures the maximum scroll offset for ScrollBy.
func (w *Widget) SetMaxScroll(max core.Vec2) *Widget {
	if w != nil {
		w.maxScroll = max
	}
	return w
}

// MaxScroll returns the maximum scroll offset.
func (w *Widget) MaxScroll() core.Vec2 {
	if w == nil {
		return core.Vec2{}
	}
	return w.maxScroll
}

// SetTextColor configures an explicit text color for the widget.
func (w *Widget) SetTextColor(c core.Color) *Widget {
	if w != nil {
		w.textColor = c
		w.hasTextColor = true
	}
	return w
}

// TextColor returns the widget's configured text color and whether one was set.
func (w *Widget) TextColor() (core.Color, bool) {
	if w == nil || !w.hasTextColor {
		return core.Color{}, false
	}
	return w.textColor, true
}

// SetFormat configures a display format string for slider or progress-bar readouts.
func (w *Widget) SetFormat(format string) *Widget {
	if w != nil {
		w.format = format
	}
	return w
}

// Format returns the widget's display format string.
func (w *Widget) Format() string {
	if w == nil {
		return ""
	}
	return w.format
}

// SetFontSize sets an explicit font size in pixels for text rendering.
func (w *Widget) SetFontSize(size float32) *Widget {
	if w != nil {
		w.fontSize = size
	}
	return w
}

// FontSize returns the explicit font size, or 0 if auto-derived from bounds.
func (w *Widget) FontSize() float32 {
	if w == nil {
		return 0
	}
	return w.fontSize
}

// SetItalic configures whether the widget text uses the italic theme font.
func (w *Widget) SetItalic(italic bool) *Widget {
	if w != nil {
		w.italic = italic
	}
	return w
}

// Italic reports whether the widget text uses the italic theme font.
func (w *Widget) Italic() bool {
	if w == nil {
		return false
	}
	return w.italic
}

// SetAlign configures the horizontal text alignment within the content bounds.
func (w *Widget) SetAlign(align core.TextAlign) *Widget {
	if w != nil {
		w.align = align
	}
	return w
}

// Align reports the horizontal text alignment within the content bounds.
func (w *Widget) Align() core.TextAlign {
	if w == nil {
		return core.AlignLeft
	}
	return w.align
}

// equalStrings compares dropdown item snapshots without allocating.
func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

// equalRichSegments compares message segment snapshots field by field.
func equalRichSegments(left, right []core.RichSegment) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
