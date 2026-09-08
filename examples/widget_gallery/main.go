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

type galleryLayout struct {
	leftPanel, rightPanel     core.Rect
	title, subtitle           core.Rect
	leftTitle, rightTitle     core.Rect
	button, checkbox          core.Rect
	textbox, textboxCaption   core.Rect
	dropdown, dropdownCaption core.Rect
	slider, sliderCaption     core.Rect
	progress                  core.Rect
	panel, panelText          core.Rect
	label                     core.Rect
	tabbar, tabbarCaption     core.Rect
	chat, chatCaption         core.Rect
	frame, frameCaption       core.Rect
	frameButton               core.Rect
	scroll, scrollCaption     core.Rect
	menuHint, tooltipHint     core.Rect
	stateCanvas               core.Rect
	status                    core.Rect
}

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
	stateCanvas     *widgets.Canvas
	statusLabel     *widgets.Label
	lineGraph       *widgets.LineGraph

	status       string
	designWidth  float32
	designHeight float32
	layout       galleryLayout
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
	loadGalleryFonts(facade.Theme())
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
				if err := saveScreenshot(*screenshot); err != nil {
					// Fallback to placeholder so headless smoke still produces a file
					log.Printf("screenshot via GPU failed (%v) — writing placeholder", err)
					if err2 := createPlaceholderScreenshot(*screenshot); err2 != nil {
						log.Printf("placeholder screenshot failed: %v", err2)
					}
				} else {
					log.Printf("screenshot saved to %s", *screenshot)
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
		// Create a placeholder image that documents headless mode.
		if err := createPlaceholderScreenshot(*screenshot); err != nil {
			log.Printf("headless placeholder failed: %v", err)
			os.Exit(1)
		}
		log.Printf("headless placeholder screenshot saved to %s", *screenshot)
	}
	if *frameLimit > 0 {
		log.Printf("headless smoke completed %d frames", *frameLimit)
	}
}

// saveScreenshot saves the current framebuffer. Unlike rl.TakeScreenshot,
// this handles absolute paths correctly (TakeScreenshot prepends
// GetWorkingDirectory and fails on /tmp/...).
func saveScreenshot(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
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
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	w, h := 800, 600
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	// Background
	bg := color.RGBA{R: 13, G: 17, B: 27, A: 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, bg)
		}
	}
	// Simple panel rectangles
	panel := color.RGBA{R: 35, G: 48, B: 70, A: 255}
	drawRect(img, 20, 60, w-40, h-80, panel)
	// Title bar placeholder
	title := color.RGBA{R: 45, G: 78, B: 112, A: 255}
	drawRect(img, 30, 80, w-60, 36, title)
	// State strip placeholders
	for i := 0; i < 6; i++ {
		c := color.RGBA{R: uint8(45 + i*20), G: 58, B: 82, A: 255}
		drawRect(img, 30+i*120, h-70, 100, 30, c)
	}
	// Status line text is not rasterized here; the image documents headless mode.
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// drawRect fills a bounded rectangle in a placeholder screenshot.
func drawRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	bounds := img.Bounds()
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			if xx >= bounds.Min.X && xx < bounds.Max.X && yy >= bounds.Min.Y && yy < bounds.Max.Y {
				img.Set(xx, yy, c)
			}
		}
	}
}

// loadGalleryFonts loads the checked-in Grenze TTFs into the theme.
// Widget text uses Grenze-Regular via drawTextInContent; captions and the
// status line use Grenze-Italic through the gallery draw helpers.
func loadGalleryFonts(theme *render.Theme) {
	regular := findFontFile("Grenze-Regular.ttf")
	if regular == "" {
		log.Fatalf("gallery font not found; expected testdata/fonts/Grenze-Regular.ttf (searched cwd and exe parents)")
	}
	if err := theme.LoadFont(regular); err != nil {
		log.Fatalf("could not load gallery font %q: %v", regular, err)
	}
	italic := findFontFile("Grenze-Italic.ttf")
	if italic == "" {
		log.Fatalf("gallery italic font not found; expected testdata/fonts/Grenze-Italic.ttf (searched cwd and exe parents)")
	}
	if err := theme.LoadItalicFont(italic); err != nil {
		log.Fatalf("could not load gallery italic font %q: %v", italic, err)
	}
}

