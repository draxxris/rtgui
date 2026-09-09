package widgets_test

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/widgets"
)

// listFixture returns a stable category tree mirroring the gallery demo.
func listFixture() []widgets.ListItem {
	return []widgets.ListItem{
		{ID: "fav", Label: "Favorites", Icon: "star"},
		{ID: "weapons", Label: "Weapons", Icon: "sword", Children: []widgets.ListItem{
			{ID: "swords", Label: "Swords"},
			{ID: "axes", Label: "Axes"},
		}},
		{ID: "mats", Label: "Materials", Icon: "crate", Expanded: true, Children: []widgets.ListItem{
			{ID: "herbs", Label: "Herbs"},
			{ID: "ess", Label: "Essences"},
		}},
	}
}

// TestListDefaults checks constructor metrics and empty state.
func TestListDefaults(t *testing.T) {
	list := widgets.NewList("cats", core.Rect{X: 10, Y: 10, W: 220, H: 300})
	if list.Kind() != core.WidgetList {
		t.Fatalf("kind = %v", list.Kind())
	}
	if list.RowHeight() != widgets.DefaultListRowHeight || list.Indent() != widgets.DefaultListIndent {
		t.Fatalf("metrics = %v/%v", list.RowHeight(), list.Indent())
	}
	if list.VisibleRowCount() != 0 || list.MaxScroll() != 0 || list.ScrollOffset() != 0 {
		t.Fatal("new list must be empty without scroll")
	}
	if _, ok := list.Selected(); ok {
		t.Fatal("new list must have no selection")
	}
	if _, err := widgets.AsList(list); err != nil {
		t.Fatalf("AsList: %v", err)
	}
	if _, err := widgets.AsList(widgets.NewLabel("lbl", core.Rect{}, "x")); err == nil {
		t.Fatal("expected ErrKindMismatch casting label to List")
	}
}

// TestListNilAccessors checks zero-value reads.
func TestListNilAccessors(t *testing.T) {
	var nilList *widgets.List
	if nilList.VisibleRowCount() != 0 {
		t.Fatal("nil row count must be zero")
	}
	if nilList.RowHeight() != widgets.DefaultListRowHeight {
		t.Fatal("nil row height must fall back to default")
	}
	if nilList.VisibleRows() != nil || nilList.Items() != nil {
		t.Fatal("nil list snapshots must be nil")
	}
	if _, ok := nilList.VisibleRowAt(0); ok {
		t.Fatal("nil list row lookup must miss")
	}
	if nilList.IndexOfRow("x") != -1 {
		t.Fatal("nil index lookup must miss")
	}
	if nilList.Text() != "" || nilList.Indent() != widgets.DefaultListIndent {
		t.Fatal("nil list text and indent must be zero-safe")
	}
	if _, ok := nilList.Selected(); ok {
		t.Fatal("nil selection must be unset")
	}
}

// TestListNilMutations checks zero-value writes report no change.
func TestListNilMutations(t *testing.T) {
	var nilList *widgets.List
	if nilList.Select("x") {
		t.Fatal("nil select must report false")
	}
	if nilList.SetExpanded("x", true) || nilList.Toggle("x") {
		t.Fatal("nil expansion must report false")
	}
	if nilList.ClearSelection() || nilList.ExpandAll() || nilList.CollapseAll() {
		t.Fatal("nil bulk edits must report false")
	}
	if nilList.SetItems(listFixture()) {
		t.Fatal("nil SetItems must report false")
	}
	if nilList.SetScrollOffset(1) || nilList.ScrollBy(1) || nilList.EnsureScrollBounds(100) {
		t.Fatal("nil scroll edits must report false")
	}
	if nilList.MaxScroll() != 0 || nilList.ScrollOffset() != 0 || nilList.IsExpanded("x") {
		t.Fatal("nil scroll and expansion reads must be zero")
	}
}

