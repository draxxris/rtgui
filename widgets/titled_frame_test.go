package widgets_test

import (
	"math"
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/layout"
	"github.com/draxxris/rtgui/widgets"
)

// TestTitledFrameConstruction checks names, title, close control, chrome
// transparency, and default variant classes.
func TestTitledFrameConstruction(t *testing.T) {
	tf := widgets.NewTitledFrame("questLog", core.Rect{X: 430, Y: 230, W: 420, H: 320}, "Quest Log")
	owned := tf.Widgets()
	if len(owned) != 5 {
		t.Fatalf("widget count = %d, want 5", len(owned))
	}
	wantNames := []string{"questLog", "questLog/titlebar", "questLog/title", "questLog/close", "questLog/content"}
	for i, want := range wantNames {
		if owned[i].Name() != want {
			t.Fatalf("widget %d = %q, want %q", i, owned[i].Name(), want)
		}
	}
	if tf.Title() != "Quest Log" {
		t.Fatalf("title = %q", tf.Title())
	}
	closeBtn := tf.CloseButton()
	if closeBtn.Text() != "X" || closeBtn.Tooltip() != "Close" {
		t.Fatalf("close = %q/%q", closeBtn.Text(), closeBtn.Tooltip())
	}
	if closeBtn.InputTransparent() {
		t.Fatal("close button must stay opaque")
	}
	if tf.ContentName() != "questLog/content" || tf.Content().Name() != "questLog/content" {
		t.Fatalf("content handle = %q", tf.ContentName())
	}
}

// TestTitledFrameChrome checks input transparency and default variant classes.
func TestTitledFrameChrome(t *testing.T) {
	tf := widgets.NewTitledFrame("questLog", core.Rect{}, "Quest Log")
	byName := titledByName(t, tf)
	for _, name := range []string{"questLog/titlebar", "questLog/title", "questLog/content"} {
		if !byName[name].InputTransparent() {
			t.Fatalf("%s must be input transparent", name)
		}
	}
	wantClasses := map[string]string{
		"questLog":          "titled-frame",
		"questLog/titlebar": "titled-titlebar",
		"questLog/title":    "titled-title",
		"questLog/close":    "titled-close",
		"questLog/content":  "titled-content",
	}
	for name, want := range wantClasses {
		if byName[name].Class() != want {
			t.Fatalf("%s class = %q, want %q", name, byName[name].Class(), want)
		}
	}
}

// TestTitledFrameLayoutGeometry checks chrome placement inside a 420x320 root.
func TestTitledFrameLayoutGeometry(t *testing.T) {
	tf := widgets.NewTitledFrame("questLog", core.Rect{X: 430, Y: 230, W: 420, H: 320}, "Quest Log")
	if err := tf.Layout(); err != nil {
		t.Fatal(err)
	}
	byName := titledByName(t, tf)
	want := map[string]core.Rect{
		"questLog":          {X: 430, Y: 230, W: 420, H: 320},
		"questLog/titlebar": {X: 438, Y: 238, W: 404, H: 32},
		"questLog/title":    {X: 448, Y: 238, W: 362, H: 32},
		"questLog/close":    {X: 814, Y: 242, W: 24, H: 24},
		"questLog/content":  {X: 438, Y: 270, W: 404, H: 272},
	}
	for name, wantRect := range want {
		if got := byName[name].Bounds(); got != wantRect {
			t.Errorf("%s = %+v, want %+v", name, got, wantRect)
		}
	}
}

// titledByName indexes owned widgets by registry name.
func titledByName(t *testing.T, tf *widgets.TitledFrame) map[string]widgets.Widget {
	t.Helper()
	out := map[string]widgets.Widget{}
	for _, w := range tf.Widgets() {
		out[w.Name()] = w
	}
	if len(out) != 5 {
		t.Fatalf("indexed %d widgets", len(out))
	}
	return out
}

