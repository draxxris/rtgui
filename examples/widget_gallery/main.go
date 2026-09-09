// widget_gallery is the runnable demo proving the textured rtgui library end-to-end.
//
// It uses stock Go + github.com/gen2brain/raylib-go/raylib plus the github.com/draxxris/rtgui/ui
// facade. The UI owns transform, theme, registry, focus, and
// callbacks; the gallery only polls raylib, forwards to the facade, and draws
// app-specific layers (scroll contents and state samples).
// There is no package-global UI state.
//
// Gallery typography uses the Grenze family (SIL OFL, see
// testdata/fonts/Grenze-OFL.txt): Grenze-Regular for titles, values, and rows,
// Grenze-Italic for captions and the status line. Widget text rendered
// through render.Theme uses Grenze-Regular once the theme font loads.
//
// Gallery contract: two panels, button, checkbox (toggles button enabled),
// textbox (typing + backspace including multi-byte UTF-8),
// dropdown popup, slider driving progress, tab bar switching demo pages (label,
// panel text, frame caption, scroll rows, right-panel heading),
// chat message with clickable item/player/URL links plus link tooltips,
// right-click context menu, hover tooltips plus a T-pinned tooltip scoped to
// the focused demo frame, scroll panel with wheel + scissor,
// frame with click-to-focus glow and relative-move child (layout.MoveFrame,
// R only while demoFrame holds container focus), state-sample strip,
// status line; flags -frames / -screenshot for headless smoke.
//
// Headless / window behavior:
//   - With a display (or xvfb-run) the gallery opens a window, renders textured
//     widgets via DrawWidget nine-patch, and saves screenshots via
//     LoadImageFromScreen + ExportImage (which handles both relative and
//     absolute paths — rl.TakeScreenshot fails on absolute paths because it
//     prepends GetWorkingDirectory).
//   - Without a display (no WAYLAND_DISPLAY / DISPLAY) raylib fails to init;
//     the gallery does NOT crash: it runs a headless smoke path that exercises
//     the facade dispatch and bounded draw recorder and writes a placeholder
//     PNG via the stdlib image/png if -screenshot was requested.
//     The window path requires a display — use xvfb-run on headless CI.
//     `go run ./examples/widget_gallery -frames 3 -screenshot out.png` therefore
//     never crashes; under xvfb it produces a real framebuffer screenshot.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/layout"
	"github.com/draxxris/rtgui/render"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/ui"
	"github.com/draxxris/rtgui/widgets"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	windowWidth  int32 = 1280
	windowHeight int32 = 780
)

var (
	frameLimit = flag.Int("frames", 0, "close after this many frames (0 means run until closed)")
	screenshot = flag.String("screenshot", "", "save the final frame to this PNG path")
)

// tabPage is one app-side tab content set. The TabBar widget owns only
// selection (SelectedTab + OnTabSelect); the gallery swaps these fields
// on selection, browser-tab style. Index-aligned with demoTabs labels.
type tabPage struct {
	heading      string
	label        string
	panelText    string
	frameCaption string
	rowPrefix    string
}

type gallery struct {
	facade *ui.UI

	leftPanel       *widgets.Frame
	rightPanel      *widgets.Frame
	titleLabel      *widgets.Label
	subtitleLabel   *widgets.Label
	leftTitle       *widgets.Label
	rightTitle      *widgets.Label
	button          *widgets.Button
	checkbox        *widgets.Checkbox
	textbox         *widgets.Textbox
	textboxCaption  *widgets.Label
	dropdown        *widgets.Dropdown
	dropdownCaption *widgets.Label
	slider          *widgets.Slider
	sliderCaption   *widgets.Label
	progress        *widgets.ProgressBar
	panel           *widgets.Frame
	panelText       *widgets.Label
	label           *widgets.Label
	tabbar          *widgets.TabBar
	tabbarCaption   *widgets.Label
	chat            *widgets.RichText
	chatCaption     *widgets.Label
	frame           *widgets.Frame
	frameCaption    *widgets.Label
	frameButton     *widgets.Button
	scroll          *widgets.ScrollPanel
	scrollCaption   *widgets.Label
	menuHint        *widgets.Label
	tooltipHint     *widgets.Label
	questLog        *widgets.TitledFrame
	stateCanvas     *widgets.Canvas
	statusLabel     *widgets.Label
	lineGraph       *widgets.LineGraph

	status       string
	designWidth  float32
	designHeight float32
	slots        []gallerySlot
	frameOrigin  core.Vec2
	tabPages     []tabPage
	// lastMouse is the logical pointer at the latest input frame. Scoped
	// hotkey callbacks read it so the T-pinned tooltip anchors where the
	// user pressed T instead of capturing coordinates at registration.
	lastMouse core.Vec2
}

