// widget_gallery is the single runnable demo proving the pure-Go rtgui library end-to-end.
//
// It uses stock Go + github.com/gen2brain/raylib-go/raylib plus the rtgui/ui
// facade. The UI owns transform, capture, theme, registry, focus, and
// callbacks; the gallery only polls raylib, forwards to the facade, and draws
// app-specific layers (scroll contents, dropdown popup, state samples).
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
//     the facade dispatch and draw log and writes a placeholder PNG via the
//     stdlib image/png if -screenshot was requested.
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
	"rtgui/layout"
	"rtgui/render"
	"rtgui/skin"
	"rtgui/ui"
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

	// layout nodes to demonstrate MoveFrame (relative-move child)
	frameNode      *layout.Node
	frameChildNode *layout.Node

	dropdownOpen bool
	status       string
	designWidth  float32
	designHeight float32
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
	kenney := loadKenneyTextures()
	defer kenney.unload()
	auxAtlas := makeAuxAtlas()
	defer rl.UnloadTexture(auxAtlas)
	facade := ui.New(int(windowWidth), int(windowHeight))
	if err := registerTheme(facade.Theme(), auxAtlas, kenney); err != nil {
		log.Fatal(err)
	}
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
// touched, but the DrawCall log still runs), and writes a placeholder PNG if
// requested.
func runHeadlessSmoke() {
	facade := ui.New(800, 600)
	button := widgets.NewButton("smokeButton", core.Rect{X: 10, Y: 10, W: 100, H: 40}, "OK")
	checkbox := widgets.NewCheckbox("smokeCheckbox", core.Rect{X: 10, Y: 60, W: 120, H: 40}, true)
	slider := widgets.NewSlider("smokeSlider", core.Rect{X: 10, Y: 110, W: 120, H: 40}, 0.5)
	field := widgets.NewTextbox("smokeField", core.Rect{X: 10, Y: 160, W: 200, H: 30}, 64)
	facade.Add(button, checkbox, slider, field)
	clicks := 0
	facade.OnClick("smokeButton", func() { clicks++ })
	center := core.Vec2{X: 60, Y: 30}
	facade.HandleMouse(ui.MouseEvent{Pos: center, Pressed: true})
	facade.HandleMouse(ui.MouseEvent{Pos: center, Released: true})
	facade.HandleKey(ui.KeyEvent{Chars: []rune("hi")})
	facade.Draw()
	calls := facade.Theme().DrawLog()
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

// kenneySet holds the Kenney UI textures (testdata/skins/kenney) that give
// the gallery its button states, checkbox icons, slider handle, and arrow.
type kenneySet struct {
	button, buttonHover, buttonPressed rl.Texture2D
	checkEmpty, checkCross             rl.Texture2D
	sliderHandle, arrow                rl.Texture2D
	panelBorder                        rl.Texture2D
}

// loadKenneyTextures loads every Kenney file the gallery needs.
func loadKenneyTextures() kenneySet {
	return kenneySet{
		button:        loadSkinTexture("kenney/blue/button_rectangle_line.png"),
		buttonHover:   loadSkinTexture("kenney/blue/button_rectangle_border.png"),
		buttonPressed: loadSkinTexture("kenney/red/button_rectangle_border.png"),
		checkEmpty:    loadSkinTexture("kenney/blue/check_square_grey.png"),
		checkCross:    loadSkinTexture("kenney/blue/check_square_grey_cross.png"),
		sliderHandle:  loadSkinTexture("kenney/blue/slide_hangle.png"),
		arrow:         loadSkinTexture("kenney/blue/arrow_basic_s_small.png"),
		panelBorder:   loadSkinTexture("kenney/panel_border_grey.png"),
	}
}

// unload releases every Kenney texture.
func (k kenneySet) unload() {
	rl.UnloadTexture(k.button)
	rl.UnloadTexture(k.buttonHover)
	rl.UnloadTexture(k.buttonPressed)
	rl.UnloadTexture(k.checkEmpty)
	rl.UnloadTexture(k.checkCross)
	rl.UnloadTexture(k.sliderHandle)
	rl.UnloadTexture(k.arrow)
	rl.UnloadTexture(k.panelBorder)
}

// loadSkinTexture loads one file below testdata/skins, fatal on failure.
func loadSkinTexture(rel string) rl.Texture2D {
	path := findSkinFile(rel)
	if path == "" {
		log.Fatalf("gallery skin not found; expected testdata/skins/%s (searched cwd and exe parents)", rel)
	}
	tex := rl.LoadTexture(path)
	if tex.ID == 0 {
		log.Fatalf("could not load gallery skin %q", path)
	}
	rl.SetTextureFilter(tex, rl.FilterPoint)
	return tex
}

// findSkinFile locates a file below testdata/skins, mirroring font lookup.
func findSkinFile(rel string) string {
	return firstExistingFile(skinCandidates(rel))
}

// skinCandidates searches RTG_SKIN_DIR, then cwd and exe parents.
func skinCandidates(rel string) []string {
	candidates := []string{}
	if configured := os.Getenv("RTG_SKIN_DIR"); configured != "" {
		candidates = append(candidates, filepath.Join(configured, rel))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = appendParentSkinCandidates(candidates, cwd, rel)
	}
	if exe, err := os.Executable(); err == nil {
		candidates = appendParentSkinCandidates(candidates, filepath.Dir(exe), rel)
	}
	return candidates
}

// appendParentSkinCandidates walks up to 5 parents for testdata/skins/rel.
func appendParentSkinCandidates(candidates []string, root, rel string) []string {
	for dir, depth := root, 0; dir != filepath.Dir(dir) && depth < 5; dir, depth = filepath.Dir(dir), depth+1 {
		if dir == "" {
			continue
		}
		candidates = append(candidates,
			filepath.Join(dir, "testdata", "skins", rel),
			filepath.Join(dir, "rtgui", "testdata", "skins", rel),
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

// fullIconDescriptor wraps a whole texture as an icon descriptor.
func fullIconDescriptor(tex skin.Texture, tint core.Color) skin.SkinDescriptor {
	return iconDescriptor(tex, core.Rect{W: float32(tex.Width), H: float32(tex.Height)}, tint)
}

func iconDescriptor(tex skin.Texture, region core.Rect, tint core.Color) skin.SkinDescriptor {
	return skin.SkinDescriptor{Texture: tex, AtlasRegion: region, Tint: tint, Alpha: 1, HasTexture: true}
}

// registerTheme skins every widget kind. Panel fills, tracks, and progress
// come from the procedural atlas; buttons, checkbox icons, slider handle,
// and dropdown arrow come from the Kenney set (testdata/skins/kenney).
func registerTheme(theme *render.Theme, auxAtlas rl.Texture2D, kenney kenneySet) error {
	auxTex := textureOf(auxAtlas)
	states := []core.WidgetState{core.StateNormal, core.StateFocused, core.StateHovered, core.StatePressed, core.StateDisabled, core.StateSelected}
	stateTints := []core.Color{
		{R: 255, G: 255, B: 255, A: 255}, {R: 225, G: 242, B: 255, A: 255},
		{R: 255, G: 255, B: 255, A: 255}, {R: 235, G: 245, B: 255, A: 255},
		{R: 185, G: 185, B: 195, A: 255}, {R: 245, G: 255, B: 245, A: 255},
	}
	backgroundKinds := []core.WidgetKind{
		core.WidgetLabel,
		core.WidgetCheckbox, core.WidgetTextbox, core.WidgetScrollPanel, core.WidgetDropdown,
		core.WidgetFrame,
	}
	for _, kind := range backgroundKinds {
		for i, state := range states {
			if err := registerBackground(theme, kind, state, stateTints[i], auxTex, i); err != nil {
				return err
			}
		}
	}

	for i, state := range states {
		if err := registerStateParts(theme, state, stateTints[i], auxTex, kenney); err != nil {
			return err
		}
	}
	if err := registerKenneyButtons(theme, states, stateTints, kenney); err != nil {
		return err
	}
	if err := registerPanelBorder(theme, states, stateTints, kenney); err != nil {
		return err
	}
	return registerFrameBackground(theme, states, stateTints, auxTex)
}

// registerPanelBorder skins frame borders from the Kenney grey panel
// ring (64x64, 8px edges, empty center) as an 8-patch: nine-patch insets with
// CenterFill false so the transparent middle is never drawn.
func registerPanelBorder(theme *render.Theme, states []core.WidgetState, tints []core.Color, kenney kenneySet) error {
	tex := textureOf(kenney.panelBorder)
	region := core.Rect{W: float32(tex.Width), H: float32(tex.Height)}
	for i, state := range states {
		border := patchDescriptor(tex, region, tints[i], 8, false)
		if err := theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetFrame, Part: skin.PartBorder, State: state}, border); err != nil {
			return err
		}
	}
	return nil
}

// registerFrameBackground gives frames a flat fill.
// The aux state patches carry a painted 2px outline that would read as a
// second border behind the Kenney ring, so the region is cropped 3px on each
// side to cut the outline off and drawn unstretched (no nine-patch).
func registerFrameBackground(theme *render.Theme, states []core.WidgetState, tints []core.Color, auxTex skin.Texture) error {
	for i, state := range states {
		region := core.Rect{X: float32(i*68 + 3), Y: 3, W: 54, H: 30}
		background := patchDescriptor(auxTex, region, tints[i], 0, true)
		if err := theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetFrame, Part: skin.PartBackground, State: state}, background); err != nil {
			return err
		}
	}
	return nil
}

