package render

import (
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	"github.com/draxxris/rtgui/transform"
)

type fakeTextureBackend struct {
	isReady     bool
	nextID      uint32
	uploadCalls int
	failUpload  int
	filterCalls int
	failFilter  int
	unloaded    []uint32
	uploaded    [][]byte
}

func (f *fakeTextureBackend) ready() bool { return f.isReady }

// upload records original bytes and can inject a deterministic failure.
func (f *fakeTextureBackend) upload(_ string, data []byte) (skin.Texture, error) {
	f.uploadCalls++
	if f.uploadCalls == f.failUpload {
		return skin.Texture{}, errors.New("injected upload failure")
	}
	f.nextID++
	f.uploaded = append(f.uploaded, append([]byte(nil), data...))
	return skin.Texture{ID: f.nextID, Width: 2, Height: 2, Mipmaps: 1, Format: 7}, nil
}

func (f *fakeTextureBackend) setFilter(skin.Texture) error {
	f.filterCalls++
	if f.filterCalls == f.failFilter {
		return errors.New("injected filter failure")
	}
	return nil
}

func (f *fakeTextureBackend) unload(texture skin.Texture) {
	f.unloaded = append(f.unloaded, texture.ID)
}

func newFakeTheme(backend *fakeTextureBackend) *Theme {
	theme := NewTheme(transform.New(core.Viewport{}))
	theme.textureBackend = backend
	return theme
}

// writeTestPNG creates a small tracked-independent CSS source image.
func writeTestPNG(t *testing.T, directory, name string, fill color.RGBA) string {
	t.Helper()
	path := filepath.Join(directory, name)
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			img.SetRGBA(x, y, fill)
		}
	}
	if err := png.Encode(file, img); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

// writeCSS writes one temporary candidate stylesheet.
func writeCSS(t *testing.T, directory, text string) string {
	t.Helper()
	path := filepath.Join(directory, "theme.css")
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func mustLoadCSS(t *testing.T, theme *Theme, path string) {
	t.Helper()
	if err := theme.LoadCSSFile(path, ""); err != nil {
		t.Fatal(err)
	}
}

// TestMergeSkinRules checks field merging and normal-state inheritance.
func TestMergeSkinRules(t *testing.T) {
	rules := []skin.SkinRule{
		{Kind: core.WidgetButton, Part: skin.PartBorder, State: core.StateNormal,
			Image: "line.png", HasImage: true, Slice: 8, HasSlice: true,
			Padding: [4]float32{8, 8, 8, 8}, HasPadding: true},
		{Kind: core.WidgetButton, Part: skin.PartBorder, State: core.StateHovered,
			Image: "hover.png", HasImage: true},
		{Kind: core.WidgetButton, Part: skin.PartBorder, State: core.StateDisabled,
			Tint: core.Color{R: 185, G: 185, B: 195, A: 255}, HasTint: true},
	}
	merged, order := mergeSkinRules(rules)
	if len(order) != 3 {
		t.Fatalf("order=%v", order)
	}
	hover := merged[skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBorder, State: core.StateHovered}]
	if hover.image != "hover.png" || !hover.hasSlice || hover.slice != 8 || !hover.hasPadding {
		t.Fatalf("hover inheritance = %+v", hover)
	}
	disabled := merged[skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBorder, State: core.StateDisabled}]
	if !disabled.hasImage || disabled.image != "line.png" || !disabled.hasTint {
		t.Fatalf("disabled inheritance = %+v", disabled)
	}
}

