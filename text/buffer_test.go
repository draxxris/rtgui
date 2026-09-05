package text

import "testing"

// TestBufferStoresValidASCIIAndMultibyteText verifies normal UTF-8 storage
// and that a returned string remains independent of subsequent edits.
func TestBufferStoresValidASCIIAndMultibyteText(t *testing.T) {
	b := NewBuffer(16, "hello")
	if got, want := b.String(), "hello"; got != want {
		t.Fatalf("initial value = %q, want %q", got, want)
	}
	if got, want := b.Len(), len("hello"); got != want {
		t.Fatalf("Len() = %d, want %d", got, want)
	}
	if !b.Set("aé😀") {
		t.Fatal("Set must report a changed multibyte value")
	}
	if got, want := b.String(), "aé😀"; got != want {
		t.Fatalf("Set value = %q, want %q", got, want)
	}
	if b.Set("aé😀") {
		t.Fatal("Set must not report an unchanged value")
	}
	value := b.String()
	if !b.AppendRune('中') {
		t.Fatal("AppendRune must append a fitting rune")
	}
	if value != "aé😀" {
		t.Fatalf("String result changed after edit: %q", value)
	}
}

// TestBufferRejectsInvalidUTF8WithoutMutation verifies invalid input cannot
// alter either initialized or populated buffers.
func TestBufferRejectsInvalidUTF8WithoutMutation(t *testing.T) {
	invalid := string([]byte{'x', 0xff})
	b := NewBuffer(8, "keep")
	if b.Set(invalid) {
		t.Fatal("Set accepted invalid UTF-8")
	}
	if got, want := b.String(), "keep"; got != want {
		t.Fatalf("invalid Set changed value to %q, want %q", got, want)
	}
	if b.AppendRune(0xd800) {
		t.Fatal("AppendRune accepted an invalid rune")
	}
	if got, want := NewBuffer(8, invalid).String(), ""; got != want {
		t.Fatalf("invalid initial value = %q, want empty", got)
	}
}

// TestBufferHonorsLimitAtRuneBoundaries verifies exact limits, safe
// truncation, and constructor limits that cannot grow for initial text.
func TestBufferHonorsLimitAtRuneBoundaries(t *testing.T) {
	exact := NewBuffer(7, "aé😀")
	if got, want := exact.String(), "aé😀"; got != want {
		t.Fatalf("exact-limit initial value = %q, want %q", got, want)
	}
	if got, want := exact.Len(), 7; got != want {
		t.Fatalf("exact-limit Len() = %d, want %d", got, want)
	}
	if got, want := exact.Limit(), 7; got != want {
		t.Fatalf("Limit() = %d, want %d", got, want)
	}

	truncated := NewBuffer(6, "")
	if !truncated.Set("aé😀") {
		t.Fatal("truncated Set must report a changed value")
	}
	if got, want := truncated.String(), "aé"; got != want {
		t.Fatalf("rune-boundary truncation = %q, want %q", got, want)
	}
	negative := NewBuffer(-1, "x")
	if got, want := negative.Limit(), 0; got != want {
		t.Fatalf("negative limit = %d, want %d", got, want)
	}
	if got := negative.String(); got != "" {
		t.Fatalf("negative-limit initial value = %q, want empty", got)
	}
	limitedInitial := NewBuffer(1, "é")
	if got, want := limitedInitial.Limit(), 1; got != want {
		t.Fatalf("initial limit grew to %d, want %d", got, want)
	}
	if got := limitedInitial.String(); got != "" {
		t.Fatalf("oversized initial value = %q, want empty", got)
	}
}

// TestBufferAppendAndBackspaceResults verifies edit results at full and empty
// boundaries and removal of both one-byte and four-byte runes.
func TestBufferAppendAndBackspaceResults(t *testing.T) {
	b := NewBuffer(5, "")
	if !b.AppendRune('😀') || !b.AppendRune('a') {
		t.Fatal("fitting runes must append")
	}
	if b.AppendRune('b') {
		t.Fatal("full buffer append must return false")
	}
	if !b.Backspace() {
		t.Fatal("backspace must remove the final one-byte rune")
	}
	if got, want := b.String(), "😀"; got != want {
		t.Fatalf("one-byte backspace value = %q, want %q", got, want)
	}
	if !b.Backspace() {
		t.Fatal("backspace must remove the final four-byte rune")
	}
	if got := b.String(); got != "" {
		t.Fatalf("four-byte backspace value = %q, want empty", got)
	}
	if b.Backspace() {
		t.Fatal("empty-buffer backspace must return false")
	}
}

// TestBufferEditsReuseCapacity verifies Set, append, and backspace retain
// storage capacity, while normal append/backspace perform no allocations.
func TestBufferEditsReuseCapacity(t *testing.T) {
	b := NewBuffer(16, "")
	capacity := cap(b.bytes)
	for range 100 {
		if !b.Set("aé") || !b.AppendRune('😀') || !b.Backspace() || !b.Set("") {
			t.Fatal("expected each repeated edit to change the value")
		}
	}
	if got := cap(b.bytes); got != capacity {
		t.Fatalf("buffer capacity grew from %d to %d", capacity, got)
	}
	if allocs := testing.AllocsPerRun(1000, func() {
		b.Set("aé")
		b.Set("")
	}); allocs != 0 {
		t.Fatalf("Set allocated %v times per run", allocs)
	}
	if allocs := testing.AllocsPerRun(1000, func() {
		b.AppendRune('x')
		b.Backspace()
	}); allocs != 0 {
		t.Fatalf("append/backspace allocated %v times per run", allocs)
	}
}
