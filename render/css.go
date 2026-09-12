// CSS file skinning: parse and mapping stay headless in skin, while this file
// validates encoded assets and transactionally owns uploaded textures.
package render

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/png" // Register PNG decoding for CSS asset validation.
	"os"
	"path/filepath"
	"strings"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/skin"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type textureBackend interface {
	ready() bool
	upload(fileType string, data []byte) (skin.Texture, error)
	setFilter(texture skin.Texture) error
	unload(texture skin.Texture)
}

type raylibTextureBackend struct{}

func (raylibTextureBackend) ready() bool { return rl.IsWindowReady() }

// upload decodes original encoded bytes in raylib and uploads one texture.
func (raylibTextureBackend) upload(fileType string, data []byte) (skin.Texture, error) {
	loaded := rl.LoadImageFromMemory(fileType, data, int32(len(data)))
	if loaded == nil {
		return skin.Texture{}, errors.New("raylib image decode failed")
	}
	defer rl.UnloadImage(loaded)
	uploaded := rl.LoadTextureFromImage(loaded)
	if uploaded.ID == 0 {
		return skin.Texture{}, errors.New("raylib texture upload failed")
	}
	return fromRaylibTexture(uploaded), nil
}

func (raylibTextureBackend) setFilter(texture skin.Texture) error {
	rl.SetTextureFilter(toRaylibTexture(texture), rl.FilterPoint)
	return nil
}

func (raylibTextureBackend) unload(texture skin.Texture) {
	rl.UnloadTexture(toRaylibTexture(texture))
}

// mergedRule is one registry key's resolved declarations.
type mergedRule struct {
	image              string
	hasImage           bool
	noTexture          bool
	slice              int32
	hasSlice           bool
	tint               core.Color
	hasTint            bool
	padding            [4]float32
	hasPadding         bool
	backgroundColor    core.Color
	hasBackgroundColor bool
	radius             float32
	hasRadius          bool
	gradients          [skin.MaxGradientLayers]skin.LinearGradient
	gradientCount      int
	hasImageDecl       bool
	font               string
	hasFont            bool
	italicFont         string
	hasItalicFont      bool
	fontSize           float32
	hasFontSize        bool
	textColor          core.Color
	hasTextColor       bool
}

type cssImage struct {
	name     string
	fileType string
	data     []byte
}

// LoadCSSFile builds and uploads a complete CSS candidate before publication.
// Failure unloads candidate textures and preserves the published CSS layer;
// success swaps layers before unloading the previous owned textures.
func (t *Theme) LoadCSSFile(cssPath, assetDir string) error {
	if t == nil {
		return errors.New("render: nil theme")
	}
	text, err := os.ReadFile(cssPath)
	if err != nil {
		return fmt.Errorf("render: css %q: %w", cssPath, err)
	}
	base := assetDir
	if base == "" {
		base = filepath.Dir(cssPath)
	}
	rules, err := skin.ParseCSS(string(text))
	if err != nil {
		return err
	}
	merged, order := mergeSkinRules(rules)
	if err := t.loadCSSFonts(base, merged, order); err != nil {
		return err
	}
	images, imageOrder, err := loadRuleImages(base, merged, order)
	if err != nil {
		return err
	}
	if (len(imageOrder) > 0 || len(t.ownedSkinTextures) > 0) && !t.ensureTextureBackend().ready() {
		return errors.New("render: window not ready")
	}
	textures, owned, err := t.uploadCSSImages(images, imageOrder)
	if err != nil {
		t.unloadTextures(owned)
		return err
	}
	candidate, err := t.buildCSSRegistry(merged, order, base, textures)
	if err != nil {
		t.unloadTextures(owned)
		return err
	}
	if t.css == nil {
		t.css = skin.NewRegistry()
	}
	oldTextures := t.ownedSkinTextures
	t.css.Replace(candidate)
	t.textRevision++
	t.ownedSkinTextures = owned
	t.unloadTextures(oldTextures)
	return nil
}

// mergeSkinRules groups rules by first key appearance, applies later fields,
// and resolves state entries over their same-kind, same-part normal entry.
func mergeSkinRules(rules []skin.SkinRule) (map[skin.SkinKey]mergedRule, []skin.SkinKey) {
	merged := map[skin.SkinKey]mergedRule{}
	var order []skin.SkinKey
	for _, rule := range rules {
		key := skin.SkinKey{Widget: rule.Kind, Class: rule.Class, Part: rule.Part, State: rule.State}
		entry, seen := merged[key]
		if !seen {
			order = append(order, key)
		}
		mergeRuleImage(&entry, rule)
		mergeRuleStyle(&entry, rule)
		merged[key] = entry
	}
	inheritNormalRules(merged, order)
	return merged, order
}

