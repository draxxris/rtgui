package widgets_test

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/widgets"
)

func chatTestSegments() []core.RichSegment {
	return []core.RichSegment{
		{Text: "Need "},
		{Text: "iron plate", Link: core.Link{Kind: core.LinkItem, Target: "iron-plate"}},
		{Text: " whisper "},
		{Text: "Mor'nor", Link: core.Link{Kind: core.LinkPlayer, Target: "Mor'nor"}},
	}
}

// TestChatLogAppendsWithStableIDs checks append IDs and counts.
func TestChatLogAppendsWithStableIDs(t *testing.T) {
	log := widgets.NewChatLog("chat", core.Rect{W: 200, H: 100}, 0)
	if log.MaxMessages() != widgets.DefaultChatLogMaxMessages {
		t.Fatalf("default capacity = %d", log.MaxMessages())
	}
	first, ok := log.AddText("hello")
	if !ok || first == 0 || log.MessageCount() != 1 {
		t.Fatalf("add text = %d/%v count %d", first, ok, log.MessageCount())
	}
	second, ok := log.AddMessage(chatTestSegments())
	if !ok || second == first || log.MessageCount() != 2 {
		t.Fatalf("add segments = %d/%v count %d", second, ok, log.MessageCount())
	}
	if got := log.IndexOfMessage(first); got != 0 {
		t.Fatalf("index of first = %d", got)
	}
	if _, ok := log.MessageByID(second); !ok {
		t.Fatal("second message not found by ID")
	}
	if got := log.Text(); got != "hello\nNeed iron plate whisper Mor'nor" {
		t.Fatalf("plain text = %q", got)
	}
}

// TestChatLogCopiesSegments checks input and output mutation isolation.
func TestChatLogCopiesSegments(t *testing.T) {
	log := widgets.NewChatLog("chat", core.Rect{}, 10)
	input := chatTestSegments()
	if _, ok := log.AddMessage(input); !ok {
		t.Fatal("add failed")
	}
	input[0].Text = "mutated"
	snapshot, ok := log.MessageAt(0)
	if !ok || snapshot.Segments[0].Text != "Need " {
		t.Fatalf("input mutation leaked: %+v", snapshot)
	}
	snapshot.Segments[0].Text = "mutated"
	again, _ := log.MessageAt(0)
	if again.Segments[0].Text != "Need " {
		t.Fatal("output mutation leaked into log")
	}
	scratch := log.CopyMessageSegmentsInto(0, nil)
	if len(scratch) != len(chatTestSegments()) {
		t.Fatalf("scratch copy len = %d", len(scratch))
	}
}

// TestChatLogEvictsOldest checks bounded history and ID stability.
func TestChatLogEvictsOldest(t *testing.T) {
	log := widgets.NewChatLog("chat", core.Rect{}, 2)
	first, _ := log.AddText("one")
	second, _ := log.AddText("two")
	third, _ := log.AddText("three")
	if log.MessageCount() != 2 {
		t.Fatalf("count = %d", log.MessageCount())
	}
	if log.IndexOfMessage(first) >= 0 {
		t.Fatal("evicted ID still resolves")
	}
	if log.IndexOfMessage(second) != 0 || log.IndexOfMessage(third) != 1 {
		t.Fatal("survivor indices wrong")
	}
	if !log.Clear() || log.MessageCount() != 0 || log.Clear() {
		t.Fatal("clear semantics wrong")
	}
}

