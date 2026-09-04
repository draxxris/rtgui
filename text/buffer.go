// Package text provides text editing primitives used by widgets.
package text

import "unicode/utf8"

// Buffer stores a NUL-terminated UTF-8 string. Cap includes the terminator.
type Buffer struct {
	Buf []byte
	Cap int
}

func NewBuffer(capacity int, initial string) *Buffer {
	if capacity <= len(initial) {
		capacity = len(initial) + 1
	}
	if capacity < 1 {
		capacity = 1
	}
	initial = truncateUTF8(initial, capacity-1)
	b := make([]byte, capacity)
	copy(b, initial)
	b[len(initial)] = 0
	return &Buffer{Buf: b, Cap: capacity}
}

func (b *Buffer) String() string {
	n := 0
	for n < len(b.Buf) && b.Buf[n] != 0 {
		n++
	}
	return string(b.Buf[:n])
}

func (b *Buffer) Set(s string) {
	if b.Cap < 1 {
		b.Cap = 1
	}
	s = truncateUTF8(s, b.Cap-1)
	b.Buf = make([]byte, b.Cap)
	copy(b.Buf, s)
	b.Buf[len(s)] = 0
}

func (b *Buffer) Bytes() []byte { return b.Buf }

func truncateUTF8(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(s) <= maxBytes {
		return s
	}
	s = s[:maxBytes]
	for len(s) > 0 && !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}