// blue border for focus/hover/selected, red border for pressed. All are
// 192x64 with an 8-pixel nine-patch border, like the retired button image.
func registerKenneyButtons(theme *render.Theme, states []core.WidgetState, tints []core.Color, kenney kenneySet) error {
	byState := map[core.WidgetState]skin.Texture{
		core.StateNormal:   textureOf(kenney.button),
		core.StateFocused:  textureOf(kenney.buttonHover),
		core.StateHovered:  textureOf(kenney.buttonHover),
		core.StatePressed:  textureOf(kenney.buttonPressed),
		core.StateDisabled: textureOf(kenney.button),
		core.StateSelected: textureOf(kenney.buttonHover),
	}
	for i, state := range states {
		tex := byState[state]
		region := core.Rect{W: float32(tex.Width), H: float32(tex.Height)}
		background := patchDescriptor(tex, region, tints[i], 8, true)
		if err := theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBackground, State: state}, background); err != nil {
			return err
		}
		border := patchDescriptor(tex, region, tints[i], 8, false)
		if err := theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBorder, State: state}, border); err != nil {
			return err
		}
	}
	return nil
}

func registerBackground(theme *render.Theme, kind core.WidgetKind, state core.WidgetState, tint core.Color, auxTex skin.Texture, stateIndex int) error {
	backgroundTexture, backgroundRegion := backgroundSource(stateIndex, auxTex)
	background := patchDescriptor(backgroundTexture, backgroundRegion, tint, 8, true)
	if err := theme.SetSkinPart(skin.SkinKey{Widget: kind, Part: skin.PartBackground, State: state}, background); err != nil {
		return err
	}
	borderTexture, borderRegion := borderSource(auxTex)
	border := patchDescriptor(borderTexture, borderRegion, tint, 8, false)
	return theme.SetSkinPart(skin.SkinKey{Widget: kind, Part: skin.PartBorder, State: state}, border)
}

