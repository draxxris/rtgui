// Package main implements the MMORPG quest window example.
package main

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/ui"
	"github.com/draxxris/rtgui/widgets"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Reward is one quest reward row. Icon names a UI-whitelisted inline
// graphic; Amount holds the count text and Label holds the reward name.
type Reward struct {
	Icon   string
	Amount string
	Label  string
}

// Quest is one journal entry. Status is Current or Completed and selects
// which tab shows the quest. Important selects the gold bang list icon
// while plain tales use the hollow circle.
type Quest struct {
	ID          string
	Title       string
	Level       int
	Zone        string
	Status      string
	Important   bool
	Intro       string
	Description string
	Objectives  []string
	Rewards     []Reward
}

// questWindow owns the quest journal widgets and the demo game state.
// The facade owns hover, press, focus, and popup state; this struct keeps
// the quest list, selection, tracked marker, farm art, and status text.
// The v6 ornate shell lives on the root border and paints after children,
// so the moldings draw over the full-bleed titlebar and content edges.
type questWindow struct {
	facade        *ui.UI
	titled        *widgets.TitledFrame
	tabs          *widgets.TabBar
	list          *widgets.List
	sheet         *widgets.Frame
	rules         *widgets.Canvas
	titleLabel    *widgets.Label
	subtitleLabel *widgets.Label
	body          *widgets.RichText
	image         *widgets.Canvas
	rewardArt     *widgets.Canvas
	rewards       *widgets.RichText
	rewardCells   [3]*widgets.RichText
	rewardSet     []Reward
	showMap       *widgets.Button
	share         *widgets.Button
	abandon       *widgets.Button
	track         *widgets.Button
	reopen        *widgets.Button
	quests        []Quest
	trackedID     string
	farmTexture   rl.Texture2D
	farmReady     bool
	status        string
}

// questTabLabels are the filter tabs in selection order. Indices align with
// questTabStatus so applyTab can map selection to a status filter.
func questTabLabels() []string {
	return []string{"Current (6)", "Completed (24)"}
}

// questTabStatus maps a tab index to its quest status filter. Out-of-range
// indices fall back to Current so drawing never branches on missing content.
func questTabStatus(index int) string {
	if index == 1 {
		return "Completed"
	}
	return "Current"
}

// questSampleData returns the reference journal: six Current tales led by
// the troubled farmstead plus twenty-four generated Completed tales.
func questSampleData() []Quest {
	quests := []Quest{
		{ID: "troubled-farmstead", Title: "A Troubled Farmstead", Level: 12, Zone: "Westfall", Status: "Current", Important: true, Intro: "The Jansen farm has been overrun by riverpaw\nbandits. Speak with Farmer Jansen and see what\naid he needs.", Description: "The Jansen farm has always been a peaceful place,\nbut lately riverpaw bandits have been causing\ntrouble. Farmer Jansen could use some help\nsecuring his land and ensuring his family's safety.", Objectives: []string{"Speak with Farmer Jansen at the Jansen\nStead in Westfall."}, Rewards: []Reward{{Icon: "reward-xp", Amount: "1,200", Label: "Experience"}, {Icon: "reward-silver", Amount: "35", Label: "Silver"}, {Icon: "reward-satchel", Amount: "", Label: "Farmer's Supply Satchel"}}},
		{ID: "wolves-door", Title: "Wolves at the Door", Level: 10, Zone: "Elwynn Forest", Status: "Current", Important: true, Intro: "Wolf packs press against the village palisade at night.", Description: "The village guard needs the packs thinned before winter stores run out.", Objectives: []string{"Slay 6 Starving Wolves"}, Rewards: []Reward{{Icon: "reward-xp", Amount: "800", Label: "Experience"}}},
		{ID: "supplies-watch", Title: "Supplies for the Watch", Level: 12, Zone: "Westfall", Status: "Current", Intro: "The watch post runs short on bandages and lamp oil.", Description: "Carry the supply crate from Sentinel Hill to the watch post.", Objectives: []string{"Deliver the Supply Crate"}, Rewards: []Reward{{Icon: "reward-xp", Amount: "900", Label: "Experience"}}},
		{ID: "lost-scout", Title: "The Lost Scout", Level: 15, Zone: "Redridge Mountains", Status: "Current", Intro: "A scout failed to return from the ridge trail.", Description: "Search the ridge trail for signs of the missing scout.", Objectives: []string{"Find the Lost Scout"}, Rewards: []Reward{{Icon: "reward-xp", Amount: "1,100", Label: "Experience"}}},
		{ID: "healers-touch", Title: "A Healer's Touch", Level: 10, Zone: "Stormwind City", Status: "Current", Important: true, Intro: "The cathedral clinic needs swift hands.", Description: "Aid the healers by gathering restorative herbs.", Objectives: []string{"Gather 5 Restorative Herbs"}, Rewards: []Reward{{Icon: "reward-xp", Amount: "850", Label: "Experience"}}},
		{ID: "old-friends", Title: "Old Friends", Level: 18, Zone: "Duskwood", Status: "Current", Intro: "An old friend waits at the crossroads inn.", Description: "Meet your friend at the crossroads inn after dark.", Objectives: []string{"Meet at the Crossroads Inn"}, Rewards: []Reward{{Icon: "reward-xp", Amount: "1,300", Label: "Experience"}}},
	}
	zones := []string{"Westfall", "Elwynn Forest", "Duskwood", "Redridge Mountains", "Stormwind City"}
	for i := 1; i <= 24; i++ {
		zone := zones[(i-1)%len(zones)]
		quests = append(quests, Quest{ID: fmt.Sprintf("completed-%02d", i), Title: fmt.Sprintf("Completed Tale %02d", i), Level: 5 + (i % 15), Zone: zone, Status: "Completed", Intro: "A finished errand from an earlier season.", Description: "This tale is closed and kept for remembrance.", Objectives: []string{"Tale complete"}, Rewards: []Reward{{Icon: "reward-xp", Amount: "100", Label: "Experience"}}})
	}
	return quests
}