// TestListCopiesItems checks deep-copy semantics for inputs and outputs.
func TestListCopiesItems(t *testing.T) {
	items := listFixture()
	list := widgets.NewList("cats", core.Rect{W: 220, H: 300})
	if !list.SetItems(items) {
		t.Fatal("initial SetItems must report a change")
	}
	items[0].Label = "mutated input"
	items[2].Children[0].Label = "mutated child"
	rows := list.VisibleRows()
	if rows[0].Label != "Favorites" {
		t.Fatalf("input mutation leaked into list: %q", rows[0].Label)
	}
	snapshot := list.Items()
	snapshot[0].Label = "mutated output"
	snapshot[2].Children[0].Label = "mutated child output"
	if list.Items()[0].Label != "Favorites" || list.Items()[2].Children[0].Label != "Herbs" {
		t.Fatal("output mutation changed list items")
	}
	if list.SetItems(list.Items()) {
		t.Fatal("identical SetItems must report no change")
	}
}

// TestListFlattensVisibleRows checks collapsed categories hide children.
func TestListFlattensVisibleRows(t *testing.T) {
	list := widgets.NewList("cats", core.Rect{W: 220, H: 300})
	list.SetItems(listFixture())
	// fav + weapons(collapsed) + mats(expanded) + herbs + ess = 5 visible.
	if got := list.VisibleRowCount(); got != 5 {
		t.Fatalf("visible rows = %d, want 5", got)
	}
	rows := list.VisibleRows()
	want := []string{"fav", "weapons", "mats", "herbs", "ess"}
	for i, id := range want {
		if rows[i].ID != id {
			t.Fatalf("row %d = %q, want %q", i, rows[i].ID, id)
		}
	}
	if rows[3].Depth != 1 || rows[0].Depth != 0 {
		t.Fatalf("depths = %d/%d", rows[0].Depth, rows[3].Depth)
	}
	if !rows[1].HasChildren || rows[3].HasChildren {
		t.Fatal("category/leaf flags mismatch")
	}
	if rows[2].Expanded != true || rows[1].Expanded != false {
		t.Fatal("expanded flags mismatch")
	}
	if got := list.IndexOfRow("axes"); got != -1 {
		t.Fatalf("hidden child index = %d, want -1", got)
	}
	if got := list.IndexOfRow("ess"); got != 4 {
		t.Fatalf("ess index = %d, want 4", got)
	}
	reused := list.AppendVisibleRows(nil)
	if len(reused) != 5 || reused[4].ID != "ess" {
		t.Fatalf("AppendVisibleRows = %v", reused)
	}
	if _, ok := list.VisibleRowAt(9); ok {
		t.Fatal("out-of-range row must miss")
	}
}

// TestListSelectsLeavesOnly checks toggle-only categories.
func TestListSelectsLeavesOnly(t *testing.T) {
	list := widgets.NewList("cats", core.Rect{W: 220, H: 300})
	list.SetItems(listFixture())
	if !list.Select("ess") {
		t.Fatal("leaf select must report a change")
	}
	if id, ok := list.Selected(); !ok || id != "ess" {
		t.Fatalf("selected = %q/%v", id, ok)
	}
	if list.Text() != "Essences" {
		t.Fatalf("Text = %q, want selected label", list.Text())
	}
	if list.Select("ess") || list.Select("mats") || list.Select("missing") || list.Select("") {
		t.Fatal("duplicate, category, unknown, or empty select must report false")
	}
	if id, _ := list.Selected(); id != "ess" {
		t.Fatalf("failed select changed selection to %q", id)
	}
	if !list.ClearSelection() || list.ClearSelection() {
		t.Fatal("clear must report change once")
	}
	if _, ok := list.Selected(); ok {
		t.Fatal("selection must be unset after clear")
	}
}