// TestTitledFrameShowHideToggleClose checks visibility transitions and that
// only Close fires the callback after hiding.
func TestTitledFrameShowHideToggleClose(t *testing.T) {
	tf := widgets.NewTitledFrame("questLog", core.Rect{W: 420, H: 320}, "Quest Log")
	calls := 0
	tf.OnClose(func() {
		calls++
		if tf.IsOpen() {
			t.Error("OnClose must fire after hiding")
		}
	})
	if !tf.IsOpen() {
		t.Fatal("new window must start open")
	}
	tf.Hide()
	if tf.IsOpen() || calls != 0 {
		t.Fatalf("Hide: open=%v calls=%d", tf.IsOpen(), calls)
	}
	tf.Toggle()
	if !tf.IsOpen() || calls != 0 {
		t.Fatalf("Toggle show: open=%v calls=%d", tf.IsOpen(), calls)
	}
	tf.Toggle()
	if tf.IsOpen() || calls != 1 {
		t.Fatalf("Toggle close: open=%v calls=%d", tf.IsOpen(), calls)
	}
	tf.Show()
	tf.Close()
	if tf.IsOpen() || calls != 2 {
		t.Fatalf("Close: open=%v calls=%d", tf.IsOpen(), calls)
	}
}

// TestTitledFrameSetTitle checks title mutation reporting.
func TestTitledFrameSetTitle(t *testing.T) {
	tf := widgets.NewTitledFrame("questLog", core.Rect{}, "Quest Log")
	if !tf.SetTitle("Spellbook") || tf.Title() != "Spellbook" {
		t.Fatalf("title = %q", tf.Title())
	}
	if tf.SetTitle("Spellbook") {
		t.Fatal("unchanged title must report false")
	}
}

// TestTitledFrameSetVariant checks class derivation and snapshot propagation.
func TestTitledFrameSetVariant(t *testing.T) {
	tf := widgets.NewTitledFrame("questLog", core.Rect{}, "Quest Log")
	tf.SetVariant("guild")
	byName := titledByName(t, tf)
	if byName["questLog"].Class() != "guild-frame" || byName["questLog/close"].Class() != "guild-close" {
		t.Fatalf("variant classes = %q/%q", byName["questLog"].Class(), byName["questLog/close"].Class())
	}
	snapshot := byName["questLog/content"].Snapshot(core.StateNormal)
	if snapshot.Class != "guild-content" {
		t.Fatalf("snapshot class = %q", snapshot.Class)
	}
	before := byName["questLog"].Class()
	tf.SetVariant("")
	if byName["questLog"].Class() != before {
		t.Fatal("empty variant must be ignored")
	}
}

// TestTitledFrameBorderInset checks chrome offsets track the inset.
func TestTitledFrameBorderInset(t *testing.T) {
	tf := widgets.NewTitledFrame("questLog", core.Rect{X: 100, Y: 100, W: 420, H: 320}, "Quest Log")
	tf.SetBorderInset(0)
	if err := tf.Layout(); err != nil {
		t.Fatal(err)
	}
	if got := titledByName(t, tf)["questLog/titlebar"].Bounds(); got.X != 100 || got.Y != 100 {
		t.Fatalf("zero inset titlebar = %+v", got)
	}
	tf.SetBorderInset(12)
	if err := tf.Layout(); err != nil {
		t.Fatal(err)
	}
	byName := titledByName(t, tf)
	if got := byName["questLog/titlebar"].Bounds(); got.X != 112 || got.W != 396 {
		t.Fatalf("inset titlebar = %+v", got)
	}
	if got := byName["questLog/content"].Bounds(); got.X != 112 || got.W != 396 {
		t.Fatalf("inset content = %+v", got)
	}
	tf.SetBorderInset(float32(math.NaN()))
	tf.SetBorderInset(-4)
	if err := tf.Layout(); err != nil {
		t.Fatal(err)
	}
	if got := titledByName(t, tf)["questLog/titlebar"].Bounds(); got.X != 100 {
		t.Fatalf("sanitized inset titlebar = %+v", got)
	}
}

