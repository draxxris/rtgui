package render

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
)

// TestRichTooltipClassBustsCache verifies a class change rebuilds layout.
func TestRichTooltipClassBustsCache(t *testing.T) {
	theme := richTooltipTestTheme()
	logical := core.Vec2{X: 800, Y: 600}
	anchor := core.Vec2{X: 30, Y: 40}
	data := richTooltipTestData()
	var cache RichTooltipCache
	if !cache.Update(theme, data, anchor, logical, 8) {
		t.Fatal("first update must build")
	}
	if cache.Update(theme, data, anchor, logical, 8) {
		t.Fatal("steady update must not rebuild")
	}
	classed := data
	classed.Class = "item"
	if !cache.Update(theme, classed, anchor, logical, 8) {
		t.Fatal("class change must rebuild")
	}
	if cache.Update(theme, classed, anchor, logical, 8) {
		t.Fatal("steady classed update must not rebuild")
	}
	if cache.data.Class != "item" {
		t.Fatalf("retained class = %q", cache.data.Class)
	}
}

// TestRichTooltipClassedShellRecordsClass verifies draws carry the class.
func TestRichTooltipClassedShellRecordsClass(t *testing.T) {
	theme := richTooltipTestTheme()
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetTooltip, Class: "item", Part: skin.PartBackground, State: core.StateNormal}, skin.SkinDescriptor{
		BackgroundColor: core.Color{R: 11, G: 21, B: 36, A: 255}, HasBackgroundColor: true,
	})
	recorder, err := NewDrawRecorder(256)
	if err != nil {
		t.Fatal(err)
	}
	theme.SetDrawRecorder(recorder)
	logical := core.Vec2{X: 800, Y: 600}
	data := richTooltipTestData()
	data.Class = "item"
	var cache RichTooltipCache
	cache.Update(theme, data, core.Vec2{X: 30, Y: 40}, logical, 8)
	theme.BeginFrame()
	theme.DrawRichTooltip(core.WidgetInfo{Name: "tip", Kind: core.WidgetTooltip, State: core.StateNormal}, &cache)
	if got := recorder.LastWidgetInfo().Class; got != "item" {
		t.Fatalf("drawn class = %q, want item", got)
	}
	if !hasPart(recorder.Calls(), skin.PartBackground) {
		t.Fatal("classed tooltip missing shell")
	}
}

// TestRichTooltipClassedPaddingShapesBounds verifies class insets lay out.
func TestRichTooltipClassedPaddingShapesBounds(t *testing.T) {
	theme := richTooltipTestTheme()
	theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetTooltip, Class: "item", Part: skin.PartBackground, State: core.StateNormal}, skin.SkinDescriptor{
		PaddingLeft: 24, PaddingTop: 24, PaddingRight: 24, PaddingBottom: 24, HasPadding: true,
		BackgroundColor: core.Color{R: 11, G: 21, B: 36, A: 255}, HasBackgroundColor: true,
	})
	logical := core.Vec2{X: 800, Y: 600}
	anchor := core.Vec2{X: 30, Y: 40}
	data := richTooltipTestData()
	var plain RichTooltipCache
	plain.Update(theme, data, anchor, logical, 8)
	classed := data
	classed.Class = "item"
	var styled RichTooltipCache
	styled.Update(theme, classed, anchor, logical, 8)
	if styled.Bounds() == plain.Bounds() {
		t.Fatalf("classed bounds %+v must differ from plain %+v", styled.Bounds(), plain.Bounds())
	}
}
