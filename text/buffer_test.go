package text

import (
	"testing"
	"unicode/utf8"
)

func TestBufferEditing(t *testing.T) {
	b := NewBuffer(32, "hello")
	if b.String() != "hello" {
		t.Fatalf("initial value %q", b.String())
	}
	b.Set("aé😀中")
	if b.String() != "aé😀中" {
		t.Fatalf("unicode value %q", b.String())
	}
	b.Set("hello world overflow")
	if len(b.String()) >= b.Cap {
		t.Fatalf("buffer was not truncated: %q", b.String())
	}
}

func TestBufferCapacityAndUTF8(t *testing.T) {
	for _, capacity := range []int{-1, 0, 1, 2, 3} {
		b := NewBuffer(capacity, "😀é")
		if !utf8.ValidString(b.String()) {
			t.Fatalf("invalid initial UTF-8 for capacity %d: %q", capacity, b.String())
		}
		b.Set("😀é中")
		if !utf8.ValidString(b.String()) {
			t.Fatalf("invalid truncated UTF-8 for capacity %d: %q", capacity, b.String())
		}
	}
}
