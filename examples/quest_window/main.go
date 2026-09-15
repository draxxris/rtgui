// quest_window recreates a classic MMORPG quest journal using only the
// rtgui widget library: an 800x800 logical ornate TitledFrame titled Quests,
// with filter tabs, a dark collapsible list, parchment details, and glowing
// bottom action buttons. The journal root carries a synthetic 24px layout
// margin inside a full-window viewport. See quest.go for the journal
// model and layout.go for the fixed slot table.
//
// Gallery contract: ornate gold border, textured blue title bar, parchment
// tale background, 3D glowing buttons, tab filtering, leaf selection,
// link clicks, abandon/share/track actions, and close/reopen. Flags
// -frames / -screenshot drive the headless smoke and screenshots.
//
// Headless behavior mirrors the gallery: without a display the binary runs
// a facade smoke through a bounded recorder and writes an 800x800 stdlib
// placeholder PNG when -screenshot is supplied. With a display (or xvfb-run)
// it opens an 800x800 window and saves real framebuffer shots.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/render"
	"github.com/draxxris/rtgui/ui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	// questDesignWidth and questDesignHeight are the fixed logical layout size.
	questDesignWidth  int32 = 800
	questDesignHeight int32 = 800
	// questWindowWidth and questWindowHeight are the default physical window size.
	questWindowWidth  int32 = 800
	questWindowHeight int32 = 800
	// questWindowMinSize keeps the journal usable while allowing resize.
	questWindowMinSize int32 = 640
	// questLayoutMargin is the synthetic layout inset around the journal
	// root. The viewport stays full-window; this margin is authored in
	// the slot table so the backdrop remains visible.
	questLayoutMargin float32 = 24
	// A real framebuffer is available after the first presentation cycle.
	questScreenshotWarmupFrames = 3
)

var (
	questFrames = flag.Int("frames", 0, "close after this many frames (0 means run until closed)")
	questShot   = flag.String("screenshot", "", "save the final frame to this PNG path")
)

