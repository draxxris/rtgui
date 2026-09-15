package main

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/ui"
	"github.com/draxxris/rtgui/widgets"
)

// TestQuestScrollbarSlices verifies the list scrollbar keeps its caps with a
// three-patch slice on the custom art.
func TestQuestScrollbarSlices(t *testing.T) {
	text, err := os.ReadFile(filepath.Join("..", "..", "testdata", "skins", "quest.css"))
	if err != nil {
		t.Fatal(err)
	}
	rules, err := skin.ParseCSS(string(text))
	if err != nil {
		t.Fatal(err)
	}
	assertQuestListScrollCaps(t, rules)
}

// assertQuestListScrollCaps verifies track and thumb keep caps via slice.
func assertQuestListScrollCaps(t *testing.T, rules []skin.SkinRule) {
	t.Helper()
	assertQuestScrollSlice(t, rules, skin.PartTrack, "track")
	assertQuestScrollSlice(t, rules, skin.PartThumb, "thumb")
}

// assertQuestSelectedInnerGlow verifies the selected row blends a
// translucent four-sided inner gradient over its vertical base.
func assertQuestSelectedInnerGlow(t *testing.T, rules []skin.SkinRule) {
	t.Helper()
	for _, rule := range rules {
		if rule.Class != "quest-list" || rule.Part != skin.PartOverlay || rule.State != core.StateSelected {
			continue
		}
		if rule.GradientCount != 2 || rule.Gradients[0].Kind != skin.GradientInner {
			t.Fatalf("selected highlight must lead with inner-gradient, got %+v", rule)
		}
		return
	}
	t.Fatal("quest-list selected highlight rule is missing")
}

// assertQuestScrollSlice verifies one scrollbar part carries slice 8.
func assertQuestScrollSlice(t *testing.T, rules []skin.SkinRule, part skin.SkinPart, name string) {
	t.Helper()
	for _, rule := range rules {
		if rule.Class != "quest-list" || rule.Part != part || rule.State != core.StateNormal {
			continue
		}
		if !rule.HasSlice || rule.Slice != [4]int32{8, 8, 8, 8} {
			t.Fatalf("quest-list::%s must carry slice 8, got %+v", name, rule)
		}
		return
	}
	t.Fatalf("quest-list::%s rule is missing", name)
}

// TestQuestRootBorder verifies the v6 ornate shell lives on the root
// border as a 95/75/75/75 slice drawn at 64/45/45/45. Frame borders paint
// after children, so the moldings finish over titlebar and content bleed
// with no overlay widget.
func TestQuestRootBorder(t *testing.T) {
	text, err := os.ReadFile(filepath.Join("..", "..", "testdata", "skins", "quest.css"))
	if err != nil {
		t.Fatal(err)
	}
	rules, err := skin.ParseCSS(string(text))
	if err != nil {
		t.Fatal(err)
	}
	for _, rule := range rules {
		if rule.Class != "quest-frame" || rule.Part != skin.PartBorder || rule.State != core.StateNormal {
			continue
		}
		if !rule.HasSlice || rule.Slice != [4]int32{95, 75, 75, 75} {
			t.Fatalf("shell slice = %+v, want 95/75/75/75", rule)
		}
		if !rule.HasWidth || rule.Width != [4]int32{64, 45, 45, 45} {
			t.Fatalf("shell width = %+v, want 64/45/45/45", rule)
		}
		if !rule.HasImage || rule.Image != "quest/ornate_frame_v6.png" {
			t.Fatalf("shell image = %+v, want v6 shell", rule)
		}
		return
	}
	t.Fatal("quest-frame border rule is missing")
}