// TestLoadRuleImagesValidatesGeneratedAssetsBeforeUpload checks decode staging.
func TestLoadRuleImagesValidatesGeneratedAssetsBeforeUpload(t *testing.T) {
	directory := t.TempDir()
	writeTestPNG(t, directory, "pixel.png", color.RGBA{R: 255, A: 255})
	key := skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBackground, State: core.StateNormal}
	images, order, err := loadRuleImages(directory, map[skin.SkinKey]mergedRule{key: {image: "pixel.png", hasImage: true}}, []skin.SkinKey{key})
	if err != nil || len(images) != 1 || len(order) != 1 || len(images[order[0]].data) == 0 {
		t.Fatalf("generated image validation = %d/%v", len(images), err)
	}
	if _, _, err := loadRuleImages(directory, map[skin.SkinKey]mergedRule{key: {image: "missing.png", hasImage: true}}, []skin.SkinKey{key}); err == nil {
		t.Fatal("missing image must fail")
	}
	if _, _, err := loadRuleImages(directory, map[skin.SkinKey]mergedRule{key: {hasTint: true}}, []skin.SkinKey{key}); err == nil {
		t.Fatal("tint without image must fail")
	}
}

// TestCSSUploadsUniqueSourceOnceAndPreservesDrawTint checks sharing and tinting.
func TestCSSUploadsUniqueSourceOnceAndPreservesDrawTint(t *testing.T) {
	directory := t.TempDir()
	originalPath := writeTestPNG(t, directory, "pixel.png", color.RGBA{R: 200, G: 100, A: 255})
	cssPath := writeCSS(t, directory, `
Button { background-image: url(pixel.png); background-image-tint: #11223344; }
Button:hover { background-image-tint: #aabbccdd; }
`)
	backend := &fakeTextureBackend{isReady: true}
	theme := newFakeTheme(backend)
	mustLoadCSS(t, theme, cssPath)
	if backend.uploadCalls != 1 || backend.filterCalls != 1 || len(theme.ownedSkinTextures) != 1 {
		t.Fatalf("uploads=%d filters=%d owned=%d", backend.uploadCalls, backend.filterCalls, len(theme.ownedSkinTextures))
	}
	original, _ := os.ReadFile(originalPath)
	if len(backend.uploaded) != 1 || string(backend.uploaded[0]) != string(original) {
		t.Fatal("backend did not receive original encoded bytes")
	}
	normalKey := skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBackground, State: core.StateNormal}
	hoverKey := skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBackground, State: core.StateHovered}
	normal, err := theme.GetSkinPart(normalKey)
	if err != nil {
		t.Fatal(err)
	}
	hover, err := theme.GetSkinPart(hoverKey)
	if err != nil {
		t.Fatal(err)
	}
	if normal.Texture.ID != hover.Texture.ID || normal.Tint != (core.Color{R: 0x11, G: 0x22, B: 0x33, A: 0x44}) || hover.Tint != (core.Color{R: 0xaa, G: 0xbb, B: 0xcc, A: 0xdd}) {
		t.Fatalf("normal=%+v hover=%+v", normal, hover)
	}
	recorder, _ := NewDrawRecorder(4)
	theme.SetDrawRecorder(recorder)
	theme.BeginFrame()
	theme.DrawWidgetPart(core.WidgetButton, skin.PartBackground, core.Rect{W: 10, H: 10}, core.StateHovered)
	calls := recorder.Calls()
	if len(calls) != 1 || calls[0].Tint != hover.Tint {
		t.Fatalf("draw-time tint calls = %+v", calls)
	}
}

// TestCSSUploadFailureRollsBackCandidateAndPreservesOldLayer checks atomic failure.
func TestCSSUploadFailureRollsBackCandidateAndPreservesOldLayer(t *testing.T) {
	directory := t.TempDir()
	writeTestPNG(t, directory, "old.png", color.RGBA{R: 1, A: 255})
	writeTestPNG(t, directory, "first.png", color.RGBA{R: 2, A: 255})
	writeTestPNG(t, directory, "second.png", color.RGBA{R: 3, A: 255})
	backend := &fakeTextureBackend{isReady: true}
	theme := newFakeTheme(backend)
	oldCSS := writeCSS(t, directory, `Button { background-image: url(old.png); }`)
	mustLoadCSS(t, theme, oldCSS)
	key := skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBackground, State: core.StateNormal}
	old, _ := theme.GetSkinPart(key)
	backend.failUpload = 3
	candidateCSS := writeCSS(t, directory, `
Button { background-image: url(first.png); }
Checkbox { background-image: url(second.png); }
`)
	if err := theme.LoadCSSFile(candidateCSS, ""); err == nil {
		t.Fatal("injected second candidate upload must fail")
	}
	current, _ := theme.GetSkinPart(key)
	if current != old || len(theme.ownedSkinTextures) != 1 || theme.ownedSkinTextures[0].ID != old.Texture.ID {
		t.Fatalf("old layer changed: old=%+v current=%+v owned=%v", old, current, theme.ownedSkinTextures)
	}
	if len(backend.unloaded) != 1 || backend.unloaded[0] == old.Texture.ID {
		t.Fatalf("candidate rollback unloads = %v", backend.unloaded)
	}
}