// findQuest resolves one quest by ID. It reports false when the ID is
// unknown so callers can show an empty details pane instead of stale text.
func (q *questWindow) findQuest(id string) (Quest, bool) {
	if q == nil {
		return Quest{}, false
	}
	for _, quest := range q.quests {
		if quest.ID == id {
			return quest, true
		}
	}
	return Quest{}, false
}

// newQuestWindow builds the journal widgets, registers them in draw order,
// wires direct callbacks, places the 800 layout, and selects the troubled
// farmstead. All hover, press, and focus state lives in the facade.
func newQuestWindow(facade *ui.UI) *questWindow {
	q := &questWindow{
		facade:        facade,
		titled:        widgets.NewTitledFrame("questWindow", core.Rect{}, "Quests").SetVariant("quest").SetTitleBarHeight(42).SetBorderInsets(15, 11, 15, 7).SetTitleBarSideInsets(7, 7).SetContentTopGap(11).SetTitleLeftInset(20).SetTitleTopOffset(-4).SetCloseButtonSize(43).SetCloseInset(4, 0),
		tabs:          widgets.NewTabBar("questTabs", core.Rect{}, questTabLabels(), 0),
		list:          widgets.NewList("questList", core.Rect{}),
		sheet:         widgets.NewFrame("questSheet", core.Rect{}),
		rules:         widgets.NewCanvas("questRules", core.Rect{}, nil),
		titleLabel:    widgets.NewStyledLabel("questTitle", core.Rect{}, "A Troubled Farmstead", 35, false, core.AlignLeft),
		subtitleLabel: widgets.NewStyledLabel("questSubtitle", core.Rect{}, "Westfall • Level 12", 24, false, core.AlignLeft),
		body:          widgets.NewRichText("questBody", core.Rect{}, nil),
		image:         widgets.NewCanvas("questImage", core.Rect{}, nil),
		rewardArt:     widgets.NewCanvas("questRewardArt", core.Rect{}, nil),
		rewards:       widgets.NewRichText("questRewards", core.Rect{}, nil),
		rewardCells: [3]*widgets.RichText{
			widgets.NewRichText("questReward1", core.Rect{}, nil),
			widgets.NewRichText("questReward2", core.Rect{}, nil),
			widgets.NewRichText("questReward3", core.Rect{}, nil),
		},
		showMap: widgets.NewButton("showMapButton", core.Rect{}, "Show on Map"),
		share:   widgets.NewButton("shareButton", core.Rect{}, "Share"),
		abandon: widgets.NewButton("abandonButton", core.Rect{}, "Abandon"),
		track:   widgets.NewButton("trackButton", core.Rect{}, "Track Quest"),
		reopen:  widgets.NewButton("reopenButton", core.Rect{}, "Open Quests"),
		quests:  questSampleData(),
		status:  "Reading A Troubled Farmstead (demo)",
	}
	q.titled.CloseButton().SetText("")
	q.applyQuestClasses()
	q.rules.SetCanvasDraw(drawQuestRules)
	q.image.SetCanvasDraw(q.drawFarmstead)
	q.rewardArt.SetCanvasDraw(q.drawQuestRewardArt)
	q.registerQuestIcons()
	q.loadFarmTexture()
	q.wireQuestCallbacks()
	q.registerQuestWidgets(facade)
	q.applyQuestLayout()
	q.reopen.SetVisible(false)
	q.applyQuestTab(q.tabs.SelectedTab())
	return q
}