// mergeRuleImage applies one rule's image, gradient, and explicit none
// declarations. A mixed leading-url stack keeps both layers; URL-only and
// gradient-only declarations still replace the previous background stack.
func mergeRuleImage(entry *mergedRule, rule skin.SkinRule) {
	if rule.HasImage || rule.NoTexture || rule.GradientCount > 0 {
		entry.hasImageDecl = true
	}
	if rule.NoTexture {
		entry.image, entry.hasImage = "", false
		entry.gradients, entry.gradientCount = [skin.MaxGradientLayers]skin.LinearGradient{}, 0
		entry.noTexture = true
		return
	}
	if rule.HasImage {
		entry.image, entry.hasImage = rule.Image, true
		entry.noTexture = false
		if rule.GradientCount == 0 {
			entry.gradients, entry.gradientCount = [skin.MaxGradientLayers]skin.LinearGradient{}, 0
		}
	}
	if rule.GradientCount > 0 {
		entry.gradients, entry.gradientCount = rule.Gradients, rule.GradientCount
		entry.noTexture = false
		if !rule.HasImage {
			entry.image, entry.hasImage = "", false
		}
	}
}

// mergeRuleStyle applies one rule's scalar visual and text declarations.
// Later declarations overwrite earlier ones per field.
func mergeRuleStyle(entry *mergedRule, rule skin.SkinRule) {
	if rule.HasSlice {
		entry.slice, entry.hasSlice = rule.Slice, true
	}
	if rule.HasTint {
		entry.tint, entry.hasTint = rule.Tint, true
	}
	if rule.HasPadding {
		entry.padding, entry.hasPadding = rule.Padding, true
	}
	if rule.HasBackgroundColor {
		entry.backgroundColor, entry.hasBackgroundColor = rule.BackgroundColor, true
	}
	if rule.HasRadius {
		entry.radius, entry.hasRadius = rule.Radius, true
	}
	if rule.HasFont {
		entry.font, entry.hasFont = rule.Font, true
	}
	if rule.HasItalicFont {
		entry.italicFont, entry.hasItalicFont = rule.ItalicFont, true
	}
	if rule.HasFontSize {
		entry.fontSize, entry.hasFontSize = rule.FontSize, true
	}
	if rule.HasTextColor {
		entry.textColor, entry.hasTextColor = rule.TextColor, true
	}
}

// inheritNormalRules fills state fields omitted from an authored rule.
func inheritNormalRules(merged map[skin.SkinKey]mergedRule, order []skin.SkinKey) {
	for _, key := range order {
		if key.State == core.StateNormal {
			continue
		}
		base, ok := merged[skin.SkinKey{Widget: key.Widget, Class: key.Class, Part: key.Part, State: core.StateNormal}]
		if !ok {
			continue
		}
		entry := merged[key]
		inheritNormalVisuals(&entry, base)
		inheritNormalText(&entry, base)
		merged[key] = entry
	}
}

// inheritNormalVisuals copies omitted visual declarations from the normal-state rule.
func inheritNormalVisuals(entry *mergedRule, base mergedRule) {
	if !entry.hasSlice {
		entry.slice, entry.hasSlice = base.slice, base.hasSlice
	}
	if !entry.hasTint {
		entry.tint, entry.hasTint = base.tint, base.hasTint
	}
	if !entry.hasPadding {
		entry.padding, entry.hasPadding = base.padding, base.hasPadding
	}
	if !entry.hasBackgroundColor {
		entry.backgroundColor, entry.hasBackgroundColor = base.backgroundColor, base.hasBackgroundColor
	}
	if !entry.hasRadius {
		entry.radius, entry.hasRadius = base.radius, base.hasRadius
	}
	// An explicit background-image replaces every layer, so states inherit
	// the whole stack only when they declare no image or gradient of their own.
	if !entry.hasImageDecl {
		entry.image, entry.hasImage = base.image, base.hasImage
		entry.gradients, entry.gradientCount = base.gradients, base.gradientCount
		entry.noTexture = base.noTexture
	}
}

