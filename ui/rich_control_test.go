package ui_test

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/render"
	"github.com/draxxris/rtgui/ui"
	"github.com/draxxris/rtgui/widgets"
)

// TestRichControlsDrawWithoutLinks checks buttons and dropdown rows render.
func TestRichControlsDrawWithoutLinks(t *testing.T) {
	u := ui.New(400, 300)
	recorder, err := render.NewDrawRecorder(128)
	if err != nil {
		t.Fatal(err)
	}
	u.Theme().SetDrawRecorder(recorder)
	u.RegisterInlineIcon("iron-plate", core.TooltipIcon{Width: 32, Height: 32})
	if err := u.RegisterFont("ValleySans", "../testdata/fonts/ValleySans-Regular.ttf"); err != nil {
		t.Fatalf("facade font registration: %v", err)
	}
	u.SetLinkColor(core.LinkItem, core.Color{R: 255, G: 180, B: 70, A: 255})
	button := widgets.NewButton("richButton", core.Rect{X: 10, Y: 10, W: 200, H: 40}, "plain")
	button.SetRichSegments([]core.RichSegment{
		{Icon: "iron-plate", HasIcon: true},
		{Text: " equipped", Link: core.Link{Kind: core.LinkItem, Target: "iron-plate"}},
	})
	dropdown := widgets.NewDropdown("richDrop", core.Rect{X: 10, Y: 60, W: 200, H: 40}, []string{"Warrior", "Mage"}, 0)
	dropdown.SetRichDropdownItem(0, []core.RichSegment{{Icon: "iron-plate", HasIcon: true}, {Text: " Warrior"}})
	chat := widgets.NewRichText("chat", core.Rect{X: 10, Y: 110, W: 200, H: 60}, []core.RichSegment{
		{Text: "take "},
		{Text: "iron", Link: core.Link{Kind: core.LinkItem, Target: "iron-plate"}},
	})
	if err := u.Add(button, dropdown, chat); err != nil {
		t.Fatal(err)
	}
	clicked := ""
	u.OnLinkClick("chat", func(link core.Link) { clicked = link.Target })
	if !u.ActivateLink("chat", 0) || clicked != "iron-plate" {
		t.Fatalf("chat link did not fire: %q", clicked)
	}
	u.Draw()
	if len(recorder.Calls()) == 0 {
		t.Fatal("rich controls drew no calls")
	}
	if _, ok := u.LinkColor(core.LinkItem); !ok {
		t.Fatal("registered link color missing")
	}
}