func backgroundSource(stateIndex int, auxTex skin.Texture) (skin.Texture, core.Rect) {
	return auxTex, core.Rect{X: float32(stateIndex * 68), W: 60, H: 36}
}

func borderSource(auxTex skin.Texture) (skin.Texture, core.Rect) {
	return auxTex, core.Rect{X: 168, Y: 104, W: 96, H: 42}
}

// registerStateParts skins tracks, fills, and icons per state. The slider
// handle, dropdown arrow, and checkbox icons come from the Kenney set; the
// checkbox empty box rides PartIcon so unchecked boxes render without touching
// the checkmark path.
func registerStateParts(theme *render.Theme, state core.WidgetState, tint core.Color, atlas skin.Texture, kenney kenneySet) error {
	handle := fullIconDescriptor(textureOf(kenney.sliderHandle), tint)
	arrow := fullIconDescriptor(textureOf(kenney.arrow), tint)
	cross := fullIconDescriptor(textureOf(kenney.checkCross), tint)
	parts := []struct {
		part       skin.SkinPart
		descriptor skin.SkinDescriptor
	}{
		{skin.PartTrack, patchDescriptor(atlas, core.Rect{X: 208, Y: 48, W: 96, H: 20}, tint, 6, true)},
		{skin.PartThumb, handle},
		{skin.PartTrack, patchDescriptor(atlas, core.Rect{X: 208, Y: 48, W: 96, H: 20}, tint, 6, true)},
		{skin.PartOverlay, patchDescriptor(atlas, core.Rect{X: 408, Y: 48, W: 96, H: 20}, tint, 6, true)},
		{skin.PartArrow, arrow},
		{skin.PartCheckmark, cross},
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
	empty := fullIconDescriptor(textureOf(kenney.checkEmpty), tint)
	return theme.SetSkinPart(skin.SkinKey{Widget: core.WidgetCheckbox, Part: skin.PartIcon, State: state}, empty)
}

// newGallery builds the widget set, registers it with the facade in draw
// order, and wires the gallery callbacks. All hover/press/focus/capture
// state lives in the facade; the gallery keeps only app concerns (popup open,
// frame nodes, status text, layout cache).
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
	g.textbox.TextBuf.Set("Type here")
	facade.Add(g.leftPanel, g.rightPanel, g.button, g.checkbox, g.textbox, g.dropdown, g.slider, g.progress, g.panel, g.label, g.frame, g.frameButton, g.scroll)
	facade.OnClick("primaryButton", func() {
		g.status = "Primary button clicked"
	})
	facade.OnClick("enableCheckbox", func() {
		g.button.Enabled = g.checkbox.Checked
		g.status = fmt.Sprintf("Checkbox is %v; primary button enabled=%v", g.checkbox.Checked, g.button.Enabled)
	})
	facade.OnClick("frameChildButton", func() {
		g.status = "Frame child button clicked"
	})
	facade.OnClick("classDropdown", func() {
		g.dropdownOpen = !g.dropdownOpen
	})
	facade.OnChange("valueSlider", func(v float32) {
		g.progress.Value = v
		g.status = fmt.Sprintf("Slider value %.0f%%", v*100)
	})
	// Layout nodes for relative-move child demo (layout.MoveFrame semantics)
	g.frameNode = layout.New("frame", core.Rect{})
	g.frameChildNode = layout.New("frameChild", core.Rect{})
	g.frameChildNode.SetAnchor(layout.AnchorTopLeft)
	g.frameChildNode.SetFixedSize(core.Vec2{X: 120, Y: 40})
	g.frameNode.AddChild(g.frameChildNode)
	// Layout is computed once from the fixed design resolution; later window
	// resizes rescale around these bounds instead of reflowing them.
	logical := facade.Transform().Viewport.LogicalSize
	g.designWidth, g.designHeight = logical.X, logical.Y
	g.layout = calculateLayout(logical.X, logical.Y)
	g.applyLayout()
	return g
}