// TestQuestChromeOrder verifies content draws before the titlebar so body
// bleed slides under the title band, and the close button stays inside
// the titlebar with no overlay widget.
func TestQuestChromeOrder(t *testing.T) {
	u := ui.New(800, 800)
	_ = newQuestWindow(u)
	root := u.Lookup("questWindow")
	if root == nil {
		t.Fatal("questWindow root is missing")
	}
	if u.Lookup("questFrameArt") != nil {
		t.Fatal("questFrameArt overlay must be gone")
	}
	children := root.Frame().Children()
	contentIndex, titleBarIndex := -1, -1
	for index, child := range children {
		switch child {
		case u.Lookup("questWindow/content").Frame():
			contentIndex = index
		case u.Lookup("questWindow/titlebar").Frame():
			titleBarIndex = index
		}
	}
	if contentIndex < 0 || titleBarIndex <= contentIndex {
		t.Fatalf("content must precede titlebar: content=%d titlebar=%d", contentIndex, titleBarIndex)
	}
	close := u.Lookup("questWindow/close")
	if close == nil {
		t.Fatal("questWindow/close is missing")
	}
	if close.Frame().Parent() != u.Lookup("questWindow/titlebar").Frame() {
		t.Fatal("close must stay inside the titlebar")
	}
}

// TestQuestDefaultWindowScaling verifies the 800 logical design maps 1:1
// inside the 800px default with a synthetic 24px margin and scale 1.
func TestQuestDefaultWindowScaling(t *testing.T) {
	u := ui.New(int(questDesignWidth), int(questDesignHeight))
	u.Resize(int(questWindowWidth), int(questWindowHeight))
	sx, sy := u.Scale()
	if sx != 1 || sy != 1 {
		t.Fatalf("default scale = %v/%v, want 1/1", sx, sy)
	}
	if got := u.UIScale(); got != 1 {
		t.Fatalf("default UI scale = %v, want 1", got)
	}
	viewport := u.Transform().Viewport
	want := core.Rect{W: 800, H: 800}
	if viewport.Viewport != want {
		t.Fatalf("physical viewport = %+v, want %+v", viewport.Viewport, want)
	}
	newQuestWindow(u)
	if got := u.Lookup("questWindow").Bounds(); got != (core.Rect{X: 24, Y: 24, W: 752, H: 752}) {
		t.Fatalf("synthetic-margin root = %+v", got)
	}
}

// TestQuestPlaceholderUsesDefaultSize verifies headless screenshots use the
// default physical framebuffer dimensions rather than the logical layout.
func TestQuestPlaceholderUsesDefaultSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quest.png")
	if err := createQuestPlaceholder(path); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	img, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	bounds := img.Bounds()
	if bounds.Dx() != int(questWindowWidth) || bounds.Dy() != int(questWindowHeight) {
		t.Fatalf("placeholder bounds = %v, want %dx%d", bounds, questWindowWidth, questWindowHeight)
	}
}

// TestQuestLayoutPlacement locks pixel-identical placement at 800x800 with
// a synthetic 24px margin. Offsets are authored literally in questSlots,
// so this test is the guarantee, not arithmetic.
func TestQuestLayoutPlacement(t *testing.T) {
	u := ui.New(800, 800)
	newQuestWindow(u)
	want := map[string]core.Rect{
		"questWindow":          {X: 24, Y: 24, W: 752, H: 752},
		"questWindow/titlebar": {X: 31, Y: 35, W: 738, H: 42},
		"questWindow/title":    {X: 51, Y: 31, W: 686, H: 46},
		"questWindow/close":    {X: 722, Y: 35, W: 43, H: 43},
		"questWindow/content":  {X: 39, Y: 88, W: 722, H: 681},
		"questTabs":            {X: 54, Y: 97, W: 272, H: 35},
		"questList":            {X: 48, Y: 145, W: 283, H: 531},
		"questSheet":           {X: 339, Y: 97, W: 411, H: 583},
		"questRules":           {X: 339, Y: 97, W: 411, H: 583},
		"questTitle":           {X: 365, Y: 103, W: 372, H: 29},
		"questSubtitle":        {X: 365, Y: 135, W: 372, H: 18},
		"questBody":            {X: 365, Y: 159, W: 372, H: 340},
		"questImage":           {X: 365, Y: 505, W: 372, H: 64},
		"questRewards":         {X: 365, Y: 575, W: 372, H: 92},
		"questRewardArt":       {X: 339, Y: 97, W: 411, H: 583},
		"questReward1":         {X: 418, Y: 623, W: 73, H: 51},
		"questReward2":         {X: 544, Y: 623, W: 59, H: 51},
		"questReward3":         {X: 653, Y: 623, W: 85, H: 51},
		"showMapButton":        {X: 112, Y: 704, W: 154, H: 37},
		"shareButton":          {X: 280, Y: 704, W: 110, H: 37},
		"abandonButton":        {X: 404, Y: 704, W: 120, H: 37},
		"trackButton":          {X: 538, Y: 704, W: 150, H: 37},
		"reopenButton":         {X: 300, Y: 376, W: 200, H: 48},
	}
	for name, wantRect := range want {
		w := u.Lookup(name)
		if w == nil {
			t.Fatalf("missing widget %q", name)
		}
		if got := w.Bounds(); got != wantRect {
			t.Errorf("%s = %+v, want %+v", name, got, wantRect)
		}
	}
}