// applyQuestClasses assigns the CSS skin classes for the journal chrome,
// dark list, parchment sheet, and action buttons. The root carries the v6
// border while the titlebar bleeds beneath the moldings; the quest-list
// track/thumb use custom scroll art with an 8px three-patch slice.
func (q *questWindow) applyQuestClasses() {
	if q == nil {
		return
	}
	q.tabs.SetClass("quest-tabs")
	q.tabs.SetAlign(core.AlignCenter)
	q.list.SetClass("quest-list")
	q.sheet.SetClass("quest-sheet")
	q.titleLabel.SetClass("quest-heading")
	q.subtitleLabel.SetClass("quest-sub")
	q.body.SetClass("quest-body")
	q.rewards.SetClass("quest-rewards")
	for _, cell := range q.rewardCells {
		cell.SetClass("quest-reward")
	}
	q.showMap.SetClass("quest-action")
	q.share.SetClass("quest-action")
	q.abandon.SetClass("quest-action")
	q.track.SetClass("quest-primary")
	q.reopen.SetClass("quest-action")
	for _, button := range []*widgets.Button{q.showMap, q.share, q.abandon, q.track, q.reopen} {
		button.SetAlign(core.AlignCenter)
	}
	q.setQuestActionText(q.showMap, "action-map", "Show on Map")
	q.setQuestActionText(q.share, "action-share", "Share")
	q.abandon.SetText("Abandon")
	q.list.SetRowHeight(60)
}

// registerQuestWidgets adds every journal widget to the facade in draw
// order. TitledFrame chrome registers first, then the journal panels.
// Frame borders paint after children, so the root ring finishes over
// content bleed with no overlay widget.
func (q *questWindow) registerQuestWidgets(facade *ui.UI) {
	if q == nil || facade == nil {
		return
	}
	if err := facade.Add(q.titled.Widgets()...); err != nil {
		panic(err)
	}
	if err := facade.Add(q.tabs, q.list, q.sheet, q.rules, q.titleLabel, q.subtitleLabel, q.body, q.image, q.rewardArt, q.rewards, q.rewardCells[0], q.rewardCells[1], q.rewardCells[2], q.showMap, q.share, q.abandon, q.track, q.reopen); err != nil {
		panic(err)
	}
}

// wireQuestCallbacks attaches direct widget callbacks. Each callback mutates
// only journal state and status text; the facade owns all input routing.
func (q *questWindow) wireQuestCallbacks() {
	if q == nil {
		return
	}
	q.tabs.OnTabSelect(func(index int) {
		q.applyQuestTab(index)
	}).SetTooltip("Filter quests by status")
	q.list.OnSelect(func(id string) {
		q.updateQuestDetails(id)
	}).SetTooltip("Dark quest list — tales select")
	q.body.OnLinkClick(func(link core.Link) {
		q.setQuestStatus(fmt.Sprintf("Link %s %q opened (demo)", link.Kind, link.Target))
	}).OnLinkTooltipRequested(func(link core.Link) string {
		return questLinkTooltip(link)
	}).SetTooltip("Parchment tale — hover a reward link")
	q.rewards.SetTooltip("Rewards for the selected quest")
	for _, cell := range q.rewardCells {
		q.wireRewardCell(cell)
	}
	q.showMap.OnClick(func() {
		q.handleShowMap()
	}).SetTooltip("Show the quest on the map")
	q.share.OnClick(func() {
		q.handleShare()
	}).SetTooltip("Share the selected quest with the party")
	q.abandon.OnClick(func() {
		q.handleAbandon()
	}).SetTooltip("Abandon the selected quest (demo keeps it)")
	q.track.OnClick(func() {
		q.handleTrack()
	}).SetTooltip("Track the selected quest on the map")
	q.titled.OnClose(func() {
		q.setQuestStatus("Quests closed (demo)")
		q.reopen.SetVisible(true)
	})
	q.reopen.OnClick(func() {
		q.titled.Show()
		q.reopen.SetVisible(false)
		q.setQuestStatus("Quests opened (demo)")
	}).SetTooltip("Reopen the quest journal")
}

// wireRewardCell keeps each fixed reward label on the same link callback
// path as the main parchment text.
func (q *questWindow) wireRewardCell(cell *widgets.RichText) {
	if q == nil || cell == nil {
		return
	}
	cell.OnLinkClick(func(link core.Link) {
		q.setQuestStatus(fmt.Sprintf("Link %s %q opened (demo)", link.Kind, link.Target))
	}).OnLinkTooltipRequested(func(link core.Link) string {
		return questLinkTooltip(link)
	}).SetTooltip("Reward — click for details")
}

// questLinkTooltip returns hover text for parchment reward links. Plain
// quests fall back to the target name so tooltips never draw empty boxes.
func questLinkTooltip(link core.Link) string {
	switch link.Kind {
	case core.LinkItem:
		return "Item — " + link.Target
	case core.LinkQuest:
		return "Quest — " + link.Target
	default:
		if link.Target == "" {
			return ""
		}
		return link.Target
	}
}