// main opens the quest journal when a display is available and otherwise
// runs the headless interaction smoke.
func main() {
	flag.Parse()
	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(questWindowWidth, questWindowHeight, "Quests — rtgui MMORPG journal")
	if !rl.IsWindowReady() {
		log.Printf("raylib window not ready (no display) — running headless smoke")
		runQuestHeadlessSmoke()
		return
	}
	defer rl.CloseWindow()
	rl.SetWindowMinSize(int(questWindowMinSize), int(questWindowMinSize))
	rl.SetTargetFPS(60)
	facade := ui.New(int(questDesignWidth), int(questDesignHeight))
	facade.Theme().SetPixelSnap(true)
	cssPath := findQuestCSS()
	if cssPath == "" {
		log.Fatal("quest css not found; expected testdata/skins/quest.css (searched cwd and exe parents)")
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
	journal := newQuestWindow(facade)
	defer journal.unloadArtwork()
	for frame := 0; !rl.WindowShouldClose(); frame++ {
		facade.Resize(rl.GetScreenWidth(), rl.GetScreenHeight())
		journal.handleQuestInput()
		journal.drawQuestWindow()
		if *questFrames > 0 && frame+1 >= *questFrames {
			if *questShot != "" {
				for frame+1 < questScreenshotWarmupFrames {
					journal.drawQuestWindow()
					frame++
				}
				if err := saveQuestScreenshot(*questShot); err != nil {
					log.Printf("screenshot failed: %v", err)
				}
			}
			break
		}
	}
}

// handleQuestInput forwards raylib input through the UI facade on the owning
// goroutine. The journal needs no per-frame animation; scoped hotkeys and
// widget gestures arrive through PollRaylibInput.
func (q *questWindow) handleQuestInput() {
	if q == nil || q.facade == nil {
		return
	}
	_ = q.facade.PollRaylibInput()
}

// drawQuestWindow renders the logical journal 1:1. The viewport equals the
// window, so no host matrix is needed; the synthetic layout margin around
// the root leaves the backdrop visible.
func (q *questWindow) drawQuestWindow() {
	if q == nil || q.facade == nil {
		return
	}
	rl.BeginDrawing()
	rl.ClearBackground(color.RGBA{R: 20, G: 29, B: 42, A: 255})
	q.facade.Draw()
	rl.EndDrawing()
}

// runQuestHeadlessSmoke exercises the journal facade without a GL context.
// It drives tab filtering, leaf selection, link activation, and the four
// bottom actions through synthetic input and a bounded recorder, then writes
// a placeholder PNG when requested.
func runQuestHeadlessSmoke() {
	facade := ui.New(int(questDesignWidth), int(questDesignHeight))
	recorder, err := render.NewDrawRecorder(128)
	if err != nil {
		log.Fatal(err)
	}
	facade.Theme().SetDrawRecorder(recorder)
	journal := newQuestWindow(facade)
	facade.SelectTab("questTabs", 1)
	facade.Draw()
	facade.SelectTab("questTabs", 0)
	facade.Draw()
	journal.list.Select("troubled-farmstead")
	journal.updateQuestDetails("troubled-farmstead")
	facade.ActivateLink("questBody", 0)
	facade.ActivateLink("questRewards", 0)
	pressQuestButton(facade, "showMapButton")
	pressQuestButton(facade, "shareButton")
	pressQuestButton(facade, "trackButton")
	pressQuestButton(facade, "abandonButton")
	pressQuestButton(facade, "questWindow/close")
	pressQuestButton(facade, "reopenButton")
	facade.Draw()
	calls := recorder.Calls()
	log.Printf("quest headless smoke: %d draw calls (status=%q fallback=%v)", len(calls), journal.status, len(calls) > 0 && calls[0].Fallback)
	if *questShot != "" {
		if err := saveQuestScreenshot(*questShot); err != nil {
			log.Printf("headless placeholder failed: %v", err)
			os.Exit(1)
		}
	}
	if *questFrames > 0 {
		log.Printf("quest headless smoke completed %d frames", *questFrames)
	}
}

// pressQuestButton presses and releases one journal button by center. The
// helper shares one UI click path with the windowed build and the tests.
func pressQuestButton(facade *ui.UI, name string) {
	if facade == nil {
		return
	}
	w := facade.Lookup(name)
	if w == nil {
		return
	}
	bounds := w.Bounds()
	center := core.Vec2{X: bounds.X + bounds.W/2, Y: bounds.Y + bounds.H/2}
	facade.HandleMouse(ui.MouseEvent{Pos: center, Pressed: true})
	facade.HandleMouse(ui.MouseEvent{Pos: center, Released: true})
}

// saveQuestScreenshot saves the framebuffer or a placeholder when headless.
// Unlike rl.TakeScreenshot it handles absolute paths; both call sites share
// this so window and headless smoke stay one-liners. Headless callers skip
// the GPU path entirely because LoadImageFromScreen segfaults without GL.
func saveQuestScreenshot(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if rl.IsWindowReady() {
		if err := saveQuestFrame(path); err == nil {
			log.Printf("screenshot saved to %s", path)
			return nil
		} else {
			log.Printf("screenshot via GPU failed (%v) — writing placeholder", err)
		}
	}
	if err := createQuestPlaceholder(path); err != nil {
		return err
	}
	log.Printf("placeholder screenshot saved to %s", path)
	return nil
}

// saveQuestFrame saves the current framebuffer via LoadImageFromScreen plus
// ExportImage, which handles absolute and relative paths alike.
func saveQuestFrame(path string) error {
	img := rl.LoadImageFromScreen()
	if img == nil {
		return fmt.Errorf("LoadImageFromScreen returned nil")
	}
	defer rl.UnloadImage(img)
	if !rl.ExportImage(*img, path) {
		return fmt.Errorf("ExportImage failed for %q", path)
	}
	return nil
}

// createQuestPlaceholder writes a default-size stdlib PNG sketching the
// logical quest layout so headless -screenshot remains viewable without GL.
// Coordinates are 1:1 with the synthetic-margin layout.
func createQuestPlaceholder(path string) error {
	w, h := int(questWindowWidth), int(questWindowHeight)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	fill := func(r image.Rectangle, c color.RGBA) {
		draw.Draw(img, r.Intersect(img.Bounds()), &image.Uniform{C: c}, image.Point{}, draw.Src)
	}
	fill(image.Rect(0, 0, w, h), color.RGBA{R: 20, G: 29, B: 42, A: 255})
	fill(image.Rect(24, 24, 776, 776), color.RGBA{R: 16, G: 21, B: 31, A: 255})
	fill(image.Rect(24, 24, 776, 62), color.RGBA{R: 46, G: 90, B: 138, A: 255})
	fill(image.Rect(40, 84, 288, 660), color.RGBA{R: 26, G: 29, B: 36, A: 255})
	fill(image.Rect(300, 84, 700, 660), color.RGBA{R: 230, G: 211, B: 163, A: 255})
	fill(image.Rect(300, 100, 700, 130), color.RGBA{R: 210, G: 188, B: 138, A: 255})
	for i := 0; i < 3; i++ {
		fill(image.Rect(40+i*236, 676, 270+i*236, 712), color.RGBA{R: 42, G: 74, B: 107, A: 255})
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// findQuestCSS locates testdata/skins/quest.css, mirroring the gallery font
// lookup. It searches the working directory and executable parents upward.
func findQuestCSS() string {
	return findQuestFile(filepath.Join("testdata", "skins", "quest.css"), filepath.Join("rtgui", "testdata", "skins", "quest.css"))
}

// findQuestFile returns the first existing file matching one of rel,
// searching upward from the working directory and executable directory.
func findQuestFile(rel ...string) string {
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
			for _, r := range rel {
				if path := filepath.Join(dir, r); isQuestFile(path) {
					return path
				}
			}
		}
	}
	return ""
}

// isQuestFile reports whether path is an existing regular file.
func isQuestFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