// TestListTextSurvivesCollapse checks the selected label outlives hiding.
func TestListTextSurvivesCollapse(t *testing.T) {
	list := widgets.NewList("cats", core.Rect{W: 220, H: 300})
	list.SetItems(listFixture())
	list.Select("ess")
	if !list.SetExpanded("mats", false) {
		t.Fatal("collapse must report a change")
	}
	if list.VisibleRowCount() != 3 {
		t.Fatalf("collapsed rows = %d, want 3", list.VisibleRowCount())
	}
	if id, ok := list.Selected(); !ok || id != "ess" {
		t.Fatal("collapse must preserve the selection")
	}
	if list.Text() != "Essences" {
		t.Fatalf("Text after collapse = %q, want selected label", list.Text())
	}
}

// TestListPreservesSelectionAcrossUpdates checks identity-stable refreshes.
func TestListPreservesSelectionAcrossUpdates(t *testing.T) {
	list := widgets.NewList("cats", core.Rect{W: 220, H: 300})
	list.SetItems(listFixture())
	list.Select("ess")
	updated := listFixture()
	updated[2].Children[1].Label = "Greater Essences"
	if !list.SetItems(updated) {
		t.Fatal("label update must report a change")
	}
	if id, _ := list.Selected(); id != "ess" {
		t.Fatal("selection must survive a label refresh")
	}
	if list.Text() != "Greater Essences" {
		t.Fatalf("Text = %q after refresh", list.Text())
	}
	// Removing the selected leaf clears the selection.
	pruned := listFixture()
	pruned[2].Children = pruned[2].Children[:1]
	if !list.SetItems(pruned) {
		t.Fatal("prune must report a change")
	}
	if _, ok := list.Selected(); ok {
		t.Fatal("selection must clear when the leaf disappears")
	}
	// Removing a leaf while unselected still reports the row change.
	if !list.SetItems(listFixture()) {
		t.Fatal("restore must report a change")
	}
	if _, ok := list.Selected(); ok {
		t.Fatal("restore must not invent a selection")
	}
}

// TestListTogglesCategories checks expansion state and visibility.
func TestListTogglesCategories(t *testing.T) {
	list := widgets.NewList("cats", core.Rect{W: 220, H: 300})
	list.SetItems(listFixture())
	if !list.Toggle("weapons") || !list.IsExpanded("weapons") {
		t.Fatal("toggle must expand a collapsed category")
	}
	if got := list.VisibleRowCount(); got != 7 {
		t.Fatalf("expanded rows = %d, want 7", got)
	}
	if !list.Toggle("weapons") || list.IsExpanded("weapons") {
		t.Fatal("toggle must collapse an expanded category")
	}
	if list.Toggle("ess") || list.Toggle("missing") {
		t.Fatal("leaf or unknown toggle must report false")
	}
	if list.SetExpanded("weapons", false) {
		t.Fatal("redundant collapse must report false")
	}
	if !list.SetExpanded("weapons", true) {
		t.Fatal("explicit expand must report a change")
	}
	if list.IsExpanded("ess") || list.IsExpanded("missing") {
		t.Fatal("leaf or unknown IsExpanded must be false")
	}
}

// TestListExpandCollapseAll checks bulk expansion helpers.
func TestListExpandCollapseAll(t *testing.T) {
	list := widgets.NewList("cats", core.Rect{W: 220, H: 300})
	list.SetItems(listFixture())
	if !list.CollapseAll() || list.IsExpanded("mats") || list.IsExpanded("weapons") {
		t.Fatal("CollapseAll must close every category")
	}
	if list.VisibleRowCount() != 3 {
		t.Fatalf("collapsed rows = %d, want 3", list.VisibleRowCount())
	}
	if list.CollapseAll() {
		t.Fatal("redundant CollapseAll must report false")
	}
	if !list.ExpandAll() || !list.IsExpanded("mats") || list.VisibleRowCount() != 7 {
		t.Fatal("ExpandAll must open every category")
	}
	if list.ExpandAll() {
		t.Fatal("redundant ExpandAll must report false")
	}
	if widgets.NewList("empty", core.Rect{}).ExpandAll() {
		t.Fatal("empty ExpandAll must report false")
	}
}