// setQuestStatus updates the demo status message for headless logs and
// windowed callbacks. The reference window shows no status line, so the
// text stays in memory instead of a label widget.
func (q *questWindow) setQuestStatus(msg string) {
	if q == nil {
		return
	}
	q.status = msg
}

// applyQuestTab rebuilds the dark list for one tab filter, selects the
// first tale, and syncs the bottom action buttons.
func (q *questWindow) applyQuestTab(index int) {
	if q == nil || q.list == nil {
		return
	}
	status := questTabStatus(index)
	q.list.SetItems(q.questListItems(status))
	first := q.firstQuestID(status)
	if first == "" {
		q.showEmptyDetails(status)
		return
	}
	q.list.Select(first)
	q.updateQuestDetails(first)
	if label, ok := q.tabs.TabSelection(); ok {
		q.setQuestStatus(fmt.Sprintf("%s tales listed (demo)", label))
	}
}

// firstQuestID returns the first tale ID for one status in list order.
func (q *questWindow) firstQuestID(status string) string {
	if q == nil {
		return ""
	}
	for _, quest := range q.quests {
		if quest.Status == status {
			return quest.ID
		}
	}
	return ""
}

// questListItems builds the flat dark list for one status filter. Each row
// carries a borrowed quest-row frame (title, zone, level, status icon) that
// the list draws render-only; row selection stays ID-based on the quest
// model. A gold bang marks important tales, a hollow circle plain ones.
func (q *questWindow) questListItems(status string) []widgets.ListItem {
	if q == nil {
		return nil
	}
	items := make([]widgets.ListItem, 0, 8)
	for _, quest := range q.quests {
		if quest.Status != status {
			continue
		}
		items = append(items, widgets.ListItem{ID: quest.ID, Content: q.questRowContent(quest)})
	}
	return items
}

// questRowContent builds one render-only journal row in content-local
// coordinates. Widths target the narrowest rows viewport (list width minus
// skin padding and scrollbar track), so the fixed layout never clips when
// the track hides; the spare pixels read as right padding instead. The
// list positions the frame per row on draw; children are never registered.
func (q *questWindow) questRowContent(quest Quest) *widgets.Frame {
	const rowW = float32(266)
	frame := widgets.NewFrame("quest-row-"+quest.ID, core.Rect{})
	icon := widgets.NewLabel("quest-row-"+quest.ID+"/icon", core.Rect{X: 5, Y: 15, W: 29, H: 29}, "")
	icon.SetRichSegments([]core.RichSegment{{Icon: questListIcon(quest), HasIcon: true, HasIconSize: true, IconSize: 29}})
	title := widgets.NewLabel("quest-row-"+quest.ID+"/title", core.Rect{X: 41, Y: 1, W: rowW - 41 - 5, H: 28}, quest.Title)
	subtitle := widgets.NewLabel("quest-row-"+quest.ID+"/zone", core.Rect{X: 41, Y: 29, W: 140, H: 19}, quest.Zone)
	subtitle.SetTextColor(core.Color{R: 154, G: 163, B: 178, A: 255})
	detail := widgets.NewLabel("quest-row-"+quest.ID+"/level", core.Rect{X: 181, Y: 29, W: rowW - 181 - 5, H: 19}, fmt.Sprintf("Level %d", quest.Level))
	detail.SetTextColor(core.Color{R: 154, G: 163, B: 178, A: 255})
	detail.SetAlign(core.AlignRight)
	return frame.Attach(icon, title, subtitle, detail)
}

// questListIcon selects the dark-list status icon for one quest: a gold
// bang for important tales and a hollow circle for plain tales.
func questListIcon(quest Quest) string {
	if quest.Important {
		return "quest-bang"
	}
	return "quest-circle"
}

// showEmptyDetails clears the parchment pane when a filter holds no tales.
func (q *questWindow) showEmptyDetails(status string) {
	if q == nil {
		return
	}
	if q.titleLabel != nil {
		q.titleLabel.SetText("No " + status + " tales")
	}
	if q.subtitleLabel != nil {
		q.subtitleLabel.SetText("")
	}
	if q.body != nil {
		q.body.SetRichSegments([]core.RichSegment{{Text: "No tales under this banner yet."}})
	}
	q.rewardSet = nil
	q.updateQuestRewardCells(nil)
	if q.rewards != nil {
		q.rewards.SetRichSegments(nil)
	}
	q.updateQuestActions("")
}

// updateQuestDetails swaps the parchment pane to one tale and syncs the
// action buttons. Unknown IDs clear the pane instead of keeping stale text.
func (q *questWindow) updateQuestDetails(id string) {
	if q == nil {
		return
	}
	quest, ok := q.findQuest(id)
	if !ok {
		q.showEmptyDetails(questTabStatus(q.tabs.SelectedTab()))
		return
	}
	if q.titleLabel != nil {
		q.titleLabel.SetText(quest.Title)
	}
	if q.subtitleLabel != nil {
		q.subtitleLabel.SetText(fmt.Sprintf("%s • Level %d", quest.Zone, quest.Level))
	}
	if q.body != nil {
		q.body.SetRichSegments(questBodySegments(quest))
	}
	q.rewardSet = quest.Rewards
	if q.rewards != nil {
		q.rewards.SetRichSegments(questRewardSegments(quest))
	}
	q.updateQuestRewardCells(quest.Rewards)
	q.updateQuestActions(id)
	q.setQuestStatus(fmt.Sprintf("Reading %q (demo)", quest.Title))
}