// findFontFile locates a checked-in gallery font, mirroring texture lookup.
func findFontFile(name string) string {
	return firstExistingFile(fontCandidates(name))
}

// fontCandidates searches RTG_FONT_DIR, then cwd and exe parents.
func fontCandidates(name string) []string {
	candidates := []string{}
	if configured := os.Getenv("RTG_FONT_DIR"); configured != "" {
		candidates = append(candidates, filepath.Join(configured, name))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = appendParentFontCandidates(candidates, cwd, name)
	}
	if exe, err := os.Executable(); err == nil {
		candidates = appendParentFontCandidates(candidates, filepath.Dir(exe), name)
	}
	return candidates
}

// appendParentFontCandidates walks up to 5 parents for testdata/fonts/name.
func appendParentFontCandidates(candidates []string, root, name string) []string {
	for dir, depth := root, 0; dir != filepath.Dir(dir) && depth < 5; dir, depth = filepath.Dir(dir), depth+1 {
		if dir == "" {
			continue
		}
		candidates = append(candidates,
			filepath.Join(dir, "testdata", "fonts", name),
			filepath.Join(dir, "rtgui", "testdata", "fonts", name),
		)
	}
	return candidates
}

// findGalleryCSS locates testdata/skins/gallery.css, mirroring font lookup.
func findGalleryCSS() string {
	candidates := []string{}
	if cwd, err := os.Getwd(); err == nil {
		candidates = appendGalleryCSSCandidates(candidates, cwd)
	}
	if exe, err := os.Executable(); err == nil {
		candidates = appendGalleryCSSCandidates(candidates, filepath.Dir(exe))
	}
	return firstExistingFile(candidates)
}

// appendGalleryCSSCandidates walks up to 5 parents for the gallery skin file.
func appendGalleryCSSCandidates(candidates []string, root string) []string {
	for dir, depth := root, 0; dir != filepath.Dir(dir) && depth < 5; dir, depth = filepath.Dir(dir), depth+1 {
		if dir == "" {
			continue
		}
		candidates = append(candidates,
			filepath.Join(dir, "testdata", "skins", "gallery.css"),
			filepath.Join(dir, "rtgui", "testdata", "skins", "gallery.css"),
		)
	}
	return candidates
}