// inheritNormalText copies omitted text and font declarations from the normal-state rule.
func inheritNormalText(entry *mergedRule, base mergedRule) {
	if !entry.hasFont {
		entry.font, entry.hasFont = base.font, base.hasFont
	}
	if !entry.hasItalicFont {
		entry.italicFont, entry.hasItalicFont = base.italicFont, base.hasItalicFont
	}
	if !entry.hasFontSize {
		entry.fontSize, entry.hasFontSize = base.fontSize, base.hasFontSize
	}
	if !entry.hasTextColor {
		entry.textColor, entry.hasTextColor = base.textColor, base.hasTextColor
	}
}

// loadRuleImages reads original bytes and decodes each unique source before
// any upload. It returns deterministic first-reference upload order.
func loadRuleImages(base string, merged map[skin.SkinKey]mergedRule, order []skin.SkinKey) (map[string]cssImage, []string, error) {
	images := map[string]cssImage{}
	imageOrder := make([]string, 0)
	for _, key := range order {
		entry := merged[key]
		if entry.hasTint && !entry.hasImage {
			return nil, nil, fmt.Errorf("render: css tint without an image for %v", key)
		}
		if !entry.hasImage {
			continue
		}
		resolved := filepath.Join(base, entry.image)
		if _, ok := images[resolved]; ok {
			continue
		}
		data, err := os.ReadFile(resolved)
		if err != nil {
			return nil, nil, fmt.Errorf("render: css image %q: %w", entry.image, err)
		}
		_, format, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, nil, fmt.Errorf("render: css image %q: %w", entry.image, err)
		}
		fileType := filepath.Ext(resolved)
		if fileType == "" {
			fileType = "." + format
		}
		images[resolved] = cssImage{name: entry.image, fileType: fileType, data: data}
		imageOrder = append(imageOrder, resolved)
	}
	return images, imageOrder, nil
}

// uploadCSSImages uploads and filters each validated source exactly once.
// Its returned owned slice includes every successful upload, including one
// whose later filtering fails, so the caller can roll all candidates back.
func (t *Theme) uploadCSSImages(images map[string]cssImage, order []string) (map[string]skin.Texture, []skin.Texture, error) {
	backend := t.ensureTextureBackend()
	if len(order) > 0 && !backend.ready() {
		return nil, nil, errors.New("render: window not ready")
	}
	textures := make(map[string]skin.Texture, len(order))
	owned := make([]skin.Texture, 0, len(order))
	for _, path := range order {
		asset := images[path]
		texture, err := backend.upload(asset.fileType, asset.data)
		if err != nil || texture.ID == 0 {
			if err == nil {
				err = errors.New("empty texture handle")
			}
			return nil, owned, fmt.Errorf("render: css image %q failed to upload: %w", asset.name, err)
		}
		owned = append(owned, texture)
		if err := backend.setFilter(texture); err != nil {
			return nil, owned, fmt.Errorf("render: css image %q failed to set filter: %w", asset.name, err)
		}
		textures[path] = texture
	}
	return textures, owned, nil
}

// buildCSSRegistry materializes every CSS descriptor from CSS declarations
// alone. Programmatic descriptors are never seeded here so every pixel
// requires a CSS->texture pathway.
func (t *Theme) buildCSSRegistry(merged map[skin.SkinKey]mergedRule, order []skin.SkinKey, base string, textures map[string]skin.Texture) (*skin.Registry, error) {
	candidate := skin.NewRegistry()
	for _, key := range order {
		descriptor, err := t.buildCSSDescriptor(key, merged[key], base, textures)
		if err != nil {
			return nil, err
		}
		candidate.Set(key, descriptor)
	}
	return candidate, nil
}

// buildCSSDescriptor applies one merged CSS rule starting from a zero descriptor.
func (t *Theme) buildCSSDescriptor(key skin.SkinKey, entry mergedRule, base string, textures map[string]skin.Texture) (skin.SkinDescriptor, error) {
	descriptor := skin.SkinDescriptor{}
	if entry.noTexture {
		descriptor.NoTexture = true
	}
	if entry.hasImage {
		texture, ok := textures[filepath.Join(base, entry.image)]
		if !ok {
			return skin.SkinDescriptor{}, fmt.Errorf("render: css image %q was not uploaded", entry.image)
		}
		descriptor.Texture = texture
		descriptor.AtlasRegion = core.Rect{W: float32(texture.Width), H: float32(texture.Height)}
		descriptor.HasTexture = true
		descriptor.Tint = cssTint(entry)
	}
	applyCSSBox(&descriptor, key, entry)
	if entry.hasBackgroundColor {
		descriptor.BackgroundColor = entry.backgroundColor
		descriptor.HasBackgroundColor = true
	}
	if entry.hasRadius {
		descriptor.Radius = entry.radius
		descriptor.HasRadius = true
	}
	if entry.gradientCount > 0 {
		descriptor.Gradients = entry.gradients
		descriptor.GradientCount = entry.gradientCount
	}
	applyDescriptorText(&descriptor, entry)
	return descriptor, nil
}