// drawQuestRules paints the three parchment divider lines and their centered
// diamond ornaments. It is a decorative Canvas child, leaving text and
// interaction on the reusable RichText and Label widgets.
func drawQuestRules(bounds core.Rect) {
	if !rl.IsWindowReady() || bounds.W <= 0 || bounds.H <= 0 {
		return
	}
	// Rule offsets are sheet-relative for the 372-wide parchment stack. The
	// body starts at Y 62, so the first two underline the intro objectives
	// flow; the third sits between the rewards header and the icon row.
	for _, y := range []float32{152, 235, 522} {
		drawQuestRule(bounds, y)
	}
	rl.DrawCircleLines(int32(bounds.X+40), int32(bounds.Y+178), 8, color.RGBA{R: 104, G: 91, B: 67, A: 210})
}

// drawQuestRule paints one thin divider inside the parchment sheet.
func drawQuestRule(bounds core.Rect, offsetY float32) {
	ink := color.RGBA{R: 126, G: 98, B: 61, A: 190}
	x0 := bounds.X + 26
	x1 := bounds.X + bounds.W - 29
	y := bounds.Y + offsetY
	rl.DrawLineEx(rl.NewVector2(x0, y), rl.NewVector2(x1, y), 1, ink)
	cx := bounds.X + bounds.W/2
	diamond := float32(5)
	rl.DrawPoly(rl.NewVector2(cx, y), 4, diamond, 0, ink)
	inner := color.RGBA{R: 233, G: 211, B: 163, A: 255}
	rl.DrawPoly(rl.NewVector2(cx, y), 4, 2, 0, inner)
}

// questBodySegments renders the parchment tale as rich runs: plain intro,
// bold brown Objectives and Description headers, circled objectives, and
// body description text. Decorative section rules come from questRules.
func questBodySegments(quest Quest) []core.RichSegment {
	brown := core.Color{R: 90, G: 58, B: 26, A: 255}
	segments := make([]core.RichSegment, 0, 8+len(quest.Objectives))
	segments = append(segments, questBodyText(quest.Intro+"\n", core.Color{}, false, 19))
	segments = append(segments, questBodyText("Objectives\n", brown, true, 26))
	for _, objective := range quest.Objectives {
		segments = append(segments, questBodyText(questObjectiveText(objective), core.Color{}, false, 26))
	}
	segments = append(segments, questBodyText("Description\n", brown, true, 32))
	segments = append(segments, questBodyText(quest.Description, core.Color{}, false, 19))
	return segments
}

// questObjectiveText indents every objective line after the decorative ring.
func questObjectiveText(value string) string {
	return "     " + strings.ReplaceAll(value, "\n", "\n     ") + "\n"
}

// questBodyText creates one consistently sized parchment text run.
func questBodyText(value string, tint core.Color, bold bool, size float32) core.RichSegment {
	segment := core.RichSegment{Text: value, Bold: bold, HasFontSize: true, FontSize: size}
	if bold || tint != (core.Color{}) {
		segment.Color = tint
		segment.HasColor = true
	}
	return segment
}

// questRewardSegments renders the parchment rewards heading and lead line.
// Individual rewards use fixed child RichText cells so icons and labels keep
// the three columns from the reference instead of flowing into a wrap.
func questRewardSegments(quest Quest) []core.RichSegment {
	brown := core.Color{R: 90, G: 58, B: 26, A: 255}
	return []core.RichSegment{
		questBodyText("Rewards\n", brown, true, 26),
		questBodyText("You will receive:\n", core.Color{}, false, 19),
	}
}

// updateQuestRewardCells refreshes the fixed reward columns and their art
// model. Empty cells are hidden so short quests do not show stale rewards.
func (q *questWindow) updateQuestRewardCells(rewards []Reward) {
	if q == nil {
		return
	}
	for index, cell := range q.rewardCells {
		if index < len(rewards) {
			cell.SetRichSegments(questRewardCellSegments(rewards[index]))
			cell.SetVisible(true)
			continue
		}
		cell.SetRichSegments(nil)
		cell.SetVisible(false)
	}
}

// questRewardCellSegments renders one clean reward label in its fixed cell.
func questRewardCellSegments(reward Reward) []core.RichSegment {
	text := reward.Label
	if reward.Label == "Farmer's Supply Satchel" {
		text = "Farmer's\nSupply Satchel"
	}
	if reward.Amount != "" {
		text = reward.Amount + "\n" + text
	}
	return []core.RichSegment{{Text: text, HasColor: true, Color: core.Color{R: 42, G: 90, B: 42, A: 255}, HasFontSize: true, FontSize: 19}}
}

