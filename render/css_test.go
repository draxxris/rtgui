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
			Image: "line.png", HasImage: true, Slice: [4]int32{8, 8, 8, 8}, HasSlice: true,
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
	if hover.image != "hover.png" || !hover.hasSlice || hover.slice != [4]int32{8, 8, 8, 8} || !hover.hasPadding {
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

// TestCSSLoadsLeadingTextureStack verifies render materialization preserves
// one uploaded texture and both CSS gradient layers in their source order.
func TestCSSLoadsLeadingTextureStack(t *testing.T) {
	directory := t.TempDir()
	writeTestPNG(t, directory, "surface.png", color.RGBA{R: 80, G: 100, B: 140, A: 255})
	cssPath := writeCSS(t, directory, `Button {
		background-image: url("surface.png"), radial-gradient(circle at 30% 20%, #ffffff66, #ffffff00 60%), linear-gradient(to bottom, #2b3d54aa, #0b1524ee);
	}`)
	backend := &fakeTextureBackend{isReady: true}
	theme := newFakeTheme(backend)
	mustLoadCSS(t, theme, cssPath)
	key := skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBackground, State: core.StateNormal}
	descriptor, err := theme.GetSkinPart(key)
	if err != nil {
		t.Fatal(err)
	}
	if !descriptor.HasTexture || descriptor.Texture.ID == 0 || descriptor.GradientCount != 2 {
		t.Fatalf("materialized mixed stack = %+v", descriptor)
	}
	if backend.uploadCalls != 1 || descriptor.Gradients[0].Kind != skin.GradientRadial || descriptor.Gradients[1].Kind != skin.GradientLinear {
		t.Fatalf("uploads/layer order = %d/%+v", backend.uploadCalls, descriptor.Gradients)
	}
	if descriptor.Gradients[0].Stops[0].Color.A != 0x66 || descriptor.Gradients[1].Stops[0].Color.A != 0xaa {
		t.Fatalf("materialized alpha = %+v/%+v", descriptor.Gradients[0].Stops[0], descriptor.Gradients[1].Stops[0])
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

// assertThreePatchDescriptor verifies that a descriptor has valid 3-patch slicing.
func assertThreePatchDescriptor(t *testing.T, desc skin.SkinDescriptor, ok bool, capVal int32) {
	t.Helper()
	if !ok || !desc.HasThreePatch || desc.ThreePatch.Top != capVal || desc.ThreePatch.Bottom != capVal {
		t.Fatalf("three-patch descriptor mismatch: ok=%v desc=%+v", ok, desc)
	}
}

// TestScrollbarThreePatchCSS verifies ScrollPanel track and thumb descriptors support 3-patch.
func TestScrollbarThreePatchCSS(t *testing.T) {
	directory := t.TempDir()
	writeTestPNG(t, directory, "panel.png", color.RGBA{R: 50, G: 60, B: 70, A: 255})
	writeTestPNG(t, directory, "track.png", color.RGBA{R: 20, G: 30, B: 40, A: 255})
	writeTestPNG(t, directory, "thumb.png", color.RGBA{R: 100, G: 110, B: 120, A: 255})
	cssText := `
		ScrollPanel {
			border-image-source: url("panel.png");
			border-image-slice: 8;
			padding: 8;
		}
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

	borderDesc, ok := theme.Lookup(core.WidgetScrollPanel, skin.PartBorder, core.StateNormal)
	if !ok || !borderDesc.HasNinePatch || borderDesc.NinePatch.Left != 8 {
		t.Fatalf("panel border descriptor mismatch: ok=%v desc=%+v", ok, borderDesc)
	}
	content := theme.ScrollContent(core.Rect{X: 100, Y: 100, W: 200, H: 200}, core.StateNormal)
	if expected := (core.Rect{X: 108, Y: 108, W: 184, H: 184}); content != expected {
		t.Fatalf("ScrollContent mismatch: got %v, want %v", content, expected)
	}

	trackDesc, ok := theme.Lookup(core.WidgetScrollPanel, skin.PartTrack, core.StateNormal)
	assertThreePatchDescriptor(t, trackDesc, ok, 8)

	thumbDesc, ok := theme.Lookup(core.WidgetScrollPanel, skin.PartThumb, core.StateNormal)
	assertThreePatchDescriptor(t, thumbDesc, ok, 8)

	thumbHover, ok := theme.Lookup(core.WidgetScrollPanel, skin.PartThumb, core.StateHovered)
	assertThreePatchDescriptor(t, thumbHover, ok, 8)
	if thumbHover.Tint != (core.Color{R: 0xE1, G: 0xF2, B: 0xFF, A: 255}) {
		t.Fatalf("thumb hover tint mismatch: %v", thumbHover.Tint)
	}
}

// TestNinePatchPerSideCSS verifies a 4-value slice maps top/right/bottom/left.
func TestNinePatchPerSideCSS(t *testing.T) {
	directory := t.TempDir()
	writeTestPNG(t, directory, "frame.png", color.RGBA{R: 50, G: 60, B: 70, A: 255})
	cssText := `Frame { border-image-source: url("frame.png"); border-image-slice: 95 75 75 75; }`
	cssPath := writeCSS(t, directory, cssText)
	theme := newFakeTheme(&fakeTextureBackend{isReady: true})
	if err := theme.LoadCSSFile(cssPath, ""); err != nil {
		t.Fatalf("LoadCSSFile: %v", err)
	}
	desc, ok := theme.Lookup(core.WidgetFrame, skin.PartBorder, core.StateNormal)
	if !ok || !desc.HasNinePatch {
		t.Fatalf("frame border missing: ok=%v desc=%+v", ok, desc)
	}
	if desc.NinePatch.Top != 95 || desc.NinePatch.Right != 75 || desc.NinePatch.Bottom != 75 || desc.NinePatch.Left != 75 {
		t.Fatalf("per-side nine-patch = %+v", desc.NinePatch)
	}
	if desc.HasBorderWidth {
		t.Fatalf("destination must default to slice: %+v", desc)
	}
	if got := destBorders(desc); got.Top != 95 || got.Right != 75 || got.Bottom != 75 || got.Left != 75 {
		t.Fatalf("default dest borders = %+v", got)
	}
}

// TestNinePatchWidthCSS verifies border-image-width separates destination
// sizes from source slices for scaled frame art.
func TestNinePatchWidthCSS(t *testing.T) {
	directory := t.TempDir()
	writeTestPNG(t, directory, "frame.png", color.RGBA{R: 50, G: 60, B: 70, A: 255})
	cssText := `Frame { border-image-source: url("frame.png"); border-image-slice: 95 75 75 75; border-image-width: 64 45 45 45; }`
	cssPath := writeCSS(t, directory, cssText)
	theme := newFakeTheme(&fakeTextureBackend{isReady: true})
	if err := theme.LoadCSSFile(cssPath, ""); err != nil {
		t.Fatalf("LoadCSSFile: %v", err)
	}
	desc, ok := theme.Lookup(core.WidgetFrame, skin.PartBorder, core.StateNormal)
	if !ok || !desc.HasNinePatch || !desc.HasBorderWidth {
		t.Fatalf("frame border missing: ok=%v desc=%+v", ok, desc)
	}
	if desc.NinePatch.Top != 95 || desc.NinePatch.Left != 75 {
		t.Fatalf("source slice must stay 95/75: %+v", desc.NinePatch)
	}
	if desc.BorderWidth != [4]int32{64, 45, 45, 45} {
		t.Fatalf("dest width = %+v", desc.BorderWidth)
	}
	if got := destBorders(desc); got.Top != 64 || got.Right != 45 || got.Bottom != 45 || got.Left != 45 {
		t.Fatalf("dest borders = %+v", got)
	}
	content := ContentRect(core.Rect{W: 752, H: 752}, desc)
	if expected := (core.Rect{X: 45, Y: 64, W: 662, H: 643}); content != expected {
		t.Fatalf("ContentRect = %+v, want %+v", content, expected)
	}
}

// TestInnerGradientCSS verifies an inner layer survives the registry with
// its kind intact so the mesh dispatcher draws four-sided falloff.
func TestInnerGradientCSS(t *testing.T) {
	directory := t.TempDir()
	cssText := `List { background-image: inner-gradient(#7fb2f0B0, #1e3a5a00 70%), linear-gradient(to bottom, #3a6a9a, #1e3a5a); }`
	cssPath := writeCSS(t, directory, cssText)
	theme := newFakeTheme(&fakeTextureBackend{isReady: true})
	if err := theme.LoadCSSFile(cssPath, ""); err != nil {
		t.Fatalf("LoadCSSFile: %v", err)
	}
	desc, ok := theme.Lookup(core.WidgetList, skin.PartBackground, core.StateNormal)
	if !ok || desc.GradientCount != 2 {
		t.Fatalf("list background missing stack: ok=%v desc=%+v", ok, desc)
	}
	if desc.Gradients[0].Kind != skin.GradientInner {
		t.Fatalf("top kind = %v", desc.Gradients[0].Kind)
	}
}

// TestCSSBackgroundColorAndGradientMergeAndInherit tests merge rules and state inheritance for color and gradient.
func TestCSSBackgroundColorAndGradientMergeAndInherit(t *testing.T) {
	rules := []skin.SkinRule{
		{
			Kind: core.WidgetButton, Part: skin.PartBackground, State: core.StateNormal,
			BackgroundColor:    core.Color{R: 0x10, G: 0x20, B: 0x30, A: 0xFF},
			HasBackgroundColor: true,
			Gradients: [skin.MaxGradientLayers]skin.LinearGradient{{
				Direction: skin.GradientToBottom,
				Stops: [skin.MaxGradientStops]skin.ColorStop{
					{Color: core.Color{R: 0x00, G: 0x11, B: 0x22, A: 0x80}, Position: 0},
					{Color: core.Color{R: 0x33, G: 0x44, B: 0x55, A: 0x80}, Position: 1},
				},
				StopCount: 2,
			}},
			GradientCount: 1,
		},
		{
			Kind: core.WidgetButton, Part: skin.PartBackground, State: core.StateHovered,
			BackgroundColor:    core.Color{R: 0x50, G: 0x60, B: 0x70, A: 0xFF},
			HasBackgroundColor: true,
		},
		{
			Kind: core.WidgetButton, Part: skin.PartBackground, State: core.StatePressed,
			Gradients: [skin.MaxGradientLayers]skin.LinearGradient{{
				Direction: skin.GradientToTop,
				Stops: [skin.MaxGradientStops]skin.ColorStop{
					{Color: core.Color{R: 0xAA, G: 0xBB, B: 0xCC, A: 0xFF}, Position: 0},
					{Color: core.Color{R: 0xDD, G: 0xEE, B: 0xFF, A: 0xFF}, Position: 1},
				},
				StopCount: 2,
			}},
			GradientCount: 1,
		},
	}
	merged, _ := mergeSkinRules(rules)
	hoverKey := skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBackground, State: core.StateHovered}
	hover := merged[hoverKey]
	if !hover.hasBackgroundColor || hover.backgroundColor != (core.Color{R: 0x50, G: 0x60, B: 0x70, A: 0xFF}) {
		t.Fatalf("hover background-color mismatch: %+v", hover)
	}
	if hover.gradientCount != 1 || hover.gradients[0].Direction != skin.GradientToBottom {
		t.Fatalf("hover inherited gradient mismatch: %+v", hover)
	}

	pressKey := skin.SkinKey{Widget: core.WidgetButton, Part: skin.PartBackground, State: core.StatePressed}
	press := merged[pressKey]
	if !press.hasBackgroundColor || press.backgroundColor != (core.Color{R: 0x10, G: 0x20, B: 0x30, A: 0xFF}) {
		t.Fatalf("pressed inherited background-color mismatch: %+v", press)
	}
	if press.gradientCount != 1 || press.gradients[0].Direction != skin.GradientToTop {
		t.Fatalf("pressed gradient mismatch: %+v", press)
	}
}

// TestTitledNoneClearsInheritedTexture verifies class none rules drop base
// textures through Overlay while keeping their own background color.
func TestTitledNoneClearsInheritedTexture(t *testing.T) {
	rules, err := skin.ParseCSS(`
		Frame {
			background-image: url("bg.png");
			border-image-source: url("ring.png");
			border-image-slice: 8;
		}
		Frame.titled-titlebar {
			background-image: none;
			background-color: #24354c;
			border-image-source: none;
		}
		Frame.titled-titlebar:hover {
			background-color: #2c425e;
		}
	`)
	if err != nil {
		t.Fatal(err)
	}
	merged, _ := mergeSkinRules(rules)
	fakes := map[string]skin.Texture{
		"bg.png":   {ID: 7, Width: 64, Height: 64},
		"ring.png": {ID: 8, Width: 32, Height: 32},
	}
	theme := newFakeTheme(&fakeTextureBackend{isReady: false})
	background := func(key skin.SkinKey) skin.SkinDescriptor {
		t.Helper()
		desc, err := theme.buildCSSDescriptor(key, merged[key], ".", fakes)
		if err != nil {
			t.Fatal(err)
		}
		return desc
	}
	baseKey := skin.SkinKey{Widget: core.WidgetFrame, Part: skin.PartBackground, State: core.StateNormal}
	classKey := skin.SkinKey{Widget: core.WidgetFrame, Class: "titled-titlebar", Part: skin.PartBackground, State: core.StateNormal}
	resolved := background(baseKey).Overlay(background(classKey))
	if resolved.HasTexture || resolved.HasGradient() {
		t.Fatalf("none must drop inherited texture, got %+v", resolved)
	}
	if !resolved.HasBackgroundColor || resolved.BackgroundColor != (core.Color{R: 0x24, G: 0x35, B: 0x4c, A: 0xff}) {
		t.Fatalf("none must keep class color, got %+v", resolved)
	}
	borderBase := skin.SkinKey{Widget: core.WidgetFrame, Part: skin.PartBorder, State: core.StateNormal}
	borderClass := skin.SkinKey{Widget: core.WidgetFrame, Class: "titled-titlebar", Part: skin.PartBorder, State: core.StateNormal}
	resolvedBorder := background(borderBase).Overlay(background(borderClass))
	if resolvedBorder.HasTexture {
		t.Fatalf("border none must drop ring, got %+v", resolvedBorder)
	}
	hoverKey := skin.SkinKey{Widget: core.WidgetFrame, Class: "titled-titlebar", Part: skin.PartBackground, State: core.StateHovered}
	if entry := merged[hoverKey]; !entry.noTexture {
		t.Fatalf("hover must inherit clear, got %+v", entry)
	}
}

// TestCSSBackgroundColorAndGradientHeadlessLoadAndDraw verifies CSS files with color and gradient load headlessly and draw.
func TestCSSBackgroundColorAndGradientHeadlessLoadAndDraw(t *testing.T) {
	directory := t.TempDir()
	cssText := `
		Button {
			background-color: #20406080;
			background-image: linear-gradient(to bottom, #112233AA, #445566BB);
		}
		Button:hover {
			background-color: #306090;
		}
		Slider::track {
			background-color: #101010;
		}
	`
	cssPath := writeCSS(t, directory, cssText)
	theme := newFakeTheme(&fakeTextureBackend{isReady: false})
	if err := theme.LoadCSSFile(cssPath, ""); err != nil {
		t.Fatalf("LoadCSSFile failed on headless theme without images: %v", err)
	}

	btnDesc, ok := theme.Lookup(core.WidgetButton, skin.PartBackground, core.StateNormal)
	if !ok || !btnDesc.HasBackgroundColor || !btnDesc.HasGradient() {
		t.Fatalf("expected button descriptor with color and gradient, got: ok=%v desc=%+v", ok, btnDesc)
	}
	if btnDesc.BackgroundColor != (core.Color{R: 0x20, G: 0x40, B: 0x60, A: 0x80}) {
		t.Fatalf("button background-color mismatch: %v", btnDesc.BackgroundColor)
	}

	recorder := newTestRecorder(t, 8)
	theme.SetDrawRecorder(recorder)
	button := core.WidgetInfo{Name: "submit", Bounds: core.Rect{W: 100, H: 30}, Kind: core.WidgetButton, State: core.StateNormal}
	theme.BeginFrame()
	theme.DrawWidget(button, "Submit", 0, false)

	calls := recorder.Calls()
	if len(calls) < 2 {
		t.Fatalf("expected at least 2 draw calls, got %v", calls)
	}
	bgCall := calls[0]
	if bgCall.Part != skin.PartBackground || bgCall.Fallback {
		t.Fatalf("expected skinned background without fallback, got: %+v", bgCall)
	}
	if bgCall.Tint != (core.Color{R: 0x20, G: 0x40, B: 0x60, A: 0x80}) {
		t.Fatalf("expected alpha-preserved background tint in DrawCall, got: %v", bgCall.Tint)
	}
}

// TestThemeCSSClassLookupAndOverlay verifies CSS class lookup, specificity, and property overlay.
func TestThemeCSSClassLookupAndOverlay(t *testing.T) {
	theme := setupClassTestTheme(t)

	t.Run("BaseButton", func(t *testing.T) {
		assertBaseButton(t, theme)
	})
	t.Run("DangerClass", func(t *testing.T) {
		assertDangerClass(t, theme)
	})
	t.Run("PrimaryClass", func(t *testing.T) {
		assertPrimaryClass(t, theme)
	})
	t.Run("DrawWidgetWithClass", func(t *testing.T) {
		assertDrawWidgetWithClass(t, theme)
	})
}

// setupClassTestTheme creates a theme loaded with CSS defining base Button, .danger, and Button.primary rules.
func setupClassTestTheme(t *testing.T) *Theme {
	directory := t.TempDir()
	cssText := `
		Button {
			background-color: #112233;
			padding: 8;
		}
		Button:hover {
			background-color: #223344;
		}
		.danger {
			background-color: #FF0000;
		}
		.danger:hover {
			background-color: #AA0000;
		}
		Button.primary {
			background-color: #0000FF;
			padding: 12;
		}
	`
	cssPath := writeCSS(t, directory, cssText)
	theme := newFakeTheme(&fakeTextureBackend{isReady: false})
	if err := theme.LoadCSSFile(cssPath, ""); err != nil {
		t.Fatalf("LoadCSSFile failed: %v", err)
	}
	return theme
}

// assertBaseButton verifies normal and hover styles for the base Button selector.
func assertBaseButton(t *testing.T, theme *Theme) {
	normal, ok := theme.Lookup(core.WidgetButton, skin.PartBackground, core.StateNormal)
	if !ok || normal.BackgroundColor != (core.Color{R: 0x11, G: 0x22, B: 0x33, A: 0xFF}) || normal.PaddingLeft != 8 {
		t.Fatalf("base button normal mismatch: ok=%v, desc=%+v", ok, normal)
	}
	hover, ok := theme.Lookup(core.WidgetButton, skin.PartBackground, core.StateHovered)
	if !ok || hover.BackgroundColor != (core.Color{R: 0x22, G: 0x33, B: 0x44, A: 0xFF}) {
		t.Fatalf("base button hover mismatch: ok=%v, desc=%+v", ok, hover)
	}
}

// assertDangerClass verifies universal .danger class overrides and overlays onto base Button properties.
func assertDangerClass(t *testing.T, theme *Theme) {
	normal, ok := theme.Lookup(core.WidgetButton, skin.PartBackground, core.StateNormal, "danger")
	if !ok || normal.BackgroundColor != (core.Color{R: 0xFF, G: 0x00, B: 0x00, A: 0xFF}) || normal.PaddingLeft != 8 {
		t.Fatalf("button.danger normal mismatch: ok=%v, desc=%+v", ok, normal)
	}
	hover, ok := theme.Lookup(core.WidgetButton, skin.PartBackground, core.StateHovered, "danger")
	if !ok || hover.BackgroundColor != (core.Color{R: 0xAA, G: 0x00, B: 0x00, A: 0xFF}) || hover.PaddingLeft != 8 {
		t.Fatalf("button.danger hover mismatch: ok=%v, desc=%+v", ok, hover)
	}
}

// assertPrimaryClass verifies kind-scoped Button.primary class overrides.
func assertPrimaryClass(t *testing.T, theme *Theme) {
	normal, ok := theme.Lookup(core.WidgetButton, skin.PartBackground, core.StateNormal, "primary")
	if !ok || normal.BackgroundColor != (core.Color{R: 0x00, G: 0x00, B: 0xFF, A: 0xFF}) || normal.PaddingLeft != 12 {
		t.Fatalf("button.primary normal mismatch: ok=%v, desc=%+v", ok, normal)
	}
}

// assertDrawWidgetWithClass verifies DrawWidget applies the class style when rendering.
func assertDrawWidgetWithClass(t *testing.T, theme *Theme) {
	recorder := newTestRecorder(t, 8)
	theme.SetDrawRecorder(recorder)
	btn := core.WidgetInfo{Name: "del", Bounds: core.Rect{W: 100, H: 30}, Kind: core.WidgetButton, State: core.StateNormal, Class: "danger"}
	theme.BeginFrame()
	theme.DrawWidget(btn, "Delete", 0, false)

	calls := recorder.Calls()
	if len(calls) < 2 {
		t.Fatalf("expected at least 2 draw calls, got %v", calls)
	}
	if calls[0].Tint != (core.Color{R: 0xFF, G: 0x00, B: 0x00, A: 0xFF}) {
		t.Fatalf("expected red tint for danger button, got: %v", calls[0].Tint)
	}
}

// TestUniversalSelectorAndFontLoading verifies universal * cascade and CSS font loading.
func TestUniversalSelectorAndFontLoading(t *testing.T) {
	theme := setupUniversalTestTheme(t)

	t.Run("FontsLoaded", func(t *testing.T) {
		assertFontsLoaded(t, theme)
	})
	t.Run("UniversalDefaultOnLabel", func(t *testing.T) {
		assertUniversalLabel(t, theme)
	})
	t.Run("KindOverrideOnButton", func(t *testing.T) {
		assertKindButton(t, theme)
	})
	t.Run("ClassOverrideOnLabel", func(t *testing.T) {
		assertClassLabel(t, theme)
	})
	t.Run("ScopedClassOverrideOnButton", func(t *testing.T) {
		assertScopedButton(t, theme)
	})
}

// setupUniversalTestTheme initializes a theme loaded with universal selector CSS and fonts.
func setupUniversalTestTheme(t *testing.T) *Theme {
	directory := t.TempDir()
	cssText := `
		* {
			background-color: #101010;
			color: #eeeeee;
			font-size: 20;
			font-family: url("../fonts/Grenze-Regular.ttf");
			font-italic-family: url("../fonts/Grenze-Italic.ttf");
			padding: 4;
		}
		Button {
			background-color: #202020;
			padding: 8;
		}
		.danger {
			color: #ff0000;
		}
		Button.danger {
			background-color: #990000;
		}
	`
	cssPath := writeCSS(t, directory, cssText)
	theme := newFakeTheme(&fakeTextureBackend{isReady: false})
	if err := theme.LoadCSSFile(cssPath, "../testdata/skins"); err != nil {
		t.Fatalf("LoadCSSFile failed: %v", err)
	}
	return theme
}

// assertFontsLoaded checks that CSS font-family and font-italic-family were registered.
func assertFontsLoaded(t *testing.T, theme *Theme) {
	if !theme.HasFont() {
		t.Fatal("expected theme.HasFont() to be true after loading CSS font-family")
	}
	if !theme.HasItalicFont() {
		t.Fatal("expected theme.HasItalicFont() to be true after loading CSS font-italic-family")
	}
}

// assertUniversalLabel checks universal property inheritance on an unstyled widget kind.
func assertUniversalLabel(t *testing.T, theme *Theme) {
	desc, ok := theme.Lookup(core.WidgetLabel, skin.PartBackground, core.StateNormal)
	if !ok {
		t.Fatal("expected lookup for WidgetLabel to succeed via universal selector")
	}
	if desc.BackgroundColor != (core.Color{R: 0x10, G: 0x10, B: 0x10, A: 0xFF}) {
		t.Errorf("label background mismatch: got %+v", desc.BackgroundColor)
	}
	if desc.TextColor != (core.Color{R: 0xEE, G: 0xEE, B: 0xEE, A: 0xFF}) {
		t.Errorf("label text color mismatch: got %+v", desc.TextColor)
	}
	if desc.FontSize != 20 || desc.PaddingLeft != 4 {
		t.Errorf("label font size/padding mismatch: size=%v pad=%v", desc.FontSize, desc.PaddingLeft)
	}
}

// assertKindButton checks element kind overrides onto universal selector defaults.
func assertKindButton(t *testing.T, theme *Theme) {
	desc, ok := theme.Lookup(core.WidgetButton, skin.PartBackground, core.StateNormal)
	if !ok {
		t.Fatal("expected lookup for WidgetButton to succeed")
	}
	if desc.BackgroundColor != (core.Color{R: 0x20, G: 0x20, B: 0x20, A: 0xFF}) {
		t.Errorf("button background mismatch: got %+v", desc.BackgroundColor)
	}
	if desc.PaddingLeft != 8 {
		t.Errorf("button padding mismatch: got %v", desc.PaddingLeft)
	}
	if desc.TextColor != (core.Color{R: 0xEE, G: 0xEE, B: 0xEE, A: 0xFF}) {
		t.Errorf("button inherited text color mismatch: got %+v", desc.TextColor)
	}
	if desc.FontSize != 20 {
		t.Errorf("button inherited font size mismatch: got %v", desc.FontSize)
	}
}

// assertClassLabel checks universal class overrides on an element kind.
func assertClassLabel(t *testing.T, theme *Theme) {
	desc, ok := theme.Lookup(core.WidgetLabel, skin.PartBackground, core.StateNormal, "danger")
	if !ok {
		t.Fatal("expected lookup for WidgetLabel.danger to succeed")
	}
	if desc.TextColor != (core.Color{R: 0xFF, G: 0x00, B: 0x00, A: 0xFF}) {
		t.Errorf("label.danger text color mismatch: got %+v", desc.TextColor)
	}
	if desc.BackgroundColor != (core.Color{R: 0x10, G: 0x10, B: 0x10, A: 0xFF}) {
		t.Errorf("label.danger background mismatch: got %+v", desc.BackgroundColor)
	}
}

// assertScopedButton checks kind-scoped class overrides in the full cascade.
func assertScopedButton(t *testing.T, theme *Theme) {
	desc, ok := theme.Lookup(core.WidgetButton, skin.PartBackground, core.StateNormal, "danger")
	if !ok {
		t.Fatal("expected lookup for WidgetButton.danger to succeed")
	}
	if desc.BackgroundColor != (core.Color{R: 0x99, G: 0x00, B: 0x00, A: 0xFF}) {
		t.Errorf("button.danger background mismatch: got %+v", desc.BackgroundColor)
	}
	if desc.TextColor != (core.Color{R: 0xFF, G: 0x00, B: 0x00, A: 0xFF}) {
		t.Errorf("button.danger text color mismatch: got %+v", desc.TextColor)
	}
	if desc.PaddingLeft != 8 || desc.FontSize != 20 {
		t.Errorf("button.danger padding/size mismatch: pad=%v size=%v", desc.PaddingLeft, desc.FontSize)
	}
}
