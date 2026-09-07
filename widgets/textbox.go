package widgets

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/text"
)

// Textbox is an editable text input widget with bounded UTF-8 storage.
type Textbox struct {
	base
	textBuf *text.Buffer
	onText  func(string)
}

// NewTextbox returns an enabled textbox with bounded private UTF-8 storage.
func NewTextbox(name string, bounds core.Rect, capacity int) *Textbox {
	return &Textbox{
		base:    newBase(name, core.WidgetTextbox, bounds),
		textBuf: text.NewBuffer(capacity, ""),
	}
}

// Text returns the textbox text without exposing internal storage.
func (t *Textbox) Text() string {
	if t == nil || t.textBuf == nil {
		return ""
	}
	return t.textBuf.String()
}

// SetText replaces the textbox content and reports whether it changed.
func (t *Textbox) SetText(value string) bool {
	if t == nil || t.textBuf == nil {
		return false
	}
	return t.textBuf.Set(value)
}

// TypeChar inserts ch at the caret, replacing any selection, and reports whether text changed.
func (t *Textbox) TypeChar(ch rune) bool {
	if t == nil || t.textBuf == nil {
		return false
	}
	return t.textBuf.AppendRune(ch)
}

// Backspace removes selection or the rune before caret and reports whether text changed.
func (t *Textbox) Backspace() bool {
	if t == nil || t.textBuf == nil {
		return false
	}
	return t.textBuf.Backspace()
}

// Delete removes selection or the rune after caret and reports whether text changed.
func (t *Textbox) Delete() bool {
	if t == nil || t.textBuf == nil {
		return false
	}
	return t.textBuf.Delete()
}

// DeleteSelection removes selected runes and reports whether text changed.
func (t *Textbox) DeleteSelection() bool {
	if t == nil || t.textBuf == nil {
		return false
	}
	return t.textBuf.DeleteSelection()
}

// InsertString inserts valid UTF-8 at the caret and reports whether text changed.
func (t *Textbox) InsertString(value string) bool {
	if t == nil || t.textBuf == nil {
		return false
	}
	return t.textBuf.InsertString(value)
}

// Caret returns the textbox caret as a rune index.
func (t *Textbox) Caret() int {
	if t == nil || t.textBuf == nil {
		return 0
	}
	return t.textBuf.Caret()
}

// SetCaret moves the caret, clears selection, and reports whether it changed.
func (t *Textbox) SetCaret(pos int) bool {
	if t == nil || t.textBuf == nil {
		return false
	}
	return t.textBuf.SetCaret(pos)
}

// MoveCaret moves caret by delta runes, optionally extending selection.
func (t *Textbox) MoveCaret(delta int, extend bool) bool {
	if t == nil || t.textBuf == nil {
		return false
	}
	return t.textBuf.MoveCaret(delta, extend)
}

// MoveCaretTo moves caret to pos, optionally extending selection.
func (t *Textbox) MoveCaretTo(pos int, extend bool) bool {
	if t == nil || t.textBuf == nil {
		return false
	}
	return t.textBuf.MoveCaretTo(pos, extend)
}

// SelectAll selects every rune and reports whether selection changed.
func (t *Textbox) SelectAll() bool {
	if t == nil || t.textBuf == nil {
		return false
	}
	return t.textBuf.SelectAll()
}

// ClearSelection forgets selection without moving caret.
func (t *Textbox) ClearSelection() bool {
	if t == nil || t.textBuf == nil {
		return false
	}
	return t.textBuf.ClearSelection()
}

// HasSelection reports whether a non-collapsed selection is held.
func (t *Textbox) HasSelection() bool {
	return t != nil && t.textBuf != nil && t.textBuf.HasSelection()
}

// Selection returns sorted selection rune bounds.
func (t *Textbox) Selection() (int, int) {
	if t == nil || t.textBuf == nil {
		return 0, 0
	}
	return t.textBuf.Selection()
}

// SelectedText returns the selected substring.
func (t *Textbox) SelectedText() string {
	if t == nil || t.textBuf == nil {
		return ""
	}
	return t.textBuf.SelectedText()
}

// RuneCount returns the number of runes currently stored.
func (t *Textbox) RuneCount() int {
	if t == nil || t.textBuf == nil {
		return 0
	}
	return t.textBuf.RuneCount()
}

// OnText attaches an edit callback directly to the textbox.
func (t *Textbox) OnText(fn func(string)) *Textbox {
	if t != nil {
		t.onText = fn
	}
	return t
}

// OnTextHandler returns the direct text mutation callback.
func (t *Textbox) OnTextHandler() func(string) {
	if t == nil {
		return nil
	}
	return t.onText
}

// SetTooltip attaches a hover tooltip string directly to the textbox.
func (t *Textbox) SetTooltip(text string) *Textbox {
	t.base.SetTooltip(text)
	return t
}

// SetTextColor configures an explicit text color for the textbox.
func (t *Textbox) SetTextColor(color core.Color) *Textbox {
	t.base.SetTextColor(color)
	return t
}

// SetFontSize sets an explicit font size in pixels for the textbox.
func (t *Textbox) SetFontSize(size float32) *Textbox {
	t.base.SetFontSize(size)
	return t
}

// SetItalic configures whether the textbox uses the italic theme font.
func (t *Textbox) SetItalic(italic bool) *Textbox {
	t.base.SetItalic(italic)
	return t
}

// SetAlign configures the horizontal text alignment for the textbox.
func (t *Textbox) SetAlign(align core.TextAlign) *Textbox {
	t.base.SetAlign(align)
	return t
}