// applyLayout assigns the cached design-resolution bounds to widgets and
// syncs the frame layout nodes. It runs once at startup; bounds never follow
// the live window size after that.
func (g *gallery) applyLayout() {
	g.leftPanel.Bounds, g.rightPanel.Bounds = g.layout.leftPanel, g.layout.rightPanel
	g.button.Bounds, g.checkbox.Bounds = g.layout.button, g.layout.checkbox
	g.textbox.Bounds, g.dropdown.Bounds = g.layout.textbox, g.layout.dropdown
	g.slider.Bounds, g.progress.Bounds = g.layout.slider, g.layout.progress
	g.panel.Bounds, g.label.Bounds = g.layout.panel, g.layout.label
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
		panel:       core.Rect{X: widgetX, Y: left.Y + 440, W: widgetW, H: 94},
		label:       core.Rect{X: right.X + 24, Y: right.Y + 40, W: right.W - 48, H: 32},
		frame:       core.Rect{X: right.X + 24, Y: right.Y + 92, W: right.W - 48, H: 164},
		frameButton: core.Rect{X: right.X + 48, Y: right.Y + 166, W: right.W - 96, H: 42},
		scroll:      core.Rect{X: right.X + 24, Y: right.Y + 282, W: right.W - 48, H: 232},
	}
}