// TestSuccessfulCSSReloadUnloadsEachPriorTexture checks repeated ownership swaps.
func TestSuccessfulCSSReloadUnloadsEachPriorTexture(t *testing.T) {
	directory := t.TempDir()
	writeTestPNG(t, directory, "pixel.png", color.RGBA{R: 1, A: 255})
	cssPath := writeCSS(t, directory, `Button { background-image: url(pixel.png); }`)
	backend := &fakeTextureBackend{isReady: true}
	theme := newFakeTheme(backend)
	for reload := 0; reload < 3; reload++ {
		mustLoadCSS(t, theme, cssPath)
	}
	if backend.uploadCalls != 3 || len(backend.unloaded) != 2 || backend.unloaded[0] != 1 || backend.unloaded[1] != 2 {
		t.Fatalf("uploads=%d unloads=%v", backend.uploadCalls, backend.unloaded)
	}
	if len(theme.ownedSkinTextures) != 1 || theme.ownedSkinTextures[0].ID != 3 {
		t.Fatalf("retained handles = %v", theme.ownedSkinTextures)
	}
}

// TestUnloadAndClearSkinNeverUnloadBorrowedTextures checks cleanup contracts.
func TestUnloadAndClearSkinNeverUnloadBorrowedTextures(t *testing.T) {
	directory := t.TempDir()
	writeTestPNG(t, directory, "pixel.png", color.RGBA{R: 1, A: 255})
	cssPath := writeCSS(t, directory, `Button { background-image: url(pixel.png); }`)
	backend := &fakeTextureBackend{isReady: true}
	theme := newFakeTheme(backend)
	key := skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBackground, State: core.StateNormal}
	borrowed := skin.SkinDescriptor{Texture: skin.Texture{ID: 999}, HasTexture: true, Tint: core.Color{A: 255}}
	theme.SetSkinPart(key, borrowed)
	mustLoadCSS(t, theme, cssPath)
	laterBorrowed := borrowed
	laterBorrowed.Texture.ID = 1000
	theme.SetSkinPart(key, laterBorrowed)
	if got, _ := theme.GetSkinPart(key); got.Texture.ID == laterBorrowed.Texture.ID {
		t.Fatal("later programmatic write was not hidden by materialized CSS")
	}
	if err := theme.UnloadSkin(); err != nil {
		t.Fatal(err)
	}
	if got, _ := theme.GetSkinPart(key); got != laterBorrowed {
		t.Fatalf("programmatic layer after unload = %+v", got)
	}
	if err := theme.UnloadSkin(); err != nil || len(backend.unloaded) != 1 {
		t.Fatalf("repeated UnloadSkin = %v unloads=%v", err, backend.unloaded)
	}
	mustLoadCSS(t, theme, cssPath)
	backend.isReady = false
	if err := theme.UnloadSkin(); err == nil || len(theme.ownedSkinTextures) != 1 {
		t.Fatal("headless cleanup must fail and retain owned handles")
	}
	backend.isReady = true
	if err := theme.ClearSkin(); err != nil {
		t.Fatal(err)
	}
	if _, err := theme.GetSkinPart(key); err == nil {
		t.Fatal("ClearSkin retained a descriptor")
	}
	if err := theme.ClearSkin(); err != nil {
		t.Fatal(err)
	}
	for _, id := range backend.unloaded {
		if id == 999 || id == 1000 {
			t.Fatalf("borrowed texture %d was unloaded", id)
		}
	}
}