// TestQuestCSSParsesVariant checks the quest skin carries the TitledFrame
// variant plus the dark list, parchment sheet, tab, and glowing buttons.
func TestQuestCSSParsesVariant(t *testing.T) {
	text, err := os.ReadFile(filepath.Join("..", "..", "testdata", "skins", "quest.css"))
	if err != nil {
		t.Fatal(err)
	}
	rules, err := skin.ParseCSS(string(text))
	if err != nil {
		t.Fatal(err)
	}
	seen := questCSSSeen(rules)
	assertQuestCSSClasses(t, seen)
	assertQuestCSSParts(t, seen)
	assertQuestCSSShell(t, rules)
	assertQuestCSSBorderOptOut(t, rules)
	assertQuestCSSActionSurface(t, rules)
	assertQuestSelectedInnerGlow(t, rules)
	assertQuestButtonRadius(t, rules)
}

// questCSSSeen indexes class and part combinations for concise stylesheet
// coverage checks.
func questCSSSeen(rules []skin.SkinRule) map[string]bool {
	seen := map[string]bool{}
	for _, rule := range rules {
		if rule.Class == "" {
			continue
		}
		seen[rule.Class+"|"+questPartBucket(rule)] = true
		seen[rule.Class] = true
	}
	return seen
}

// assertQuestCSSClasses checks the classes consumed by the journal widgets.
func assertQuestCSSClasses(t *testing.T, seen map[string]bool) {
	t.Helper()
	for _, class := range []string{"quest-frame", "quest-titlebar", "quest-title", "quest-close", "quest-content", "quest-tabs", "quest-list", "quest-sheet", "quest-action", "quest-primary", "quest-heading", "quest-sub", "quest-body", "quest-rewards", "quest-reward"} {
		if !seen[class] {
			t.Fatalf("quest css misses .%s", class)
		}
	}
}

// assertQuestCSSParts checks the list, tab, and parchment coverage buckets.
func assertQuestCSSParts(t *testing.T, seen map[string]bool) {
	t.Helper()
	for _, key := range []string{"quest-list|highlight", "quest-list|track", "quest-list|thumb", "quest-tabs|tab", "quest-sheet|base", "quest-body|base", "quest-rewards|base"} {
		if !seen[key] {
			t.Fatalf("quest css misses %s (seen=%v)", key, seen)
		}
	}
}

// assertQuestCSSShell verifies the root owns the dark fill plus the v6
// shell border, with no overlay widget.
func assertQuestCSSShell(t *testing.T, rules []skin.SkinRule) {
	t.Helper()
	root := questCSSBackgroundRule(rules, "quest-frame")
	assertQuestCSSRootShell(t, root)
	assertQuestRootBorder(t, rules)
}

// questCSSBackgroundRule returns one normal background rule for a class.
func questCSSBackgroundRule(rules []skin.SkinRule, class string) *skin.SkinRule {
	for index := range rules {
		rule := &rules[index]
		if rule.Class == class && rule.State == core.StateNormal && rule.Part == skin.PartBackground {
			return rule
		}
	}
	return nil
}