// applyCSSBox populates nine-patch, three-patch, and padding geometry from
// merged rule declarations.
func applyCSSBox(descriptor *skin.SkinDescriptor, key skin.SkinKey, entry mergedRule) {
	if (key.Part == skin.PartBorder || key.Part == skin.PartPopupBorder) && entry.hasSlice {
		descriptor.NinePatch.Left, descriptor.NinePatch.Top = entry.slice, entry.slice
		descriptor.NinePatch.Right, descriptor.NinePatch.Bottom = entry.slice, entry.slice
		descriptor.HasNinePatch = true
		descriptor.CenterFill = false
	} else if (key.Widget == core.WidgetScrollPanel && (key.Part == skin.PartTrack || key.Part == skin.PartThumb)) && entry.hasSlice {
		descriptor.ThreePatch.Top = entry.slice
		descriptor.ThreePatch.Bottom = entry.slice
		descriptor.HasThreePatch = true
	}
	if entry.hasPadding {
		descriptor.PaddingTop, descriptor.PaddingRight = entry.padding[0], entry.padding[1]
		descriptor.PaddingBottom, descriptor.PaddingLeft = entry.padding[2], entry.padding[3]
		descriptor.HasPadding = true
	}
}

// applyDescriptorText populates optional text styling from merged rule declarations.
func applyDescriptorText(descriptor *skin.SkinDescriptor, entry mergedRule) {
	if entry.hasTextColor {
		descriptor.TextColor = entry.textColor
		descriptor.HasTextColor = true
	}
	if entry.hasFontSize {
		descriptor.FontSize = entry.fontSize
		descriptor.HasFontSize = true
	}
	if entry.hasFont {
		descriptor.Font = entry.font
		descriptor.HasFont = true
	}
	if entry.hasItalicFont {
		descriptor.ItalicFont = entry.italicFont
		descriptor.HasItalicFont = true
	}
}

// loadCSSFonts loads any TTF/OTF font files authored in CSS font-family or font-italic-family rules.
func (t *Theme) loadCSSFonts(base string, merged map[skin.SkinKey]mergedRule, order []skin.SkinKey) error {
	for _, key := range order {
		entry := merged[key]
		if entry.hasFont && isFontFilePath(entry.font) {
			path := entry.font
			if !filepath.IsAbs(path) {
				path = filepath.Join(base, path)
			}
			if err := t.LoadFont(path); err != nil {
				return fmt.Errorf("render: load font %q: %w", entry.font, err)
			}
		}
		if entry.hasItalicFont && isFontFilePath(entry.italicFont) {
			path := entry.italicFont
			if !filepath.IsAbs(path) {
				path = filepath.Join(base, path)
			}
			if err := t.LoadItalicFont(path); err != nil {
				return fmt.Errorf("render: load italic font %q: %w", entry.italicFont, err)
			}
		}
	}
	return nil
}

// isFontFilePath reports whether value looks like a file path to a font rather than a font face name.
func isFontFilePath(value string) bool {
	ext := strings.ToLower(filepath.Ext(value))
	return ext == ".ttf" || ext == ".otf" || strings.Contains(value, "/") || strings.Contains(value, "\\")
}

// cssTint returns exact authored tint or explicit no-tint opaque white.
func cssTint(entry mergedRule) core.Color {
	if entry.hasTint {
		return entry.tint
	}
	return core.Color{R: 255, G: 255, B: 255, A: 255}
}

func (t *Theme) ensureTextureBackend() textureBackend {
	if t.textureBackend == nil {
		t.textureBackend = raylibTextureBackend{}
	}
	return t.textureBackend
}

// unloadTextures releases each texture through the current backend.
func (t *Theme) unloadTextures(textures []skin.Texture) {
	if len(textures) == 0 {
		return
	}
	backend := t.ensureTextureBackend()
	for _, texture := range textures {
		backend.unload(texture)
	}
}