// TestThemeLookupOrderAcrossCSSAndProgrammaticLayers checks exact/fallback order.
// CSS normal wins over a programmatic exact match so unauthored states inherit
// their base rule instead of borrowed art.
func TestThemeLookupOrderAcrossCSSAndProgrammaticLayers(t *testing.T) {
	theme := NewTheme(transform.New(core.Viewport{}))
	key := func(state core.WidgetState) skin.SkinKey {
		return skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBackground, State: state}
	}
	descriptor := func(red uint8) skin.SkinDescriptor { return skin.SkinDescriptor{Tint: core.Color{R: red, A: 255}} }
	theme.SetSkinPart(key(core.StateNormal), descriptor(1))
	theme.SetSkinPart(key(core.StateHovered), descriptor(2))
	theme.css.Set(key(core.StateNormal), descriptor(3))
	theme.css.Set(key(core.StatePressed), descriptor(4))
	for state, want := range map[core.WidgetState]uint8{
		core.StateNormal: 3, core.StateHovered: 3, core.StatePressed: 4, core.StateFocused: 3,
	} {
		got, ok := theme.Lookup(core.WidgetButton, skin.PartBackground, state)
		if !ok || got.Tint.R != want {
			t.Fatalf("Lookup(%v) = %+v/%v, want red %d", state, got, ok, want)
		}
	}
}

// TestCheckboxHoverInheritsBaseCSS verifies unauthored checkbox states resolve
// to the base kenney texture and draw without fallback.
func TestCheckboxHoverInheritsBaseCSS(t *testing.T) {
	directory := t.TempDir()
	writeTestPNG(t, directory, "box.png", color.RGBA{R: 10, A: 255})
	writeTestPNG(t, directory, "check.png", color.RGBA{R: 20, A: 255})
	cssPath := writeCSS(t, directory, `
Checkbox::box { background-image: url(box.png); }
Checkbox::checkmark { background-image: url(check.png); }
Checkbox::box:disabled { background-image-tint: #b9b9c3; }
Checkbox::checkmark:disabled { background-image-tint: #b9b9c3; }
`)
	backend := &fakeTextureBackend{isReady: true}
	theme := newFakeTheme(backend)
	mustLoadCSS(t, theme, cssPath)
	boxNormal, err := theme.GetSkinPart(skin.SkinKey{Widget: core.WidgetCheckbox, Part: skin.PartIcon, State: core.StateNormal})
	if err != nil {
		t.Fatal(err)
	}
	boxHover, ok := theme.Lookup(core.WidgetCheckbox, skin.PartIcon, core.StateHovered)
	if !ok || boxHover.Texture.ID != boxNormal.Texture.ID {
		t.Fatalf("box hover=%+v/%v normal=%+v", boxHover, ok, boxNormal)
	}
	checkNormal, err := theme.GetSkinPart(skin.SkinKey{Widget: core.WidgetCheckbox, Part: skin.PartCheckmark, State: core.StateNormal})
	if err != nil {
		t.Fatal(err)
	}
	checkHover, ok := theme.Lookup(core.WidgetCheckbox, skin.PartCheckmark, core.StateHovered)
	if !ok || checkHover.Texture.ID != checkNormal.Texture.ID {
		t.Fatalf("check hover=%+v/%v normal=%+v", checkHover, ok, checkNormal)
	}
	boxDisabled, ok := theme.Lookup(core.WidgetCheckbox, skin.PartIcon, core.StateDisabled)
	if !ok || boxDisabled.Texture.ID != boxNormal.Texture.ID || boxDisabled.Tint != (core.Color{R: 0xb9, G: 0xb9, B: 0xc3, A: 255}) {
		t.Fatalf("box disabled=%+v/%v", boxDisabled, ok)
	}
	recorder, _ := NewDrawRecorder(8)
	theme.SetDrawRecorder(recorder)
	theme.BeginFrame()
	box := core.Rect{X: 10, Y: 10, W: 120, H: 40}
	checked := core.WidgetInfo{Name: "agreeBox", Bounds: box, Kind: core.WidgetCheckbox, State: core.StateHovered}
	theme.DrawWidget(checked, "", 0, true)
	found := false
	for _, call := range recorder.Calls() {
		if call.Part == skin.PartCheckmark && call.State == core.StateHovered {
			found = true
			if call.Fallback {
				t.Fatalf("hover checkmark must not fall back: %+v", call)
			}
		}
	}
	if !found {
		t.Fatalf("hover checkmark not drawn: %+v", recorder.Calls())
	}
}