// assertQuestCSSRootShell verifies the root keeps its dark fill.
func assertQuestCSSRootShell(t *testing.T, root *skin.SkinRule) {
	t.Helper()
	if root == nil {
		t.Fatal("quest-frame background rule is missing")
	}
	if !root.HasBackgroundColor {
		t.Fatalf("root must keep its dark fill: %+v", root)
	}
}

// assertQuestRootBorder verifies the v6 shell border slices and widths on
// the root with no overlay widget.
func assertQuestRootBorder(t *testing.T, rules []skin.SkinRule) {
	t.Helper()
	for _, rule := range rules {
		if rule.Class != "quest-frame" || rule.State != core.StateNormal {
			continue
		}
		if rule.Part != skin.PartBorder {
			continue
		}
		if !rule.HasImage || rule.Image != "quest/ornate_frame_v6.png" {
			t.Fatalf("root border image = %+v, want v6 shell", rule)
		}
		if !rule.HasSlice || rule.Slice != [4]int32{95, 75, 75, 75} {
			t.Fatalf("root border slice = %+v, want 95/75/75/75", rule)
		}
		if !rule.HasWidth || rule.Width != [4]int32{64, 45, 45, 45} {
			t.Fatalf("root border width = %+v, want 64/45/45/45", rule)
		}
		return
	}
	t.Fatal("quest-frame border rule is missing")
}

// assertQuestCSSBorderOptOut verifies chrome without a separate ring does not
// accidentally inherit the outer frame border.
func assertQuestCSSBorderOptOut(t *testing.T, rules []skin.SkinRule) {
	t.Helper()
	for _, rule := range rules {
		if questChromeClass(rule.Class) && rule.Part == skin.PartBorder && !rule.NoTexture {
			t.Fatalf("%s border must opt out with none", rule.Class)
		}
	}
}

// questChromeClass reports classes whose border is intentionally transparent.
func questChromeClass(class string) bool {
	return class == "quest-titlebar" || class == "quest-content" || class == "quest-sheet"
}

// assertQuestCSSActionSurface checks every authored button state uses its
// intended v2 texture and two alpha gradient layers.
func assertQuestCSSActionSurface(t *testing.T, rules []skin.SkinRule) {
	t.Helper()
	assertQuestButtonSurface(t, rules, "quest-action", core.StateNormal, "quest/button_surface_v3.png")
	assertQuestButtonSurface(t, rules, "quest-action", core.StateHovered, "quest/button_surface_hover_v3.png")
	assertQuestButtonSurface(t, rules, "quest-action", core.StatePressed, "quest/button_surface_pressed_v3.png")
	assertQuestButtonSurface(t, rules, "quest-primary", core.StateNormal, "quest/button_surface_primary_v3.png")
	assertQuestButtonSurface(t, rules, "quest-primary", core.StateHovered, "quest/button_surface_primary_v3.png")
	assertQuestButtonSurface(t, rules, "quest-primary", core.StatePressed, "quest/button_surface_primary_pressed_v3.png")
}

// assertQuestButtonRadius verifies action buttons clip all layers to a
// rounded rect so the square fill and gradient stack cannot poke past the
// chamfered surface art.
func assertQuestButtonRadius(t *testing.T, rules []skin.SkinRule) {
	t.Helper()
	for _, class := range []string{"quest-action", "quest-primary"} {
		found := false
		for _, rule := range rules {
			if rule.Class != class || rule.Part != skin.PartBackground || rule.State != core.StateNormal {
				continue
			}
			found = true
			if !rule.HasRadius || rule.Radius != 5 {
				t.Fatalf("%s radius = %+v, want 5px clip", class, rule)
			}
		}
		if !found {
			t.Fatalf("%s background rule is missing", class)
		}
	}
}

// assertQuestButtonSurface finds one authored background rule and verifies
// its texture plus the two alpha gradient layers.
func assertQuestButtonSurface(t *testing.T, rules []skin.SkinRule, class string, state core.WidgetState, image string) {
	t.Helper()
	for _, rule := range rules {
		if rule.Class != class || rule.Part != skin.PartBackground || rule.State != state {
			continue
		}
		if !rule.HasImage || rule.Image != image || rule.GradientCount != 2 {
			t.Fatalf("%s state %v mixed surface = %+v", class, state, rule)
		}
		return
	}
	t.Fatalf("%s state %v background rule is missing", class, state)
}