// drawQuestRewardArt paints framed reward icons in the fixed parchment
// columns. The linked labels remain RichText children for interaction.
func (q *questWindow) drawQuestRewardArt(bounds core.Rect) {
	if q == nil || q.facade == nil || !rl.IsWindowReady() || bounds.W <= 0 || bounds.H <= 0 {
		return
	}
	positions := []float32{29, 155, 264}
	for index, reward := range q.rewardSet {
		if index >= len(positions) {
			break
		}
		q.drawQuestRewardIcon(bounds, positions[index], reward.Icon)
	}
}

// drawQuestRewardIcon draws one dark framed icon from the UI whitelist.
func (q *questWindow) drawQuestRewardIcon(bounds core.Rect, offsetX float32, name string) {
	if name == "" {
		return
	}
	icon, ok := q.facade.LookupInlineIcon(name)
	if !ok || icon.ID == 0 {
		return
	}
	// Icon rows share the reward cells' sheet-relative Y so each framed icon
	// lands left of its amount and label text. Origins snap to whole pixels
	// so point-filtered art never straddles a physical texel.
	dest := rl.Rectangle{X: float32(math.Round(float64(bounds.X + offsetX))), Y: float32(math.Round(float64(bounds.Y + 526))), Width: 43, Height: 43}
	rl.DrawRectangleRec(dest, color.RGBA{R: 24, G: 29, B: 36, A: 255})
	rl.DrawRectangleLinesEx(dest, 2, color.RGBA{R: 81, G: 67, B: 42, A: 255})
	source := rl.Rectangle{X: 0, Y: 0, Width: float32(icon.Width), Height: float32(icon.Height)}
	rl.DrawTexturePro(rl.NewTexture2D(icon.ID, icon.Width, icon.Height, 1, 7), source, rl.Rectangle{X: dest.X + 2, Y: dest.Y + 2, Width: 39, Height: 39}, rl.NewVector2(0, 0), 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})
}

// questRewardText formats one reward as padded inline text following its
// icon run. Amounts without counts render the bare label for satchels.
func questRewardText(reward Reward) string {
	if reward.Amount == "" {
		return "  " + reward.Label + "   "
	}
	return fmt.Sprintf("  %s %s   ", reward.Amount, reward.Label)
}

// updateQuestActions enables the bottom buttons for one selection and names
// the track toggle. Empty selections disable all four actions.
func (q *questWindow) updateQuestActions(id string) {
	if q == nil {
		return
	}
	_, ok := q.findQuest(id)
	enabled := ok && id != ""
	for _, button := range []*widgets.Button{q.showMap, q.share, q.abandon, q.track} {
		if button != nil {
			button.SetEnabled(enabled)
		}
	}
	if q.track != nil {
		label := "Track"
		if q.trackedID != "" && q.trackedID == id {
			label = "Untrack"
		} else if id == "troubled-farmstead" {
			label = "Track Quest"
		}
		q.setQuestActionText(q.track, "action-track", label)
	}
	if q.abandon != nil {
		quest, found := q.findQuest(id)
		q.abandon.SetEnabled(enabled && found && quest.Status == "Current")
	}
}

// selectedQuestID returns the dark list selection, or false when unset.
func (q *questWindow) selectedQuestID() (string, bool) {
	if q == nil || q.list == nil {
		return "", false
	}
	return q.list.Selected()
}

// handleShowMap reports a show-on-map request for the selected tale.
func (q *questWindow) handleShowMap() {
	if q == nil {
		return
	}
	id, ok := q.selectedQuestID()
	if !ok {
		q.setQuestStatus("No tale selected to show")
		return
	}
	quest, found := q.findQuest(id)
	if !found {
		q.setQuestStatus("No tale selected to show")
		return
	}
	q.setQuestStatus(fmt.Sprintf("Showing %q on the map (demo)", quest.Title))
}

// handleAbandon reports an abandon request without deleting the demo tale.
func (q *questWindow) handleAbandon() {
	if q == nil {
		return
	}
	id, ok := q.selectedQuestID()
	if !ok {
		q.setQuestStatus("No tale selected to abandon")
		return
	}
	quest, found := q.findQuest(id)
	if !found {
		q.setQuestStatus("No tale selected to abandon")
		return
	}
	q.setQuestStatus(fmt.Sprintf("Abandoned %q (demo — tale kept)", quest.Title))
}

// handleShare reports a party-share request for the selected tale.
func (q *questWindow) handleShare() {
	if q == nil {
		return
	}
	id, ok := q.selectedQuestID()
	if !ok {
		q.setQuestStatus("No tale selected to share")
		return
	}
	quest, found := q.findQuest(id)
	if !found {
		q.setQuestStatus("No tale selected to share")
		return
	}
	q.setQuestStatus(fmt.Sprintf("Shared %q with the party (demo)", quest.Title))
}

