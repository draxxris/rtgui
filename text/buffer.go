// Package text provides text editing primitives used by widgets.
package text

import "unicode/utf8"

// Buffer stores a bounded UTF-8 value with a caret and an optional selection.
// Its slice capacity is the maximum byte length, and the slice is reused by
// normal editing operations. The caret and selection anchor are rune indices;
// the anchor is -1 when no selection exists.
//
// Caret and selection live here as part of the textbox editing model rather
// than in UI-owned focus state, so truncation, insertion, and deletion keep
// one invariant. UI owns focus: the caret draws only while focused, and
// focus transfer or loss clears the selection.
type Buffer struct {
	bytes  []byte
	caret  int
	anchor int
}

// NewBuffer returns a buffer whose limit is capacity bytes. Invalid initial
// text is discarded, and valid text is truncated at a rune boundary. The
// caret starts at the end with no selection.
func NewBuffer(capacity int, initial string) *Buffer {
	if capacity < 0 {
		capacity = 0
	}
	b := &Buffer{bytes: make([]byte, 0, capacity), anchor: -1}
	b.Set(initial)
	return b
}

// String returns the buffer's current immutable string value.
func (b *Buffer) String() string {
	if b == nil {
		return ""
	}
	return string(b.bytes)
}

// Set replaces the value with valid UTF-8 truncated to the buffer limit. It
// moves the caret to the end, clears any selection, and reports whether the
// stored value changed.
func (b *Buffer) Set(value string) bool {
	if b == nil || !utf8.ValidString(value) {
		return false
	}
	value = truncateUTF8(value, b.Limit())
	changed := true
	if len(value) == len(b.bytes) {
		changed = false
		for i := range b.bytes {
			if b.bytes[i] != value[i] {
				changed = true
				break
			}
		}
	}
	b.bytes = append(b.bytes[:0], value...)
	b.caret = utf8.RuneCount(b.bytes)
	b.anchor = -1
	return changed
}

// AppendRune inserts r at the caret, replacing any selection when its UTF-8
// encoding fits in the buffer. It reports whether the stored value changed.
func (b *Buffer) AppendRune(r rune) bool {
	if b == nil || !utf8.ValidRune(r) {
		return false
	}
	b.clampCaret()
	size := utf8.RuneLen(r)
	if b.HasSelection() {
		start, end := b.Selection()
		selBytes := b.byteOffsetForRune(end) - b.byteOffsetForRune(start)
		if b.Len()-selBytes+size > b.Limit() {
			return false
		}
		b.DeleteSelection()
	} else if b.Len()+size > b.Limit() {
		return false
	}
	return b.insertRuneAtCaret(r)
}

// Backspace removes the selection or the rune before the caret. It reports
// whether the stored value changed.
func (b *Buffer) Backspace() bool {
	if b == nil || len(b.bytes) == 0 && !b.HasSelection() {
		return false
	}
	b.clampCaret()
	if b.HasSelection() {
		return b.DeleteSelection()
	}
	if b.caret <= 0 {
		return false
	}
	endByte := b.byteOffsetForRune(b.caret)
	startByte := b.byteOffsetForRune(b.caret - 1)
	b.deleteByteRange(startByte, endByte)
	b.caret--
	return true
}

// Delete removes the selection or the rune after the caret. It reports
// whether the stored value changed.
func (b *Buffer) Delete() bool {
	if b == nil {
		return false
	}
	b.clampCaret()
	if b.HasSelection() {
		return b.DeleteSelection()
	}
	if b.caret >= utf8.RuneCount(b.bytes) {
		return false
	}
	startByte := b.byteOffsetForRune(b.caret)
	endByte := b.byteOffsetForRune(b.caret + 1)
	b.deleteByteRange(startByte, endByte)
	return true
}

// DeleteSelection removes the selected runes, collapses the caret to the
// selection start, and reports whether text was removed.
func (b *Buffer) DeleteSelection() bool {
	if b == nil || !b.HasSelection() {
		return false
	}
	start, end := b.Selection()
	startByte := b.byteOffsetForRune(start)
	endByte := b.byteOffsetForRune(end)
	b.deleteByteRange(startByte, endByte)
	b.caret = start
	b.anchor = -1
	return true
}

// InsertString inserts valid UTF-8 at the caret, replacing any selection with
// as many leading runes as fit. It reports whether any rune was inserted or
// a selection was removed.
func (b *Buffer) InsertString(value string) bool {
	if b == nil || value == "" || !utf8.ValidString(value) {
		return false
	}
	b.clampCaret()
	if b.HasSelection() {
		// Remove the selection first so pasted runes reuse its bytes.
		// An empty paste never reaches here, so removal always pairs
		// with an insertion attempt; the removal alone counts as a
		// mutation even when no rune fits afterwards.
		b.DeleteSelection()
		for _, r := range value {
			if !b.insertRuneAtCaret(r) {
				break
			}
		}
		return true
	}
	mutated := false
	for _, r := range value {
		if !b.insertRuneAtCaret(r) {
			break
		}
		mutated = true
	}
	return mutated
}

// Caret returns the caret position as a rune index from 0 to RuneCount.
func (b *Buffer) Caret() int {
	if b == nil {
		return 0
	}
	b.clampCaret()
	return b.caret
}

// SetCaret moves the caret to pos, clears any selection, and reports whether
// the caret or selection changed.
func (b *Buffer) SetCaret(pos int) bool {
	if b == nil {
		return false
	}
	return b.MoveCaretTo(pos, false)
}

