package ui

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/widgets"
)

// richTestMessage returns a registered message with plain and linked runs.
func richTestMessage(t *testing.T, u *UI) *widgets.RichText {
	t.Helper()
	message := widgets.NewRichText("chat", core.Rect{X: 20, Y: 20, W: 300, H: 60}, []core.RichSegment{
		{Text: "Need "},
		{Text: "Thunderfury", Link: core.Link{Kind: core.LinkItem, Target: "item:19019"}},
		{Text: " thanks"},
		{Text: "wiki", Link: core.Link{Kind: core.LinkURL, Target: "https://example.com"}},
	})
	mustAdd(t, u, message)
	return message
}

// richSegCenter resolves the first fragment center of one segment for tests.
func richSegCenter(u *UI, message *widgets.RichText, segment int) core.Vec2 {
	for _, span := range u.richSpans(message) {
		if span.Segment == segment {
			return core.Vec2{X: span.Bounds.X + span.Bounds.W/2, Y: span.Bounds.Y + span.Bounds.H/2}
		}
	}
	return core.Vec2{}
}

// TestRichTextLinkClickCommits checks press-release selection semantics.
func TestRichTextLinkClickCommits(t *testing.T) {
	u := New(400, 200)
	message := richTestMessage(t, u)
	var clicked core.Link
	clicks := 0
	u.OnLinkClick("chat", func(link core.Link) { clicked = link })
	u.OnClick("chat", func() { clicks++ })
	clickAt(u, richSegCenter(u, message, 1))
	if clicked.Target != "item:19019" || clicks != 0 {
		t.Fatalf("link click = %+v/%d", clicked, clicks)
	}
	clickAt(u, richSegCenter(u, message, 0))
	if clicks != 1 || clicked.Target != "item:19019" {
		t.Fatalf("plain click = %d/%+v", clicks, clicked)
	}
	linkPoint := richSegCenter(u, message, 3)
	plainPoint := richSegCenter(u, message, 2)
	u.HandleMouse(MouseEvent{Pos: linkPoint, Pressed: true})
	u.HandleMouse(MouseEvent{Pos: plainPoint, Released: true})
	if clicks != 1 || clicked.Target != "item:19019" {
		t.Fatal("link drag-cancel must consume without callbacks")
	}
}

// TestRichTextTooltipRequestedOnce checks the hover edge protocol.
func TestRichTextTooltipRequestedOnce(t *testing.T) {
	u := New(400, 200)
	message := richTestMessage(t, u)
	requests := 0
	u.OnLinkTooltipRequested("chat", func(core.Link) string {
		requests++
		return "dynamic tip"
	})
	point := richSegCenter(u, message, 1)
	u.HandleMouse(MouseEvent{Pos: point})
	u.HandleMouse(MouseEvent{Pos: point})
	u.HandleMouse(MouseEvent{Pos: point})
	if requests != 1 || u.tipText != "dynamic tip" {
		t.Fatalf("tooltip requests = %d/%q", requests, u.tipText)
	}
	u.HandleMouse(MouseEvent{Pos: richSegCenter(u, message, 3)})
	if requests != 2 {
		t.Fatalf("second link hover requests = %d", requests)
	}
	u.HandleMouse(MouseEvent{Pos: richSegCenter(u, message, 0)})
	if u.tipText != "" {
		t.Fatal("plain hover must clear the link tip")
	}
}

// TestRichTextStaticTooltipBeatsCallback checks static link text precedence.
func TestRichTextStaticTooltipBeatsCallback(t *testing.T) {
	u := New(400, 200)
	message := widgets.NewRichText("chat", core.Rect{X: 20, Y: 20, W: 300, H: 60}, []core.RichSegment{
		{Text: "hi ", Link: core.Link{Kind: core.LinkItem, Target: "item:1", Tooltip: "static tip"}},
	})
	mustAdd(t, u, message)
	requests := 0
	u.OnLinkTooltipRequested("chat", func(core.Link) string {
		requests++
		return "dynamic tip"
	})
	u.HandleMouse(MouseEvent{Pos: richSegCenter(u, message, 0)})
	if requests != 0 || u.tipText != "static tip" {
		t.Fatalf("static tooltip = %d/%q", requests, u.tipText)
	}
	recorder := attachDrawRecorder(t, u)
	u.DrawWidgets()
	u.DrawPopup()
	if !hasKindPart(recorder.Calls(), core.WidgetTooltip, skin.PartText) {
		t.Fatal("link tip did not draw")
	}
}

// TestActivateLinkSemantics checks pointer-free link activation.
func TestActivateLinkSemantics(t *testing.T) {
	u := New(400, 200)
	message := richTestMessage(t, u)
	var clicked core.Link
	u.OnLinkClick("chat", func(link core.Link) { clicked = link })
	if !u.ActivateLink("chat", 1) || clicked.Target != "https://example.com" {
		t.Fatalf("semantic link = %+v", clicked)
	}
	if u.ActivateLink("chat", 2) || u.ActivateLink("missing", 0) || u.ActivateLink("chat", -1) {
		t.Fatal("invalid semantic link accepted")
	}
	button := widgets.NewButton("button", core.Rect{}, "button")
	mustAdd(t, u, button)
	if u.ActivateLink("button", 0) {
		t.Fatal("non-rich widget accepted link activation")
	}
	message.SetEnabled(false)
	if u.ActivateLink("chat", 0) {
		t.Fatal("disabled message accepted link activation")
	}
}

// TestHoveredLinkSupportsCursor checks the cursor polling accessor.
func TestHoveredLinkSupportsCursor(t *testing.T) {
	u := New(400, 200)
	message := richTestMessage(t, u)
	u.HandleMouse(MouseEvent{Pos: richSegCenter(u, message, 1)})
	name, link, seg, ok := u.HoveredLink()
	if !ok || name != "chat" || link.Target != "item:19019" || seg != 1 {
		t.Fatalf("hovered link = %q/%+v/%d/%v", name, link, seg, ok)
	}
	u.HandleMouse(MouseEvent{Pos: richSegCenter(u, message, 0)})
	if _, _, _, ok := u.HoveredLink(); ok {
		t.Fatal("plain hover reported a link")
	}
	var nilUI *UI
	if _, _, _, ok := nilUI.HoveredLink(); ok {
		t.Fatal("nil UI reported a link")
	}
	if !u.Activate("chat") {
		t.Fatal("rich Activate must re-fire OnClick")
	}
}