// questPartBucket classifies one quest rule for coverage checks.
func questPartBucket(rule skin.SkinRule) string {
	switch {
	case rule.Part == skin.PartBackground:
		return "base"
	case rule.Part == skin.PartBorder:
		return "border"
	case rule.Part == skin.PartOverlay:
		return "highlight"
	case rule.Part == skin.PartTrack:
		return "track"
	case rule.Part == skin.PartThumb:
		return "thumb"
	case rule.Part == skin.PartTab:
		return "tab"
	default:
		return "other"
	}
}

// TestQuestCSSReferencesReplacementAssets checks the stylesheet names the
// externally supplied v6 shell and stable journal assets.
func TestQuestCSSReferencesReplacementAssets(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "skins", "quest.css")
	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		`quest/ornate_frame_v6.png`,
		`quest/titlebar_blue_v2.png`,
		`quest/parchment_v2.png`,
		`quest/button_surface_v3.png`,
		`quest/button_surface_primary_pressed_v3.png`,
		`quest/close_button_v3.png`,
		`quest/scroll_track_v3.png`,
		`quest/scroll_thumb_v3.png`,
	} {
		if !strings.Contains(string(text), name) {
			t.Fatalf("quest.css does not reference %s", name)
		}
	}
}

// TestQuestActionIconRegistration verifies the three action assets enter the
// same parent-owned inline-icon whitelist as list and reward art. The
// renderer-free metadata remains available in headless construction.
func TestQuestActionIconRegistration(t *testing.T) {
	u := ui.New(800, 800)
	newQuestWindow(u)
	for _, name := range []string{"action-map", "action-share", "action-track"} {
		icon, ok := u.LookupInlineIcon(name)
		if !ok {
			t.Fatalf("missing registered action icon %q", name)
		}
		if icon.Width != 32 || icon.Height != 32 {
			t.Fatalf("headless action icon %q size = %dx%d, want 32x32", name, icon.Width, icon.Height)
		}
	}
}

// TestQuestActionRichButtonContent verifies action labels use rich segments
// for icons, while Abandon remains an ordinary text-only button.
func TestQuestActionRichButtonContent(t *testing.T) {
	u := ui.New(800, 800)
	q := newQuestWindow(u)
	assertQuestActionSegments(t, q.showMap, "action-map", "Show on Map")
	assertQuestActionSegments(t, q.share, "action-share", "Share")
	assertQuestActionSegments(t, q.track, "action-track", "Track Quest")
	if q.abandon.HasRichText() || q.abandon.Text() != "Abandon" {
		t.Fatalf("abandon content = %q/%v, want plain text", q.abandon.Text(), q.abandon.HasRichText())
	}
}

// assertQuestActionSegments checks one icon run followed by one sized label.
func assertQuestActionSegments(t *testing.T, button *widgets.Button, icon, label string) {
	t.Helper()
	segments := button.RichSegments()
	if len(segments) != 2 {
		t.Fatalf("%s rich segments = %+v, want icon+label", button.Name(), segments)
	}
	if !segments[0].HasIcon || segments[0].Icon != icon || segments[0].Text != "" || !segments[0].HasIconSize || segments[0].IconSize != 35 {
		t.Fatalf("%s icon segment = %+v", button.Name(), segments[0])
	}
	if segments[1].HasIcon || segments[1].Text != label || !segments[1].HasFontSize || segments[1].FontSize != 21 || !segments[1].HasColor {
		t.Fatalf("%s label segment = %+v", button.Name(), segments[1])
	}
	if button.Text() != label {
		t.Fatalf("%s text = %q, want %q", button.Name(), button.Text(), label)
	}
}

