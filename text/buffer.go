// Package text provides text editing primitives used by widgets.
package text

import "unicode/utf8"

// Buffer stores a bounded UTF-8 value. Its slice capacity is the maximum byte
// length, and the slice is reused by normal editing operations.
type Buffer struct {
	bytes []byte
}

// NewBuffer returns a buffer whose limit is capacity bytes. Invalid initial
// text is discarded, and valid text is truncated at a rune boundary.
func NewBuffer(capacity int, initial string) *Buffer {
	if capacity < 0 {
		capacity = 0
	}
	b := &Buffer{bytes: make([]byte, 0, capacity)}
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
// reports whether the stored value changed.
func (b *Buffer) Set(value string) bool {
	if b == nil || !utf8.ValidString(value) {
		return false
	}
	value = truncateUTF8(value, b.Limit())
	if len(value) == len(b.bytes) {
		unchanged := true
		for i := range b.bytes {
			if b.bytes[i] != value[i] {
				unchanged = false
				break
			}
		}
		if unchanged {
			return false
		}
	}
	b.bytes = append(b.bytes[:0], value...)
	return true
}

// AppendRune appends r when its UTF-8 encoding fits in the buffer. It reports
// whether the stored value changed.
func (b *Buffer) AppendRune(r rune) bool {
	if b == nil || !utf8.ValidRune(r) {
		return false
	}
	var encoded [utf8.UTFMax]byte
	size := utf8.EncodeRune(encoded[:], r)
	if b.Len()+size > b.Limit() {
		return false
	}
	b.bytes = append(b.bytes, encoded[:size]...)
	return true
}

// Backspace removes the final UTF-8 rune. It reports whether the stored value
// changed.
func (b *Buffer) Backspace() bool {
	if b == nil || len(b.bytes) == 0 {
		return false
	}
	_, size := utf8.DecodeLastRune(b.bytes)
	if size <= 0 {
		return false
	}
	b.bytes = b.bytes[:len(b.bytes)-size]
	return true
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