// TestListScrollBounds checks derived limits and clamping.
func TestListScrollBounds(t *testing.T) {
	list := widgets.NewList("cats", core.Rect{W: 220, H: 100})
	list.SetItems(listFixture())
	// 5 visible rows * 28px = 140px WO 100px viewport -> max 40.
	if got := list.MaxScroll(); got != 40 {
		t.Fatalf("max scroll = %v, want 40", got)
	}
	if !list.SetScrollOffset(20) || list.ScrollOffset() != 20 {
		t.Fatalf("scroll offset = %v", list.ScrollOffset())
	}
	if list.SetScrollOffset(20) {
		t.Fatal("duplicate scroll must report false")
	}
	if !list.ScrollBy(100) || list.ScrollOffset() != 40 {
		t.Fatalf("clamped scroll = %v, want 40", list.ScrollOffset())
	}
	if list.ScrollBy(0) || !list.ScrollBy(-100) || list.ScrollOffset() != 0 {
		t.Fatalf("scroll home = %v", list.ScrollOffset())
	}
}

// TestListScrollViewportSync checks viewport reconciliation and limits.
func TestListScrollViewportSync(t *testing.T) {
	list := widgets.NewList("cats", core.Rect{W: 220, H: 100})
	list.SetItems(listFixture())
	// A taller viewport removes scrolling without losing rows.
	if !list.EnsureScrollBounds(500) || list.MaxScroll() != 0 {
		t.Fatalf("roomy viewport max = %v", list.MaxScroll())
	}
	if list.EnsureScrollBounds(500) {
		t.Fatal("redundant bounds sync must report false")
	}
	list.EnsureScrollBounds(100)
	if list.MaxScroll() != 40 {
		t.Fatalf("restored max = %v, want 40", list.MaxScroll())
	}
	list.SetMaxScroll(10)
	if list.MaxScroll() != 10 {
		t.Fatalf("explicit max = %v", list.MaxScroll())
	}
	short := widgets.NewList("short", core.Rect{W: 220, H: 500})
	short.SetItems([]widgets.ListItem{{ID: "only", Label: "Only"}})
	if short.MaxScroll() != 0 {
		t.Fatalf("fitting list max = %v", short.MaxScroll())
	}
}

// TestListFluentSetters checks chaining and metric guards.
func TestListFluentSetters(t *testing.T) {
	list := widgets.NewList("cats", core.Rect{W: 220, H: 300})
	list.SetItems(listFixture())
	rowsBefore := list.VisibleRowCount()
	list.SetRowHeight(0).SetRowHeight(-4).SetIndent(-1)
	if list.RowHeight() != widgets.DefaultListRowHeight || list.VisibleRowCount() != rowsBefore {
		t.Fatal("invalid metrics must be ignored")
	}
	list.SetRowHeight(32).SetIndent(8)
	if list.RowHeight() != 32 || list.Indent() != 8 {
		t.Fatal("valid metrics must apply")
	}
	list.OnSelect(func(string) {})
	list.OnToggle(func(string, bool) {})
	if list.OnSelectHandler() == nil || list.OnToggleHandler() == nil {
		t.Fatal("handlers must be stored")
	}
	list.OnSelect(nil).OnToggle(nil)
	if list.OnSelectHandler() != nil || list.OnToggleHandler() != nil {
		t.Fatal("nil handlers must clear the slots")
	}
	list.SetTooltip("tip").SetFontSize(18).SetItalic(true).SetAlign(core.AlignLeft).
		SetTextColor(core.Color{R: 1, G: 2, B: 3, A: 255})
	if list.Tooltip() != "tip" || list.FontSize() != 18 || !list.Italic() || list.Align() != core.AlignLeft {
		t.Fatal("fluent styling must apply")
	}
	if c, ok := list.TextColor(); !ok || c.R != 1 {
		t.Fatal("explicit text color must apply")
	}
}
