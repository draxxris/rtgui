// CSS file skinning: upload half of Theme.LoadCSSFile. Parse and mapping live
// headless in skin/css.go; this file resolves asset files, bakes tints,
// uploads each unique file+tint once, and writes registry descriptors.
//
// Application is two-phase and atomic: parse, merge, and validate everything
// (including file reads and PNG decodes) before the first texture upload or
// SetSkinPart. Headless callers get a loud early error instead of a
// half-registered theme, since upload needs a GL context.
package render

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	rl "github.com/gen2brain/raylib-go/raylib"
	"rtgui/core"
	"rtgui/skin"
)

// mergedRule is one registry key's resolved declarations: later file rules win
// per field, then states inherit unset siblings from their kind+part normal.
type mergedRule struct {
	image      string
	hasImage   bool
	slice      int32
	hasSlice   bool
	tint       core.Color
	hasTint    bool
	padding    [4]float32
	hasPadding bool
}

// LoadCSSFile skins the theme from a LOOK-only CSS file. Asset paths resolve
// against assetDir, or the CSS file's own directory when assetDir is empty.
// Programmatic registrations stay underneath: CSS patches each key's
// descriptor, so properties a file never mentions keep their current values.
func (t *Theme) LoadCSSFile(cssPath, assetDir string) error {
	if t == nil {
		return errors.New("render: nil theme")
	}
	if !rl.IsWindowReady() {
		return errors.New("render: window not ready")
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
	images, err := loadRuleImages(base, merged, order)
	if err != nil {
		return err
	}
	cache := map[string]skin.Texture{}
	for _, key := range order {
		descriptor, err := t.buildCSSDescriptor(key, merged[key], base, images, cache)
		if err != nil {
			return err
		}
		if err := t.SetSkinPart(key, descriptor); err != nil {
			return err
		}
	}
	return nil
}

// mergeSkinRules groups rules by key in first-appearance order with later
// rules winning per field, then resolves each state key over its kind+part
// normal key so tint-only or padding-only state rules inherit siblings.
func mergeSkinRules(rules []skin.SkinRule) (map[skin.SkinKey]mergedRule, []skin.SkinKey) {
	merged := map[skin.SkinKey]mergedRule{}
	var order []skin.SkinKey
	for _, rule := range rules {
		key := skin.SkinKey{Widget: rule.Kind, Part: rule.Part, State: rule.State}
		entry, seen := merged[key]
		if !seen {
			order = append(order, key)
		}
		if rule.HasImage {
			entry.image, entry.hasImage = rule.Image, true
		}
		if rule.HasSlice {
			entry.slice, entry.hasSlice = rule.Slice, true
		}
		if rule.HasTint {
			entry.tint, entry.hasTint = rule.Tint, true
		}
		if rule.HasPadding {
			entry.padding, entry.hasPadding = rule.Padding, true
		}
		merged[key] = entry
	}
	for _, key := range order {
		if key.State == core.StateNormal {
			continue
		}
		base, ok := merged[skin.SkinKey{Widget: key.Widget, Part: key.Part, State: core.StateNormal}]
		if !ok {
			continue
		}
		entry := merged[key]
		if !entry.hasImage {
			entry.image, entry.hasImage = base.image, base.hasImage
		}
		if !entry.hasSlice {
			entry.slice, entry.hasSlice = base.slice, base.hasSlice
		}
		if !entry.hasTint {
			entry.tint, entry.hasTint = base.tint, base.hasTint
		}
		if !entry.hasPadding {
			entry.padding, entry.hasPadding = base.padding, base.hasPadding
		}
		merged[key] = entry
	}
	return merged, order
}

// loadRuleImages reads and decodes every referenced image before any upload,
// so missing or corrupt files fail before the theme is touched.
func loadRuleImages(base string, merged map[skin.SkinKey]mergedRule, order []skin.SkinKey) (map[string]image.Image, error) {
	images := map[string]image.Image{}
	for _, key := range order {
		entry := merged[key]
		if entry.hasTint && !entry.hasImage {
			return nil, fmt.Errorf("render: css tint without an image for %v", key)
		}
		if !entry.hasImage {
			continue
		}
		resolved := filepath.Join(base, entry.image)
		if _, ok := images[resolved]; ok {
			continue
		}
		file, err := os.Open(resolved)
		if err != nil {
			return nil, fmt.Errorf("render: css image %q: %w", entry.image, err)
		}
		img, _, err := image.Decode(file)
		_ = file.Close()
		if err != nil {
			return nil, fmt.Errorf("render: css image %q: %w", entry.image, err)
		}
		images[resolved] = img
	}
	return images, nil
}

// buildCSSDescriptor overlays one merged key onto the current registry
// descriptor and uploads its texture (cached per file+tint). Descriptor Tint
// stays white on baked textures so nothing tints twice.
func (t *Theme) buildCSSDescriptor(key skin.SkinKey, entry mergedRule, base string, images map[string]image.Image, cache map[string]skin.Texture) (skin.SkinDescriptor, error) {
	descriptor, _ := t.GetSkinPart(key)
	if !entry.hasImage && !entry.hasSlice && !entry.hasPadding {
		return descriptor, nil
	}
	if entry.hasImage {
		texture, err := uploadCSSImage(entry, base, images, cache)
		if err != nil {
			return skin.SkinDescriptor{}, err
		}
		region := core.Rect{W: float32(texture.Width), H: float32(texture.Height)}
		descriptor.Texture, descriptor.AtlasRegion = texture, region
		descriptor.HasTexture = true
		descriptor.Tint, descriptor.Alpha = core.Color{R: 255, G: 255, B: 255, A: 255}, 1
		if key.Part == skin.PartBorder {
			descriptor.NinePatch.Source = region
		}
	}
	if key.Part == skin.PartBorder && entry.hasSlice {
		descriptor.NinePatch.Left, descriptor.NinePatch.Top = entry.slice, entry.slice
		descriptor.NinePatch.Right, descriptor.NinePatch.Bottom = entry.slice, entry.slice
		descriptor.HasNinePatch = true
		descriptor.CenterFill = false
	}
	if entry.hasPadding {
		descriptor.PaddingTop, descriptor.PaddingRight = entry.padding[0], entry.padding[1]
		descriptor.PaddingBottom, descriptor.PaddingLeft = entry.padding[2], entry.padding[3]
	}
	return descriptor, nil
}

// uploadCSSImage encodes the (optionally tint-baked) image to PNG bytes and
// uploads once per file+tint pair.
func uploadCSSImage(entry mergedRule, base string, images map[string]image.Image, cache map[string]skin.Texture) (skin.Texture, error) {
	resolved := filepath.Join(base, entry.image)
	cacheKey := resolved + "\x00" + tintKey(entry)
	if texture, ok := cache[cacheKey]; ok {
		return texture, nil
	}
	img := images[resolved]
	var encoded bytes.Buffer
	pixels := img
	if entry.hasTint {
		pixels = skin.TintImage(img, entry.tint)
	}
	if err := png.Encode(&encoded, pixels); err != nil {
		return skin.Texture{}, fmt.Errorf("render: css image %q: %w", entry.image, err)
	}
	raw := encoded.Bytes()
	loaded := rl.LoadImageFromMemory(".png", raw, int32(len(raw)))
	if loaded == nil {
		return skin.Texture{}, fmt.Errorf("render: css image %q failed to load", entry.image)
	}
	defer rl.UnloadImage(loaded)
	uploaded := rl.LoadTextureFromImage(loaded)
	if uploaded.ID == 0 {
		return skin.Texture{}, fmt.Errorf("render: css image %q failed to upload", entry.image)
	}
	rl.SetTextureFilter(uploaded, rl.FilterPoint)
	texture := fromRaylibTexture(uploaded)
	cache[cacheKey] = texture
	return texture, nil
}

// tintKey renders a tint cache-safe: white for untinted entries.
func tintKey(entry mergedRule) string {
	if !entry.hasTint {
		return "none"
	}
	return fmt.Sprintf("%02x%02x%02x%02x", entry.tint.R, entry.tint.G, entry.tint.B, entry.tint.A)
}
