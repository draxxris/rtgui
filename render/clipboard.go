package render

import rl "github.com/gen2brain/raylib-go/raylib"

// IsWindowReady reports whether a graphics context exists for drawing and
// system clipboard access. It keeps raylib confined to render.
func IsWindowReady() bool {
	return rl.IsWindowReady()
}

// GetClipboardText returns the system clipboard text when a window exists,
// or "" headless. UI falls back to its in-memory clipboard headless.
func GetClipboardText() string {
	if !rl.IsWindowReady() {
		return ""
	}
	return rl.GetClipboardText()
}

// SetClipboardText stores text in the system clipboard when a window exists.
// Headless calls are no-ops; UI always mirrors into its in-memory clipboard.
func SetClipboardText(text string) {
	if !rl.IsWindowReady() {
		return
	}
	rl.SetClipboardText(text)
}