// handleTrack toggles the map-track marker for the selected tale.
func (q *questWindow) handleTrack() {
	if q == nil {
		return
	}
	id, ok := q.selectedQuestID()
	if !ok {
		q.setQuestStatus("No tale selected to track")
		return
	}
	quest, found := q.findQuest(id)
	if !found {
		q.setQuestStatus("No tale selected to track")
		return
	}
	if q.trackedID == id {
		q.trackedID = ""
		q.setQuestStatus(fmt.Sprintf("Stopped tracking %q (demo)", quest.Title))
	} else {
		q.trackedID = id
		q.setQuestStatus(fmt.Sprintf("Tracking %q (demo)", quest.Title))
	}
	q.updateQuestActions(id)
}

// registerQuestIcons whitelists the dark-list status icons, action-button
// icons, and reward textures. Missing files fall back to empty art so
// headless tests still exercise selection, callbacks, and rich content.
func (q *questWindow) registerQuestIcons() {
	if q == nil || q.facade == nil {
		return
	}
	for _, icon := range []struct {
		name string
		file string
	}{
		{name: "quest-bang", file: "quest_bang.png"},
		{name: "quest-circle", file: "quest_circle.png"},
		{name: "action-map", file: "action_map.png"},
		{name: "action-share", file: "action_share.png"},
		{name: "action-track", file: "action_track.png"},
		{name: "reward-xp", file: "reward_xp.png"},
		{name: "reward-silver", file: "reward_silver.png"},
		{name: "reward-satchel", file: "reward_satchel.png"},
	} {
		q.facade.RegisterInlineIcon(icon.name, loadQuestIconFile(icon.file))
	}
}

// setQuestActionText keeps an action button's icon and changing label in one
// rich-segment value. Empty icon names deliberately use plain text, which is
// the Abandon button's behavior and preserves the normal button fallback.
func (q *questWindow) setQuestActionText(button *widgets.Button, icon, label string) {
	if button == nil {
		return
	}
	if icon == "" {
		button.SetText(label)
		return
	}
	button.SetRichSegments([]core.RichSegment{
		{Icon: icon, HasIcon: true, HasIconSize: true, IconSize: 35},
		{Text: label, HasColor: true, Color: core.Color{R: 223, G: 232, B: 245, A: 255}, HasFontSize: true, FontSize: 21},
	})
}

// loadQuestIconFile uploads one optional quest, action, or reward icon at
// native resolution with point filtering, matching the skin path. Single
// GPU scaling stays crisp; it returns an empty placeholder when headless
// or unavailable.
func loadQuestIconFile(name string) core.TooltipIcon {
	const size = int32(32)
	path := findQuestAsset(name)
	if path == "" || !rl.IsWindowReady() {
		return core.TooltipIcon{Width: size, Height: size}
	}
	img := rl.LoadImage(path)
	if img == nil {
		return core.TooltipIcon{Width: size, Height: size}
	}
	defer rl.UnloadImage(img)
	texture := rl.LoadTextureFromImage(img)
	if texture.ID == 0 {
		return core.TooltipIcon{Width: size, Height: size}
	}
	rl.SetTextureFilter(texture, rl.FilterPoint)
	return core.TooltipIcon{ID: texture.ID, Width: texture.Width, Height: texture.Height}
}

// loadFarmTexture uploads the optional farmstead painting for the parchment
// illustration. Headless builds keep the procedural fallback.
func (q *questWindow) loadFarmTexture() {
	if q == nil {
		return
	}
	path := findQuestAsset("farmstead.png")
	if path == "" || !rl.IsWindowReady() {
		return
	}
	img := rl.LoadImage(path)
	if img == nil {
		return
	}
	defer rl.UnloadImage(img)
	q.farmTexture = rl.LoadTextureFromImage(img)
	q.farmReady = q.farmTexture.ID != 0
}

// drawFarmstead renders the parchment quest illustration inside its canvas.
// AI art draws when uploaded; otherwise a small procedural farm keeps the
// headless path allocation-free and the windowed demo recognizable.
func (q *questWindow) drawFarmstead(bounds core.Rect) {
	if q == nil || bounds.W <= 0 || bounds.H <= 0 || !rl.IsWindowReady() {
		return
	}
	if q.farmReady && q.farmTexture.ID != 0 {
		q.drawFarmPhoto(bounds)
		return
	}
	drawProceduralFarm(bounds)
}

