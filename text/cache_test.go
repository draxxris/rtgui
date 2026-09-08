package text

import "testing"

// TestStringCacheTracksEveryMutation verifies immutable old snapshots and invalidation.
func TestStringCacheTracksEveryMutation(t *testing.T) {
	b := NewBuffer(128, "abc")
	old := b.String()
	b.AppendRune('d')
	if b.String() != "abcd" || old != "abc" {
		t.Fatal("insertion cache or immutable snapshot failed")
	}
	b.Backspace()
	if b.String() != "abc" {
		t.Fatal("backspace cache failed")
	}
	b.SetCaret(0)
	b.Delete()
	if b.String() != "bc" {
		t.Fatal("delete cache failed")
	}
	b.SelectAll()
	b.InsertString("日本")
	if b.String() != "日本" {
		t.Fatal("paste cache failed")
	}
	b.SelectAll()
	b.DeleteSelection()
	if b.String() != "" {
		t.Fatal("selection cache failed")
	}
	b.Set("value")
	if b.String() != "value" {
		t.Fatal("set cache failed")
	}
	if n := testing.AllocsPerRun(100, func() { _ = b.String() }); n != 0 {
		t.Fatalf("cached string allocates %g", n)
	}
}