// TestTitledFrameCustomChrome checks title inset, vertical offset, and close
// sizing remain layout-controlled rather than renderer-specific.
func TestTitledFrameCustomChrome(t *testing.T) {
	tf := widgets.NewTitledFrame("questLog", core.Rect{X: 100, Y: 100, W: 420, H: 320}, "Quest Log")
	tf.SetTitleLeftInset(20).SetTitleTopOffset(-3).SetCloseButtonSize(40)
	if err := tf.Layout(); err != nil {
		t.Fatal(err)
	}
	byName := titledByName(t, tf)
	if got := byName["questLog/title"].Bounds(); got.X != 128 || got.Y != 105 || got.W != 352 || got.H != 35 {
		t.Fatalf("custom title = %+v", got)
	}
	if got := byName["questLog/close"].Bounds(); got != (core.Rect{X: 468, Y: 112, W: 40, H: 40}) {
		t.Fatalf("custom close = %+v", got)
	}
}

// TestTitledFramePerSideInsets checks asymmetric rings place the titlebar
// and content with a divider gap between them.
func TestTitledFramePerSideInsets(t *testing.T) {
	tf := widgets.NewTitledFrame("questLog", core.Rect{X: 100, Y: 100, W: 420, H: 320}, "Quest Log")
	tf.SetBorderInsets(30, 18, 30, 75).SetTitleBarHeight(60).SetContentTopGap(17)
	if err := tf.Layout(); err != nil {
		t.Fatal(err)
	}
	byName := titledByName(t, tf)
	if got := byName["questLog/titlebar"].Bounds(); got != (core.Rect{X: 130, Y: 118, W: 360, H: 60}) {
		t.Fatalf("titlebar = %+v", got)
	}
	if got := byName["questLog/content"].Bounds(); got != (core.Rect{X: 130, Y: 195, W: 360, H: 150}) {
		t.Fatalf("content = %+v", got)
	}
	tf.SetCloseInset(-19, -8).SetCloseButtonSize(35)
	if err := tf.Layout(); err != nil {
		t.Fatal(err)
	}
	if got := titledByName(t, tf)["questLog/close"].Bounds(); got != (core.Rect{X: 474, Y: 110, W: 35, H: 35}) {
		t.Fatalf("close overlap = %+v", got)
	}
	tf.SetBorderInsets(float32(math.NaN()), -1, float32(math.Inf(1)), 12)
	if err := tf.Layout(); err != nil {
		t.Fatal(err)
	}
	if got := titledByName(t, tf)["questLog/titlebar"].Bounds(); got.X != 100 || got.W != 420 {
		t.Fatalf("sanitized insets titlebar = %+v", got)
	}
}

// TestTitledFrameTitleBarSideInsets checks the titlebar keeps its own side
// margins while the content follows the body insets.
func TestTitledFrameTitleBarSideInsets(t *testing.T) {
	tf := widgets.NewTitledFrame("questLog", core.Rect{X: 100, Y: 100, W: 420, H: 320}, "Quest Log")
	tf.SetBorderInsets(15, 11, 15, 18).SetTitleBarSideInsets(25, 25).SetTitleBarHeight(42).SetContentTopGap(11)
	if err := tf.Layout(); err != nil {
		t.Fatal(err)
	}
	byName := titledByName(t, tf)
	if got := byName["questLog/titlebar"].Bounds(); got != (core.Rect{X: 125, Y: 111, W: 370, H: 42}) {
		t.Fatalf("titlebar = %+v", got)
	}
	if got := byName["questLog/content"].Bounds(); got != (core.Rect{X: 115, Y: 164, W: 390, H: 238}) {
		t.Fatalf("content = %+v", got)
	}
}