// TestQuestActionStateTextUpdates verifies Track Quest, Track, and Untrack
// update the rich label without dropping the crosshair icon.
func TestQuestActionStateTextUpdates(t *testing.T) {
	u := ui.New(800, 800)
	q := newQuestWindow(u)
	if q.track.Text() != "Track Quest" {
		t.Fatalf("initial track text = %q", q.track.Text())
	}
	q.trackedID = "troubled-farmstead"
	q.updateQuestActions("troubled-farmstead")
	assertQuestActionSegments(t, q.track, "action-track", "Untrack")
	q.trackedID = ""
	q.updateQuestActions("wolves-door")
	assertQuestActionSegments(t, q.track, "action-track", "Track")
	q.updateQuestActions("")
	if q.track.Text() != "Track" || !q.track.HasRichText() || q.track.Enabled() {
		t.Fatalf("empty selection track = %q/rich=%v/enabled=%v", q.track.Text(), q.track.HasRichText(), q.track.Enabled())
	}
}

// TestQuestAssetsExist verifies every image consumed by the quest example is
// present, including action icons, the externally supplied v6 shell, and
// button surfaces.
func TestQuestAssetsExist(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "skins", "quest")
	for _, name := range []string{
		"ornate_frame_v6.png", "titlebar_blue_v2.png", "parchment_v2.png",
		"close_button_v3.png",
		"scroll_track_v3.png", "scroll_thumb_v3.png",
		"button_surface_v3.png", "button_surface_hover_v3.png",
		"button_surface_pressed_v3.png", "button_surface_primary_v3.png",
		"button_surface_primary_pressed_v3.png",
		"quest_bang.png", "quest_circle.png", "action_map.png",
		"action_share.png", "action_track.png", "farmstead.png",
		"reward_xp.png", "reward_silver.png", "reward_satchel.png",
	} {
		info, err := os.Stat(filepath.Join(root, name))
		if err != nil {
			t.Fatalf("missing quest asset %s: %v", name, err)
		}
		if info.IsDir() || info.Size() == 0 {
			t.Fatalf("quest asset %s is empty", name)
		}
	}
}

// clickQuestButton presses and releases one journal button by center.
func clickQuestButton(t *testing.T, u *ui.UI, name string) {
	t.Helper()
	w := u.Lookup(name)
	if w == nil {
		t.Fatalf("missing widget %q", name)
	}
	bounds := w.Bounds()
	center := core.Vec2{X: bounds.X + bounds.W/2, Y: bounds.Y + bounds.H/2}
	u.HandleMouse(ui.MouseEvent{Pos: center, Pressed: true})
	u.HandleMouse(ui.MouseEvent{Pos: center, Released: true})
}

// TestQuestTabFiltering exercises the two filter tabs: each rebuilds the
// dark list and selects the first tale in reference order.
func TestQuestTabFiltering(t *testing.T) {
	u := ui.New(800, 800)
	q := newQuestWindow(u)
	if got := q.tabs.SelectedTab(); got != 0 {
		t.Fatalf("initial tab = %d, want 0", got)
	}
	if id, ok := q.list.Selected(); !ok || id != "troubled-farmstead" {
		t.Fatalf("initial selection = %q/%v, want troubled-farmstead", id, ok)
	}
	if label, ok := q.tabs.TabSelection(); !ok || label != "Current (6)" {
		t.Fatalf("initial tab label = %q/%v", label, ok)
	}
	if q.list.VisibleRowCount() != 6 {
		t.Fatalf("current rows = %d, want 6", q.list.VisibleRowCount())
	}
	if !u.SelectTab("questTabs", 1) {
		t.Fatal("SelectTab Completed failed")
	}
	if id, ok := q.list.Selected(); !ok || id != "completed-01" {
		t.Fatalf("completed selection = %q/%v, want completed-01", id, ok)
	}
	if q.list.VisibleRowCount() != 24 {
		t.Fatalf("completed rows = %d, want 24", q.list.VisibleRowCount())
	}
	if q.titleLabel.Text() != "Completed Tale 01" {
		t.Fatalf("completed title = %q", q.titleLabel.Text())
	}
	if !u.SelectTab("questTabs", 0) {
		t.Fatal("SelectTab Current failed")
	}
	if id, ok := q.list.Selected(); !ok || id != "troubled-farmstead" {
		t.Fatalf("current selection = %q/%v, want troubled-farmstead", id, ok)
	}
}

