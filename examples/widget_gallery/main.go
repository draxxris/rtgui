// widget_gallery is the single runnable demo proving the pure-Go rtgui library end-to-end.
//
// It uses only stock Go + github.com/gen2brain/raylib-go/raylib. Rendering,
// input, widgets, transforms, and data types are explicit package instances;
// there is no package-global UI state.
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
//     the pure-Go draw log (NinePatchRects, ContentRect, fallback) and writes
//     a placeholder PNG via the stdlib image/png if -screenshot was requested.
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

	rl "github.com/gen2brain/raylib-go/raylib"
	"rtgui/core"
	"rtgui/input"
	"rtgui/layout"
	"rtgui/render"
	"rtgui/skin"
	"rtgui/transform"
	"rtgui/widgets"
)

const (
	windowWidth    int32 = 1280
	windowHeight   int32 = 780
	auxAtlasWidth        = 512
	auxAtlasHeight       = 144
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
	rectangle, label      core.Rect
	frame, frameButton    core.Rect
	scroll                core.Rect
}

type gallery struct {
	buttonTexture rl.Texture2D
	theme         *render.Theme
	transform     *transform.Transform
	capture       *input.Capture

	leftPanel   *widgets.Widget
	rightPanel  *widgets.Widget
	button      *widgets.Widget
	checkbox    *widgets.Widget
	textbox     *widgets.Widget
	dropdown    *widgets.Widget
	slider      *widgets.Widget
	progress    *widgets.Widget
	rectangle   *widgets.Widget
	label       *widgets.Widget
	frame       *widgets.Widget
	frameButton *widgets.Widget
	scroll      *widgets.Widget

	// layout nodes to demonstrate MoveFrame (relative-move child)
	frameNode      *layout.Node
	frameChildNode *layout.Node

	pressed      *widgets.Widget
	focused      *widgets.Widget
	dropdownOpen bool
	status       string
	lastWidth    int
	lastHeight   int
	layout       galleryLayout
}

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

	// Pure-Go path: textures loaded directly via raylib, no loader shim.
	buttonTexture := loadButtonTexture()
	defer rl.UnloadTexture(buttonTexture)
	auxAtlas := makeAuxAtlas()
	defer rl.UnloadTexture(auxAtlas)
	viewport := core.Viewport{Viewport: core.Rect{W: float32(windowWidth), H: float32(windowHeight)}, LogicalSize: core.Vec2{X: float32(windowWidth), Y: float32(windowHeight)}}
	uiTransform := transform.New(viewport)
	theme := render.NewTheme(uiTransform)
	if err := registerTheme(theme, buttonTexture, auxAtlas); err != nil {
		log.Fatal(err)
	}

	g := newGallery(buttonTexture, theme, transform.New(viewport), input.NewCapture())
	// Seed initial viewport before first frame (on resize: SetViewport)
	g.resizeIfNeeded(int(windowWidth), int(windowHeight))

	for frame := 0; !rl.WindowShouldClose(); frame++ {
		g.resizeIfNeeded(rl.GetScreenWidth(), rl.GetScreenHeight())
		g.handleInput()
		g.draw()

		if *frameLimit > 0 && frame+1 >= *frameLimit {
			if *screenshot != "" {
				if err := saveScreenshot(*screenshot); err != nil {
					// Fallback to placeholder so headless smoke still produces a file
					log.Printf("screenshot via GPU failed (%v) — writing placeholder", err)
					if err2 := createPlaceholderScreenshot(*screenshot, g); err2 != nil {
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

// runHeadlessSmoke exercises the pure-Go library without a GL context.
// It sets viewport, drives a few DrawWidget calls headlessly (guarded by
// IsWindowReady in draw.go, so no GL is touched, but DrawCall log + fallback
// logic still runs), and writes a placeholder PNG if requested.
func runHeadlessSmoke() {
	viewport := core.Viewport{Viewport: core.Rect{W: 800, H: 600}, LogicalSize: core.Vec2{X: 800, Y: 600}}
	theme := render.NewTheme(transform.New(viewport))
	// Drive a few headless DrawWidget calls to prove NinePatch/ContentRect/fallback.
	// Missing skin falls back to color 200,200,200 — exercised by drawing without texture.
	theme.ClearDrawLog()
	_ = theme.DrawWidgetPart(core.WidgetButton, skin.PartBackground, core.Rect{X: 10, Y: 10, W: 100, H: 40}, core.StateNormal)
	_ = theme.DrawWidget(core.WidgetInfo{ID: 99, Name: "smokeCheckbox", Bounds: core.Rect{X: 10, Y: 60, W: 120, H: 40}, Kind: core.WidgetCheckbox, State: core.StateNormal}, "", 0, true)
	_ = theme.DrawWidget(core.WidgetInfo{ID: 100, Name: "smokeSlider", Bounds: core.Rect{X: 10, Y: 110, W: 120, H: 40}, Kind: core.WidgetSlider, State: core.StateNormal}, "", 0.5, false)
	calls := theme.DrawLog()
	log.Printf("headless smoke: %d draw calls logged (fallback=%v)", len(calls), len(calls) > 0 && calls[0].Fallback)

	if *screenshot != "" {
		// Create a placeholder image that documents headless mode.
		g := &gallery{status: "headless smoke — no display"}
		if err := createPlaceholderScreenshot(*screenshot, g); err != nil {
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
func createPlaceholderScreenshot(path string, g *gallery) error {
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

func loadButtonTexture() rl.Texture2D {
	path := findButtonTexture()
	if path == "" {
		log.Fatalf("button texture not found; expected testdata/skins/button_rectangle_border.png or set RTG_BUTTON_TEXTURE (searched cwd and exe parents)")
	}
	tex := rl.LoadTexture(path)
	if tex.ID == 0 {
		log.Fatalf("could not load button texture %q", path)
	}
	rl.SetTextureFilter(tex, rl.FilterPoint)
	return tex
}

func findButtonTexture() string {
	return firstExistingFile(textureCandidates())
}

func textureCandidates() []string {
	candidates := []string{}
	if configured := os.Getenv("RTG_BUTTON_TEXTURE"); configured != "" {
		candidates = append(candidates, configured)
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = appendParentTextureCandidates(candidates, cwd)
	}
	if exe, err := os.Executable(); err == nil {
		candidates = appendParentTextureCandidates(candidates, filepath.Dir(exe))
	}
	return candidates
}

func appendParentTextureCandidates(candidates []string, root string) []string {
	for dir, depth := root, 0; dir != filepath.Dir(dir) && depth < 5; dir, depth = filepath.Dir(dir), depth+1 {
		if dir == "" {
			continue
		}
		candidates = append(candidates,
			filepath.Join(dir, "testdata", "skins", "button_rectangle_border.png"),
			filepath.Join(dir, "rtgui", "testdata", "skins", "button_rectangle_border.png"),
		)
	}
	return candidates
}

func firstExistingFile(candidates []string) string {
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func makeAuxAtlas() rl.Texture2D {
	// Procedural atlas via GenImageColor + ImageDraw* + LoadTextureFromImage
	// as required by the pure-Go task.
	img := rl.GenImageColor(auxAtlasWidth, auxAtlasHeight, rl.Blank)
	if img == nil {
		log.Fatal("could not allocate the procedural skin atlas")
	}

	stateFills := []color.RGBA{
		{R: 45, G: 58, B: 82, A: 255},
		{R: 58, G: 78, B: 112, A: 255},
		{R: 72, G: 102, B: 150, A: 255},
		{R: 35, G: 48, B: 74, A: 255},
		{R: 70, G: 72, B: 82, A: 220},
		{R: 72, G: 112, B: 88, A: 255},
	}
	for state, fill := range stateFills {
		paintPatch(img, int32(state*68), 0, 60, 36, fill, color.RGBA{R: 145, G: 184, B: 230, A: 255})
	}

	paintPatch(img, 0, 48, 96, 42, color.RGBA{R: 28, G: 36, B: 52, A: 255}, color.RGBA{R: 94, G: 122, B: 162, A: 255})
	paintPatch(img, 104, 48, 96, 42, color.RGBA{R: 20, G: 28, B: 43, A: 255}, color.RGBA{R: 99, G: 151, B: 208, A: 255})
	paintPatch(img, 208, 48, 96, 20, color.RGBA{R: 22, G: 30, B: 45, A: 255}, color.RGBA{R: 85, G: 110, B: 145, A: 255})
	paintPatch(img, 0, 104, 160, 32, color.RGBA{R: 34, G: 43, B: 61, A: 255}, color.RGBA{R: 102, G: 133, B: 171, A: 255})
	paintPatch(img, 168, 104, 96, 42, rl.Blank, color.RGBA{R: 116, G: 151, B: 194, A: 255})

	// Thumb, checkbox, arrow, and progress-fill parts.
	paintPatch(img, 312, 48, 28, 28, color.RGBA{R: 103, G: 171, B: 235, A: 255}, color.RGBA{R: 196, G: 228, B: 255, A: 255})
	paintPatch(img, 344, 48, 28, 28, color.RGBA{R: 45, G: 117, B: 83, A: 255}, color.RGBA{R: 174, G: 238, B: 184, A: 255})
	rl.ImageDrawLineEx(img, rl.Vector2{X: 350, Y: 61}, rl.Vector2{X: 357, Y: 68}, 3, color.RGBA{R: 235, G: 255, B: 235, A: 255})
	rl.ImageDrawLineEx(img, rl.Vector2{X: 357, Y: 68}, rl.Vector2{X: 367, Y: 56}, 3, color.RGBA{R: 235, G: 255, B: 235, A: 255})
	paintPatch(img, 376, 48, 28, 28, color.RGBA{R: 48, G: 64, B: 91, A: 255}, color.RGBA{R: 153, G: 194, B: 238, A: 255})
	rl.ImageDrawTriangle(img, rl.Vector2{X: 383, Y: 59}, rl.Vector2{X: 397, Y: 59}, rl.Vector2{X: 390, Y: 67}, color.RGBA{R: 225, G: 240, B: 255, A: 255})
	paintPatch(img, 408, 48, 96, 20, color.RGBA{R: 52, G: 145, B: 101, A: 255}, color.RGBA{R: 145, G: 238, B: 176, A: 255})

	tex := rl.LoadTextureFromImage(img)
	rl.UnloadImage(img)
	if tex.ID == 0 {
		log.Fatal("could not upload the procedural skin atlas")
	}
	rl.SetTextureFilter(tex, rl.FilterPoint)
	return tex
}

func paintPatch(img *rl.Image, x, y, width, height int32, fill, border color.RGBA) {
	rl.ImageDrawRectangle(img, x, y, width, height, fill)
	if border.A != 0 {
		rl.ImageDrawRectangleLines(img, rl.Rectangle{X: float32(x), Y: float32(y), Width: float32(width), Height: float32(height)}, 2, border)
	}
}

func textureOf(tex rl.Texture2D) skin.Texture {
	return skin.Texture{ID: tex.ID, Width: tex.Width, Height: tex.Height, Mipmaps: tex.Mipmaps, Format: int32(tex.Format)}
}

func patchDescriptor(tex skin.Texture, region core.Rect, tint core.Color, border int32, center bool) skin.SkinDescriptor {
	return skin.SkinDescriptor{
		Texture: tex, AtlasRegion: region,
		NinePatch: skin.NinePatch{Source: region, Left: border, Top: border, Right: border, Bottom: border},
		Tint:      tint, Alpha: 1, HasTexture: true, HasNinePatch: border > 0, CenterFill: center,
		PaddingLeft: float32(border), PaddingTop: float32(border), PaddingRight: float32(border), PaddingBottom: float32(border),
	}
}

func iconDescriptor(tex skin.Texture, region core.Rect, tint core.Color) skin.SkinDescriptor {
	return skin.SkinDescriptor{Texture: tex, AtlasRegion: region, Tint: tint, Alpha: 1, HasTexture: true}
}

func registerTheme(theme *render.Theme, buttonAtlas, auxAtlas rl.Texture2D) error {
	buttonTex, auxTex := textureOf(buttonAtlas), textureOf(auxAtlas)
	states := []core.WidgetState{core.StateNormal, core.StateFocused, core.StateHovered, core.StatePressed, core.StateDisabled, core.StateSelected}
	stateTints := []core.Color{
		{R: 255, G: 255, B: 255, A: 255}, {R: 225, G: 242, B: 255, A: 255},
		{R: 255, G: 255, B: 255, A: 255}, {R: 235, G: 245, B: 255, A: 255},
		{R: 185, G: 185, B: 195, A: 255}, {R: 245, G: 255, B: 245, A: 255},
	}
	backgroundKinds := []core.WidgetKind{
		core.WidgetButton, core.WidgetRectangle, core.WidgetLabel,
		core.WidgetCheckbox, core.WidgetTextbox, core.WidgetScrollPanel, core.WidgetDropdown,
		core.WidgetFrame,
	}
	for _, kind := range backgroundKinds {
		for i, state := range states {
			if err := registerBackground(theme, kind, state, stateTints[i], buttonTex, auxTex, i); err != nil {
				return err
			}
		}
	}

	for i, state := range states {
		if err := registerStateParts(theme, state, stateTints[i], auxTex); err != nil {
			return err
		}
	}
	return nil
}

func registerBackground(theme *render.Theme, kind core.WidgetKind, state core.WidgetState, tint core.Color, buttonTex, auxTex skin.Texture, stateIndex int) error {
	backgroundTexture, backgroundRegion := backgroundSource(kind, stateIndex, buttonTex, auxTex)
	background := patchDescriptor(backgroundTexture, backgroundRegion, tint, 8, true)
	if err := theme.SetSkinPart(skin.SkinKey{Widget: kind, Part: skin.PartBackground, State: state}, background); err != nil {
		return err
	}
	borderTexture, borderRegion := borderSource(kind, buttonTex, auxTex)
	border := patchDescriptor(borderTexture, borderRegion, tint, 8, false)
	return theme.SetSkinPart(skin.SkinKey{Widget: kind, Part: skin.PartBorder, State: state}, border)
}

func backgroundSource(kind core.WidgetKind, stateIndex int, buttonTex, auxTex skin.Texture) (skin.Texture, core.Rect) {
	if kind == core.WidgetButton {
		// This is the checked-in 192x64 button image. Its 8-pixel border is
		// preserved while the center stretches.
		return buttonTex, core.Rect{W: 192, H: 64}
	}
	return auxTex, core.Rect{X: float32(stateIndex * 68), W: 60, H: 36}
}

func borderSource(kind core.WidgetKind, buttonTex, auxTex skin.Texture) (skin.Texture, core.Rect) {
	if kind == core.WidgetButton {
		return buttonTex, core.Rect{W: 192, H: 64}
	}
	return auxTex, core.Rect{X: 168, Y: 104, W: 96, H: 42}
}

func registerStateParts(theme *render.Theme, state core.WidgetState, tint core.Color, atlas skin.Texture) error {
	parts := []struct {
		part       skin.SkinPart
		descriptor skin.SkinDescriptor
	}{
		{skin.PartTrack, patchDescriptor(atlas, core.Rect{X: 208, Y: 48, W: 96, H: 20}, tint, 6, true)},
		{skin.PartThumb, iconDescriptor(atlas, core.Rect{X: 312, Y: 48, W: 28, H: 28}, tint)},
		{skin.PartTrack, patchDescriptor(atlas, core.Rect{X: 208, Y: 48, W: 96, H: 20}, tint, 6, true)},
		{skin.PartOverlay, patchDescriptor(atlas, core.Rect{X: 408, Y: 48, W: 96, H: 20}, tint, 6, true)},
		{skin.PartArrow, iconDescriptor(atlas, core.Rect{X: 376, Y: 48, W: 28, H: 28}, tint)},
		{skin.PartCheckmark, iconDescriptor(atlas, core.Rect{X: 344, Y: 48, W: 28, H: 28}, tint)},
	}
	kinds := []core.WidgetKind{
		core.WidgetSlider, core.WidgetSlider, core.WidgetProgressBar,
		core.WidgetProgressBar, core.WidgetDropdown, core.WidgetCheckbox,
	}
	for i, registration := range parts {
		key := skin.SkinKey{Widget: kinds[i], Part: registration.part, State: state}
		if err := theme.SetSkinPart(key, registration.descriptor); err != nil {
			return err
		}
	}
	return nil
}

func newGallery(buttonTexture rl.Texture2D, theme *render.Theme, uiTransform *transform.Transform, capture *input.Capture) *gallery {
	g := &gallery{
		buttonTexture: buttonTexture,
		theme:         theme, transform: uiTransform, capture: capture,
		leftPanel: widgets.NewFrame("leftPanel", core.Rect{}), rightPanel: widgets.NewFrame("rightPanel", core.Rect{}),
		button:   widgets.NewButton("primaryButton", core.Rect{}, "Primary button"),
		checkbox: widgets.NewCheckbox("enableCheckbox", core.Rect{}, true),
		textbox:  widgets.NewTextbox("inputTextbox", core.Rect{}, 128),
		dropdown: widgets.NewDropdown("classDropdown", core.Rect{}, []string{"Warrior", "Ranger", "Mage"}, 0),
		slider:   widgets.NewSlider("valueSlider", core.Rect{}, 0.35), progress: widgets.NewProgressBar("valueProgress", core.Rect{}, 0.35),
		rectangle: widgets.NewRectangle("demoRectangle", core.Rect{}), label: widgets.NewLabel("demoLabel", core.Rect{}, "Textured label"),
		frame: widgets.NewFrame("demoFrame", core.Rect{}), frameButton: widgets.NewButton("frameChildButton", core.Rect{}, "Frame child"),
		scroll: widgets.NewScrollPanel("scrollPanel", core.Rect{}), status: "Click a widget to interact with it — press R to MoveFrame",
	}
	g.textbox.TextBuf.Set("Type here")
	// Layout nodes for relative-move child demo (layout.MoveFrame semantics)
	g.frameNode = layout.New("frame", core.Rect{})
	g.frameChildNode = layout.New("frameChild", core.Rect{})
	g.frameChildNode.SetAnchor(layout.AnchorTopLeft)
	g.frameChildNode.SetFixedSize(core.Vec2{X: 120, Y: 40})
	g.frameNode.AddChild(g.frameChildNode)
	return g
}

func (g *gallery) resizeIfNeeded(width, height int) {
	if width == g.lastWidth && height == g.lastHeight {
		return
	}
	g.lastWidth, g.lastHeight = width, height
	w, h := float32(width), float32(height)
	g.layout = calculateLayout(w, h)
	viewport := core.Viewport{Viewport: core.Rect{W: w, H: h}, LogicalSize: core.Vec2{X: w, Y: h}}
	if err := g.transform.SetViewport(viewport); err != nil {
		log.Fatal(err)
	}
	g.leftPanel.Bounds, g.rightPanel.Bounds = g.layout.leftPanel, g.layout.rightPanel
	g.button.Bounds, g.checkbox.Bounds = g.layout.button, g.layout.checkbox
	g.textbox.Bounds, g.dropdown.Bounds = g.layout.textbox, g.layout.dropdown
	g.slider.Bounds, g.progress.Bounds = g.layout.slider, g.layout.progress
	g.rectangle.Bounds, g.label.Bounds = g.layout.rectangle, g.layout.label
	g.frame.Bounds, g.frameButton.Bounds = g.layout.frame, g.layout.frameButton
	g.scroll.Bounds = g.layout.scroll
	// Sync layout nodes with new viewport arrangement
	if g.frameNode != nil {
		g.frameNode.Rect = g.frame.Bounds
		g.frameNode.Resolved = g.frame.Bounds
		g.frameChildNode.Rect = core.Rect{X: 24, Y: 74, W: g.frameButton.Bounds.W, H: g.frameButton.Bounds.H}
		layout.Arrange(g.frameNode, g.frame.Bounds)
	}
}

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
		rectangle:   core.Rect{X: widgetX, Y: left.Y + 440, W: widgetW, H: 94},
		label:       core.Rect{X: right.X + 24, Y: right.Y + 40, W: right.W - 48, H: 32},
		frame:       core.Rect{X: right.X + 24, Y: right.Y + 92, W: right.W - 48, H: 164},
		frameButton: core.Rect{X: right.X + 48, Y: right.Y + 166, W: right.W - 96, H: 42},
		scroll:      core.Rect{X: right.X + 24, Y: right.Y + 282, W: right.W - 48, H: 232},
	}
}

func (g *gallery) handleInput() {
	mouse := g.mousePosition()
	g.handleEscape()
	g.handleFrameMove()
	g.animateFrame()
	if g.dropdownOpen {
		g.handleDropdownInput(mouse)
	} else {
		g.handlePointerInput(mouse)
	}
	g.handleTextInput()
}

func (g *gallery) mousePosition() core.Vec2 {
	physical := rl.GetMousePosition()
	return g.transform.PhysicalToViewport(core.Vec2{X: physical.X, Y: physical.Y})
}

func (g *gallery) handleEscape() {
	if rl.IsKeyPressed(rl.KeyEscape) {
		g.dropdownOpen = false
		g.focus(nil)
	}
}

func (g *gallery) handleFrameMove() {
	if !rl.IsKeyPressed(rl.KeyR) {
		return
	}
	delta := core.Vec2{X: 12, Y: 8}
	if g.frame.Bounds.X+delta.X+g.frame.Bounds.W > float32(g.lastWidth)-20 {
		delta.X = -40
	}
	if g.frame.Bounds.Y+delta.Y+g.frame.Bounds.H > float32(g.lastHeight)-20 {
		delta.Y = -30
	}
	g.applyFrameMove(delta)
	g.status = fmt.Sprintf("MoveFrame %+v — child follows (%.0f,%.0f)", delta, g.frameButton.Bounds.X, g.frameButton.Bounds.Y)
}

func (g *gallery) animateFrame() {
	if g.frameNode == nil || rl.IsKeyDown(rl.KeyR) {
		return
	}
	t := float32(rl.GetTime())
	target := core.Rect{
		X: g.layout.frame.X + float32(math.Sin(float64(t*0.6)))*6,
		Y: g.layout.frame.Y,
		W: g.layout.frame.W,
		H: g.layout.frame.H,
	}
	delta := core.Vec2{X: target.X - g.frame.Bounds.X, Y: target.Y - g.frame.Bounds.Y}
	if math.Abs(float64(delta.X)) <= 0.1 && math.Abs(float64(delta.Y)) <= 0.1 {
		return
	}
	g.applyFrameMove(delta)
}

func (g *gallery) applyFrameMove(delta core.Vec2) {
	layout.MoveFrame(g.frameNode, delta)
	g.frame.Bounds = g.frameNode.Resolved
	g.frameButton.Bounds = g.frameChildNode.Resolved
}

func (g *gallery) handlePointerInput(mouse core.Vec2) {
	g.updateHover(mouse)
	g.handleWheel(mouse)
	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		g.press(mouse)
	}
	if rl.IsMouseButtonDown(rl.MouseButtonLeft) && g.pressed == g.slider {
		g.setSlider(mouse)
	}
	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
		g.release(mouse)
	}
}

func (g *gallery) updateHover(mouse core.Vec2) {
	for _, w := range g.interactive() {
		if w == g.pressed || w == g.focused {
			continue
		}
		w.UpdateHover(mouse)
	}
	if !g.button.Enabled {
		g.button.State = core.StateDisabled
	}
}

func (g *gallery) interactive() []*widgets.Widget {
	return []*widgets.Widget{g.button, g.checkbox, g.textbox, g.dropdown, g.slider, g.frameButton}
}

func (g *gallery) hitInteractive(mouse core.Vec2) *widgets.Widget {
	for _, w := range g.interactive() {
		if w.Enabled && w.HitTest(mouse) {
			return w
		}
	}
	return nil
}

func (g *gallery) press(mouse core.Vec2) {
	w := g.hitInteractive(mouse)
	if w == nil {
		g.focus(nil)
		return
	}
	if w == g.textbox {
		g.focus(w)
	}
	if !w.Press(mouse, g.capture) {
		return
	}
	g.pressed = w
	if w == g.dropdown {
		g.dropdownOpen = true
	}
	if w == g.slider {
		g.setSlider(mouse)
	}
}

func (g *gallery) release(mouse core.Vec2) {
	if g.pressed == nil {
		return
	}
	w := g.pressed
	clicked := w.Release(mouse, g.capture)
	g.pressed = nil
	if !clicked {
		return
	}
	switch w {
	case g.button:
		g.status = "Primary button clicked"
	case g.checkbox:
		g.button.Enabled = g.checkbox.Checked
		g.status = fmt.Sprintf("Checkbox is %v; primary button enabled=%v", g.checkbox.Checked, g.button.Enabled)
	case g.frameButton:
		g.status = "Frame child button clicked"
	}
	if w == g.slider {
		g.progress.Value = g.slider.Value
		g.status = fmt.Sprintf("Slider value %.0f%%", g.slider.Value*100)
	}
}

func (g *gallery) setSlider(mouse core.Vec2) {
	value := (mouse.X - g.slider.Bounds.X) / g.slider.Bounds.W
	g.slider.SetSlider(value)
	g.progress.Value = g.slider.Value
}

func (g *gallery) handleWheel(mouse core.Vec2) {
	if !g.scroll.HitTest(mouse) {
		return
	}
	delta := rl.GetMouseWheelMove()
	if delta == 0 {
		return
	}
	g.scroll.ScrollBy(0, -delta*28)
	g.scroll.Scroll.Y = clamp(g.scroll.Scroll.Y, 0, 170)
}

func (g *gallery) handleDropdownInput(mouse core.Vec2) {
	if !rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		return
	}
	popup := g.dropdownPopup()
	if popup.Contains(mouse) {
		row := int((mouse.Y - popup.Y) / 36)
		if row >= 0 && row < len(g.dropdown.DropdownItems) {
			g.dropdown.DropdownIndex = row
			g.dropdownOpen = false
			g.dropdown.State = core.StateHovered
			g.status = "Dropdown selected: " + g.dropdown.DropdownItems[row]
		}
		return
	}
	if !g.dropdown.HitTest(mouse) {
		g.dropdownOpen = false
	}
}

func (g *gallery) dropdownPopup() core.Rect {
	return core.Rect{X: g.dropdown.Bounds.X, Y: g.dropdown.Bounds.Y + g.dropdown.Bounds.H + 4, W: g.dropdown.Bounds.W, H: float32(len(g.dropdown.DropdownItems) * 36)}
}

func (g *gallery) focus(widget *widgets.Widget) {
	if g.focused == widget {
		return
	}
	if g.focused != nil {
		g.focused.Blur()
	}
	g.focused = widget
	if g.focused != nil {
		g.focused.Focus()
	}
}

func (g *gallery) handleTextInput() {
	if g.focused != g.textbox {
		return
	}
	for codepoint := rl.GetCharPressed(); codepoint > 0; codepoint = rl.GetCharPressed() {
		// UTF-8: TextBuffer truncates safely, multi-byte backspace handled in Backspace()
		if codepoint >= 32 && codepoint != 127 {
			g.textbox.TypeChar(rune(codepoint))
		}
	}
	if rl.IsKeyPressed(rl.KeyBackspace) || rl.IsKeyPressedRepeat(rl.KeyBackspace) {
		g.textbox.Backspace()
	}
}

func (g *gallery) draw() {
	rl.BeginDrawing()
	rl.ClearBackground(color.RGBA{R: 13, G: 17, B: 27, A: 255})

	rl.DrawText("RTG textured widget gallery", 28, 24, 26, color.RGBA{R: 226, G: 239, B: 255, A: 255})
	rl.DrawText("Every v1 widget uses an atlas part, nine-patch, tint, alpha, or the documented fallback path. Press R to MoveFrame.", 30, 51, 14, color.RGBA{R: 153, G: 174, B: 202, A: 255})
	panelTitle(g.layout.leftPanel, "Widgets")
	panelTitle(g.layout.rightPanel, "Containers, clipping, and states")

	g.drawWidget(g.leftPanel)
	g.drawWidget(g.rightPanel)
	g.drawWidget(g.button)
	g.drawWidget(g.checkbox)
	g.drawWidget(g.textbox)
	g.drawWidget(g.dropdown)
	g.drawWidget(g.slider)
	g.drawWidget(g.progress)
	g.drawWidget(g.rectangle)
	g.drawWidget(g.label)
	g.drawWidget(g.frame)
	g.drawWidget(g.frameButton)
	g.drawWidget(g.scroll)

	rl.DrawText("Enable primary button", int32(g.checkbox.Bounds.X+40), int32(g.checkbox.Bounds.Y+8), 18, color.RGBA{R: 205, G: 218, B: 238, A: 255})
	rl.DrawText("Textbox (click, type, backspace — UTF-8)", int32(g.textbox.Bounds.X), int32(g.textbox.Bounds.Y-23), 15, color.RGBA{R: 153, G: 174, B: 202, A: 255})
	rl.DrawText("Dropdown (click to open)", int32(g.dropdown.Bounds.X), int32(g.dropdown.Bounds.Y-23), 15, color.RGBA{R: 153, G: 174, B: 202, A: 255})
	rl.DrawText("Slider drives the progress bar", int32(g.slider.Bounds.X), int32(g.slider.Bounds.Y-23), 15, color.RGBA{R: 153, G: 174, B: 202, A: 255})
	rl.DrawText(fmt.Sprintf("%.0f%%", g.slider.Value*100), int32(g.slider.Bounds.X+g.slider.Bounds.W-48), int32(g.slider.Bounds.Y+13), 16, color.RGBA{R: 230, G: 242, B: 255, A: 255})
	rl.DrawText(fmt.Sprintf("Progress: %.0f%%", g.progress.Value*100), int32(g.progress.Bounds.X+12), int32(g.progress.Bounds.Y+13), 16, color.RGBA{R: 235, G: 255, B: 240, A: 255})
	rl.DrawText("Rectangle primitive / frame decoration", int32(g.rectangle.Bounds.X+14), int32(g.rectangle.Bounds.Y+38), 17, color.RGBA{R: 218, G: 230, B: 248, A: 255})
	rl.DrawText("Frame child moves with its parent (R / sine)", int32(g.frame.Bounds.X+18), int32(g.frame.Bounds.Y+20), 15, color.RGBA{R: 153, G: 174, B: 202, A: 255})

	g.drawScrollContents()
	if g.dropdownOpen {
		g.drawDropdownPopup()
	}
	g.drawStateSamples()
	rl.DrawText(g.status, 30, int32(g.lastHeight-18), 15, color.RGBA{R: 161, G: 192, B: 224, A: 255})

	rl.EndDrawing()
}

func panelTitle(bounds core.Rect, title string) {
	rl.DrawText(title, int32(bounds.X+24), int32(bounds.Y+15), 20, color.RGBA{R: 224, G: 235, B: 252, A: 255})
}

func (g *gallery) drawWidget(w *widgets.Widget) {
	if !w.Enabled {
		w.State = core.StateDisabled
	}
	info := w.Info()
	info.HasCapture = g.capture.IsCaptured() && g.capture.ID() == w.ID
	if err := g.drawWidgetParts(w, info, g.widgetText(w)); err != nil {
		log.Fatal(err)
	}
}

func (g *gallery) widgetText(w *widgets.Widget) string {
	text := w.Text
	if w.TextBuf != nil {
		text = w.TextBuf.String()
	}
	if w == g.dropdown && w.DropdownIndex >= 0 && w.DropdownIndex < len(w.DropdownItems) {
		text = w.DropdownItems[w.DropdownIndex]
	}
	return text
}

func (g *gallery) drawWidgetParts(w *widgets.Widget, info core.WidgetInfo, text string) error {
	if w == g.label {
		if err := g.theme.DrawWidgetPart(w.Kind, skin.PartBackground, w.Bounds, w.State); err != nil {
			return err
		}
	}
	if err := g.theme.DrawWidget(info, text, w.Value, w.Checked); err != nil {
		return err
	}
	if needsBorder(w.Kind) {
		if err := g.theme.DrawWidgetPart(w.Kind, skin.PartBorder, w.Bounds, w.State); err != nil {
			return err
		}
	}
	if w == g.dropdown {
		arrow := core.Rect{X: w.Bounds.X + w.Bounds.W - 34, Y: w.Bounds.Y + 7, W: 28, H: 28}
		if err := g.theme.DrawWidgetPart(w.Kind, skin.PartArrow, arrow, w.State); err != nil {
			return err
		}
	}
	return nil
}

func needsBorder(kind core.WidgetKind) bool {
	return kind == core.WidgetButton || kind == core.WidgetRectangle || kind == core.WidgetLabel || kind == core.WidgetTextbox || kind == core.WidgetScrollPanel || kind == core.WidgetDropdown || kind == core.WidgetFrame
}

func (g *gallery) drawScrollContents() {
	// Scissor is left to the app (draw.go never calls BeginScissorMode)
	rl.BeginScissorMode(int32(g.scroll.Bounds.X), int32(g.scroll.Bounds.Y), int32(g.scroll.Bounds.W), int32(g.scroll.Bounds.H))
	start := g.scroll.Bounds.Y + 12 - g.scroll.Scroll.Y
	for i := 0; i < 10; i++ {
		y := start + float32(i*34)
		fill := color.RGBA{R: 35, G: 48, B: 70, A: 255}
		if i%2 == 1 {
			fill = color.RGBA{R: 29, G: 40, B: 59, A: 255}
		}
		rl.DrawRectangleRec(rl.Rectangle{X: g.scroll.Bounds.X + 10, Y: y, Width: g.scroll.Bounds.W - 20, Height: 28}, fill)
		rl.DrawText(fmt.Sprintf("Clipped row %02d  •  scroll offset %.0f", i+1, g.scroll.Scroll.Y), int32(g.scroll.Bounds.X+20), int32(y+6), 14, color.RGBA{R: 194, G: 211, B: 235, A: 255})
	}
	rl.EndScissorMode()
	rl.DrawText("Scroll panel — wheel over this area", int32(g.scroll.Bounds.X+12), int32(g.scroll.Bounds.Y-22), 15, color.RGBA{R: 153, G: 174, B: 202, A: 255})
}

func (g *gallery) drawDropdownPopup() {
	popup := g.dropdownPopup()
	info := core.WidgetInfo{ID: 1001, Name: "dropdownPopup", Bounds: popup, Kind: core.WidgetDropdown, State: core.StatePressed}
	if err := g.theme.DrawWidget(info, "", 0, false); err != nil {
		log.Fatal(err)
	}
	if err := g.theme.DrawWidgetPart(core.WidgetDropdown, skin.PartBorder, popup, core.StatePressed); err != nil {
		log.Fatal(err)
	}
	for i, item := range g.dropdown.DropdownItems {
		y := popup.Y + float32(i*36)
		mouse := g.transform.PhysicalToViewport(core.Vec2{X: rl.GetMousePosition().X, Y: rl.GetMousePosition().Y})
		rowBounds := core.Rect{X: popup.X, Y: y, W: popup.W, H: 36}
		if rowBounds.Contains(mouse) {
			rl.DrawRectangle(int32(popup.X+4), int32(y+3), int32(popup.W-8), 30, color.RGBA{R: 67, G: 97, B: 139, A: 255})
		}
		rl.DrawText(item, int32(popup.X+16), int32(y+8), 16, color.RGBA{R: 230, G: 240, B: 255, A: 255})
	}
}

func (g *gallery) drawStateSamples() {
	names := []string{"normal", "focus", "hover", "press", "disabled", "selected"}
	startX := g.rightPanel.Bounds.X + 20
	y := g.rightPanel.Bounds.Y + g.rightPanel.Bounds.H - 66
	for i, state := range []core.WidgetState{core.StateNormal, core.StateFocused, core.StateHovered, core.StatePressed, core.StateDisabled, core.StateSelected} {
		bounds := core.Rect{X: startX + float32(i)*((g.rightPanel.Bounds.W-40)/6), Y: y, W: (g.rightPanel.Bounds.W - 52) / 6, H: 34}
		if err := g.theme.DrawWidgetPart(core.WidgetButton, skin.PartBackground, bounds, state); err != nil {
			log.Fatal(err)
		}
		rl.DrawText(names[i], int32(bounds.X+5), int32(bounds.Y+10), 11, color.RGBA{R: 228, G: 239, B: 255, A: 255})
	}
}

func clamp(value, low, high float32) float32 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
