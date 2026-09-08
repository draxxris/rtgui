package render

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"testing"
)

// TestRichTooltipOffscreenAnchorClamps checks pointers outside the logical viewport.
func TestRichTooltipOffscreenAnchorClamps(t *testing.T) {
	theme := richTooltipTestTheme()
	var cache RichTooltipCache
	cache.Update(theme, richTooltipTestData(), core.Vec2{X: 10000, Y: 10000}, core.Vec2{X: 300, Y: 200}, 8)
	b := cache.Bounds()
	if b.X < 0 || b.Y < 0 || b.X+b.W > 300 || b.Y+b.H > 200 {
		t.Fatalf("offscreen tooltip: %+v", b)
	}
}

// TestTooltipBodyDoesNotApplyRichInsetsTwice checks style isolation between widgets.
func TestTooltipBodyDoesNotApplyRichInsetsTwice(t *testing.T) {
	theme := richTooltipTestTheme()
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetRichText, Part: skin.PartBackground}, skin.SkinDescriptor{PaddingLeft: 12, PaddingTop: 9})
	spans := layoutRichTooltipBody(theme, []core.RichSegment{{Text: "hello"}}, 200, 100)
	if len(spans) != 1 || spans[0].Bounds.X != 0 || spans[0].Bounds.Y != 0 {
		t.Fatalf("tooltip inherited extra chat offsets: %+v", spans)
	}
}

// TestClipStackIntersectsAndRestores verifies the headless scissor ownership stack.
func TestClipStackIntersectsAndRestores(t *testing.T) {
	theme := richTooltipTestTheme()
	theme.PushClip(core.Rect{X: 10, Y: 10, W: 100, H: 100})
	theme.PushClip(core.Rect{X: 50, Y: 50, W: 100, H: 100})
	if theme.clips[1] != (core.Rect{X: 50, Y: 50, W: 60, H: 60}) {
		t.Fatalf("nested clip: %+v", theme.clips)
	}
	theme.PopClip()
	if len(theme.clips) != 1 {
		t.Fatal("inner pop removed outer clip")
	}
	theme.PopClip()
	if len(theme.clips) != 0 {
		t.Fatal("clip stack retained active clipping")
	}
}