// main opens the gallery when a display is available and otherwise runs the
// headless interaction smoke.
func main() {
	flag.Parse()

	// Window path requires a display. Use xvfb-run on headless CI:
	//   xvfb-run go run ./examples/widget_gallery -frames 3 -screenshot out.png
	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(windowWidth, windowHeight, "RTG textured widget gallery")
	if !rl.IsWindowReady() {
		// Headless smoke: at least do not crash for -frames/-screenshot.
		log.Printf("raylib window not ready (no display) — running headless smoke")
		runHeadlessSmoke()
		return
	}
	defer rl.CloseWindow()
	rl.SetWindowMinSize(1100, 720)
	rl.SetTargetFPS(60)

	facade := ui.New(int(windowWidth), int(windowHeight))
	// PixelSnap keeps nine-patch borders on whole pixels so the drifting demo
	// frame never renders fractional seam positions while it moves.
	facade.Theme().SetPixelSnap(true)
	// CSS-only skinning: every pixel requires a CSS->texture pathway.
	// Unauthored keys stay invisible (text still draws); unauthored states
	// inherit their base rule.
	cssPath := findGalleryCSS()
	if cssPath == "" {
		log.Fatal("gallery css not found; expected testdata/skins/gallery.css (searched cwd and exe parents)")
	}
	if err := facade.Theme().LoadCSSFile(cssPath, ""); err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := facade.Theme().UnloadSkin(); err != nil {
			log.Printf("skin texture cleanup failed: %v", err)
			return
		}
		log.Printf("skin texture cleanup complete")
	}()
	defer facade.Theme().UnloadFonts()

	g := newGallery(facade)

	for frame := 0; !rl.WindowShouldClose(); frame++ {
		// Forward the live window size; the logical design resolution stays
		// fixed, so the window rescales the UI instead of reflowing it.
		g.facade.Resize(rl.GetScreenWidth(), rl.GetScreenHeight())
		g.handleInput()
		g.draw()

		if *frameLimit > 0 && frame+1 >= *frameLimit {
			if *screenshot != "" {
				if err := saveGalleryScreenshot(*screenshot); err != nil {
					log.Printf("screenshot failed: %v", err)
				}
			}
			break
		}
	}
}

// runHeadlessSmoke exercises the facade without a GL context.
// It drives synthetic press/release, typing, and slider drag through
// HandleMouse/HandleKey (guarded by IsWindowReady in draw.go, so no GL is
// touched) through a bounded recorder, and writes a placeholder PNG if
// requested.
func runHeadlessSmoke() {
	facade := ui.New(800, 600)
	recorder, err := render.NewDrawRecorder(64)
	if err != nil {
		log.Fatal(err)
	}
	facade.Theme().SetDrawRecorder(recorder)
	button := widgets.NewButton("smokeButton", core.Rect{X: 10, Y: 10, W: 100, H: 40}, "OK")
	checkbox := widgets.NewCheckbox("smokeCheckbox", core.Rect{X: 10, Y: 60, W: 120, H: 40}, true)
	slider := widgets.NewSlider("smokeSlider", core.Rect{X: 10, Y: 110, W: 120, H: 40}, 0.5)
	field := widgets.NewTextbox("smokeField", core.Rect{X: 10, Y: 160, W: 200, H: 30}, 64)
	tabs := widgets.NewTabBar("smokeTabs", core.Rect{X: 10, Y: 210, W: 200, H: 36}, []string{"A", "B"}, 0)
	chat := widgets.NewRichText("smokeChat", core.Rect{X: 230, Y: 10, W: 200, H: 60}, []core.RichSegment{
		{Text: "hi "},
		{Text: "item", Link: core.Link{Kind: core.LinkItem, Target: "item:1"}},
	})
	smokeScroll := widgets.NewScrollPanel("smokeScroll", core.Rect{X: 10, Y: 260, W: 200, H: 80})
	smokeScroll.SetMaxScroll(core.Vec2{Y: 50})
	if err := facade.Add(button, checkbox, slider, field, tabs, chat, smokeScroll); err != nil {
		log.Fatal(err)
	}
	clicks := 0
	facade.OnClick("smokeButton", func() { clicks++ })
	tabSelected := -1
	facade.OnTabSelect("smokeTabs", func(index int) { tabSelected = index })
	facade.SelectTab("smokeTabs", 1)
	linkClicked := ""
	facade.OnLinkClick("smokeChat", func(link core.Link) { linkClicked = link.Target })
	facade.ActivateLink("smokeChat", 0)
	facade.SetTooltip("smokeButton", "headless tip")
	center := core.Vec2{X: 60, Y: 30}
	facade.HandleMouse(ui.MouseEvent{Pos: center, Pressed: true})
	facade.HandleMouse(ui.MouseEvent{Pos: center, Released: true})
	facade.HandleKey(ui.KeyEvent{Chars: []rune("hi")})
	facade.ShowContextMenu([]ui.MenuItem{{ID: "a", Label: "Alpha"}}, core.Vec2{X: 50, Y: 50}, nil)
	facade.Draw()
	facade.CloseMenu()
	facade.Draw()
	calls := recorder.Calls()
	log.Printf("headless smoke: %d draw calls logged (clicks=%d tab=%d link=%q fallback=%v)", len(calls), clicks, tabSelected, linkClicked, len(calls) > 0 && calls[0].Fallback)

	if *screenshot != "" {
		if err := saveGalleryScreenshot(*screenshot); err != nil {
			log.Printf("headless placeholder failed: %v", err)
			os.Exit(1)
		}
	}
	if *frameLimit > 0 {
		log.Printf("headless smoke completed %d frames", *frameLimit)
	}
}

