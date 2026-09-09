package render

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/transform"
	"github.com/draxxris/rtgui/widgets"
)

// TestChatLogContentIsNilSafe verifies the viewport degrades to bounds.
func TestChatLogContentIsNilSafe(t *testing.T) {
	bounds := core.Rect{X: 10, Y: 20, W: 220, H: 100}
	var nilTheme *Theme
	if got := nilTheme.ChatLogContent(bounds, core.StateNormal); got != bounds {
		t.Fatalf("nil theme content = %+v", got)
	}
	theme := NewTheme(transform.New(core.Viewport{}))
	if got := theme.ChatLogContent(bounds, core.StateNormal); got != bounds {
		t.Fatalf("unskinned content = %+v", got)
	}
}

// TestChatMessageHeightMinimums checks empty and narrow inputs stay visible.
func TestChatMessageHeightMinimums(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	if got := theme.ChatMessageHeight(200, nil); got != RichLineHeight {
		t.Fatalf("empty height = %v", got)
	}
	if got := theme.ChatMessageHeight(0, []core.RichSegment{{Text: "hi"}}); got != RichLineHeight {
		t.Fatalf("zero width height = %v", got)
	}
	single := theme.ChatMessageHeight(200, []core.RichSegment{{Text: "hi"}})
	if single != RichLineHeight {
		t.Fatalf("single line height = %v, want %v", single, RichLineHeight)
	}
	long := theme.ChatMessageHeight(40, []core.RichSegment{{Text: "a very long chat line that must wrap onto several rows of text"}})
	if long <= single {
		t.Fatalf("wrapped height %v must exceed single %v", long, single)
	}
}

// TestChatTotalHeightAndWindow checks stacking math and visible windows.
func TestChatTotalHeightAndWindow(t *testing.T) {
	heights := []float32{24, 48, 24}
	if got := ChatTotalHeight(heights, 4); got != 24+48+24+8 {
		t.Fatalf("total = %v", got)
	}
	if got := ChatTotalHeight(nil, 4); got != 0 {
		t.Fatalf("empty total = %v", got)
	}
	first, last := ChatWindow(heights, 4, 0, 100)
	if first != 0 || last != 2 {
		t.Fatalf("window = %d/%d", first, last)
	}
	first, last = ChatWindow(heights, 4, 28, 24)
	if first != 1 || last != 1 {
		t.Fatalf("scrolled window = %d/%d", first, last)
	}
	if first, last := ChatWindow(nil, 4, 0, 100); last >= first {
		t.Fatal("empty window must be empty")
	}
	content := core.Rect{X: 10, Y: 20, W: 200, H: 100}
	rect, ok := ChatMessageRect(content, heights, 4, 0, 1)
	if !ok || rect.Y != 20+24+4 || rect.H != 48 {
		t.Fatalf("message rect = %+v/%v", rect, ok)
	}
	if _, ok := ChatMessageRect(content, heights, 4, 0, 9); ok {
		t.Fatal("out-of-range rect must miss")
	}
}

// TestChatLogDrawsMessages checks the shell and text calls through a recorder.
func TestChatLogDrawsMessages(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	recorder, err := NewDrawRecorder(256)
	if err != nil {
		t.Fatal(err)
	}
	theme.SetDrawRecorder(recorder)
	log := widgets.NewChatLog("chat", core.Rect{X: 10, Y: 20, W: 220, H: 100}, 10)
	log.AddText("hello world")
	rows := core.Rect{X: 10, Y: 20, W: 220, H: 100}
	heights := []float32{theme.ChatMessageHeight(rows.W, []core.RichSegment{{Text: "hello world"}})}
	info := log.Snapshot(core.StateNormal)
	theme.DrawChatLog(info, log, rows, heights, widgets.ChatLogMessageGap, 0, nil, -1, -1, core.StateNormal)
	foundText := false
	for _, call := range recorder.Calls() {
		if call.Kind == core.WidgetChatLog {
			foundText = true
		}
	}
	if !foundText {
		t.Fatal("chat draw recorded no chat calls")
	}
}