// firstExistingFile returns the first regular file in candidates.
func firstExistingFile(candidates []string) string {
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
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
	g.button.OnClick(func() {
		g.setStatus("Primary button clicked")
	}).SetTooltip("Primary action — fires OnClick")

	g.checkbox.OnClick(func() {
		g.button.SetEnabled(g.checkbox.Checked())
		g.setStatus(fmt.Sprintf("Checkbox is %v; primary button enabled=%v", g.checkbox.Checked(), g.button.Enabled()))
	})

	g.frameButton.OnClick(func() {
		g.setStatus("Frame child button clicked")
	})

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
		g.stateCanvas,
		g.statusLabel,
	); err != nil {
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

	// The visual widgets own the layout nodes used by the MoveFrame demo.
	if err := g.frame.Frame().AddChild(g.frameCaption.Frame()); err != nil {
		panic(err)
	}
	if err := g.frameCaption.SetPoint(layout.AnchorTopLeft, nil, layout.AnchorTopLeft, core.Vec2{X: 18, Y: 20}); err != nil {
		panic(err)
	}
	if err := g.frame.Frame().AddChild(g.frameButton.Frame()); err != nil {
		panic(err)
	}
	if err := g.frameButton.SetPoint(layout.AnchorTopLeft, nil, layout.AnchorTopLeft, core.Vec2{X: 24, Y: 74}); err != nil {
		panic(err)
	}
	// Layout is computed once from the fixed design resolution; later window
	// resizes rescale around these bounds instead of reflowing them.
	logical := facade.Transform().Viewport.LogicalSize
	g.designWidth, g.designHeight = logical.X, logical.Y
	g.layout = calculateLayout(logical.X, logical.Y)
	g.tabPages = defaultTabPages()
	g.applyLayout()
	g.setupGameWidgets()
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

// applyLayout assigns cached design-resolution bounds and arranges the frame
// widget's owned child. Bounds never follow the live window size after this.
func (g *gallery) applyLayout() {
	g.leftPanel.SetBounds(g.layout.leftPanel)
	g.rightPanel.SetBounds(g.layout.rightPanel)
	g.titleLabel.SetBounds(g.layout.title)
	g.subtitleLabel.SetBounds(g.layout.subtitle)
	g.leftTitle.SetBounds(g.layout.leftTitle)
	g.rightTitle.SetBounds(g.layout.rightTitle)
	g.button.SetBounds(g.layout.button)
	g.checkbox.SetBounds(g.layout.checkbox)
	g.textboxCaption.SetBounds(g.layout.textboxCaption)
	g.textbox.SetBounds(g.layout.textbox)
	g.dropdownCaption.SetBounds(g.layout.dropdownCaption)
	g.dropdown.SetBounds(g.layout.dropdown)
	g.sliderCaption.SetBounds(g.layout.sliderCaption)
	g.slider.SetBounds(g.layout.slider)
	g.progress.SetBounds(g.layout.progress)
	g.panel.SetBounds(g.layout.panel)
	g.panelText.SetBounds(g.layout.panelText)
	g.label.SetBounds(g.layout.label)
	g.tabbarCaption.SetBounds(g.layout.tabbarCaption)
	g.tabbar.SetBounds(g.layout.tabbar)
	g.chatCaption.SetBounds(g.layout.chatCaption)
	g.chat.SetBounds(g.layout.chat)
	g.frame.SetBounds(g.layout.frame)
	g.frameCaption.SetBounds(core.Rect{W: g.layout.frameCaption.W, H: g.layout.frameCaption.H})
	g.frameButton.SetBounds(core.Rect{W: g.layout.frameButton.W, H: g.layout.frameButton.H})
	g.scrollCaption.SetBounds(g.layout.scrollCaption)
	g.scroll.SetBounds(g.layout.scroll)
	g.menuHint.SetBounds(g.layout.menuHint)
	g.tooltipHint.SetBounds(g.layout.tooltipHint)
	g.stateCanvas.SetBounds(g.layout.stateCanvas)
	g.statusLabel.SetBounds(g.layout.status)
	if err := layout.Arrange(g.frame.Frame(), core.Rect{}); err != nil {
		panic(err)
	}
}

// calculateLayout returns fixed logical design bounds for the gallery widgets.
func calculateLayout(width, height float32) galleryLayout {
	margin, gap := float32(28), float32(24)
	panelWidth := (width - 2*margin - gap) / 2
	left := core.Rect{X: margin, Y: 74, W: panelWidth, H: height - 98}
	right := core.Rect{X: margin + panelWidth + gap, Y: 74, W: panelWidth, H: height - 98}
	widgetX, widgetW := left.X+24, left.W-48
	rightX, rightW := right.X+24, right.W-48
	return galleryLayout{
		leftPanel:       left,
		rightPanel:      right,
		title:           core.Rect{X: 28, Y: 24, W: 600, H: 26},
		subtitle:        core.Rect{X: 30, Y: 51, W: 1200, H: 20},
		leftTitle:       core.Rect{X: left.X + 24, Y: left.Y + 15, W: left.W - 48, H: 24},
		rightTitle:      core.Rect{X: right.X + 24, Y: right.Y + 15, W: right.W - 48, H: 24},
		button:          core.Rect{X: widgetX, Y: left.Y + 48, W: widgetW, H: 42},
		checkbox:        core.Rect{X: widgetX, Y: left.Y + 112, W: widgetW, H: 38},
		textboxCaption:  core.Rect{X: widgetX, Y: left.Y + 176 - 23, W: widgetW, H: 20},
		textbox:         core.Rect{X: widgetX, Y: left.Y + 176, W: widgetW, H: 42},
		dropdownCaption: core.Rect{X: widgetX, Y: left.Y + 240 - 23, W: widgetW, H: 20},
		dropdown:        core.Rect{X: widgetX, Y: left.Y + 240, W: widgetW, H: 42},
		sliderCaption:   core.Rect{X: widgetX, Y: left.Y + 304 - 23, W: widgetW, H: 20},
		slider:          core.Rect{X: widgetX, Y: left.Y + 304, W: widgetW, H: 42},
		progress:        core.Rect{X: widgetX, Y: left.Y + 368, W: widgetW, H: 42},
		panel:           core.Rect{X: widgetX, Y: left.Y + 440, W: widgetW, H: 70},
		panelText:       core.Rect{X: widgetX + 14, Y: left.Y + 440 + 30, W: widgetW - 28, H: 24},
		tabbarCaption:   core.Rect{X: widgetX, Y: left.Y + 536 - 23, W: widgetW, H: 20},
		tabbar:          core.Rect{X: widgetX, Y: left.Y + 536, W: widgetW, H: 36},
		chatCaption:     core.Rect{X: widgetX, Y: left.Y + 600 - 23, W: widgetW, H: 20},
		chat:            core.Rect{X: widgetX, Y: left.Y + 600, W: widgetW, H: 68},
		label:           core.Rect{X: rightX, Y: right.Y + 40, W: rightW, H: 32},
		frame:           core.Rect{X: rightX, Y: right.Y + 92, W: rightW, H: 164},
		frameCaption:    core.Rect{W: rightW - 36, H: 22},
		frameButton:     core.Rect{W: rightW - 96, H: 42},
		scrollCaption:   core.Rect{X: rightX + 12, Y: right.Y + 282 - 22, W: rightW - 24, H: 20},
		scroll:          core.Rect{X: rightX, Y: right.Y + 282, W: rightW, H: 232},
		menuHint:        core.Rect{X: rightX + 12, Y: right.Y + 282 + 232 + 10, W: rightW - 24, H: 20},
		tooltipHint:     core.Rect{X: rightX + 12, Y: right.Y + 282 + 232 + 28, W: rightW - 24, H: 20},
		stateCanvas:     core.Rect{X: rightX, Y: right.Y + right.H - 66, W: rightW, H: 34},
		status:          core.Rect{X: 30, Y: height - 18, W: width - 60, H: 20},
	}
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
	// frame back toward g.layout.frame every tick, so without this the nudge
	// is fully reverted on the very next frame.
	g.layout.frame.X += delta.X
	g.layout.frame.Y += delta.Y
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
	target := core.Rect{
		X: g.layout.frame.X + float32(math.Sin(float64(t*0.6)))*6,
		Y: g.layout.frame.Y,
		W: g.layout.frame.W,
		H: g.layout.frame.H,
	}
	frameBounds := g.frame.Bounds()
	delta := core.Vec2{X: target.X - frameBounds.X, Y: target.Y - frameBounds.Y}
	if math.Abs(float64(delta.X)) <= 0.1 && math.Abs(float64(delta.Y)) <= 0.1 {
		return
	}
	g.applyFrameMove(delta)
}

// applyFrameMove authors movement and performs the required arrangement.
func (g *gallery) applyFrameMove(delta core.Vec2) {
	layout.MoveFrame(g.frame.Frame(), delta)
	if err := layout.Arrange(g.frame.Frame(), core.Rect{}); err != nil {
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

// clamp confines value to the inclusive low/high interval.
func clamp(value, low, high float32) float32 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