// saveGalleryScreenshot saves the current framebuffer, falling back to a
// placeholder PNG when no GL context is available. Unlike rl.TakeScreenshot,
// the GPU path handles absolute paths correctly (TakeScreenshot prepends
// GetWorkingDirectory and fails on /tmp/...). Both call sites share this so
// window and headless smoke stay one-liners.
func saveGalleryScreenshot(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if err := saveScreenshot(path); err == nil {
		log.Printf("screenshot saved to %s", path)
		return nil
	} else {
		log.Printf("screenshot via GPU failed (%v) — writing placeholder", err)
	}
	if err := createPlaceholderScreenshot(path); err != nil {
		return err
	}
	log.Printf("placeholder screenshot saved to %s", path)
	return nil
}

// saveScreenshot saves the current framebuffer via LoadImageFromScreen +
// ExportImage.
func saveScreenshot(path string) error {
	img := rl.LoadImageFromScreen()
	if img == nil {
		return fmt.Errorf("LoadImageFromScreen returned nil")
	}
	defer rl.UnloadImage(img)
	// ExportImage handles both absolute and relative paths.
	if !rl.ExportImage(*img, path) {
		return fmt.Errorf("ExportImage failed for %q", path)
	}
	return nil
}

// createPlaceholderScreenshot writes a stdlib PNG so headless -screenshot
// still produces a viewable file even without GL.
func createPlaceholderScreenshot(path string) error {
	w, h := 800, 600
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	fill := func(r image.Rectangle, c color.RGBA) {
		draw.Draw(img, r.Intersect(img.Bounds()), &image.Uniform{C: c}, image.Point{}, draw.Src)
	}
	fill(img.Bounds(), color.RGBA{R: 13, G: 17, B: 27, A: 255})
	fill(image.Rect(20, 60, w-20, h-20), color.RGBA{R: 35, G: 48, B: 70, A: 255})
	fill(image.Rect(30, 80, w-30, 116), color.RGBA{R: 45, G: 78, B: 112, A: 255})
	// State strip placeholders.
	for i := 0; i < 6; i++ {
		fill(image.Rect(30+i*120, h-70, 30+i*120+100, h-40), color.RGBA{R: uint8(45 + i*20), G: 58, B: 82, A: 255})
	}
	// Status line text is not rasterized here; the image documents headless mode.
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// findFontFile locates a checked-in gallery font, honoring RTG_FONT_DIR first.
func findFontFile(name string) string {
	if configured := os.Getenv("RTG_FONT_DIR"); configured != "" {
		if path := filepath.Join(configured, name); isDemoFile(path) {
			return path
		}
	}
	return findDemoFile(filepath.Join("testdata", "fonts", name), filepath.Join("rtgui", "testdata", "fonts", name))
}

// findGalleryCSS locates testdata/skins/gallery.css, mirroring font lookup.
func findGalleryCSS() string {
	return findDemoFile(filepath.Join("testdata", "skins", "gallery.css"), filepath.Join("rtgui", "testdata", "skins", "gallery.css"))
}

// findDemoFile returns the first existing file matching one of rel, searching
// upward (up to 5 parents) from the working directory and the executable
// directory. It replaces the per-asset candidate walkers for CSS and fonts.
func findDemoFile(rel ...string) string {
	roots := []string{}
	if cwd, err := os.Getwd(); err == nil {
		roots = append(roots, cwd)
	}
	if exe, err := os.Executable(); err == nil {
		roots = append(roots, filepath.Dir(exe))
	}
	for _, root := range roots {
		for dir, depth := root, 0; dir != filepath.Dir(dir) && depth < 5; dir, depth = filepath.Dir(dir), depth+1 {
			if dir == "" {
				continue
			}
			for _, r := range rel {
				if path := filepath.Join(dir, r); isDemoFile(path) {
					return path
				}
			}
		}
	}
	return ""
}

// isDemoFile reports whether path is an existing regular file.
func isDemoFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// newGallery builds the widget set, registers it with the facade in draw
// order, and wires the gallery callbacks. All hover, press, focus, and
// dropdown-popup state lives in the facade; the gallery keeps status and its
// fixed design-resolution layout.
func newGallery(facade *ui.UI) *gallery {
	captionColor := core.Color{R: 153, G: 174, B: 202, A: 255}
	g := &gallery{
		facade:          facade,
		leftPanel:       widgets.NewFrame("leftPanel", core.Rect{}),
		rightPanel:      widgets.NewFrame("rightPanel", core.Rect{}),
		titleLabel:      widgets.NewStyledLabel("galleryTitle", core.Rect{}, "RTG textured widget gallery", 26, false, core.AlignLeft).SetTextColor(core.Color{R: 226, G: 239, B: 255, A: 255}),
		subtitleLabel:   widgets.NewStyledLabel("gallerySubtitle", core.Rect{}, "Every pixel needs a CSS texture; missing skins stay invisible and states inherit their base rule. Click demoFrame then R to MoveFrame, T to pin.", 18, true, core.AlignLeft).SetTextColor(captionColor),
		leftTitle:       widgets.NewStyledLabel("leftTitle", core.Rect{}, "Widgets", 20, false, core.AlignLeft).SetTextColor(core.Color{R: 224, G: 235, B: 252, A: 255}),
		rightTitle:      widgets.NewStyledLabel("rightTitle", core.Rect{}, "Widgets output", 20, false, core.AlignLeft).SetTextColor(core.Color{R: 224, G: 235, B: 252, A: 255}),
		button:          widgets.NewButton("primaryButton", core.Rect{}, "Primary button"),
		checkbox:        widgets.NewCheckboxWithLabel("enableCheckbox", core.Rect{}, "Enable primary button", true),
		textboxCaption:  widgets.NewStyledLabel("textboxCaption", core.Rect{}, "Textbox (arrows/Home/End, Shift-select, Ctrl-A/C/X/V, Del — UTF-8)", 20, true, core.AlignLeft).SetTextColor(captionColor),
		textbox:         widgets.NewTextbox("inputTextbox", core.Rect{}, 128),
		dropdownCaption: widgets.NewStyledLabel("dropdownCaption", core.Rect{}, "Dropdown (click to open)", 20, true, core.AlignLeft).SetTextColor(captionColor),
		dropdown:        widgets.NewDropdown("classDropdown", core.Rect{}, []string{"Warrior", "Ranger", "Mage"}, 0),
		sliderCaption:   widgets.NewStyledLabel("sliderCaption", core.Rect{}, "Slider drives the progress bar", 20, true, core.AlignLeft).SetTextColor(captionColor),
		slider:          widgets.NewSlider("valueSlider", core.Rect{}, 0.35),
		progress:        widgets.NewProgressBar("valueProgress", core.Rect{}, 0.35),
		panel:           widgets.NewFrame("demoPanel", core.Rect{}),
		panelText:       widgets.NewStyledLabel("panelText", core.Rect{}, "Panel frame decoration", 22, false, core.AlignLeft).SetTextColor(core.Color{R: 218, G: 230, B: 248, A: 255}),
		label:           widgets.NewLabel("demoLabel", core.Rect{}, "Textured label"),
		tabbarCaption:   widgets.NewStyledLabel("tabbarCaption", core.Rect{}, "Tab bar — click to switch demo pages", 20, true, core.AlignLeft).SetTextColor(captionColor),
		tabbar:          widgets.NewTabBar("demoTabs", core.Rect{}, []string{"Widgets", "Style", "About"}, 0),
		chatCaption:     widgets.NewStyledLabel("chatCaption", core.Rect{}, "Chat message — links clickable", 20, true, core.AlignLeft).SetTextColor(captionColor),
		chat: widgets.NewRichText("chatMessage", core.Rect{}, []core.RichSegment{
			{Text: "Guild: need "},
			{Text: "Thunderfury", Color: core.Color{R: 255, G: 140, B: 40, A: 255}, HasColor: true,
				Link: core.Link{Kind: core.LinkItem, Target: "item:19019"}},
			{Text: " for tonight — whisper "},
			{Text: "Mor'nor", Color: core.Color{R: 120, G: 220, B: 120, A: 255}, HasColor: true,
				Link: core.Link{Kind: core.LinkPlayer, Target: "Mor'nor"}},
			{Text: " or see "},
			{Text: "the wiki", Link: core.Link{Kind: core.LinkURL, Target: "https://example.com/guide"}},
			{Text: "."},
		}),
		frame:         widgets.NewFrame("demoFrame", core.Rect{}),
		frameCaption:  widgets.NewStyledLabel("frameCaption", core.Rect{}, "Frame child moves with its parent (focus + R / sine)", 20, false, core.AlignLeft).SetTextColor(captionColor),
		frameButton:   widgets.NewButton("frameChildButton", core.Rect{}, "Frame child"),
		scrollCaption: widgets.NewStyledLabel("scrollCaption", core.Rect{}, "Scroll panel — wheel over this area", 20, true, core.AlignLeft).SetTextColor(captionColor),
		scroll:        widgets.NewScrollPanel("scrollPanel", core.Rect{}),
		menuHint:      widgets.NewStyledLabel("menuHint", core.Rect{}, "Right-click anywhere for the context menu", 20, true, core.AlignLeft).SetTextColor(captionColor),
		tooltipHint:   widgets.NewStyledLabel("tooltipHint", core.Rect{}, "Click demoFrame then T for a pinned tooltip (Esc dismisses)", 20, true, core.AlignLeft).SetTextColor(captionColor),
		status:        "Click demoFrame to focus it — R moves, T pins a tooltip",
	}
	g.statusLabel = widgets.NewStyledLabel("statusLabel", core.Rect{}, g.status, 18, true, core.AlignLeft).SetTextColor(core.Color{R: 161, G: 192, B: 224, A: 255})
	g.stateCanvas = widgets.NewCanvas("stateSamples", core.Rect{}, g.drawStateSamples)
	g.checkbox.SetTextColor(core.Color{R: 205, G: 218, B: 238, A: 255})
	g.textbox.SetText("Type here")

	g.slider.SetFormat("%.0f%%").SetFontSize(20).SetAlign(core.AlignRight).SetTextColor(core.Color{R: 230, G: 242, B: 255, A: 255})
	g.progress.SetFormat("Progress: %.0f%%").SetFontSize(20).SetTextColor(core.Color{R: 235, G: 255, B: 240, A: 255})

	// Direct callback attachment on widget instances replaces stringly registration:
	buttonClasses := []string{"accent", "danger", ""}
	buttonClassIdx := 0
	g.button.SetClass(buttonClasses[buttonClassIdx])
	g.button.OnClick(func() {
		buttonClassIdx = (buttonClassIdx + 1) % len(buttonClasses)
		cls := buttonClasses[buttonClassIdx]
		g.button.SetClass(cls)
		desc := cls
		if desc == "" {
			desc = "default"
		}
		g.setStatus(fmt.Sprintf("Primary button clicked — class set to %q", desc))
	}).SetTooltip("Click to cycle class (accent -> danger -> default)")

	g.checkbox.OnClick(func() {
		g.button.SetEnabled(g.checkbox.Checked())
		g.setStatus(fmt.Sprintf("Checkbox is %v; primary button enabled=%v", g.checkbox.Checked(), g.button.Enabled()))
	})

	g.frameButton.SetClass("danger")
	g.frameButton.OnClick(func() {
		g.setStatus("Frame child button clicked (styled with .danger class)")
	}).SetTooltip("Frame child styled with .danger class")

	g.slider.OnChange(func(v float32) {
		g.progress.SetValue(v)
		g.setStatus(fmt.Sprintf("Slider value %.0f%%", v*100))
	}).SetTooltip("Drag to drive the progress bar")

	g.tabbar.OnTabSelect(func(index int) {
		g.applyTab(index)
	}).SetTooltip("TabBar — click a tab to switch demo pages")

	g.dropdown.SetTooltip("Dropdown — click to open")

	g.chat.OnLinkClick(func(link core.Link) {
		g.setStatus(fmt.Sprintf("Link clicked %s %q", link.Kind, link.Target))
	}).OnLinkTooltipRequested(func(link core.Link) string {
		switch link.Kind {
		case core.LinkItem:
			if link.Target == "iron-plate" {
				return "Iron plate — crafting component"
			}
			return "Item — " + link.Target
		case core.LinkPlayer:
			return link.Target + " — Level 60 Warrior"
		case core.LinkURL:
			return "Open " + link.Target + " in browser"
		default:
			return ""
		}
	}).SetTooltip("Chat log — hover a link")

	// Quest log demo: a TitledFrame window with non-functional quest buttons.
	// The X button shares the Close path; Accept/Decline only report status.
	questButton := widgets.NewButton("questButton", core.Rect{}, "Open Quest Log")
	questText := widgets.NewLabel("questText", core.Rect{}, "Thunderfury, Blessed Blade of the Windseeker — recover both bindings from Molten Core.")
	questAccept := widgets.NewButton("questAccept", core.Rect{}, "Accept")
	questDecline := widgets.NewButton("questDecline", core.Rect{}, "Decline")
	g.questLog = widgets.NewTitledFrame("questLog", core.Rect{}, "Quest Log")
	questButton.OnClick(func() {
		g.questLog.Show()
		g.facade.BringToFront("questLog")
		g.setStatus("Quest log opened (demo)")
	}).SetTooltip("Open the quest log window")
	questAccept.OnClick(func() {
		g.setStatus("Quest accepted (demo)")
	}).SetTooltip("Accept the quest (demo only)")
	questDecline.OnClick(func() {
		g.setStatus("Quest declined (demo)")
	}).SetTooltip("Decline the quest (demo only)")
	g.questLog.OnClose(func() {
		g.setStatus("Quest log closed (demo)")
	})

	// Self-contained scroll panel manages clipping and bounds:
	g.scroll.SetMaxScroll(core.Vec2{Y: 170}).SetScrollContentDrawer(g.drawScrollRows)

	if err := facade.Add(
		g.leftPanel, g.rightPanel,
		g.titleLabel, g.subtitleLabel,
		g.leftTitle, g.rightTitle,
		g.button, g.checkbox,
		g.textboxCaption, g.textbox,
		g.dropdownCaption, g.dropdown,
		g.sliderCaption, g.slider,
		g.progress,
		g.panel, g.panelText,
		g.label,
		g.tabbarCaption, g.tabbar,
		g.chatCaption, g.chat,
		g.frame, g.frameCaption, g.frameButton,
		g.scrollCaption, g.scroll,
		g.menuHint, g.tooltipHint,
		questButton,
		g.stateCanvas,
		g.statusLabel,
	); err != nil {
		panic(err)
	}
	if err := facade.Add(g.questLog.Widgets()...); err != nil {
		panic(err)
	}
	if err := facade.Add(questText, questAccept, questDecline); err != nil {
		panic(err)
	}

	// Scoped hotkeys replace direct IsKeyPressed polling. Text wins while
	// editing; otherwise the library fires these only while demoFrame holds
	// container focus, and reports consumption so a host game stays silent.
	facade.OnHotkey("demoFrameMove", 'R', ui.HotkeyOpts{Scope: "demoFrame", Consume: true}, func() {
		g.nudgeDemoFrame()
	})
	facade.OnHotkey("demoPinTooltip", 'T', ui.HotkeyOpts{Scope: "demoFrame", Consume: true}, func() {
		g.facade.ShowTooltip("Pinned tooltip — Esc dismisses it", g.facade.Pointer())
	})
	facade.OnContextMenu(g.showGalleryMenu)

	// WoW-style placement: one slot table owns both hierarchy and anchors.
	// Resolution is fixed in options; window resize only scales. A new
	// logical resolution needs a new table plus UI scale.
	logical := facade.Transform().Viewport.LogicalSize
	g.designWidth, g.designHeight = logical.X, logical.Y
	g.slots = gallerySlots(logical.X, logical.Y)
	g.tabPages = defaultTabPages()
	g.applyLayout()
	g.setupGameWidgets()
	g.applyLayout()
	if err := g.questLog.Layout(); err != nil {
		panic(err)
	}
	g.questLog.Hide()
	g.applyTab(g.tabbar.SelectedTab())
	return g
}

// defaultTabPages returns the index-aligned demo pages for demoTabs labels
// Widgets, Style, and About. Each page drives the right-panel heading,
// demo label, panel decoration, frame caption, and scroll-row prefix.
func defaultTabPages() []tabPage {
	return []tabPage{
		{
			heading:      "Widgets output",
			label:        "Widgets — live control values",
			panelText:    "Panel frame decoration",
			frameCaption: "Frame child moves with its parent (focus + R / sine)",
			rowPrefix:    "Widget row",
		},
		{
			heading:      "Style sampler",
			label:        "Style — state swatches below",
			panelText:    "Style panel — six button states",
			frameCaption: "Focused frame glows — click demoFrame",
			rowPrefix:    "Style row",
		},
		{
			heading:      "About gallery",
			label:        "About — rtgui texture demo",
			panelText:    "About this demo",
			frameCaption: "T pins a tooltip — Esc dismisses it",
			rowPrefix:    "About row",
		},
	}
}

// currentPage resolves the active demo page from the tab bar selection.
// An empty or out-of-range selection falls back to the first page so
// drawing never branches on missing content.
func (g *gallery) currentPage() tabPage {
	if g == nil || len(g.tabPages) == 0 {
		return tabPage{heading: "Containers", rowPrefix: "Clipped row"}
	}
	index := g.tabbar.SelectedTab()
	if index < 0 || index >= len(g.tabPages) {
		return g.tabPages[0]
	}
	return g.tabPages[index]
}

// setStatus updates the active status message and syncs the status label widget.
func (g *gallery) setStatus(msg string) {
	g.status = msg
	if g.statusLabel != nil {
		g.statusLabel.SetText(msg)
	}
}

// applyTab swaps app-side tab content for index and reports the selection.
// The TabBar owns selection state; this owns the page swap (label text
// plus status). Labels and widgets update their text directly.
func (g *gallery) applyTab(index int) {
	if g == nil || len(g.tabPages) == 0 {
		return
	}
	if index < 0 || index >= len(g.tabPages) {
		g.setStatus(fmt.Sprintf("Tab index %d selected", index))
		return
	}
	page := g.tabPages[index]
	g.selectGraphPage(index)
	g.label.SetText(page.label)
	if g.rightTitle != nil {
		g.rightTitle.SetText(page.heading)
	}
	if g.panelText != nil {
		g.panelText.SetText(page.panelText)
	}
	if g.frameCaption != nil {
		g.frameCaption.SetText(page.frameCaption)
	}
	if label, ok := g.tabbar.TabSelection(); ok {
		g.setStatus(fmt.Sprintf("Tab %q selected (index %d)", label, index))
		return
	}
	g.setStatus(fmt.Sprintf("Tab index %d selected", index))
}


// handleInput polls raylib input through the UI driver and animates the demo frame.
func (g *gallery) handleInput() {
	g.lastMouse = g.facade.Pointer()
	_ = g.facade.PollRaylibInput()
	g.animateFrame()
}

// showGalleryMenu opens the demo context menu at a logical point. The
// selection callback reports into the gallery status line.
func (g *gallery) showGalleryMenu(pos core.Vec2) {
	g.facade.ShowContextMenu([]ui.MenuItem{
		{ID: "inspect", Label: "Inspect widget"},
		{ID: "edit", Label: "Edit (locked)", Disabled: true},
		{Separator: true},
		{ID: "about", Label: "About gallery"},
	}, pos, func(id string) {
		g.setStatus(fmt.Sprintf("Menu selected %q", id))
	})
}

// nudgeDemoFrame authors one scoped MoveFrame step and performs the required
// arrangement. The library calls it only for in-scope R presses while no
// textbox holds keyboard focus, so typing r never arrives here.
func (g *gallery) nudgeDemoFrame() {
	delta := core.Vec2{X: 12, Y: 8}
	frameBounds := g.frame.Bounds()
	if frameBounds.X+delta.X+frameBounds.W > g.designWidth-20 {
		delta.X = -40
	}
	if frameBounds.Y+delta.Y+frameBounds.H > g.designHeight-20 {
		delta.Y = -30
	}
	// Shift the sine baseline with the manual move. animateFrame servos the
	// frame back toward frameOrigin every tick, so without this the nudge
	// is fully reverted on the very next frame.
	g.frameOrigin.X += delta.X
	g.frameOrigin.Y += delta.Y
	g.applyFrameMove(delta)
	childBounds := g.frameButton.Bounds()
	g.setStatus(fmt.Sprintf("MoveFrame %+v — child follows (%.0f,%.0f)", delta, childBounds.X, childBounds.Y))
}

// animateFrame drifts the demo frame on a sine wave. It runs every frame
// ahead of keys so a scoped R nudge owns the final word on manual-move
// frames; exact resting position does not matter, only that the child
// follows its parent.
func (g *gallery) animateFrame() {
	if g.frame == nil {
		return
	}
	t := float32(rl.GetTime())
	frameSize := g.frame.Bounds()
	target := core.Rect{
		X: g.frameOrigin.X + float32(math.Sin(float64(t*0.6)))*6,
		Y: g.frameOrigin.Y,
		W: frameSize.W,
		H: frameSize.H,
	}
	frameBounds := g.frame.Bounds()
	delta := core.Vec2{X: target.X - frameBounds.X, Y: target.Y - frameBounds.Y}
	if math.Abs(float64(delta.X)) <= 0.1 && math.Abs(float64(delta.Y)) <= 0.1 {
		return
	}
	g.applyFrameMove(delta)
}

// applyFrameMove authors movement and performs the required arrangement.
// The frame is nested under rightPanel, so arrange the ownership root.
func (g *gallery) applyFrameMove(delta core.Vec2) {
	layout.MoveFrame(g.frame.Frame(), delta)
	if err := layout.Arrange(g.rightPanel.Frame(), core.Rect{}); err != nil {
		panic(err)
	}
}

// draw renders the facade widgets. Everything is expressed in logical design
// coordinates; the matrix stretch maps it onto the live window, so resizing
// scales the UI instead of reflowing it.
func (g *gallery) draw() {
	rl.BeginDrawing()
	rl.ClearBackground(color.RGBA{R: 13, G: 17, B: 27, A: 255})

	sx, sy := g.facade.Scale()
	rl.PushMatrix()
	rl.Scalef(sx, sy, 1)

	g.facade.Draw()

	rl.PopMatrix()
	rl.EndDrawing()
}

// drawScrollRows renders the clipped demo rows inside the scroll panel.
func (g *gallery) drawScrollRows(bounds core.Rect, scroll core.Vec2) {
	prefix := g.currentPage().rowPrefix
	if prefix == "" {
		prefix = "Clipped row"
	}
	start := bounds.Y + 12 - scroll.Y
	theme := g.facade.Theme()
	trackPad := float32(0)
	if g.scroll.MaxScroll().Y > 0 {
		trackPad = 24
	}
	for i := 0; i < 10; i++ {
		y := start + float32(i*34)
		fill := color.RGBA{R: 35, G: 48, B: 70, A: 255}
		if i%2 == 1 {
			fill = color.RGBA{R: 29, G: 40, B: 59, A: 255}
		}
		rl.DrawRectangleRec(rl.Rectangle{X: bounds.X + 10, Y: y, Width: bounds.W - 20 - trackPad, Height: 28}, fill)
		theme.DrawText(fmt.Sprintf("%s %02d  •  scroll offset %.0f", prefix, i+1, scroll.Y), bounds.X+20, y+6, 18, false, color.RGBA{R: 194, G: 211, B: 235, A: 255})
	}
}

// drawStateSamples renders the six state swatches inside the state canvas.
func (g *gallery) drawStateSamples(bounds core.Rect) {
	theme := g.facade.Theme()
	names := []string{"normal", "focus", "hover", "press", "disabled", "selected"}
	count := float32(len(names))
	itemWidth := (bounds.W - 12) / count
	for i, state := range []core.WidgetState{core.StateNormal, core.StateFocused, core.StateHovered, core.StatePressed, core.StateDisabled, core.StateSelected} {
		itemBounds := core.Rect{X: bounds.X + float32(i)*itemWidth, Y: bounds.Y, W: itemWidth - 2, H: bounds.H}
		theme.DrawWidgetPart(core.WidgetButton, skin.PartBackground, itemBounds, state)
		theme.DrawText(names[i], itemBounds.X+5, itemBounds.Y+10, 14, false, color.RGBA{R: 228, G: 239, B: 255, A: 255})
	}
}
