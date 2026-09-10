package ui

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/widgets"
)

// TestClassedRichTooltipDrawsOnHover verifies a classed shell draws for hover.
func TestClassedRichTooltipDrawsOnHover(t *testing.T) {
	u := New(800, 600)
	button := widgets.NewButton("slot", core.Rect{X: 50, Y: 50, W: 100, H: 30}, "Slot")
	mustAdd(t, u, button)
	u.SetRichTooltip("slot", core.RichTooltip{Title: "Ashwood Thorn Shortbow", Class: "item"})
	u.HandleMouse(MouseEvent{Pos: centerOf(button)})
	recorder := attachDrawRecorder(t, u)
	u.DrawWidgets()
	u.DrawPopup()
	if !hasKindPart(recorder.Calls(), core.WidgetTooltip, skin.PartBackground) {
		t.Fatal("classed tooltip shell did not draw")
	}
	if !u.richTipCache.Valid() || u.richTipCache.Bounds().W <= 0 {
		t.Fatal("classed tooltip did not lay out")
	}
}
