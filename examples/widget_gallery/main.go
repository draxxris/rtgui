// widget_gallery is the runnable demo proving the textured rtgui library end-to-end.
//
// It uses stock Go + github.com/gen2brain/raylib-go/raylib plus the github.com/draxxris/rtgui/ui
// facade. The UI owns transform, theme, registry, focus, and
// callbacks; the gallery only polls raylib, forwards to the facade, and draws
// app-specific layers (scroll contents and state samples).
// There is no package-global UI state.
//
// Gallery typography uses the Grenze family (SIL OFL, see
// testdata/fonts/Grenze-OFL.txt): Grenze-Light for titles, values, and rows,
// Grenze-LightItalic for captions and the status line. Widget text rendered
// through render.Theme uses Grenze-Light once the theme font loads.
//
// Gallery contract: two panels, button, checkbox (toggles button enabled),
// textbox (typing + backspace including multi-byte UTF-8),
// dropdown popup, slider driving progress, scroll panel with wheel + scissor,
// frame with relative-move child (layout.MoveFrame), state-sample strip,
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
	leftPanel, rightPanel core.Rect
	button, checkbox      core.Rect
	textbox, dropdown     core.Rect
	slider, progress      core.Rect
	panel, label          core.Rect
	frame, frameButton    core.Rect
	scroll                core.Rect
}