// TestQuestSelectionUpdatesDetails exercises leaf selection through the
// shared callback path: title, subtitle, parchment body, farmstead art,
// rewards, and actions sync.
func TestQuestSelectionUpdatesDetails(t *testing.T) {
	u := ui.New(800, 800)
	q := newQuestWindow(u)
	if !u.SelectListItem("questList", "wolves-door") {
		t.Fatal("SelectListItem wolves-door failed")
	}
	if q.titleLabel.Text() != "Wolves at the Door" {
		t.Fatalf("title = %q, want Wolves at the Door", q.titleLabel.Text())
	}
	if !strings.Contains(q.subtitleLabel.Text(), "Elwynn Forest") {
		t.Fatalf("subtitle = %q, want Elwynn Forest", q.subtitleLabel.Text())
	}
	if !strings.Contains(q.body.RichPlainText(), "Objectives") {
		t.Fatalf("body missing objectives: %q", q.body.RichPlainText())
	}
	if !strings.Contains(q.rewards.RichPlainText(), "You will receive") {
		t.Fatalf("rewards missing header: %q", q.rewards.RichPlainText())
	}
	if !q.showMap.Enabled() || !q.share.Enabled() || !q.track.Enabled() || !q.abandon.Enabled() {
		t.Fatal("actions must enable for a Current tale")
	}
	if !u.SelectTab("questTabs", 1) {
		t.Fatal("SelectTab Completed failed")
	}
	if !u.SelectListItem("questList", "completed-02") {
		t.Fatal("SelectListItem completed-02 failed")
	}
	if q.abandon.Enabled() {
		t.Fatal("abandon must disable for Completed tales")
	}
}

// TestQuestButtons exercises the bottom action row: show and share report,
// track toggles its label, abandon reports while keeping the demo tale,
// and body-link activation reports through the parchment pane.
func TestQuestButtons(t *testing.T) {
	u := ui.New(800, 800)
	q := newQuestWindow(u)
	if !u.SelectListItem("questList", "troubled-farmstead") {
		t.Fatal("SelectListItem failed")
	}
	clickQuestButton(t, u, "showMapButton")
	if !strings.Contains(q.status, "Showing") {
		t.Fatalf("show status = %q", q.status)
	}
	clickQuestButton(t, u, "shareButton")
	if !strings.Contains(q.status, "Shared") {
		t.Fatalf("share status = %q", q.status)
	}
	clickQuestButton(t, u, "trackButton")
	if q.trackedID != "troubled-farmstead" || q.track.Text() != "Untrack" {
		t.Fatalf("track on = %q/%q", q.trackedID, q.track.Text())
	}
	clickQuestButton(t, u, "trackButton")
	if q.trackedID != "" || q.track.Text() != "Track Quest" {
		t.Fatalf("track off = %q/%q", q.trackedID, q.track.Text())
	}
	clickQuestButton(t, u, "abandonButton")
	if !strings.Contains(q.status, "Abandoned") {
		t.Fatalf("abandon status = %q", q.status)
	}
	if q.rewardCells[0].LinkCount() != 0 {
		t.Fatal("reward labels must not draw hyperlink underlines")
	}
}

// TestQuestCloseReopen exercises the title X close path and the reopen
// button: closing hides the ornate window and reveals the opener, reopening
// restores the journal without losing the selection.
func TestQuestCloseReopen(t *testing.T) {
	u := ui.New(800, 800)
	q := newQuestWindow(u)
	if !q.titled.IsOpen() {
		t.Fatal("journal must start open")
	}
	if q.reopen.Visible() {
		t.Fatal("reopen must start hidden")
	}
	clickQuestButton(t, u, "questWindow/close")
	if q.titled.IsOpen() || !q.reopen.Visible() {
		t.Fatalf("close: open=%v reopen=%v", q.titled.IsOpen(), q.reopen.Visible())
	}
	clickQuestButton(t, u, "reopenButton")
	if !q.titled.IsOpen() || q.reopen.Visible() {
		t.Fatalf("reopen: open=%v reopen=%v", q.titled.IsOpen(), q.reopen.Visible())
	}
	if id, ok := q.list.Selected(); !ok || id == "" {
		t.Fatal("selection must survive close/reopen")
	}
}