// TestLoadCSSFileRequiresReadyBackendOnlyForUpload checks context validation.
func TestLoadCSSFileRequiresReadyBackendOnlyForUpload(t *testing.T) {
	directory := t.TempDir()
	writeTestPNG(t, directory, "pixel.png", color.RGBA{A: 255})
	cssPath := writeCSS(t, directory, `Button { background-image: url(pixel.png); }`)
	theme := newFakeTheme(&fakeTextureBackend{isReady: false})
	if err := theme.LoadCSSFile(cssPath, ""); err == nil {
		t.Fatal("unready backend must reject upload")
	}
	var nilTheme *Theme
	if err := nilTheme.LoadCSSFile(cssPath, ""); err == nil {
		t.Fatal("nil theme must fail")
	}
}

// TestScrollbarThreePatchCSS verifies ScrollPanel track and thumb descriptors support 3-patch.
func TestScrollbarThreePatchCSS(t *testing.T) {
	directory := t.TempDir()
	writeTestPNG(t, directory, "track.png", color.RGBA{R: 20, G: 30, B: 40, A: 255})
	writeTestPNG(t, directory, "thumb.png", color.RGBA{R: 100, G: 110, B: 120, A: 255})
	cssText := `
		ScrollPanel::track {
			border-image-source: url("track.png");
			border-image-slice: 8;
		}
		ScrollPanel::thumb {
			border-image-source: url("thumb.png");
			border-image-slice: 8;
		}
		ScrollPanel::thumb:hover {
			border-image-source-tint: #E1F2FF;
		}
	`
	cssPath := writeCSS(t, directory, cssText)
	theme := newFakeTheme(&fakeTextureBackend{isReady: true})
	if err := theme.LoadCSSFile(cssPath, ""); err != nil {
		t.Fatalf("LoadCSSFile: %v", err)
	}

	trackDesc, ok := theme.Lookup(core.WidgetScrollPanel, skin.PartTrack, core.StateNormal)
	if !ok || !trackDesc.HasThreePatch || trackDesc.ThreePatch.Top != 8 || trackDesc.ThreePatch.Bottom != 8 {
		t.Fatalf("track descriptor mismatch: ok=%v desc=%+v", ok, trackDesc)
	}

	thumbDesc, ok := theme.Lookup(core.WidgetScrollPanel, skin.PartThumb, core.StateNormal)
	if !ok || !thumbDesc.HasThreePatch || thumbDesc.ThreePatch.Top != 8 || thumbDesc.ThreePatch.Bottom != 8 {
		t.Fatalf("thumb descriptor mismatch: ok=%v desc=%+v", ok, thumbDesc)
	}

	thumbHover, ok := theme.Lookup(core.WidgetScrollPanel, skin.PartThumb, core.StateHovered)
	if !ok || !thumbHover.HasThreePatch || thumbHover.ThreePatch.Top != 8 || thumbHover.ThreePatch.Bottom != 8 {
		t.Fatalf("thumb hover descriptor mismatch: ok=%v desc=%+v", ok, thumbHover)
	}
	if thumbHover.Tint != (core.Color{R: 0xE1, G: 0xF2, B: 0xFF, A: 255}) {
		t.Fatalf("thumb hover tint mismatch: %v", thumbHover.Tint)
	}
}