// TestChatLogScrollClamps checks offset clamping and bottom stick state.
func TestChatLogScrollClamps(t *testing.T) {
	log := widgets.NewChatLog("chat", core.Rect{}, 10)
	if !log.IsAtBottom() || !log.AutoStick() {
		t.Fatal("new log must start glued with stick on")
	}
	log.SetMaxScroll(100)
	if log.IsAtBottom() {
		t.Fatal("zero offset with limit must not read as bottom")
	}
	if !log.ScrollToBottom() || !log.IsAtBottom() {
		t.Fatal("scroll to bottom failed")
	}
	if log.ScrollToBottom() {
		t.Fatal("second scroll to bottom must report no change")
	}
	if !log.ScrollBy(-10) || log.ScrollOffset() != 90 {
		t.Fatalf("scroll by = %v", log.ScrollOffset())
	}
	if log.SetScrollOffset(90) {
		t.Fatal("same offset must report no change")
	}
	if !log.EnsureScrollBounds(300, 100) || log.MaxScroll() != 200 {
		t.Fatalf("ensure bounds max = %v", log.MaxScroll())
	}
	if log.EnsureScrollBounds(300, 100) {
		t.Fatal("steady bounds must report no change")
	}
	log.SetAutoStick(false)
	if log.AutoStick() {
		t.Fatal("stick disable failed")
	}
}

// TestChatLogLinks checks per-message link accessors.
func TestChatLogLinks(t *testing.T) {
	log := widgets.NewChatLog("chat", core.Rect{}, 10)
	if _, ok := log.AddMessage(chatTestSegments()); !ok {
		t.Fatal("add failed")
	}
	if got := log.MessageLinkCount(0); got != 2 {
		t.Fatalf("link count = %d", got)
	}
	link, seg, ok := log.MessageLinkAt(0, 1)
	if !ok || link.Kind != core.LinkPlayer || seg != 3 {
		t.Fatalf("link 1 = %+v/%d/%v", link, seg, ok)
	}
	if _, _, ok := log.MessageLinkAt(0, 2); ok {
		t.Fatal("out-of-range link accepted")
	}
	if _, _, ok := log.MessageLinkAt(9, 0); !ok {
		// out-of-range message must fail; invert check below
	} else {
		t.Fatal("out-of-range message accepted")
	}
}

// TestChatLogNilReaders checks nil receivers stay zero-valued.
func TestChatLogNilSafety(t *testing.T) {
	var log *widgets.ChatLog
	if _, ok := log.AddText("x"); ok {
		t.Fatal("nil add must fail")
	}
	if _, ok := log.AddMessage(nil); ok {
		t.Fatal("nil add must fail")
	}
	if log.MessageCount() != 0 || log.Text() != "" || log.MaxMessages() != widgets.DefaultChatLogMaxMessages {
		t.Fatal("nil readers must be zero")
	}
	if _, ok := log.MessageAt(0); ok {
		t.Fatal("nil message lookup must fail")
	}
	if log.Clear() || log.ScrollToBottom() {
		t.Fatal("nil mutations must report no change")
	}
}

// TestChatLogNilMutations checks nil scroll and setter paths stay quiet.
func TestChatLogNilMutations(t *testing.T) {
	var log *widgets.ChatLog
	if log.SetScrollOffset(1) || log.ScrollBy(1) || log.EnsureScrollBounds(1, 1) {
		t.Fatal("nil mutations must report no change")
	}
	if log.IsAtBottom() || log.AutoStick() {
		t.Fatal("nil stick state must be false")
	}
	if log.OnLinkClick(nil) != nil || log.OnLinkTooltipRequested(nil) != nil {
		t.Fatal("nil fluent setters must return nil")
	}
}

// TestChatLogSetMaxMessages checks capacity shrink and defaults.
func TestChatLogSetMaxMessages(t *testing.T) {
	log := widgets.NewChatLog("chat", core.Rect{}, 3)
	log.AddText("a")
	log.AddText("b")
	log.AddText("c")
	log.SetMaxMessages(2)
	if log.MessageCount() != 2 || log.MaxMessages() != 2 {
		t.Fatalf("shrink count = %d cap %d", log.MessageCount(), log.MaxMessages())
	}
	log.SetMaxMessages(0)
	if log.MaxMessages() != widgets.DefaultChatLogMaxMessages {
		t.Fatal("non-positive capacity must select default")
	}
	if _, err := widgets.AsChatLog(log); err != nil {
		t.Fatalf("cast failed: %v", err)
	}
	if _, err := widgets.AsChatLog(widgets.NewLabel("label", core.Rect{}, "x")); err == nil {
		t.Fatal("wrong-kind cast must fail")
	}
}