// handleInput polls raylib once per frame and forwards to the facade.
// The dropdown popup gets first refusal on left-press; everything else flows
// through ui.HandleMouse/HandleKey, which report handled for game gating.
// Frame-move (R) and the idle sine animation mutate frame bounds directly.
func (g *gallery) handleInput() {
	physical := rl.GetMousePosition()
	mouse := g.facade.ToLogical(core.Vec2{X: physical.X, Y: physical.Y})
	pressedEdge := rl.IsMouseButtonPressed(rl.MouseButtonLeft)
	if g.dropdownOpen && pressedEdge {
		if g.handleDropdownClick(mouse) {
			// Popup consumed this press (row selection or outside close):
			// still pump keys so Escape works, but skip mouse dispatch to
			// avoid opening a second gesture underneath.
			g.facade.HandleMouse(ui.MouseEvent{Pos: mouse, Down: rl.IsMouseButtonDown(rl.MouseButtonLeft), Wheel: rl.GetMouseWheelMove()})
			g.handleKeys()
			g.handleFrameMove()
			g.animateFrame()
			return
		}
		// else: the press landed on the dropdown widget itself — fall through
		// to normal dispatch so press/release fires OnClick and toggles the
		// popup closed.
	}
	mouseHandled := g.facade.HandleMouse(ui.MouseEvent{
		Pos:      mouse,
		Pressed:  pressedEdge,
		Down:     rl.IsMouseButtonDown(rl.MouseButtonLeft),
		Released: rl.IsMouseButtonReleased(rl.MouseButtonLeft),
		Wheel:    rl.GetMouseWheelMove(),
	})
	_ = mouseHandled
	// The facade scrolls unclamped; the gallery bounds its demo list.
	g.scroll.Scroll.Y = clamp(g.scroll.Scroll.Y, 0, 170)
	g.handleKeys()
	g.handleFrameMove()
	g.animateFrame()
}

// handleKeys forwards chars, backspace, and escape to the facade.
// Escape also closes the dropdown popup when it is open.
func (g *gallery) handleKeys() {
	escape := rl.IsKeyPressed(rl.KeyEscape)
	if escape && g.dropdownOpen {
		g.dropdownOpen = false
	}
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
	if g.frame.Bounds.X+delta.X+g.frame.Bounds.W > g.designWidth-20 {
		delta.X = -40
	}
	if g.frame.Bounds.Y+delta.Y+g.frame.Bounds.H > g.designHeight-20 {
		delta.Y = -30
	}
	g.applyFrameMove(delta)
	g.status = fmt.Sprintf("MoveFrame %+v — child follows (%.0f,%.0f)", delta, g.frameButton.Bounds.X, g.frameButton.Bounds.Y)
}