// drawFarmPhoto stretches the AI farmstead painting into its canvas with a
// thin dark frame so the illustration sits cleanly on the parchment.
func (q *questWindow) drawFarmPhoto(bounds core.Rect) {
	dest := rl.Rectangle{X: bounds.X, Y: bounds.Y, Width: bounds.W, Height: bounds.H}
	source := questCoverSource(q.farmTexture.Width, q.farmTexture.Height, bounds.W, bounds.H)
	rl.DrawTexturePro(q.farmTexture, source, dest, rl.NewVector2(0, 0), 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	rl.DrawRectangleLinesEx(dest, 2, color.RGBA{R: 90, G: 66, B: 38, A: 255})
}

// questCoverSource returns a centered source crop that fills the destination
// without changing the illustration's aspect ratio.
func questCoverSource(textureWidth, textureHeight int32, destWidth, destHeight float32) rl.Rectangle {
	width, height := float32(textureWidth), float32(textureHeight)
	if width <= 0 || height <= 0 || destWidth <= 0 || destHeight <= 0 {
		return rl.Rectangle{}
	}
	if width/height > destWidth/destHeight {
		cropped := height * destWidth / destHeight
		return rl.Rectangle{X: (width - cropped) / 2, Width: cropped, Height: height}
	}
	cropped := width * destHeight / destWidth
	return rl.Rectangle{Y: (height - cropped) / 2, Width: width, Height: cropped}
}

// unloadArtwork releases textures loaded directly by the example before the
// graphics context closes. CSS-owned textures remain owned by Theme.
func (q *questWindow) unloadArtwork() {
	if q == nil || q.facade == nil || !rl.IsWindowReady() {
		return
	}
	for _, name := range []string{"quest-bang", "quest-circle", "action-map", "action-share", "action-track", "reward-xp", "reward-silver", "reward-satchel"} {
		if icon, ok := q.facade.LookupInlineIcon(name); ok && icon.ID != 0 {
			rl.UnloadTexture(rl.NewTexture2D(icon.ID, icon.Width, icon.Height, 1, 7))
		}
		q.facade.UnregisterInlineIcon(name)
	}
	if q.farmReady && q.farmTexture.ID != 0 {
		rl.UnloadTexture(q.farmTexture)
	}
	q.farmTexture = rl.Texture2D{}
	q.farmReady = false
}

// drawProceduralFarm paints a small stylized farm fallback with sky, field,
// cottage, windmill, tree, and fence shapes for headless-safe demos.
func drawProceduralFarm(bounds core.Rect) {
	r := rl.Rectangle{X: bounds.X, Y: bounds.Y, Width: bounds.W, Height: bounds.H}
	rl.DrawRectangleRec(r, color.RGBA{R: 232, G: 211, B: 163, A: 255})
	skyH := bounds.H * 0.45
	rl.DrawRectangleRec(rl.Rectangle{X: bounds.X, Y: bounds.Y, Width: bounds.W, Height: skyH}, color.RGBA{R: 140, G: 180, B: 220, A: 255})
	rl.DrawRectangleRec(rl.Rectangle{X: bounds.X, Y: bounds.Y + skyH, Width: bounds.W, Height: bounds.H - skyH}, color.RGBA{R: 210, G: 170, B: 90, A: 255})
	cx := bounds.X + bounds.W*0.55
	cy := bounds.Y + bounds.H*0.62
	rl.DrawRectangleRec(rl.Rectangle{X: cx - 70, Y: cy - 30, Width: 140, Height: 70}, color.RGBA{R: 180, G: 150, B: 110, A: 255})
	rl.DrawTriangle(rl.NewVector2(cx-85, cy-30), rl.NewVector2(cx+85, cy-30), rl.NewVector2(cx, cy-75), color.RGBA{R: 120, G: 85, B: 50, A: 255})
	rl.DrawLineEx(rl.NewVector2(bounds.X+70, cy+40), rl.NewVector2(bounds.X+70, cy-60), 6, color.RGBA{R: 110, G: 80, B: 50, A: 255})
	rl.DrawCircle(int32(cx+180), int32(cy-20), 34, color.RGBA{R: 90, G: 140, B: 70, A: 255})
	rl.DrawRectangleLinesEx(r, 2, color.RGBA{R: 90, G: 66, B: 38, A: 255})
}

// findQuestAsset locates one generated quest texture by searching the
// repository skins directory upward from the working directory and binary.
func findQuestAsset(name string) string {
	rel := filepath.Join("testdata", "skins", "quest", name)
	roots := []string{}
	if cwd, err := os.Getwd(); err == nil {
		roots = append(roots, cwd)
	}
	if exe, err := os.Executable(); err == nil {
		roots = append(roots, filepath.Dir(exe))
	}
	for _, root := range roots {
		for dir, depth := root, 0; dir != filepath.Dir(dir) && depth < 6; dir, depth = filepath.Dir(dir), depth+1 {
			if dir == "" {
				continue
			}
			if path := filepath.Join(dir, rel); isQuestAssetFile(path) {
				return path
			}
		}
	}
	return ""
}

// isQuestAssetFile reports whether path is an existing regular file.
func isQuestAssetFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