// TestQuestUsesLibraryWidgets guards the composition contract: the journal
// must use the titled window, list, tab, button, rich-text, frame, and
// canvas widgets rather than bypassing the library with custom drawing.
func TestQuestUsesLibraryWidgets(t *testing.T) {
	u := ui.New(800, 800)
	q := newQuestWindow(u)
	if q.titled == nil || q.list == nil || q.tabs == nil || q.body == nil || q.sheet == nil || q.rules == nil || q.image == nil || q.rewardArt == nil {
		t.Fatal("journal must own titled, list, tabs, body, sheet, rules, image, and rewards")
	}
	for _, name := range []string{"questWindow", "questTabs", "questList", "questSheet", "questRules", "questBody", "questImage", "questRewardArt", "questRewards", "questReward1", "questReward2", "questReward3", "showMapButton", "shareButton", "abandonButton", "trackButton"} {
		w := u.Lookup(name)
		if w == nil {
			t.Fatalf("missing widget %q", name)
		}
		switch w.(type) {
		case *widgets.Frame, *widgets.TabBar, *widgets.List, *widgets.RichText, *widgets.Button, *widgets.Label, *widgets.Canvas:
			continue
		default:
			t.Fatalf("widget %q uses unexpected type %T", name, w)
		}
	}
}

// TestQuestListWidgetRows guards the journal index contract: each tale
// carries a borrowed row frame (title, zone, level, status icon) that
// survives item copies by reference while structure stays isolated.
func TestQuestListWidgetRows(t *testing.T) {
	u := ui.New(800, 800)
	q := newQuestWindow(u)
	row, ok := q.list.VisibleRowAt(0)
	if !ok {
		t.Fatal("missing first row")
	}
	if row.ID != "troubled-farmstead" || row.Content == nil {
		t.Fatalf("first row = %+v, want farmstead widget row", row)
	}
	frame, ok := row.Content.(*widgets.Frame)
	if !ok {
		t.Fatalf("row content = %T, want *widgets.Frame", row.Content)
	}
	texts := questRowChildTexts(frame)
	if texts["title"] != "A Troubled Farmstead" || texts["zone"] != "Westfall" || texts["level"] != "Level 12" {
		t.Fatalf("row children = %q, want farmstead title/zone/level", texts)
	}
	if !questRowHasIcon(frame, "quest-bang") {
		t.Fatal("first row must carry the quest-bang status icon")
	}
	items := q.list.Items()
	if len(items) != 6 || items[0].Content != row.Content {
		t.Fatal("items copy must share row content by reference")
	}
	items[0].ID = "mutated"
	if q.list.Items()[0].ID != "troubled-farmstead" {
		t.Fatal("output structure mutation changed list state")
	}
}

// questRowChildTexts indexes one row frame's label children by their name
// suffix: title, zone, and level.
func questRowChildTexts(frame *widgets.Frame) map[string]string {
	texts := map[string]string{}
	if frame == nil {
		return texts
	}
	for _, child := range frame.Children() {
		label, ok := child.(*widgets.Label)
		if !ok {
			continue
		}
		name := label.Name()
		switch {
		case strings.HasSuffix(name, "/title"):
			texts["title"] = label.Text()
		case strings.HasSuffix(name, "/zone"):
			texts["zone"] = label.Text()
		case strings.HasSuffix(name, "/level"):
			texts["level"] = label.Text()
		}
	}
	return texts
}

// questRowHasIcon reports whether one row frame carries an icon run with
// the given whitelisted name.
func questRowHasIcon(frame *widgets.Frame, name string) bool {
	if frame == nil {
		return false
	}
	for _, child := range frame.Children() {
		provider, ok := child.(widgets.RichProvider)
		if !ok || !provider.HasRichText() {
			continue
		}
		for _, segment := range provider.RichSegments() {
			if segment.HasIcon && segment.Icon == name {
				return true
			}
		}
	}
	return false
}