// TestTitledFrameTitleBarHeight checks chrome height and content shift.
func TestTitledFrameTitleBarHeight(t *testing.T) {
	tf := widgets.NewTitledFrame("questLog", core.Rect{X: 100, Y: 100, W: 420, H: 320}, "Quest Log")
	tf.SetTitleBarHeight(48)
	if err := tf.Layout(); err != nil {
		t.Fatal(err)
	}
	byName := titledByName(t, tf)
	if got := byName["questLog/titlebar"].Bounds(); got.H != 48 {
		t.Fatalf("titlebar height = %+v", got)
	}
	if got := byName["questLog/content"].Bounds(); got.Y != 156 || got.H != 256 {
		t.Fatalf("content = %+v", got)
	}
	tf.SetTitleBarHeight(0)
	tf.SetTitleBarHeight(float32(math.NaN()))
	if err := tf.Layout(); err != nil {
		t.Fatal(err)
	}
	if got := titledByName(t, tf)["questLog/titlebar"].Bounds(); got.H != 48 {
		t.Fatalf("invalid height must be ignored, got %+v", got)
	}
}

// TestTitledFrameContentFollowsMove checks caller children ride the root.
func TestTitledFrameContentFollowsMove(t *testing.T) {
	tf := widgets.NewTitledFrame("questLog", core.Rect{X: 100, Y: 100, W: 420, H: 320}, "Quest Log")
	if err := tf.Layout(); err != nil {
		t.Fatal(err)
	}
	child := widgets.NewLabel("questBody", core.Rect{W: 200, H: 40}, "Body")
	if err := tf.Content().Frame().AddChild(child.Frame()); err != nil {
		t.Fatal(err)
	}
	if err := child.SetPoint(layout.AnchorTopLeft, nil, layout.AnchorTopLeft, core.Vec2{X: 16, Y: 12}); err != nil {
		t.Fatal(err)
	}
	if err := tf.Arrange(); err != nil {
		t.Fatal(err)
	}
	before := child.Bounds()
	layout.MoveFrame(titledRootFrame(t, tf), core.Vec2{X: 20, Y: 10})
	if err := tf.Arrange(); err != nil {
		t.Fatal(err)
	}
	if got := child.Bounds(); got.X != before.X+20 || got.Y != before.Y+10 {
		t.Fatalf("child = %+v, want %+v shifted by (20,10)", got, before)
	}
}

// titledRootFrame returns the root layout node through the owned widgets.
func titledRootFrame(t *testing.T, tf *widgets.TitledFrame) *layout.Node {
	t.Helper()
	for _, w := range tf.Widgets() {
		if w.Name() == "questLog" {
			return w.Frame()
		}
	}
	t.Fatal("missing root")
	return nil
}

// TestTitledFrameCloseButtonClick checks the wired X path hides and fires.
func TestTitledFrameCloseButtonClick(t *testing.T) {
	tf := widgets.NewTitledFrame("questLog", core.Rect{}, "Quest Log")
	fired := false
	tf.OnClose(func() { fired = true })
	tf.CloseButton().Callbacks().Click()
	if tf.IsOpen() || !fired {
		t.Fatalf("X path: open=%v fired=%v", tf.IsOpen(), fired)
	}
}

// TestTitledFrameNilSafety checks nil receivers never panic.
func TestTitledFrameNilSafety(t *testing.T) {
	var tf *widgets.TitledFrame
	if tf.Widgets() != nil || tf.ContentName() != "" || tf.Content() != nil {
		t.Fatal("nil accessors must return zero values")
	}
	if tf.CloseButton() != nil || tf.Title() != "" || tf.SetTitle("x") || tf.IsOpen() {
		t.Fatal("nil accessors must return zero values")
	}
	if err := tf.Layout(); err == nil || err != widgets.ErrNilWidget {
		t.Fatalf("nil Layout = %v", err)
	}
	if err := tf.Arrange(); err == nil || err != widgets.ErrNilWidget {
		t.Fatalf("nil Arrange = %v", err)
	}
	tf.Show()
	tf.Hide()
	tf.Toggle()
	tf.Close()
	tf.OnClose(nil)
	tf.SetVariant("")
	tf.SetTitleBarHeight(0)
	tf.SetBorderInset(0)
}