type gallery struct {
	facade *ui.UI

	leftPanel   *widgets.Widget
	rightPanel  *widgets.Widget
	button      *widgets.Widget
	checkbox    *widgets.Widget
	textbox     *widgets.Widget
	dropdown    *widgets.Widget
	slider      *widgets.Widget
	progress    *widgets.Widget
	panel       *widgets.Widget
	label       *widgets.Widget
	frame       *widgets.Widget
	frameButton *widgets.Widget
	scroll      *widgets.Widget

	status       string
	designWidth  float32
	designHeight float32
	layout       galleryLayout
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
	if err := facade.Add(button, checkbox, slider, field); err != nil {
		log.Fatal(err)
	}
	clicks := 0
	facade.OnClick("smokeButton", func() { clicks++ })
	center := core.Vec2{X: 60, Y: 30}
	facade.HandleMouse(ui.MouseEvent{Pos: center, Pressed: true})
	facade.HandleMouse(ui.MouseEvent{Pos: center, Released: true})
	facade.HandleKey(ui.KeyEvent{Chars: []rune("hi")})
	facade.Draw()
	calls := recorder.Calls()
	log.Printf("headless smoke: %d draw calls logged (clicks=%d fallback=%v)", len(calls), clicks, len(calls) > 0 && calls[0].Fallback)

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
// Widget text uses Grenze-Light via drawTextInContent; captions and the
// status line use Grenze-LightItalic through the gallery draw helpers.
func loadGalleryFonts(theme *render.Theme) {
	regular := findFontFile("Grenze-Light.ttf")
	if regular == "" {
		log.Fatalf("gallery font not found; expected testdata/fonts/Grenze-Light.ttf (searched cwd and exe parents)")
	}
	if err := theme.LoadFont(regular); err != nil {
		log.Fatalf("could not load gallery font %q: %v", regular, err)
	}
	italic := findFontFile("Grenze-LightItalic.ttf")
	if italic == "" {
		log.Fatalf("gallery italic font not found; expected testdata/fonts/Grenze-LightItalic.ttf (searched cwd and exe parents)")
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
	g := &gallery{
		facade:      facade,
		leftPanel:   widgets.NewFrame("leftPanel", core.Rect{}),
		rightPanel:  widgets.NewFrame("rightPanel", core.Rect{}),
		button:      widgets.NewButton("primaryButton", core.Rect{}, "Primary button"),
		checkbox:    widgets.NewCheckbox("enableCheckbox", core.Rect{}, true),
		textbox:     widgets.NewTextbox("inputTextbox", core.Rect{}, 128),
		dropdown:    widgets.NewDropdown("classDropdown", core.Rect{}, []string{"Warrior", "Ranger", "Mage"}, 0),
		slider:      widgets.NewSlider("valueSlider", core.Rect{}, 0.35),
		progress:    widgets.NewProgressBar("valueProgress", core.Rect{}, 0.35),
		panel:       widgets.NewFrame("demoPanel", core.Rect{}),
		label:       widgets.NewLabel("demoLabel", core.Rect{}, "Textured label"),
		frame:       widgets.NewFrame("demoFrame", core.Rect{}),
		frameButton: widgets.NewButton("frameChildButton", core.Rect{}, "Frame child"),
		scroll:      widgets.NewScrollPanel("scrollPanel", core.Rect{}),
		status:      "Click a widget to interact with it — press R to MoveFrame",
	}
	g.textbox.SetText("Type here")
	if err := facade.Add(g.leftPanel, g.rightPanel, g.button, g.checkbox, g.textbox, g.dropdown, g.slider, g.progress, g.panel, g.label, g.frame, g.frameButton, g.scroll); err != nil {
		panic(err)
	}
	facade.OnClick("primaryButton", func() {
		g.status = "Primary button clicked"
	})
	facade.OnClick("enableCheckbox", func() {
		g.button.SetEnabled(g.checkbox.Checked())
		g.status = fmt.Sprintf("Checkbox is %v; primary button enabled=%v", g.checkbox.Checked(), g.button.Enabled())
	})
	facade.OnClick("frameChildButton", func() {
		g.status = "Frame child button clicked"
	})
	facade.OnChange("valueSlider", func(v float32) {
		g.progress.SetValue(v)
		g.status = fmt.Sprintf("Slider value %.0f%%", v*100)
	})
	// The visual widgets own the layout nodes used by the MoveFrame demo.
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
	g.applyLayout()
	return g
}

// applyLayout assigns cached design-resolution bounds and arranges the frame
// widget's owned child. Bounds never follow the live window size after this.
func (g *gallery) applyLayout() {
	g.leftPanel.SetBounds(g.layout.leftPanel)
	g.rightPanel.SetBounds(g.layout.rightPanel)
	g.button.SetBounds(g.layout.button)
	g.checkbox.SetBounds(g.layout.checkbox)
	g.textbox.SetBounds(g.layout.textbox)
	g.dropdown.SetBounds(g.layout.dropdown)
	g.slider.SetBounds(g.layout.slider)
	g.progress.SetBounds(g.layout.progress)
	g.panel.SetBounds(g.layout.panel)
	g.label.SetBounds(g.layout.label)
	g.frame.SetBounds(g.layout.frame)
	g.frameButton.SetBounds(core.Rect{W: g.layout.frameButton.W, H: g.layout.frameButton.H})
	g.scroll.SetBounds(g.layout.scroll)
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
	return galleryLayout{
		leftPanel: left, rightPanel: right,
		button:      core.Rect{X: widgetX, Y: left.Y + 48, W: widgetW, H: 42},
		checkbox:    core.Rect{X: widgetX, Y: left.Y + 112, W: widgetW, H: 38},
		textbox:     core.Rect{X: widgetX, Y: left.Y + 176, W: widgetW, H: 42},
		dropdown:    core.Rect{X: widgetX, Y: left.Y + 240, W: widgetW, H: 42},
		slider:      core.Rect{X: widgetX, Y: left.Y + 304, W: widgetW, H: 42},
		progress:    core.Rect{X: widgetX, Y: left.Y + 368, W: widgetW, H: 42},
		panel:       core.Rect{X: widgetX, Y: left.Y + 440, W: widgetW, H: 94},
		label:       core.Rect{X: right.X + 24, Y: right.Y + 40, W: right.W - 48, H: 32},
		frame:       core.Rect{X: right.X + 24, Y: right.Y + 92, W: right.W - 48, H: 164},
		frameButton: core.Rect{X: right.X + 48, Y: right.Y + 166, W: right.W - 96, H: 42},
		scroll:      core.Rect{X: right.X + 24, Y: right.Y + 282, W: right.W - 48, H: 232},
	}
}

// handleInput polls raylib once per frame and forwards to the facade. The UI
// owns dropdown popup input; frame movement and animation remain gallery work.
func (g *gallery) handleInput() {
	physical := rl.GetMousePosition()
	mouse := g.facade.ToLogical(core.Vec2{X: physical.X, Y: physical.Y})
	pressedEdge := rl.IsMouseButtonPressed(rl.MouseButtonLeft)
	mouseHandled := g.facade.HandleMouse(ui.MouseEvent{
		Pos:      mouse,
		Pressed:  pressedEdge,
		Down:     rl.IsMouseButtonDown(rl.MouseButtonLeft),
		Released: rl.IsMouseButtonReleased(rl.MouseButtonLeft),
		Wheel:    rl.GetMouseWheelMove(),
	})
	_ = mouseHandled
	// The facade scrolls unclamped; the gallery bounds its demo list.
	scroll := g.scroll.Scroll()
	scroll.Y = clamp(scroll.Y, 0, 170)
	g.scroll.SetScroll(scroll)
	g.handleKeys()
	g.handleFrameMove()
	g.animateFrame()
}

// handleKeys forwards chars, backspace, and escape to the facade.
func (g *gallery) handleKeys() {
	escape := rl.IsKeyPressed(rl.KeyEscape)
	chars := drainGalleryChars()
	backspace := rl.IsKeyPressed(rl.KeyBackspace) || rl.IsKeyPressedRepeat(rl.KeyBackspace)
	_ = g.facade.HandleKey(ui.KeyEvent{Chars: chars, Backspace: backspace, Escape: escape})
}

// drainGalleryChars collects pending raylib runes for this frame.
func drainGalleryChars() []rune {
	var out []rune
	for codepoint := rl.GetCharPressed(); codepoint > 0; codepoint = rl.GetCharPressed() {
		if codepoint >= 32 && codepoint != 127 {
			out = append(out, rune(codepoint))
		}
	}
	return out
}

// handleFrameMove nudges the demo frame on R.
func (g *gallery) handleFrameMove() {
	if !rl.IsKeyPressed(rl.KeyR) {
		return
	}
	delta := core.Vec2{X: 12, Y: 8}
	frameBounds := g.frame.Bounds()
	if frameBounds.X+delta.X+frameBounds.W > g.designWidth-20 {
		delta.X = -40
	}
	if frameBounds.Y+delta.Y+frameBounds.H > g.designHeight-20 {
		delta.Y = -30
	}
	g.applyFrameMove(delta)
	childBounds := g.frameButton.Bounds()
	g.status = fmt.Sprintf("MoveFrame %+v — child follows (%.0f,%.0f)", delta, childBounds.X, childBounds.Y)
}

// animateFrame drifts the demo frame on a sine wave.
func (g *gallery) animateFrame() {
	if g.frame == nil || rl.IsKeyDown(rl.KeyR) {
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

// draw renders the facade widgets plus app-specific layers. Everything is
// expressed in logical design coordinates; the matrix stretch maps it onto
// the live window, so resizing scales the UI instead of reflowing it.
func (g *gallery) draw() {
	rl.BeginDrawing()
	rl.ClearBackground(color.RGBA{R: 13, G: 17, B: 27, A: 255})

	sx, sy := g.facade.Scale()
	rl.PushMatrix()
	rl.Scalef(sx, sy, 1)

	g.drawText("RTG textured widget gallery", 28, 24, 26, color.RGBA{R: 226, G: 239, B: 255, A: 255})
	g.drawItalic("Every pixel needs a CSS texture; missing skins stay invisible and states inherit their base rule. Press R to MoveFrame.", 30, 51, 14, color.RGBA{R: 153, G: 174, B: 202, A: 255})
	g.panelTitle(g.layout.leftPanel, "Widgets")
	g.panelTitle(g.layout.rightPanel, "Containers, clipping, and states")

	g.facade.Draw()

	checkboxBounds := g.checkbox.Bounds()
	textboxBounds := g.textbox.Bounds()
	dropdownBounds := g.dropdown.Bounds()
	sliderBounds := g.slider.Bounds()
	progressBounds := g.progress.Bounds()
	panelBounds := g.panel.Bounds()
	frameBounds := g.frame.Bounds()
	g.drawText("Enable primary button", int32(checkboxBounds.X+40), int32(checkboxBounds.Y+8), 18, color.RGBA{R: 205, G: 218, B: 238, A: 255})
	g.drawItalic("Textbox (click, type, backspace — UTF-8)", int32(textboxBounds.X), int32(textboxBounds.Y-23), 15, color.RGBA{R: 153, G: 174, B: 202, A: 255})
	g.drawItalic("Dropdown (click to open)", int32(dropdownBounds.X), int32(dropdownBounds.Y-23), 15, color.RGBA{R: 153, G: 174, B: 202, A: 255})
	g.drawItalic("Slider drives the progress bar", int32(sliderBounds.X), int32(sliderBounds.Y-23), 15, color.RGBA{R: 153, G: 174, B: 202, A: 255})
	g.drawText(fmt.Sprintf("%.0f%%", g.slider.Value()*100), int32(sliderBounds.X+sliderBounds.W-48), int32(sliderBounds.Y+13), 16, color.RGBA{R: 230, G: 242, B: 255, A: 255})
	g.drawText(fmt.Sprintf("Progress: %.0f%%", g.progress.Value()*100), int32(progressBounds.X+12), int32(progressBounds.Y+13), 16, color.RGBA{R: 235, G: 255, B: 240, A: 255})
	g.drawText("Panel frame decoration", int32(panelBounds.X+14), int32(panelBounds.Y+38), 17, color.RGBA{R: 218, G: 230, B: 248, A: 255})
	g.drawText("Frame child moves with its parent (R / sine)", int32(frameBounds.X+18), int32(frameBounds.Y+20), 15, color.RGBA{R: 153, G: 174, B: 202, A: 255})

	g.drawScrollContents()
	g.drawStateSamples()
	g.drawItalic(g.status, 30, int32(g.designHeight-18), 15, color.RGBA{R: 161, G: 192, B: 224, A: 255})

	rl.PopMatrix()
	rl.EndDrawing()
}

// drawText renders gallery chrome with the Grenze-Light theme font.
func (g *gallery) drawText(value string, x, y int32, size float32, tint color.RGBA) {
	theme := g.facade.Theme()
	if theme.HasFont() {
		rl.DrawTextEx(theme.FontForSize(size), value, rl.NewVector2(float32(x), float32(y)), size, size/10, tint)
		return
	}
	rl.DrawText(value, x, y, int32(size), tint)
}

// drawItalic renders captions and status with the Grenze-LightItalic font.
func (g *gallery) drawItalic(value string, x, y int32, size float32, tint color.RGBA) {
	theme := g.facade.Theme()
	if theme.HasItalicFont() {
		rl.DrawTextEx(theme.ItalicForSize(size), value, rl.NewVector2(float32(x), float32(y)), size, size/10, tint)
		return
	}
	rl.DrawText(value, x, y, int32(size), tint)
}

// panelTitle renders a panel heading with the Grenze-Light theme font.
func (g *gallery) panelTitle(bounds core.Rect, title string) {
	g.drawText(title, int32(bounds.X+24), int32(bounds.Y+15), 20, color.RGBA{R: 224, G: 235, B: 252, A: 255})
}

// drawScrollContents renders the clipped demo rows. Scissor stays app-side,
// and takes physical pixels, so the logical panel bounds are scaled by hand
// (the matrix stretch does not apply to the GPU scissor test).
func (g *gallery) drawScrollContents() {
	// Scissor is left to the app (draw.go never calls BeginScissorMode)
	sx, sy := g.facade.Scale()
	bounds := g.scroll.Bounds()
	scroll := g.scroll.Scroll()
	rl.BeginScissorMode(int32(bounds.X*sx), int32(bounds.Y*sy), int32(bounds.W*sx), int32(bounds.H*sy))
	start := bounds.Y + 12 - scroll.Y
	for i := 0; i < 10; i++ {
		y := start + float32(i*34)
		fill := color.RGBA{R: 35, G: 48, B: 70, A: 255}
		if i%2 == 1 {
			fill = color.RGBA{R: 29, G: 40, B: 59, A: 255}
		}
		rl.DrawRectangleRec(rl.Rectangle{X: bounds.X + 10, Y: y, Width: bounds.W - 20, Height: 28}, fill)
		g.drawText(fmt.Sprintf("Clipped row %02d  •  scroll offset %.0f", i+1, scroll.Y), int32(bounds.X+20), int32(y+6), 14, color.RGBA{R: 194, G: 211, B: 235, A: 255})
	}
	rl.EndScissorMode()
	g.drawItalic("Scroll panel — wheel over this area", int32(bounds.X+12), int32(bounds.Y-22), 15, color.RGBA{R: 153, G: 174, B: 202, A: 255})
}

// drawStateSamples renders the six state swatches.
func (g *gallery) drawStateSamples() {
	theme := g.facade.Theme()
	names := []string{"normal", "focus", "hover", "press", "disabled", "selected"}
	panelBounds := g.rightPanel.Bounds()
	startX := panelBounds.X + 20
	y := panelBounds.Y + panelBounds.H - 66
	for i, state := range []core.WidgetState{core.StateNormal, core.StateFocused, core.StateHovered, core.StatePressed, core.StateDisabled, core.StateSelected} {
		bounds := core.Rect{X: startX + float32(i)*((panelBounds.W-40)/6), Y: y, W: (panelBounds.W - 52) / 6, H: 34}
		theme.DrawWidgetPart(core.WidgetButton, skin.PartBackground, bounds, state)
		g.drawText(names[i], int32(bounds.X+5), int32(bounds.Y+10), 11, color.RGBA{R: 228, G: 239, B: 255, A: 255})
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