// animateFrame drifts the demo frame on a sine wave.
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

// applyFrameMove moves the frame node and syncs widget bounds to it.
func (g *gallery) applyFrameMove(delta core.Vec2) {
	layout.MoveFrame(g.frameNode, delta)
	g.frame.Bounds = g.frameNode.Resolved
	g.frameButton.Bounds = g.frameChildNode.Resolved
}

// handleDropdownClick selects a popup row or closes on outside click.
// It reports true when the press was consumed by the popup.
func (g *gallery) handleDropdownClick(mouse core.Vec2) bool {
	popup := g.dropdownPopup()
	if popup.Contains(mouse) {
		row := int((mouse.Y - popup.Y) / 36)
		if row >= 0 && row < len(g.dropdown.DropdownItems) {
			g.dropdown.DropdownIndex = row
			g.dropdownOpen = false
			g.dropdown.State = core.StateHovered
			g.status = "Dropdown selected: " + g.dropdown.DropdownItems[row]
		}
		return true
	}
	if !g.dropdown.HitTest(mouse) {
		g.dropdownOpen = false
		return true
	}
	return false
}

// dropdownPopup returns the popup bounds below the dropdown widget.
func (g *gallery) dropdownPopup() core.Rect {
	return core.Rect{X: g.dropdown.Bounds.X, Y: g.dropdown.Bounds.Y + g.dropdown.Bounds.H + 4, W: g.dropdown.Bounds.W, H: float32(len(g.dropdown.DropdownItems) * 36)}
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
	g.drawItalic("Every v1 widget uses an atlas part, nine-patch, tint, alpha, or the documented fallback path. Press R to MoveFrame.", 30, 51, 14, color.RGBA{R: 153, G: 174, B: 202, A: 255})
	g.panelTitle(g.layout.leftPanel, "Widgets")
	g.panelTitle(g.layout.rightPanel, "Containers, clipping, and states")

	g.facade.Draw()

	g.drawText("Enable primary button", int32(g.checkbox.Bounds.X+40), int32(g.checkbox.Bounds.Y+8), 18, color.RGBA{R: 205, G: 218, B: 238, A: 255})
	g.drawItalic("Textbox (click, type, backspace — UTF-8)", int32(g.textbox.Bounds.X), int32(g.textbox.Bounds.Y-23), 15, color.RGBA{R: 153, G: 174, B: 202, A: 255})
	g.drawItalic("Dropdown (click to open)", int32(g.dropdown.Bounds.X), int32(g.dropdown.Bounds.Y-23), 15, color.RGBA{R: 153, G: 174, B: 202, A: 255})
	g.drawItalic("Slider drives the progress bar", int32(g.slider.Bounds.X), int32(g.slider.Bounds.Y-23), 15, color.RGBA{R: 153, G: 174, B: 202, A: 255})
	g.drawText(fmt.Sprintf("%.0f%%", g.slider.Value*100), int32(g.slider.Bounds.X+g.slider.Bounds.W-48), int32(g.slider.Bounds.Y+13), 16, color.RGBA{R: 230, G: 242, B: 255, A: 255})
	g.drawText(fmt.Sprintf("Progress: %.0f%%", g.progress.Value*100), int32(g.progress.Bounds.X+12), int32(g.progress.Bounds.Y+13), 16, color.RGBA{R: 235, G: 255, B: 240, A: 255})
	g.drawText("Panel frame decoration", int32(g.panel.Bounds.X+14), int32(g.panel.Bounds.Y+38), 17, color.RGBA{R: 218, G: 230, B: 248, A: 255})
	g.drawText("Frame child moves with its parent (R / sine)", int32(g.frame.Bounds.X+18), int32(g.frame.Bounds.Y+20), 15, color.RGBA{R: 153, G: 174, B: 202, A: 255})

	g.drawScrollContents()
	if g.dropdownOpen {
		g.drawDropdownPopup()
	}
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
	rl.BeginScissorMode(int32(g.scroll.Bounds.X*sx), int32(g.scroll.Bounds.Y*sy), int32(g.scroll.Bounds.W*sx), int32(g.scroll.Bounds.H*sy))
	start := g.scroll.Bounds.Y + 12 - g.scroll.Scroll.Y
	for i := 0; i < 10; i++ {
		y := start + float32(i*34)
		fill := color.RGBA{R: 35, G: 48, B: 70, A: 255}
		if i%2 == 1 {
			fill = color.RGBA{R: 29, G: 40, B: 59, A: 255}
		}
		rl.DrawRectangleRec(rl.Rectangle{X: g.scroll.Bounds.X + 10, Y: y, Width: g.scroll.Bounds.W - 20, Height: 28}, fill)
		g.drawText(fmt.Sprintf("Clipped row %02d  •  scroll offset %.0f", i+1, g.scroll.Scroll.Y), int32(g.scroll.Bounds.X+20), int32(y+6), 14, color.RGBA{R: 194, G: 211, B: 235, A: 255})
	}
	rl.EndScissorMode()
	g.drawItalic("Scroll panel — wheel over this area", int32(g.scroll.Bounds.X+12), int32(g.scroll.Bounds.Y-22), 15, color.RGBA{R: 153, G: 174, B: 202, A: 255})
}

