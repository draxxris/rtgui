package widgets_test

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/widgets"
)

// richTestSegments returns a stable mixed message for tests.
func richTestSegments() []core.RichSegment {
	return []core.RichSegment{
		{Text: "Need "},
		{Text: "Thunderfury", Color: core.Color{R: 255, G: 140, B: 40, A: 255}, HasColor: true,
			Link: core.Link{Kind: core.LinkItem, Target: "item:19019"}},
		{Text: " whisper "},
		{Text: "Mor'nor", Link: core.Link{Kind: core.LinkPlayer, Target: "Mor'nor"}},
		{Text: "."},
	}
}

// TestRichTextCopiesSegments checks segment copy semantics for I/O slices.
func TestRichTextCopiesSegments(t *testing.T) {
	input := richTestSegments()
	message := widgets.NewRichText("chat", core.Rect{X: 10, Y: 10, W: 200, H: 60}, input)
	input[1].Text = "mutated input"
	if got := message.RichSegments()[1].Text; got != "Thunderfury" {
		t.Fatalf("input mutation changed segment to %q", got)
	}
	snapshot := message.RichSegments()
	snapshot[3].Text = "mutated output"
	if got := message.RichSegments()[3].Text; got != "Mor'nor" {
		t.Fatalf("output mutation changed segment to %q", got)
	}
	if message.SetRichSegments(richTestSegments()) {
		t.Fatal("identical segments reported a change")
	}
	changed := []core.RichSegment{{Text: "other"}}
	if !message.SetRichSegments(changed) || message.RichPlainText() != "other" {
		t.Fatalf("changed segments = %q", message.RichPlainText())
	}
	if message.SetRichSegments(changed) {
		t.Fatal("equal segments reported a change")
	}
}

// TestRichTextLinkAccessors checks plain text, counts, and indexed links.
func TestRichTextLinkAccessors(t *testing.T) {
	message := widgets.NewRichText("chat", core.Rect{}, richTestSegments())
	if got := message.RichPlainText(); got != "Need Thunderfury whisper Mor'nor." {
		t.Fatalf("plain text = %q", got)
	}
	if got := message.LinkCount(); got != 2 {
		t.Fatalf("link count = %d", got)
	}
	link, ok := message.LinkAt(1)
	if !ok || link.Kind != core.LinkPlayer || link.Target != "Mor'nor" {
		t.Fatalf("link 1 = %+v/%v", link, ok)
	}
	if _, ok := message.LinkAt(2); ok {
		t.Fatal("out-of-range link index accepted")
	}
	if _, ok := message.LinkAt(-1); ok {
		t.Fatal("negative link index accepted")
	}
	var nilWidget *widgets.Widget
	if nilWidget.RichSegments() != nil || nilWidget.RichPlainText() != "" || nilWidget.LinkCount() != 0 {
		t.Fatal("nil widget rich accessors must be zero")
	}
	if _, ok := nilWidget.LinkAt(0); ok {
		t.Fatal("nil widget link lookup must fail")
	}
	if widgets.NewLabel("label", core.Rect{}, "x").LinkCount() != 0 {
		t.Fatal("non-rich widget reported links")
	}
}