// MoveCaret moves the caret by delta runes. When extend is true the selection
// anchor is preserved or started; otherwise any selection is cleared. It
// reports whether the caret or selection changed.
func (b *Buffer) MoveCaret(delta int, extend bool) bool {
	if b == nil {
		return false
	}
	b.clampCaret()
	return b.moveTo(b.caret+delta, extend)
}

// MoveCaretTo moves the caret to pos. When extend is true the selection
// anchor is preserved or started; otherwise any selection is cleared. It
// reports whether the caret or selection changed.
func (b *Buffer) MoveCaretTo(pos int, extend bool) bool {
	if b == nil {
		return false
	}
	b.clampCaret()
	return b.moveTo(pos, extend)
}

// SelectAll selects every rune and reports whether the selection changed.
func (b *Buffer) SelectAll() bool {
	if b == nil {
		return false
	}
	n := utf8.RuneCount(b.bytes)
	if b.anchor == 0 && b.caret == n && b.HasSelection() {
		return false
	}
	// An empty buffer has no selectable range.
	if n == 0 {
		return false
	}
	b.anchor = 0
	b.caret = n
	return true
}

// ClearSelection forgets any selection without moving the caret. It reports
// whether a selection was present.
func (b *Buffer) ClearSelection() bool {
	if b == nil || !b.HasSelection() {
		return false
	}
	b.anchor = -1
	return true
}

// HasSelection reports whether a non-collapsed selection exists.
func (b *Buffer) HasSelection() bool {
	return b != nil && b.anchor >= 0 && b.anchor != b.caret
}

// Selection returns the sorted selection bounds as rune indices. It returns
// zeros when no selection exists.
func (b *Buffer) Selection() (int, int) {
	if !b.HasSelection() {
		return 0, 0
	}
	if b.anchor < b.caret {
		return b.anchor, b.caret
	}
	return b.caret, b.anchor
}

// SelectedText returns the selected substring, or "" when nothing is selected.
func (b *Buffer) SelectedText() string {
	if !b.HasSelection() {
		return ""
	}
	start, end := b.Selection()
	return string(b.bytes[b.byteOffsetForRune(start):b.byteOffsetForRune(end)])
}

// RuneCount returns the number of runes in the buffer.
func (b *Buffer) RuneCount() int {
	if b == nil {
		return 0
	}
	return utf8.RuneCount(b.bytes)
}

// Len returns the current byte length.
func (b *Buffer) Len() int {
	if b == nil {
		return 0
	}
	return len(b.bytes)
}

// Limit returns the maximum byte length.
func (b *Buffer) Limit() int {
	if b == nil {
		return 0
	}
	return cap(b.bytes)
}

// moveTo relocates the caret with selection-extension handling.
func (b *Buffer) moveTo(pos int, extend bool) bool {
	n := utf8.RuneCount(b.bytes)
	if pos < 0 {
		pos = 0
	}
	if pos > n {
		pos = n
	}
	oldCaret, oldAnchor := b.caret, b.anchor
	hadSelection := b.HasSelection()
	if extend {
		if !hadSelection {
			b.anchor = oldCaret
		}
		b.caret = pos
		if b.caret == b.anchor {
			b.anchor = -1
		}
	} else {
		b.caret = pos
		b.anchor = -1
	}
	return b.caret != oldCaret || b.anchor != oldAnchor
}

// insertRuneAtCaret inserts one rune at the caret without selection handling.
// Callers must remove any selection first. It reports whether insertion fit.
func (b *Buffer) insertRuneAtCaret(r rune) bool {
	if !utf8.ValidRune(r) {
		return false
	}
	var encoded [utf8.UTFMax]byte
	size := utf8.EncodeRune(encoded[:], r)
	if b.Len()+size > b.Limit() {
		return false
	}
	at := b.byteOffsetForRune(b.caret)
	oldLen := len(b.bytes)
	for i := 0; i < size; i++ {
		b.bytes = append(b.bytes, 0)
	}
	copy(b.bytes[at+size:], b.bytes[at:oldLen])
	copy(b.bytes[at:], encoded[:size])
	b.caret++
	return true
}

// deleteByteRange removes bytes in [start, end) without moving the caret.
// Callers set the caret explicitly afterwards.
func (b *Buffer) deleteByteRange(start, end int) {
	if start < 0 {
		start = 0
	}
	if end > len(b.bytes) {
		end = len(b.bytes)
	}
	if start >= end {
		return
	}
	copy(b.bytes[start:], b.bytes[end:])
	b.bytes = b.bytes[:len(b.bytes)-(end-start)]
}

// clampCaret keeps the caret and anchor inside the current rune range.
func (b *Buffer) clampCaret() {
	n := utf8.RuneCount(b.bytes)
	if b.caret < 0 {
		b.caret = 0
	}
	if b.caret > n {
		b.caret = n
	}
	if b.anchor < -1 {
		b.anchor = -1
	}
	if b.anchor > n {
		b.anchor = n
	}
	if b.anchor == b.caret {
		b.anchor = -1
	}
}

// byteOffsetForRune converts a rune index to its byte offset, clamped.
func (b *Buffer) byteOffsetForRune(index int) int {
	if index <= 0 {
		return 0
	}
	pos := 0
	for i := 0; i < index && pos < len(b.bytes); i++ {
		_, size := utf8.DecodeRune(b.bytes[pos:])
		if size <= 0 {
			break
		}
		pos += size
	}
	return pos
}

// truncateUTF8 keeps the largest valid rune prefix that fits in limit bytes.
func truncateUTF8(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if len(value) <= limit {
		return value
	}
	for limit > 0 && !utf8.RuneStart(value[limit]) {
		limit--
	}
	return value[:limit]
}