// drawDropdownPopup renders the open popup and its hover highlight.
func (g *gallery) drawDropdownPopup() {
	theme := g.facade.Theme()
	popup := g.dropdownPopup()
	info := core.WidgetInfo{ID: 1001, Name: "dropdownPopup", Bounds: popup, Kind: core.WidgetDropdown, State: core.StatePressed}
	if err := theme.DrawWidget(info, "", 0, false); err != nil {
		log.Fatal(err)
	}
	if err := theme.DrawWidgetPart(core.WidgetDropdown, skin.PartBorder, popup, core.StatePressed); err != nil {
		log.Fatal(err)
	}
	physical := rl.GetMousePosition()
	mouse := g.facade.ToLogical(core.Vec2{X: physical.X, Y: physical.Y})
	for i, item := range g.dropdown.DropdownItems {
		y := popup.Y + float32(i*36)
		rowBounds := core.Rect{X: popup.X, Y: y, W: popup.W, H: 36}
		if rowBounds.Contains(mouse) {
			rl.DrawRectangle(int32(popup.X+4), int32(y+3), int32(popup.W-8), 30, color.RGBA{R: 67, G: 97, B: 139, A: 255})
		}
		g.drawText(item, int32(popup.X+16), int32(y+8), 16, color.RGBA{R: 230, G: 240, B: 255, A: 255})
	}
}

// drawStateSamples renders the six state swatches.
func (g *gallery) drawStateSamples() {
	theme := g.facade.Theme()
	names := []string{"normal", "focus", "hover", "press", "disabled", "selected"}
	startX := g.rightPanel.Bounds.X + 20
	y := g.rightPanel.Bounds.Y + g.rightPanel.Bounds.H - 66
	for i, state := range []core.WidgetState{core.StateNormal, core.StateFocused, core.StateHovered, core.StatePressed, core.StateDisabled, core.StateSelected} {
		bounds := core.Rect{X: startX + float32(i)*((g.rightPanel.Bounds.W-40)/6), Y: y, W: (g.rightPanel.Bounds.W - 52) / 6, H: 34}
		if err := theme.DrawWidgetPart(core.WidgetButton, skin.PartBackground, bounds, state); err != nil {
			log.Fatal(err)
		}
		g.drawText(names[i], int32(bounds.X+5), int32(bounds.Y+10), 11, color.RGBA{R: 228, G: 239, B: 255, A: 255})
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
