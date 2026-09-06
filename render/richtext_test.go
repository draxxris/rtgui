package render

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/transform"
)

// richLayoutTheme builds a theme with a logical viewport for layout tests.
func richLayoutTheme() *Theme {
	return NewTheme(transform.New(core.Viewport{
		Viewport:    core.Rect{W: 400, H: 300},
		LogicalSize: core.Vec2{X: 400, Y: 300},
	}))
}

// richFragmentedSpans lays out a link that must wrap for layout tests.
func richFragmentedSpans(t *testing.T) []RichSpanLayout {
	t.Helper()
	segments := []core.RichSegment{
		{Text: "See "},
		{Text: "Thunderfury Blessed Blade", Link: core.Link{Kind: core.LinkItem, Target: "item:19019"}},
		{Text: " ok"},
	}
	spans := richLayoutTheme().LayoutRichSpans(core.Rect{X: 10, Y: 10, W: 90, H: 200}, segments, core.StateNormal)
	if len(spans) == 0 {
		t.Fatal("no spans laid out")
	}
	return spans
}

// TestRichLayoutFragmentsLinksAcrossRows checks wrap fragments share segments.
func TestRichLayoutFragmentsLinksAcrossRows(t *testing.T) {
	spans := richFragmentedSpans(t)
	linkFrags := 0
	rows := map[float32]bool{}
	for _, span := range spans {
		if span.Segment == 1 && span.Linked() {
			linkFrags++
			rows[span.Bounds.Y] = true
		}
		if span.Text == "" {
			t.Fatal("empty fragment text laid out")
		}
	}
	if linkFrags < 2 || len(rows) < 2 {
		t.Fatalf("link did not fragment across rows: %d frags/%d rows", linkFrags, len(rows))
	}
}

// TestRichSpanAtInvertsFragmentBounds checks hit testing per fragment.
func TestRichSpanAtInvertsFragmentBounds(t *testing.T) {
	spans := richFragmentedSpans(t)
	for _, span := range spans {
		if !span.Linked() {
			continue
		}
		middle := core.Vec2{X: span.Bounds.X + span.Bounds.W/2, Y: span.Bounds.Y + span.Bounds.H/2}
		fragment, ok := RichSpanAt(spans, middle)
		if !ok || !fragment.Linked() || fragment.Segment != span.Segment || fragment.Link.Target != "item:19019" {
			t.Fatalf("link at %+v = %+v/%v", middle, fragment, ok)
		}
	}
	if _, ok := RichSpanAt(spans, core.Vec2{X: 0, Y: 0}); ok {
		t.Fatal("outside miss reported a fragment")
	}
	if _, ok := RichSpanAt(nil, core.Vec2{}); ok {
		t.Fatal("empty span lookup hit")
	}
}

// TestRichContentHeightGrowsWithWrappedLines checks auto-height behavior.
func TestRichContentHeightGrowsWithWrappedLines(t *testing.T) {
	theme := richLayoutTheme()
	oneLine := []core.RichSegment{{Text: "hi"}}
	many := []core.RichSegment{{Text: "alpha beta gamma delta epsilon zeta eta theta"}}
	short := theme.RichContentHeight(300, oneLine)
	tall := theme.RichContentHeight(90, many)
	if short <= 0 || tall <= short {
		t.Fatalf("content heights = %v/%v", short, tall)
	}
	if got := theme.RichContentHeight(0, many); got != 0 {
		t.Fatalf("zero width height = %v", got)
	}
	if got := theme.RichContentHeight(300, nil); got != 0 {
		t.Fatalf("unskinned empty message height = %v", got)
	}
	var nilTheme *Theme
	if content := nilTheme.RichContent(core.Rect{W: 10, H: 10}, core.StateNormal); content != (core.Rect{W: 10, H: 10}) {
		t.Fatalf("nil theme content = %+v", content)
	}
}

// TestRichDrawRecordsTextUnderlineAndHover checks the fragment draw calls.
func TestRichDrawRecordsTextUnderlineAndHover(t *testing.T) {
	theme := richLayoutTheme()
	recorder, err := NewDrawRecorder(64)
	if err != nil {
		t.Fatal(err)
	}
	theme.SetDrawRecorder(recorder)
	segments := []core.RichSegment{
		{Text: "hi "},
		{Text: "item", Link: core.Link{Kind: core.LinkItem, Target: "item:1"}},
	}
	info := core.WidgetInfo{Name: "chat", Bounds: core.Rect{W: 300, H: 60}, Kind: core.WidgetRichText, State: core.StateNormal}
	theme.DrawRichText(info, segments, 1)
	calls := recorder.Calls()
	if !hasPart(calls, skin.PartText) || !hasPart(calls, skin.PartOverlay) {
		t.Fatalf("rich draw calls missing text or hover: %+v", calls)
	}
	before := len(calls)
	theme.DrawRichText(info, nil, -1)
	if got := len(recorder.Calls()); got != before {
		t.Fatalf("empty draw recorded %d calls, want %d", got, before)
	}
}

// hasPart searches recorder calls for one skin part.
func hasPart(calls []DrawCall, part skin.SkinPart) bool {
	for _, call := range calls {
		if call.Part == part {
			return true
		}
	}
	return false
}
